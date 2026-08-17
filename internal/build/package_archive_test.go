//go:build !llgo
// +build !llgo

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
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/packages"
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
