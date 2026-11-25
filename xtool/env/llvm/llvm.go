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
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/xtool/clang"
	"github.com/xgo-dev/llgo/xtool/llvm/install_name_tool"
	"github.com/xgo-dev/llgo/xtool/llvm/llvmlink"
	"github.com/xgo-dev/llgo/xtool/nm"
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

// ConfigBin returns the llvm-config executable selected for LLGo. It does not
// execute the program, so callers can use it for diagnostics even when the
// configured LLVM installation is incomplete.
func ConfigBin() string {
	return defaultLLVMConfigBin()
}

// -----------------------------------------------------------------------------

// Env represents an LLVM installation.
type Env struct {
	binDir string
}

type binDirCacheKey struct {
	bin     string
	modTime int64
	size    int64
}

var (
	binDirCacheMu sync.Mutex
	binDirCache   = map[binDirCacheKey]string{}
)

// New creates a new [Env] instance. Successful --bindir lookups are memoized
// per resolved llvm-config executable (keyed by path, mtime, and size).
func New(llvmConfigBin string) *Env {
	if llvmConfigBin == "" {
		llvmConfigBin = defaultLLVMConfigBin()
	}

	resolved := llvmConfigBin
	if !strings.ContainsAny(resolved, `/\`) {
		if found, err := exec.LookPath(resolved); err == nil {
			resolved = found
		}
	}

	if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
		key := binDirCacheKey{
			bin:     resolved,
			modTime: info.ModTime().UnixNano(),
			size:    info.Size(),
		}
		binDirCacheMu.Lock()
		if cached, ok := binDirCache[key]; ok {
			binDirCacheMu.Unlock()
			return &Env{binDir: cached}
		}
		binDirCacheMu.Unlock()

		trimmed, err := queryBinDir(resolved)
		if err == nil && trimmed != "" {
			binDirCacheMu.Lock()
			binDirCache[key] = trimmed
			binDirCacheMu.Unlock()
		}
		return &Env{binDir: trimmed}
	}

	trimmed, _ := queryBinDir(llvmConfigBin)
	return &Env{binDir: trimmed}
}

// queryBinDir runs llvm-config --bindir. An empty result is acceptable; in that
// case LLVM executables are assumed to be in PATH.
func queryBinDir(llvmConfigBin string) (string, error) {
	out, err := exec.Command(llvmConfigBin, "--bindir").Output()
	return strings.TrimSpace(string(out)), err
}

// BinDir returns the directory containing LLVM executables. An empty string
// means LLVM executables are assumed to be in PATH.
func (e *Env) BinDir() string { return e.binDir }

// SetupPath prepends the selected LLVM bin directory to the process PATH if
// it is not already present. CLI entry points and standalone tools (such as
// lldb and chore utilities) call it so external lookups find LLVM tools;
// internal/build also ensures the LLVM bin directory is present in each
// build's per-invocation environment snapshot without mutating process PATH.
func SetupPath() {
	binDir := New("").BinDir()
	if binDir == "" {
		return
	}

	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if samePath(dir, binDir) {
			return
		}
	}
	if path != "" {
		binDir += string(os.PathListSeparator) + path
	}
	_ = os.Setenv("PATH", binDir)
}

func samePath(x, y string) bool {
	x, y = filepath.Clean(x), filepath.Clean(y)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(x, y)
	}
	return x == y
}

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

// FileCheck returns a command to execute LLVM FileCheck with given arguments.
func (e *Env) FileCheck(args ...string) (*exec.Cmd, error) {
	path, err := e.toolPath("FileCheck")
	if err != nil {
		return nil, err
	}
	return exec.Command(path, args...), nil
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
