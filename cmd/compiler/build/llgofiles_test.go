/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"github.com/xgo-dev/llgo/types"
	"go/constant"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func llgoFilesTestPackage(t *testing.T, dir, value string) *packages.Package {
	t.Helper()
	goFile := filepath.Join(dir, "package.go")
	if err := os.WriteFile(goFile, []byte("package fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	typesPkg := types.NewPackage("example.com/fixture", "fixture")
	typesPkg.Scope().Insert(types.NewConst(token.NoPos, typesPkg, "LLGoFiles", types.Typ[types.UntypedString], constant.MakeString(value)))
	return &packages.Package{
		ID:         "example.com/fixture",
		PkgPath:    "example.com/fixture",
		GoFiles:    []string{goFile},
		ExportFile: filepath.Join(dir, "fixture.a"),
		Types:      typesPkg,
	}
}

func TestLLGoFileInputsResolvePathsAndFlags(t *testing.T) {
	dir := t.TempDir()
	pkg := llgoFilesTestPackage(t, dir, "$LLGO_TEST_CFLAGS: wrap.c; ; wrap.S")
	ctx := &context{commands: commandEnv{environ: []string{"LLGO_TEST_CFLAGS=-DVALUE=1"}}}
	inputs, err := llgoPkgFileInputs(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 {
		t.Fatalf("LLGoFiles inputs = %#v, want two files", inputs)
	}
	for i, name := range []string{"wrap.c", "wrap.S"} {
		if got, want := inputs[i].path, filepath.Join(dir, name); got != want {
			t.Errorf("input %d path = %q, want %q", i, got, want)
		}
		if got, want := inputs[i].compilerArgs, []string{"-DVALUE=1"}; !reflect.DeepEqual(got, want) {
			t.Errorf("input %d compiler args = %q, want %q", i, got, want)
		}
	}
	pkg.GoFiles = nil
	inputs, err = llgoPkgFileInputs(&context{}, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 0 {
		t.Fatalf("LLGoFiles inputs without a package source directory = %#v, want none", inputs)
	}
}

func TestLLGoFileInputsFrozenCache(t *testing.T) {
	pkg := llgoFilesTestPackage(t, t.TempDir(), "wrap.c")
	ctx := &context{llgoFilesFrozen: true}
	if _, err := llgoPkgFileInputs(ctx, pkg); err == nil {
		t.Fatal("frozen LLGoFiles cache miss unexpectedly succeeded")
	}

	want := []llgoFileInput{{path: "cached.c"}}
	ctx.llgoFilesCache = map[*packages.Package][]llgoFileInput{pkg: want}
	got, err := llgoPkgFileInputs(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen LLGoFiles cache hit = %#v, want %#v", got, want)
	}

	other := llgoFilesTestPackage(t, t.TempDir(), "other.c")
	if _, err := llgoPkgFileInputs(ctx, other); err == nil {
		t.Fatal("nonempty frozen LLGoFiles cache miss unexpectedly succeeded")
	}
}

func TestLLGoFilesFingerprintUsesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wrap.c")
	if err := os.WriteFile(path, []byte("int value = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := llgoFilesTestPackage(t, dir, "wrap.c")
	fingerprint := func(ctx *context) (string, llgoFileDigest) {
		manifest := newManifestBuilder()
		if err := ctx.collectPackageInputs(manifest, &aPackage{Package: pkg}); err != nil {
			t.Fatal(err)
		}
		if len(manifest.pkg.LLGoFiles) != 1 {
			t.Fatalf("manifest LLGoFiles = %#v, want one file", manifest.pkg.LLGoFiles)
		}
		return manifest.Fingerprint(), manifest.pkg.LLGoFiles[0]
	}

	before, beforeFile := fingerprint(&context{buildConf: &Config{}})
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("int value = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Preserve size and modification time to prove that content, rather than
	// the metadata used for ordinary Go inputs, invalidates the package cache.
	if err := os.Chtimes(path, time.Unix(0, info.ModTime().UnixNano()), time.Unix(0, info.ModTime().UnixNano())); err != nil {
		t.Fatal(err)
	}
	after, afterFile := fingerprint(&context{buildConf: &Config{}})
	if before == after {
		t.Fatal("LLGoFiles content change did not change the package fingerprint")
	}
	if beforeFile.PreprocessedHash == afterFile.PreprocessedHash {
		t.Fatal("LLGoFiles content change did not change its preprocessed hash")
	}
}

func TestLLGoFilesFingerprintUsesIncludedContent(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "wrap.c")
	header := filepath.Join(dir, "value.h")
	if err := os.WriteFile(source, []byte("#include \"value.h\"\nint value = VALUE;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(header, []byte("#define VALUE 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := llgoFilesTestPackage(t, dir, "wrap.c")
	fingerprint := func() (string, llgoFileDigest) {
		ctx := &context{buildConf: &Config{}}
		manifest := newManifestBuilder()
		if err := ctx.collectPackageInputs(manifest, &aPackage{Package: pkg}); err != nil {
			t.Fatal(err)
		}
		return manifest.Fingerprint(), manifest.pkg.LLGoFiles[0]
	}

	before, beforeFile := fingerprint()
	info, err := os.Stat(header)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(header, []byte("#define VALUE 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(header, time.Unix(0, info.ModTime().UnixNano()), time.Unix(0, info.ModTime().UnixNano())); err != nil {
		t.Fatal(err)
	}
	after, afterFile := fingerprint()
	if before == after {
		t.Fatal("included LLGoFiles content change did not change the package fingerprint")
	}
	if beforeFile.PreprocessedHash == afterFile.PreprocessedHash {
		t.Fatal("included LLGoFiles content change did not change its preprocessed hash")
	}
}

func TestModeGenFingerprintSkipsLLGoFiles(t *testing.T) {
	dir := t.TempDir()
	pkg := llgoFilesTestPackage(t, dir, "missing.c")
	ctx := &context{mode: ModeGen, buildConf: &Config{Mode: ModeGen}}
	manifest := newManifestBuilder()
	if err := ctx.collectPackageInputs(manifest, &aPackage{Package: pkg}); err != nil {
		t.Fatalf("ModeGen fingerprint required an unused LLGoFiles compiler: %v", err)
	}
	if manifest.pkg.LLGoFiles != nil {
		t.Fatalf("ModeGen LLGoFiles manifest = %#v, want nil", manifest.pkg.LLGoFiles)
	}
}

func TestLLGoFileOutputsAreProcessPrivate(t *testing.T) {
	first, err := genLLGoFileOutput("wrap.c", ".o")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(first)
	second, err := genLLGoFileOutput("wrap.c", ".o")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(second)
	if first == second {
		t.Fatalf("two LLGoFiles compilations selected the same object path %q", first)
	}
}

func llgoFilesBuildContext(dir string, genLL bool) *context {
	return &context{
		conf:      &packages.Config{},
		buildConf: &Config{GenLL: genLL},
		commands: commandEnv{
			dir:     dir,
			environ: os.Environ(),
		},
	}
}

func TestLLGoFileCompilationLifecycle(t *testing.T) {
	t.Run("GenLL success", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "wrap.c")
		if err := os.WriteFile(source, []byte("int answer(void) { return 42; }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		exportFile := filepath.Join(dir, "fixture.a")
		object, err := clFile(llgoFilesBuildContext(dir, true), nil, source, exportFile, "example.com/fixture", false)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(object)
		published := exportFile + filepath.Base(source) + ".ll"
		defer os.Remove(published)
		if info, err := os.Stat(published); err != nil {
			t.Fatal(err)
		} else {
			wantMode := os.FileMode(0o644)
			if runtime.GOOS == "windows" {
				// Windows records only the read-only attribute, so os.Stat reports
				// every writable regular file as 0666 even after Chmod(0644).
				wantMode = 0o666
			}
			if got := info.Mode().Perm(); got != wantMode {
				t.Fatalf("published LLVM IR mode = %o, want %o", got, wantMode)
			}
		}
	})

	t.Run("temporary output errors", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(blocker, []byte("block"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
			t.Setenv(key, blocker)
		}
		for _, tt := range []struct {
			name   string
			genLL  bool
			needle string
		}{
			{name: "LLVM IR", genLL: true, needle: "temporary LLVM IR output"},
			{name: "object", needle: "temporary object output"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				ctx := llgoFilesBuildContext("", tt.genLL)
				if _, err := clFile(ctx, nil, "wrap.c", "fixture.a", "example.com/fixture", false); err == nil || !strings.Contains(err.Error(), tt.needle) {
					t.Fatalf("clFile error = %v, want %q", err, tt.needle)
				}
			})
		}
	})

	t.Run("LLVM IR compile error", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "broken.c")
		if err := os.WriteFile(source, []byte("this is not C;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := clFile(llgoFilesBuildContext(dir, true), nil, source, filepath.Join(dir, "fixture.a"), "example.com/fixture", false); err == nil || !strings.Contains(err.Error(), "to LLVM IR") {
			t.Fatalf("clFile error = %v, want LLVM IR compile error", err)
		}
	})

	t.Run("LLVM IR publish error", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "wrap.c")
		if err := os.WriteFile(source, []byte("int answer(void) { return 42; }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		blocker := filepath.Join(dir, "not-a-directory")
		if err := os.WriteFile(blocker, []byte("block"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := clFile(llgoFilesBuildContext(dir, true), nil, source, blocker+string(os.PathSeparator), "example.com/fixture", false); err == nil || !strings.Contains(err.Error(), "publish LLVM IR") {
			t.Fatalf("clFile error = %v, want publish error", err)
		}
	})

	t.Run("object compile error", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "broken.c")
		if err := os.WriteFile(source, []byte("this is not C;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := clFile(llgoFilesBuildContext(dir, false), nil, source, filepath.Join(dir, "fixture.a"), "example.com/fixture", false); err == nil || !strings.Contains(err.Error(), "compile ") {
			t.Fatalf("clFile error = %v, want object compile error", err)
		}
	})
}

func TestConcatPkgLinkFilesCleansPartialOutputs(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.c")
	if err := os.WriteFile(valid, []byte("int answer(void) { return 42; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := llgoFilesTestPackage(t, dir, "valid.c;missing.c")
	ctx := llgoFilesBuildContext(dir, false)
	ctx.llgoFilesCache = map[*packages.Package][]llgoFileInput{
		pkg: {
			{path: valid},
			{path: filepath.Join(dir, "missing.c")},
		},
	}
	parts, err := concatPkgLinkFiles(ctx, pkg, false)
	if err == nil {
		t.Fatal("concatPkgLinkFiles unexpectedly succeeded")
	}
	if len(parts) != 1 {
		t.Fatalf("partial outputs = %v, want one compiled object", parts)
	}
	if _, statErr := os.Stat(parts[0]); !os.IsNotExist(statErr) {
		t.Fatalf("partial object was not removed: %v", statErr)
	}

	frozen := &context{llgoFilesFrozen: true}
	if _, err := concatPkgLinkFiles(frozen, pkg, false); err == nil {
		t.Fatal("concatPkgLinkFiles accepted an unprepared frozen cache")
	}
}

func TestCollectPackageInputsReportsLLGoFilesErrors(t *testing.T) {
	t.Run("primary list", func(t *testing.T) {
		pkg := llgoFilesTestPackage(t, t.TempDir(), "wrap.c")
		ctx := &context{buildConf: &Config{}, llgoFilesFrozen: true}
		err := ctx.collectPackageInputs(newManifestBuilder(), &aPackage{Package: pkg})
		if err == nil || !strings.Contains(err.Error(), "list LLGoFiles") {
			t.Fatalf("collectPackageInputs error = %v, want primary list error", err)
		}
	})

	t.Run("alternate list", func(t *testing.T) {
		pkg := llgoFilesTestPackage(t, t.TempDir(), "wrap.c")
		alt := llgoFilesTestPackage(t, t.TempDir(), "alt.c")
		ctx := &context{
			buildConf:       &Config{},
			llgoFilesCache:  map[*packages.Package][]llgoFileInput{pkg: nil},
			llgoFilesFrozen: true,
		}
		err := ctx.collectPackageInputs(newManifestBuilder(), &aPackage{
			Package: pkg,
			AltPkg:  &packages.Cached{Package: alt},
		})
		if err == nil || !strings.Contains(err.Error(), "list alternate LLGoFiles") {
			t.Fatalf("collectPackageInputs error = %v, want alternate list error", err)
		}
	})

	t.Run("digest", func(t *testing.T) {
		pkg := llgoFilesTestPackage(t, t.TempDir(), "missing.c")
		ctx := &context{buildConf: &Config{}}
		err := ctx.collectPackageInputs(newManifestBuilder(), &aPackage{Package: pkg})
		if err == nil || !strings.Contains(err.Error(), "digest LLGoFiles") {
			t.Fatalf("collectPackageInputs error = %v, want digest error", err)
		}
	})
}

func TestCleanupTemporaryObjFilesPreservesOtherMembers(t *testing.T) {
	dir := t.TempDir()
	temporary := filepath.Join(dir, "temporary.o")
	stable := filepath.Join(dir, "stable.o")
	for _, path := range []string{temporary, stable} {
		if err := os.WriteFile(path, []byte("object"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pkg := &aPackage{ObjFiles: []string{temporary, stable}, tempObjFiles: []string{temporary}}
	pkg.cleanupTemporaryObjFiles()
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatalf("temporary object still exists: %v", err)
	}
	if _, err := os.Stat(stable); err != nil {
		t.Fatalf("stable archive member was removed: %v", err)
	}
	if pkg.tempObjFiles != nil {
		t.Fatalf("temporary object tracking = %#v, want nil", pkg.tempObjFiles)
	}
}

func TestDigestLLGoFileInputsEdges(t *testing.T) {
	ctx := &context{buildConf: &Config{}}
	if digests, err := digestLLGoFileInputs(ctx, nil, nil); err != nil || digests != nil {
		t.Fatalf("empty digests = %#v, %v; want nil, nil", digests, err)
	}

	path := filepath.Join(t.TempDir(), "overlay.c")
	if err := os.WriteFile(path, []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	digests, err := digestLLGoFileInputs(ctx, []llgoFileInput{
		{path: path, compilerArgs: []string{"-DZ=1"}},
		{path: path, compilerArgs: []string{"-DA=1"}},
	}, map[string][]byte{path: []byte("overlay content")})
	if err != nil {
		t.Fatal(err)
	}
	if len(digests) != 2 || !reflect.DeepEqual(digests[0].CompilerArgs, []string{"-DA=1"}) ||
		digests[0].OverlayHash != digestBytes([]byte("overlay content")) {
		t.Fatalf("overlay digests = %#v", digests)
	}

	missing := filepath.Join(t.TempDir(), "missing.c")
	if _, err := digestLLGoFileInputs(ctx, []llgoFileInput{{path: missing}}, nil); err == nil {
		t.Fatal("digesting a missing LLGoFiles input unexpectedly succeeded")
	}
}

func TestDigestLLGoFileInputsMemoizesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cached.c")
	if err := os.WriteFile(path, []byte("int value = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{}}
	inputs := []llgoFileInput{{path: path}}
	first, err := digestLLGoFileInputs(ctx, inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	second, err := digestLLGoFileInputs(ctx, inputs, nil)
	if err != nil {
		t.Fatalf("memoized digest re-read removed input: %v", err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("memoized digest = %#v, want %#v", second, first)
	}
}
