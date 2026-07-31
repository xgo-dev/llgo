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

	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/packages"
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
	if err := normalizeToArchive(ctx, pkg, false); err != nil {
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
	err := normalizeToArchive(ctx, pkg, false)
	if err == nil {
		t.Fatal("normalizeToArchive succeeded with a missing member")
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
