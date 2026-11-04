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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	gllvm "github.com/xgo-dev/llvm"
)

func TestNormalizeToArchiveUsesMemoryBuffer(t *testing.T) {
	llvmCtx := gllvm.NewContext()
	defer llvmCtx.Dispose()
	mod := llvmCtx.NewModule("archive")
	defer mod.Dispose()
	mod.SetTarget("x86_64-unknown-linux-gnu")
	gllvm.AddFunction(mod, "memory_symbol", gllvm.FunctionType(llvmCtx.Int32Type(), nil, false))

	memoryBuf := gllvm.WriteBitcodeToMemoryBuffer(mod)
	ctx := &context{
		buildConf: &Config{Goos: "linux", Goarch: "amd64"},
		crossCompile: crosscompile.Export{
			LLVMTarget: "x86_64-unknown-linux-gnu",
		},
	}
	fileBuf := gllvm.WriteBitcodeToMemoryBuffer(mod)
	defer fileBuf.Dispose()
	filePath := filepath.Join(t.TempDir(), "file-member.bc")
	if err := os.WriteFile(filePath, fileBuf.Bytes(), 0o644); err != nil {
		memoryBuf.Dispose()
		t.Fatal(err)
	}

	pkg := &aPackage{
		Package:  &packages.Package{PkgPath: "example.com/archive"},
		ObjFiles: []string{filePath},
		ObjBuffers: []packageArchiveBuffer{{
			name:   "memory-member.bc",
			buffer: memoryBuf,
		}},
	}
	if err := normalizeToArchive(ctx, pkg, true); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pkg.ArchiveFile)

	if len(pkg.ObjFiles) != 0 || len(pkg.ObjBuffers) != 0 {
		t.Fatalf("package members were not released: files=%v buffers=%v", pkg.ObjFiles, pkg.ObjBuffers)
	}
	data, err := os.ReadFile(pkg.ArchiveFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("!<arch>\n")) {
		t.Fatalf("archive has invalid magic: %q", data[:8])
	}
	for _, name := range []string{"file-member.bc", "memory-member.bc"} {
		if !bytes.Contains(data, []byte(name)) {
			t.Errorf("archive does not contain member %q", name)
		}
	}
}

func TestCreatePackageArchiveFileWithExternalArchiver(t *testing.T) {
	for _, mode := range []lto.Mode{lto.Off, lto.Full, lto.Thin} {
		t.Run(mode.String(), func(t *testing.T) {
			testPackageArchiveWithExternalArchiver(t, mode)
		})
	}
}

func testPackageArchiveWithExternalArchiver(t *testing.T, mode lto.Mode) {
	t.Helper()
	llssa.Initialize(llssa.InitAll)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	llvmPkg := prog.NewPackage("archive", "example.com/archive")
	mod := llvmPkg.Module()
	llvmCtx := mod.Context()
	fn := gllvm.AddFunction(mod, "external_archive_symbol", gllvm.FunctionType(llvmCtx.Int32Type(), nil, false))
	builder := llvmCtx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(llvmCtx.AddBasicBlock(fn, "entry"))
	builder.CreateRet(gllvm.ConstInt(llvmCtx.Int32Type(), 42, false))

	// Match actual package emission: native objects without LTO, bitcode with
	// LTO. Feeding bitcode to a non-LTO system archiver can silently drop it.
	ctx := &context{
		prog:      prog,
		buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, LTO: mode},
		commands:  commandEnv{environ: os.Environ()},
	}
	if runtime.GOOS == "windows" {
		ctx.crossCompile.Toolchain.ObjectFormat = crosscompile.ObjectFormatCOFF
	}
	buffer, _, err := emitObjectToMemoryBuffer(ctx, llvmPkg)
	if err != nil {
		t.Fatal(err)
	}
	defer buffer.Dispose()
	pkg := &aPackage{ObjBuffers: []packageArchiveBuffer{{name: "memory-member.bc", buffer: buffer}}}
	archivePath := filepath.Join(t.TempDir(), "external.a")
	if err := ctx.createPackageArchiveFileWithExternalArchiver(archivePath, pkg, false); err != nil {
		t.Fatal(err)
	}
	checkExternalArchiveMember(t, ctx, archivePath, pkg.ObjBuffers[0])

	t.Run("dispatch", func(t *testing.T) {
		previous := useExternalPackageArchiver
		useExternalPackageArchiver = true
		t.Cleanup(func() { useExternalPackageArchiver = previous })

		archivePath := filepath.Join(t.TempDir(), "external-dispatch.a")
		if err := ctx.createPackageArchiveFile(archivePath, pkg, false); err != nil {
			t.Fatal(err)
		}
		checkExternalArchiveMember(t, ctx, archivePath, pkg.ObjBuffers[0])
	})

	t.Run("invalid members", func(t *testing.T) {
		for _, member := range []packageArchiveBuffer{
			{buffer: buffer},
			{name: "nil.bc"},
			{name: ".", buffer: buffer},
		} {
			pkg := &aPackage{ObjBuffers: []packageArchiveBuffer{member}}
			if err := ctx.createPackageArchiveFileWithExternalArchiver(filepath.Join(t.TempDir(), "invalid.a"), pkg, false); err == nil {
				t.Fatalf("createPackageArchiveFileWithExternalArchiver succeeded for member %+v", member)
			}
		}
	})

	t.Run("invalid archive parent", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(parent, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ctx.createPackageArchiveFileWithExternalArchiver(filepath.Join(parent, "invalid.a"), pkg, false); err == nil {
			t.Fatal("createPackageArchiveFileWithExternalArchiver succeeded below a regular file")
		}
	})

	t.Run("missing file member", func(t *testing.T) {
		pkg := &aPackage{
			ObjFiles:   []string{filepath.Join(t.TempDir(), "missing.o")},
			ObjBuffers: []packageArchiveBuffer{{name: "memory-member.bc", buffer: buffer}},
		}
		if err := ctx.createPackageArchiveFileWithExternalArchiver(filepath.Join(t.TempDir(), "invalid.a"), pkg, false); err == nil {
			t.Fatal("createPackageArchiveFileWithExternalArchiver succeeded with a missing file member")
		}
	})
}

func checkExternalArchiveMember(t *testing.T, ctx *context, path string, member packageArchiveBuffer) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("!<arch>\n")) {
		t.Fatalf("archive has invalid magic: %q", data[:min(len(data), 8)])
	}
	ar, err := exec.LookPath(ctx.archiver())
	if err != nil {
		t.Fatal(err)
	}
	members, err := ctx.commands.configure(exec.Command(ar, "t", path)).CombinedOutput()
	if err != nil {
		t.Fatalf("%s t: %v\n%s", ar, err, members)
	}
	count := 0
	for _, name := range strings.Fields(string(members)) {
		if name == member.name {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("archive requires exactly one %q member, got %d using %s: %q", member.name, count, ar, members)
	}
	cmd := ctx.commands.configure(exec.Command(ar, "p", path, member.name))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	payload, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s p: %v\n%s", ar, err, &stderr)
	}
	if !bytes.Equal(payload, member.buffer.Bytes()) {
		t.Fatalf("%s changed archive member %q: got %d bytes, want %d", ar, member.name, len(payload), len(member.buffer.Bytes()))
	}
}

func TestPackageArchiveEdgeCases(t *testing.T) {
	empty := &aPackage{}
	if err := (&context{}).createPackageArchiveFile(filepath.Join(t.TempDir(), "empty.a"), empty, false); err == nil {
		t.Fatal("createPackageArchiveFile succeeded without members")
	}
	fileOnly := &aPackage{ObjFiles: []string{filepath.Join(t.TempDir(), "missing.o")}}
	if err := (&context{buildConf: &Config{Goos: "linux", Goarch: "amd64"}}).createPackageArchiveFile(filepath.Join(t.TempDir(), "file-only.a"), fileOnly, false); err == nil {
		t.Fatal("file-only archive succeeded with a missing member")
	}
	empty.ObjBuffers = []packageArchiveBuffer{{}}
	empty.disposeArchiveBuffers()
	if empty.ObjBuffers != nil {
		t.Fatal("disposeArchiveBuffers retained nil members")
	}

	llvmCtx := gllvm.NewContext()
	defer llvmCtx.Dispose()
	mod := llvmCtx.NewModule("archive-errors")
	defer mod.Dispose()
	mod.SetTarget("x86_64-unknown-linux-gnu")
	gllvm.AddFunction(mod, "archive_error_symbol", gllvm.FunctionType(llvmCtx.Int32Type(), nil, false))
	buffer := gllvm.WriteBitcodeToMemoryBuffer(mod)
	defer buffer.Dispose()
	pkg := &aPackage{ObjBuffers: []packageArchiveBuffer{{name: "member.bc", buffer: buffer}}}
	ctx := &context{
		buildConf:    &Config{Goos: "linux", Goarch: "amd64"},
		crossCompile: crosscompile.Export{LLVMTarget: "x86_64-unknown-linux-gnu"},
	}

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ctx.createPackageArchiveFile(filepath.Join(blocker, "archive.a"), pkg, false); err == nil {
		t.Fatal("createPackageArchiveFile succeeded below a regular file")
	}
	tooLong := filepath.Join(t.TempDir(), strings.Repeat("a", 300)+".a")
	if err := ctx.createPackageArchiveFile(tooLong, pkg, false); err == nil {
		t.Fatal("createPackageArchiveFile succeeded with an overlong temporary-file prefix")
	}

	archiveDir := filepath.Join(t.TempDir(), "archive.a")
	if err := os.Mkdir(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ctx.createPackageArchiveFile(archiveDir, pkg, false); err == nil {
		t.Fatal("createPackageArchiveFile replaced a directory")
	}
}

func TestNormalizeToArchiveFailsWithoutObjectFallback(t *testing.T) {
	llvmCtx := gllvm.NewContext()
	defer llvmCtx.Dispose()
	mod := llvmCtx.NewModule("archive-error")
	defer mod.Dispose()
	mod.SetTarget("x86_64-unknown-linux-gnu")
	gllvm.AddFunction(mod, "memory_symbol", gllvm.FunctionType(llvmCtx.Int32Type(), nil, false))

	memoryBuf := gllvm.WriteBitcodeToMemoryBuffer(mod)
	ctx := &context{
		buildConf: &Config{Goos: "linux", Goarch: "amd64"},
		crossCompile: crosscompile.Export{
			LLVMTarget: "x86_64-unknown-linux-gnu",
		},
	}
	pkg := &aPackage{
		Package:  &packages.Package{PkgPath: "example.com/archive-error"},
		ObjFiles: []string{filepath.Join(t.TempDir(), "missing.o")},
		ObjBuffers: []packageArchiveBuffer{{
			name:   "memory-member.bc",
			buffer: memoryBuf,
		}},
	}
	err := finalizePackageBuild(ctx, &packageBuildTask{pkg: pkg}, false)
	if err == nil {
		t.Fatal("finalizePackageBuild succeeded with a missing member")
	}
	if !strings.Contains(err.Error(), "missing.o") {
		t.Fatalf("normalizeToArchive error = %v, want missing member", err)
	}
	if len(pkg.ObjBuffers) != 0 {
		t.Fatal("memory buffer was retained after archive failure")
	}
	if pkg.ArchiveFile != "" {
		t.Fatalf("ArchiveFile = %q after failure", pkg.ArchiveFile)
	}
}

func TestEmscriptenArchiverUsesSDK(t *testing.T) {
	root := t.TempDir()
	tool := func(dir, name string) string {
		t.Helper()
		if runtime.GOOS == "windows" {
			if name == "emcc" || name == "emar" {
				name += ".bat"
			} else {
				name += ".exe"
			}
		}
		path := filepath.Join(root, dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("unused tool fixture"), 0755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if runtime.GOOS == "windows" {
		t.Setenv("PATHEXT", ".EXE;.BAT;.CMD")
	}
	cc := tool("sdk", "emcc")
	emar := tool("sdk", "emar")
	hostAR := tool("host", "llvm-ar")
	otherEMAR := tool("other-sdk", "emar")
	// A different SDK on PATH must not override emcc's sibling emar.
	t.Setenv("PATH", filepath.Dir(otherEMAR)+string(os.PathListSeparator)+filepath.Dir(hostAR))
	t.Setenv("LLGO_AR", "")
	for _, provider := range []crosscompile.WasmProvider{crosscompile.WasmProviderGoJS, crosscompile.WasmProviderEmscripten} {
		ctx := &context{buildConf: &Config{Goarch: "wasm"}, crossCompile: crosscompile.Export{CC: cc, WasmProvider: provider}}
		if got := ctx.archiver(); got != emar {
			t.Fatalf("%s archiver = %q, want SDK emar %q", provider, got, emar)
		}
		if got, err := ctx.archiveMerger(); err != nil || got != emar {
			t.Fatalf("%s MRI archiver = %q, %v; want %q", provider, got, err, emar)
		}
	}
	// A command found through PATH should still select its own SDK first.
	t.Setenv("PATH", filepath.Dir(cc)+string(os.PathListSeparator)+filepath.Dir(hostAR))
	ctx := &context{crossCompile: crosscompile.Export{CC: filepath.Base(cc), WasmProvider: crosscompile.WasmProviderGoJS}}
	if got := ctx.emscriptenArchiver(); got != emar {
		t.Fatalf("PATH compiler archiver = %q, want %q", got, emar)
	}
	// A compiler wrapper outside the SDK can use emar from PATH.
	ctx.crossCompile.CC = tool("wrapper", "emcc")
	if got := ctx.emscriptenArchiver(); got != emar {
		t.Fatalf("wrapped compiler archiver = %q, want %q", got, emar)
	}
	ctx.crossCompile.WasmProvider = crosscompile.WasmProviderNone
	if got := ctx.emscriptenArchiver(); got != "" {
		t.Fatalf("non-Emscripten target selected %q", got)
	}
	t.Setenv("PATH", filepath.Dir(hostAR))
	ctx = &context{buildConf: &Config{Goarch: "wasm"}, crossCompile: crosscompile.Export{CC: "missing-emcc", WasmProvider: crosscompile.WasmProviderGoJS}}
	if got := ctx.archiver(); got != hostAR {
		t.Fatalf("missing SDK fallback = %q, want %q", got, hostAR)
	}
	if got, err := ctx.archiveMerger(); err != nil || got != hostAR {
		t.Fatalf("missing SDK MRI fallback = %q, %v; want %q", got, err, hostAR)
	}
	t.Setenv("LLGO_AR", hostAR)
	ctx = &context{crossCompile: crosscompile.Export{CC: cc, WasmProvider: crosscompile.WasmProviderGoJS}}
	if got := ctx.archiver(); got != hostAR {
		t.Fatalf("ignored explicit LLGO_AR: %q", got)
	}
	if got, err := ctx.archiveMerger(); err != nil || got != hostAR {
		t.Fatalf("MRI ignored explicit LLGO_AR: %q, %v", got, err)
	}
}
