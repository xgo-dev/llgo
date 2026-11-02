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
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/debugabi"
	llvmenv "github.com/xgo-dev/llgo/xtool/env/llvm"
)

type symbolStripFileOps struct {
	stat      func(string) (os.FileInfo, error)
	lstat     func(string) (os.FileInfo, error)
	mkdirTemp func(string, string) (string, error)
	copyFile  func(string, string) error
	writeFile func(string, []byte, os.FileMode) error
	chmod     func(string, os.FileMode) error
	openFile  func(string, int, os.FileMode) (*os.File, error)
	syncFile  func(*os.File) error
	closeFile func(*os.File) error
	rename    func(string, string) error
	removeAll func(string) error
}

func symbolStripOSFileOps() symbolStripFileOps {
	return symbolStripFileOps{
		stat: os.Stat, lstat: os.Lstat, mkdirTemp: os.MkdirTemp,
		copyFile: copyFileAtomic, writeFile: os.WriteFile, chmod: os.Chmod,
		openFile: os.OpenFile, syncFile: (*os.File).Sync, closeFile: (*os.File).Close,
		rename: os.Rename, removeAll: os.RemoveAll,
	}
}

func shouldStripNativeSymbols(conf *Config, target *crosscompile.Export) bool {
	if conf == nil || target == nil || !conf.LinkOptions.OmitSymbolTable ||
		conf.Mode == ModeGen || conf.Target != "" || conf.BuildMode == BuildModeCArchive {
		return false
	}
	return (conf.Goos == "linux" && target.Toolchain.ObjectFormat == crosscompile.ObjectFormatELF) ||
		(conf.Goos == "darwin" && target.Toolchain.ObjectFormat == crosscompile.ObjectFormatMachO)
}

func finalizeNativeSymbols(ctx *context, out *OutFmtDetails, verbose bool) error {
	if ctx == nil || !shouldStripNativeSymbols(ctx.buildConf, &ctx.crossCompile) {
		return nil
	}
	if out == nil || out.Out == "" {
		return fmt.Errorf("strip symbols: executable path is empty")
	}
	var exports []string
	if ctx.buildConf.Goos == "darwin" && shouldEmitDebugInfo(ctx.buildConf, &ctx.crossCompile) {
		// A strip keep-list also filters dyld exports. Preserve the existing
		// public interface, including explicit C exports. nm provides raw
		// linker names; debug/macho normalizes underscores in Go names.
		output, err := ctx.commands.configure(exec.Command("nm", "-gjU", out.Out)).CombinedOutput()
		if err != nil {
			return fmt.Errorf("read Darwin exports: %w: %s", err, output)
		}
		exports = strings.Split(strings.TrimSpace(string(output)), "\n")
	}
	return stripNativeSymbolsWith(ctx, out, exports, verbose, func(name string, args ...string) error {
		if verbose {
			fmt.Fprintf(os.Stderr, "%s %q\n", name, args)
		}
		output, err := ctx.commands.configure(exec.Command(name, args...)).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %w: %s", name, err, output)
		}
		return nil
	})
}

// stripNativeSymbolsWith stages both outputs. Tool failures leave the linked
// executable and any previous dSYM untouched. PCLN finalization must run first.
func stripNativeSymbolsWith(ctx *context, out *OutFmtDetails, darwinExports []string, verbose bool, run func(string, ...string) error) error {
	return stripNativeSymbolsWithIO(ctx, out, darwinExports, run, symbolStripOSFileOps())
}

func stripNativeSymbolsWithIO(ctx *context, out *OutFmtDetails, darwinExports []string, run func(string, ...string) error, files symbolStripFileOps) error {
	if out == nil || out.Out == "" {
		return fmt.Errorf("strip symbols: executable path is empty")
	}
	info, err := files.stat(out.Out)
	if err != nil {
		return err
	}
	dir, err := files.mkdirTemp(filepath.Dir(out.Out), "."+filepath.Base(out.Out)+".symbols-*")
	if err != nil {
		return err
	}
	publication := dsymPublication{destination: out.DSYM, files: files}
	defer func() {
		if !publication.keepBackup {
			_ = files.removeAll(dir)
		}
	}()
	staged := filepath.Join(dir, filepath.Base(out.Out))
	darwin := ctx.buildConf.Goos == "darwin"
	preserve := shouldEmitDebugInfo(ctx.buildConf, &ctx.crossCompile)
	stagedDSYM := ""
	debuggerSymbols := []string{debugabi.NativeRecordSymbol, debugabi.LegacyMarkerSymbol, "llgo_debugger_threads_v1"}
	if darwin {
		if err := files.copyFile(out.Out, staged); err != nil {
			return err
		}
		args := []string{"-u"}
		if preserve {
			if out.DSYM == "" {
				return fmt.Errorf("strip symbols: retained Darwin DWARF needs a dSYM output")
			}
			stagedDSYM = filepath.Join(dir, "debug.dSYM")
			if err := run("dsymutil", dsymutilInput(out.Out), "-o", stagedDSYM); err != nil {
				return err
			}
			if ctx.buildConf.LinkOptions.DWARFCompression == DWARFCompressionEnabled {
				if err := compressDSYMBundle(stagedDSYM, filepath.Base(out.Out)); err != nil {
					return err
				}
			}
			// LLDB does not expose all non-DWARF globals from a dSYM's
			// symbol table. Retain the adapter's discovery names in the image.
			keep := filepath.Join(dir, "debugger-symbols")
			names := append([]string(nil), darwinExports...)
			for _, symbol := range debuggerSymbols {
				names = append(names, "_"+symbol)
			}
			slices.Sort(names)
			if err := files.writeFile(keep, []byte(strings.Join(slices.Compact(names), "\n")+"\n"), 0o600); err != nil {
				return err
			}
			args = append(args, "-s", keep)
		}
		// Keep undefined symbols needed by dynamic binding. The export trie
		// remains intact, including C exports from shared libraries.
		if err := run("strip", append(args, staged)...); err != nil {
			return err
		}
		if err := run("codesign", "-f", "-s", "-", staged); err != nil {
			return err
		}
		if err := run("codesign", "--verify", staged); err != nil {
			return err
		}
	} else {
		tool := filepath.Join(llvmenv.New("").BinDir(), "llvm-objcopy")
		args := []string{"--strip-all"}
		if preserve {
			args = append(args, "--keep-section=.debug_*", "--keep-section=.zdebug_*", "--keep-section=.gdb_index", "--keep-section=.gnu_debuglink", "--keep-section=.gnu_debugaltlink")
			// The native debugger adapters discover these globals by symbol
			// name; their bytes alone and DWARF function entries are insufficient.
			for _, symbol := range debuggerSymbols {
				args = append(args, "--keep-symbol="+symbol)
			}
		}
		// Let objcopy read the linked image and write the staged result in
		// one pass, avoiding a separate full-file copy before stripping.
		args = append(args, out.Out, staged)
		if err := run(tool, args...); err != nil {
			return err
		}
	}
	if err := files.chmod(staged, info.Mode()); err != nil {
		return err
	}
	file, err := files.openFile(staged, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	syncErr := files.syncFile(file)
	closeErr := files.closeFile(file)
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}

	if stagedDSYM != "" {
		if err := publication.publish(stagedDSYM, filepath.Join(dir, "previous.dSYM")); err != nil {
			return err
		}
	}
	if err := files.rename(staged, out.Out); err != nil {
		if stagedDSYM != "" {
			return publication.rollback(err)
		}
		return err
	}
	if darwin && !preserve {
		// A successful -s/-w rebuild owns and removes its obsolete bundle.
		return files.removeAll(out.Out + ".dSYM")
	}
	return nil
}
