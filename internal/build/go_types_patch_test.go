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
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
)

func TestRuntimePatchGoTypesOverlayAndSizes(t *testing.T) {
	goroot := runtime.GOROOT()
	overlay, patchFiles, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), goroot, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatalf("buildSourcePatchOverlayForGOROOT: %v", err)
	}
	if len(patchFiles["go/types"]) == 0 {
		t.Fatal("expected go/types patch files in overlay")
	}

	// Verify the patched sizes.go replaces SizesFor and stdSizes in GOROOT go/types/sizes.go.
	stdSizesPath := filepath.Join(goroot, "src", "go", "types", "sizes.go")
	patchedStd, ok := overlay[stdSizesPath]
	if !ok {
		t.Fatalf("expected filtered %s in overlay", stdSizesPath)
	}
	if strings.Contains(string(patchedStd), "\nfunc SizesFor(") {
		t.Fatalf("original SizesFor was not commented out in patched %s", stdSizesPath)
	}

	// Parse and type-check the entire patched GOROOT go/types package with the
	// injected sizes.go patch to prove zero symbol/type mismatches against GOROOT.
	goTypesDir := filepath.Join(goroot, "src", "go", "types")
	entries, err := os.ReadDir(goTypesDir)
	if err != nil {
		t.Fatalf("read %s: %v", goTypesDir, err)
	}
	bctx := build.Default
	fset := token.NewFileSet()
	var parsed []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		match, err := bctx.MatchFile(goTypesDir, name)
		if err != nil || !match {
			continue
		}
		full := filepath.Join(goTypesDir, name)
		var src []byte
		if ov, ok := overlay[full]; ok {
			src = ov
		} else {
			src, err = os.ReadFile(full)
			if err != nil {
				t.Fatalf("read %s: %v", full, err)
			}
		}
		f, err := parser.ParseFile(fset, full, src, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", full, err)
		}
		parsed = append(parsed, f)
	}
	for _, patchSrcPath := range patchFiles["go/types"] {
		injectedPath := filepath.Join(goTypesDir, "z_llgo_patch_"+filepath.Base(patchSrcPath))
		injectedSrc, ok := overlay[injectedPath]
		if !ok {
			t.Fatalf("expected injected %s in overlay", injectedPath)
		}
		f, err := parser.ParseFile(fset, injectedPath, injectedSrc, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse injected %s failed: %v", injectedPath, err)
		}
		parsed = append(parsed, f)
	}
	conf := gotypes.Config{Importer: importer.Default()}
	if _, err := conf.Check("go/types", fset, parsed, nil); err != nil {
		t.Fatalf("type-check patched go/types failed: %v", err)
	}

	// Cross-verify our llgo/types reference Sizes contract matches the patched rules across arches.
	structsPkg := gotypes.NewPackage("structs", "structs")
	hostMarker := gotypes.NewNamed(gotypes.NewTypeName(0, structsPkg, "HostLayout", nil), gotypes.NewStruct(nil, nil), nil)
	hostStruct := gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, nil, "_", hostMarker, false),
		gotypes.NewField(0, nil, "b", gotypes.Typ[gotypes.Byte], false),
		gotypes.NewField(0, nil, "v", gotypes.Typ[gotypes.Int64], false),
	}, nil)
	fnSig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
	tailZero := gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, nil, "x", gotypes.Typ[gotypes.Int32], false),
		gotypes.NewField(0, nil, "z", gotypes.NewStruct(nil, nil), false),
	}, nil)

	for _, tc := range []struct {
		goos, goarch   string
		wasmProfile    string
		wantFnSize     int64
		wantHostSize   int64
		wantHostAlign  int64
		wantTailZeroSz int64
	}{
		{"linux", "amd64", "", 16, 16, 8, 8},
		{"darwin", "arm64", "", 16, 16, 8, 8},
		{"linux", "arm", "", 8, 16, 8, 8},
		{"linux", "386", "", 8, 12, 4, 8},
		{"wasip1", "wasm", "", 8, 16, 8, 8},
		{"wasip1", "wasm", "w32", 16, 16, 8, 8},
	} {
		ctx := lltypes.NewContextWithConfig(lltypes.ContextConfig{
			Spec: target.ResolveSpec(target.Spec{GOOS: tc.goos, GOARCH: tc.goarch, WasmProfile: tc.wasmProfile}),
		})
		sz := ctx.GoSizes()
		if got := sz.Sizeof(fnSig); got != tc.wantFnSize {
			t.Errorf("%s/%s Sizeof(func()) = %d, want %d", tc.goos, tc.goarch, got, tc.wantFnSize)
		}
		if got := sz.Sizeof(hostStruct); got != tc.wantHostSize {
			t.Errorf("%s/%s Sizeof(HostLayout struct) = %d, want %d", tc.goos, tc.goarch, got, tc.wantHostSize)
		}
		if got := sz.Alignof(hostStruct); got != tc.wantHostAlign {
			t.Errorf("%s/%s Alignof(HostLayout struct) = %d, want %d", tc.goos, tc.goarch, got, tc.wantHostAlign)
		}
		if got := sz.Sizeof(tailZero); got != tc.wantTailZeroSz {
			t.Errorf("%s/%s Sizeof(tailZero) = %d, want %d", tc.goos, tc.goarch, got, tc.wantTailZeroSz)
		}
	}
	_ = ast.NewIdent("ok")
}

func TestParseMakeDepfileAndLinkSnapshotSidecar(t *testing.T) {
	dep := "llgo: /tmp/foo.c \\\n  /usr/include/a\\ b.h /tmp/bar.h\n"
	got := parseMakeDepfile(dep)
	if len(got) != 3 || got[0] != "/tmp/foo.c" || got[1] != "/usr/include/a b.h" || got[2] != "/tmp/bar.h" {
		t.Fatalf("parseMakeDepfile = %#v", got)
	}

	dir := t.TempDir()
	snapPath := filepath.Join(dir, "pkg.linksnap")
	pkg := &aPackage{
		Package: &packages.Package{PkgPath: "example.com/demo", Name: "demo"},
		NeedRt:  true,
		linkSnapshot: &packageLinkSnapshot{
			needAbiInit:     2,
			hasLocalExports: true,
			methodByIndex:   map[int]none{0: {}, 3: {}},
			methodByName:    map[string]none{"Read": {}},
			abiSymbols:      map[string]none{"_llgo_demo.T": {}},
		},
	}
	c := &context{}
	if err := c.writeLinkSnapshotSidecar(snapPath, pkg); err != nil {
		t.Fatalf("writeLinkSnapshotSidecar: %v", err)
	}
	loaded, err := c.readLinkSnapshotSidecar(snapPath)
	if err != nil {
		t.Fatalf("readLinkSnapshotSidecar: %v", err)
	}
	if loaded.needAbiInit != 2 || !loaded.hasLocalExports || len(loaded.methodByIndex) != 2 || len(loaded.methodByName) != 1 || len(loaded.abiSymbols) != 1 {
		t.Fatalf("loaded link snapshot mismatch: %+v", loaded)
	}
	if _, ok := loaded.methodByIndex[3]; !ok {
		t.Fatal("missing methodByIndex[3]")
	}
	if _, ok := loaded.methodByName["Read"]; !ok {
		t.Fatal("missing methodByName[Read]")
	}
	if _, ok := loaded.abiSymbols["_llgo_demo.T"]; !ok {
		t.Fatal("missing abiSymbols[_llgo_demo.T]")
	}
	if err := writeFileAtomicBytes(filepath.Join(dir, "atomic.bin"), []byte("hello")); err != nil {
		t.Fatalf("writeFileAtomicBytes: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "atomic.bin"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("read atomic.bin = (%q, %v)", string(data), err)
	}
}
