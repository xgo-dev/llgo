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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/xtool/safesplit"
)

// darwinLTOObjectArgs keeps the native LTO output available to dsymutil after
// Clang exits. Each link owns its directory so rebuilding an output cannot
// overwrite an object still referenced by another executable's debug map.
// The output owns the enclosing sidecar, which clean and temporary-output
// cleanup reclaim along with the executable.
func darwinLTOObjectArgs(ctx *context, app string, cmd *clang.Cmd, args []string) ([]string, error) {
	conf := ctx.buildConf
	if conf.Goos != "darwin" || conf.Target != "" || !conf.ltoEnabled() ||
		!shouldEmitDebugInfo(conf, &ctx.crossCompile) {
		return nil, nil
	}
	if hasDarwinLTOObjectPath(cmd.LinkArguments(args...)) {
		return nil, nil
	}
	root := darwinLTOObjectDir(app)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(root, "link-*")
	if err != nil {
		return nil, err
	}
	return []string{"-Xlinker", "-object_path_lto", "-Xlinker", filepath.Join(dir, "lto.o")}, nil
}

func hasDarwinLTOObjectPath(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			return false
		case "-o", "-L", "-F", "-isysroot", "--sysroot":
			i++ // The next token is a driver option's value, not a linker option.
		case "-object_path_lto":
			return true
		case "-Xlinker":
			i++
			if i < len(args) && args[i] == "-object_path_lto" {
				return true
			}
		default:
			if strings.HasPrefix(arg, "-Wl,") {
				for _, option := range strings.Split(strings.TrimPrefix(arg, "-Wl,"), ",") {
					if option == "-object_path_lto" {
						return true
					}
				}
			}
		}
	}
	return false
}

func darwinLTOObjectDir(app string) string {
	return app + ".llgo-debug-lto"
}

type darwinSizeSymbolPlan struct {
	linkerArgs     []string
	stripLTOLocals bool
	hiddenGlobals  []string
}

type darwinSizeCommand func(name string, args ...string) ([]byte, error)

type darwinSizeFileOps struct {
	stat       func(string) (os.FileInfo, error)
	open       func(string) (*os.File, error)
	closeFile  func(*os.File) error
	createTemp func(string, string) (*os.File, error)
	openFile   func(string, int, os.FileMode) (*os.File, error)
	rename     func(string, string) error
}

func darwinSizeOSFileOps() darwinSizeFileOps {
	return darwinSizeFileOps{
		stat:       os.Stat,
		open:       os.Open,
		closeFile:  (*os.File).Close,
		createTemp: os.CreateTemp,
		openFile:   os.OpenFile,
		rename:     os.Rename,
	}
}

func planDarwinSizeSymbols(ctx *context, pkgs []*aPackage, linkArgs []string) darwinSizeSymbolPlan {
	return planDarwinSizeSymbolsFor(ctx, pkgs, linkArgs, runtime.GOOS == "darwin")
}

func planDarwinSizeSymbolsFor(ctx *context, pkgs []*aPackage, linkArgs []string, nativeDarwin bool) darwinSizeSymbolPlan {
	if ctx == nil || ctx.buildConf == nil {
		return darwinSizeSymbolPlan{}
	}
	conf := ctx.buildConf
	if !nativeDarwin || conf.Goos != "darwin" || conf.Target != "" ||
		conf.BuildMode != BuildModeExe || !isSizeOptLevel(conf.OptLevel) {
		return darwinSizeSymbolPlan{}
	}
	// Without either the prebuilt table or site records, Darwin symbolization
	// falls back to dlsym. Keep the exports available in that escape-hatch
	// configuration.
	if conf.PCLNMode != PCLNNone && (ctx.prog == nil || !ctx.prog.FuncInfoSitesEnabled()) {
		return darwinSizeSymbolPlan{}
	}
	if mainPackageHasExports(pkgs) {
		return darwinSizeSymbolPlan{}
	}
	allLinkArgs := append([]string(nil), linkArgs...)
	allLinkArgs = append(allLinkArgs, safesplit.SplitPkgConfigFlags(os.Getenv("LDFLAGS"))...)
	if hasDarwinExportControl(allLinkArgs) {
		return darwinSizeSymbolPlan{}
	}
	if conf.ltoEnabled() && shouldEmitDebugInfo(conf, &ctx.crossCompile) {
		// Preserve the established LTO visibility and inlining policy for
		// DWARF builds.
		return darwinSizeSymbolPlan{}
	}
	if conf.PCLNMode == PCLNEmbedded && !shouldEmitRuntimeEntrySites(ctx) {
		// DWARF disables the in-body entry anchors to preserve LLDB's lexical
		// scopes. The runtime still needs dlsym for function entries in this
		// mode. Hide data/type definitions without adding export roots for
		// otherwise dead functions, which would increase code size under LTO.
		globals := make([]string, 0)
		for name := range linkedModuleGlobals(pkgs) {
			globals = append(globals, name)
		}
		sort.Strings(globals)
		return darwinSizeSymbolPlan{hiddenGlobals: globals}
	}

	if conf.ltoEnabled() && !shouldEmitDebugInfo(conf, &ctx.crossCompile) {
		// LTO internalizes ordinary Go definitions. Keep those local symbols
		// through pclnpost, then remove them from the final signed executable.
		return darwinSizeSymbolPlan{stripLTOLocals: true}
	}
	// DWARF builds retain their debug map for dsymutil. Export visibility is
	// independent of that map, including when LTO produces the definitions.
	// A main executable does not need to publish every package definition to
	// dyld. Keeping them as private externs preserves the Mach-O symbol table
	// for pclnpost and offline tools while eliminating the large export trie.
	return darwinSizeSymbolPlan{linkerArgs: []string{"-Wl,-no_exported_symbols"}}
}

// darwinHiddenGlobalArgs hides data/type definitions without rooting functions
// or changing the DWARF runtime's function export policy.
func darwinHiddenGlobalArgs(output string, globals []string) ([]string, func(), error) {
	return darwinHiddenGlobalArgsWith(output, globals, os.MkdirTemp, os.WriteFile)
}

func darwinHiddenGlobalArgsWith(output string, globals []string, mkdir func(string, string) (string, error), write func(string, []byte, os.FileMode) error) ([]string, func(), error) {
	cleanup := func() {}
	if globals == nil {
		return nil, cleanup, nil
	}
	dir, err := mkdir(filepath.Dir(output), ".llgo-hidden-*")
	if err != nil {
		return nil, cleanup, fmt.Errorf("create Darwin hidden-global list: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "symbols")
	// Escape type syntax so the exact glob cannot hide an unrelated symbol.
	// Synthetic main-module globals are not in the package snapshots.
	escape := strings.NewReplacer("\\", "\\\\", "*", "\\*", "?", "\\?", "[", "\\[", "]", "\\]", "{", "\\{", "}", "\\}")
	var list strings.Builder
	list.WriteString("___llgo_*\n")
	for _, symbol := range globals {
		list.WriteString("[_]")
		list.WriteString(escape.Replace(symbol))
		list.WriteByte('\n')
	}
	if err := write(path, []byte(list.String()), 0o600); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("write Darwin hidden-global list: %w", err)
	}
	return []string{"-Xlinker", "-unexported_symbols_list", "-Xlinker", path}, cleanup, nil
}

func isSizeOptLevel(level optlevel.Level) bool {
	return level == optlevel.Os || level == optlevel.Oz
}

func mainPackageHasExports(pkgs []*aPackage) bool {
	for _, pkg := range pkgs {
		// Executable-level dynamic exports are declared by the root main
		// package. Runtime and standard-library //export callbacks are static
		// implementation links and must not turn every Go symbol into a dyld
		// export.
		if pkg != nil && pkg.Package != nil && pkg.Name == "main" {
			if pkg.linkSnapshot != nil && pkg.linkSnapshot.hasLocalExports {
				return true
			}
			if pkg.LPkg != nil && hasLocalCExports(pkg.LPkg) {
				return true
			}
		}
	}
	return false
}

func hasDarwinExportControl(args []string) bool {
	for _, arg := range args {
		arg = strings.ToLower(arg)
		if strings.Contains(arg, "exported_symbol") ||
			strings.Contains(arg, "export_dynamic") ||
			strings.Contains(arg, "dynamic_lookup") {
			return true
		}
	}
	return false
}

func finalizeDarwinSizeExecutable(ctx *context, path string, verbose bool) error {
	if ctx == nil || !ctx.stripDarwinLTOLocals {
		return nil
	}
	if err := stripAndSignDarwinLocals(path, verbose); err != nil {
		return fmt.Errorf("compact Darwin LTO symbols: %w", err)
	}
	return nil
}

// stripAndSignDarwinLocals stages the mutation beside the output so a failed
// strip or signature never replaces the successfully linked executable.
func stripAndSignDarwinLocals(path string, verbose bool) error {
	return stripAndSignDarwinLocalsWith(path, verbose, func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	})
}

func stripAndSignDarwinLocalsWith(path string, verbose bool, run darwinSizeCommand) error {
	return stripAndSignDarwinLocalsUsing(path, verbose, run, darwinSizeOSFileOps())
}

func stripAndSignDarwinLocalsUsing(path string, verbose bool, run darwinSizeCommand, files darwinSizeFileOps) error {
	st, err := files.stat(path)
	if err != nil {
		return err
	}
	source, err := files.open(path)
	if err != nil {
		return err
	}
	defer func() {
		if source != nil {
			_ = files.closeFile(source)
		}
	}()

	tmp, err := files.createTemp(filepath.Dir(path), "."+filepath.Base(path)+".strip-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(st.Mode()); err != nil {
		return err
	}
	if _, err := io.Copy(tmp, source); err != nil {
		return err
	}
	// Windows does not replace an existing path while that file still has an
	// open handle. The staged copy no longer needs the original after io.Copy.
	if err := files.closeFile(source); err != nil {
		return err
	}
	source = nil
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "strip -x %s\n", tmpPath)
	}
	if output, err := run("strip", "-x", tmpPath); err != nil {
		return fmt.Errorf("strip -x: %v: %s", err, output)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "codesign -f -s - %s\n", tmpPath)
	}
	if output, err := run("codesign", "-f", "-s", "-", tmpPath); err != nil {
		return fmt.Errorf("codesign: %v: %s", err, output)
	}
	signed, err := files.openFile(tmpPath, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := signed.Sync(); err != nil {
		_ = signed.Close()
		return err
	}
	if err := signed.Close(); err != nil {
		return err
	}
	return files.rename(tmpPath, path)
}
