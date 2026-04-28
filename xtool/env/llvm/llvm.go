/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package llvm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/xtool/clang"
	"github.com/goplus/llgo/xtool/llvm/install_name_tool"
	"github.com/goplus/llgo/xtool/llvm/llvmlink"
	"github.com/goplus/llgo/xtool/nm"
)

// -----------------------------------------------------------------------------

const (
	// CrosscompileClangPath is the relative path from LLGO_ROOT to the clang installation
	CrosscompileClangPath = "crosscompile/clang"
)

// -----------------------------------------------------------------------------

// defaultLLVMConfigBin returns the default path to the llvm-config binary. It
// checks the LLVM_CONFIG environment variable first, then searches in PATH. If
// not found, it returns [ldLLVMConfigBin] as a last resort.
func defaultLLVMConfigBin() string {
	bin := os.Getenv("LLVM_CONFIG")
	if bin != "" {
		return bin
	}
	bin, _ = exec.LookPath("llvm-config")
	if bin != "" {
		return bin
	}

	llgoRoot := env.LLGoROOT()
	// Check LLGO_ROOT/crosscompile/clang for llvm-config
	crossLLVMConfigBin := filepath.Join(llgoRoot, CrosscompileClangPath, "bin", "llvm-config")
	if _, err := os.Stat(crossLLVMConfigBin); err == nil {
		return crossLLVMConfigBin
	}
	return ldLLVMConfigBin
}

// -----------------------------------------------------------------------------

// Env represents an LLVM installation.
type Env struct {
	binDir string
}

var bindirCache sync.Map

// New creates a new [Env] instance.
func New(llvmConfigBin string) *Env {
	cacheKey := llvmConfigBin
	if llvmConfigBin == "" {
		llvmConfigBin = defaultLLVMConfigBin()
		cacheKey = "default\x00" + os.Getenv("LLVM_CONFIG") + "\x00" + os.Getenv("PATH") + "\x00" + llvmConfigBin
	}
	if cached, ok := bindirCache.Load(cacheKey); ok {
		return &Env{binDir: cached.(string)}
	}

	// Note that an empty binDir is acceptable. In this case, LLVM
	// executables are assumed to be in PATH.
	binDir, err := exec.Command(llvmConfigBin, "--bindir").Output()
	trimmed := strings.TrimSpace(string(binDir))
	if err == nil {
		actual, _ := bindirCache.LoadOrStore(cacheKey, trimmed)
		trimmed = actual.(string)
	}

	e := &Env{binDir: trimmed}
	return e
}

// BinDir returns the directory containing LLVM executables. An empty string
// means LLVM executables are assumed to be in PATH.
func (e *Env) BinDir() string { return e.binDir }

// Clang returns a new [clang.Cmd] instance.
func (e *Env) Clang() *clang.Cmd {
	bin := filepath.Join(e.BinDir(), "clang++")
	return clang.New(bin)
}

// Link returns a new [llvmlink.Cmd] instance.
func (e *Env) Link() *llvmlink.Cmd {
	bin := filepath.Join(e.BinDir(), "llvm-link")
	return llvmlink.New(bin)
}

// Nm returns a new [nm.Cmd] instance.
func (e *Env) Nm() *nm.Cmd {
	bin := filepath.Join(e.BinDir(), "llvm-nm")
	return nm.New(bin)
}

func (e *Env) InstallNameTool() *install_name_tool.Cmd {
	bin := filepath.Join(e.BinDir(), "llvm-install-name-tool")
	return install_name_tool.New(bin)
}

// Readelf returns a command to execute llvm-readelf with given arguments.
func (e *Env) Readelf(args ...string) (*exec.Cmd, error) {
	path, err := e.toolPath("llvm-readelf")
	if err != nil {
		return nil, err
	}
	return exec.Command(path, args...), nil
}

func (e *Env) toolPath(base string) (string, error) {
	if tool := searchTool(e.binDir, base); tool != "" {
		return tool, nil
	}
	if tool, err := exec.LookPath(base); err == nil {
		return tool, nil
	}
	if tool := searchToolInPath(base); tool != "" {
		return tool, nil
	}
	return "", fmt.Errorf("%s not found", base)
}

func searchTool(dir, base string) string {
	if dir == "" {
		return ""
	}
	candidate := filepath.Join(dir, base)
	if isExecutable(candidate) {
		return candidate
	}
	pattern := filepath.Join(dir, base+"-*")
	matches, _ := filepath.Glob(pattern)
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	for _, match := range matches {
		if isExecutable(match) {
			return match
		}
	}
	return ""
}

func searchToolInPath(base string) string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if tool := searchTool(dir, base); tool != "" {
			return tool
		}
	}
	return ""
}

func isExecutable(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// -----------------------------------------------------------------------------
