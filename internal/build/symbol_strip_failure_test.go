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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestSymbolStripPublicationFailures(t *testing.T) {
	for _, tt := range []struct {
		name, fail, bundle string
		noDebug, noOld     bool
		keepBackup         bool
	}{
		{name: "success", bundle: "new"},
		{name: "stat", fail: "stat", bundle: "old"},
		{name: "stage", fail: "stage", bundle: "old"},
		{name: "copy", fail: "copy", bundle: "old"},
		{name: "keep-list", fail: "keep", bundle: "old"},
		{name: "permissions", fail: "chmod", bundle: "old"},
		{name: "open", fail: "open", bundle: "old"},
		{name: "sync", fail: "sync", bundle: "old"},
		{name: "close", fail: "close", bundle: "old"},
		{name: "bundle-stat", fail: "lstat", bundle: "old"},
		{name: "backup", fail: "backup", bundle: "old"},
		{name: "publish-debug", fail: "publish-debug", bundle: "old"},
		{name: "publish-debug-no-backup", fail: "publish-debug", noOld: true},
		{name: "restore-after-debug-failure", fail: "publish-debug+restore", keepBackup: true},
		{name: "publish-exe", fail: "publish-exe", bundle: "old"},
		{name: "restore-after-exe-failure", fail: "publish-exe+restore", keepBackup: true},
		{name: "remove-new-bundle-with-backup", fail: "publish-exe+remove", bundle: "new", keepBackup: true},
		{name: "remove-new-bundle-without-backup", fail: "publish-exe+remove", bundle: "new", noOld: true},
		{name: "dsymutil", fail: "dsymutil", bundle: "old"},
		{name: "strip", fail: "strip", bundle: "old"},
		{name: "sign", fail: "codesign", bundle: "old"},
		{name: "verify-signature", fail: "verify", bundle: "old"},
		{name: "missing-dsym-output", bundle: "old"},
		{name: "dsym-is-file", bundle: "file"},
		{name: "no-debug", noDebug: true},
		{name: "no-debug-publish-failure", fail: "publish-exe", bundle: "old", noDebug: true},
		{name: "obsolete-bundle-cleanup", fail: "remove", bundle: "old", noDebug: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := &OutFmtDetails{Out: filepath.Join(t.TempDir(), "app")}
			bundle := out.Out + ".dSYM"
			out.DSYM = bundle
			if err := os.WriteFile(out.Out, []byte("linked"), 0o751); err != nil {
				t.Fatal(err)
			}
			if tt.bundle == "file" {
				if err := os.WriteFile(bundle, []byte("previous"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else if !tt.noOld {
				if err := os.Mkdir(bundle, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(bundle, "old"), []byte("previous"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			conf := &Config{Goos: "darwin", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
			if tt.noDebug {
				conf.LinkOptions.DWARF = DWARFOmit
				out.DSYM = ""
			}
			if tt.name == "missing-dsym-output" {
				out.DSYM = ""
			}
			fault := errors.New("injected publication failure")
			fails := func(role string) bool {
				for _, part := range strings.Split(tt.fail, "+") {
					if part == role {
						return true
					}
				}
				return false
			}
			files := symbolStripOSFileOps()
			stage := ""
			files.mkdirTemp = func(dir, pattern string) (string, error) {
				if fails("stage") {
					return "", fault
				}
				var err error
				stage, err = os.MkdirTemp(dir, pattern)
				return stage, err
			}
			if fails("stat") {
				files.stat = func(string) (os.FileInfo, error) { return nil, fault }
			}
			if fails("copy") {
				files.copyFile = func(string, string) error { return fault }
			}
			if fails("keep") {
				files.writeFile = func(string, []byte, os.FileMode) error { return fault }
			}
			if fails("chmod") {
				files.chmod = func(string, os.FileMode) error { return fault }
			}
			if fails("open") {
				files.openFile = func(string, int, os.FileMode) (*os.File, error) { return nil, fault }
			}
			if fails("sync") {
				files.syncFile = func(*os.File) error { return fault }
			}
			if fails("close") {
				files.closeFile = func(f *os.File) error { _ = f.Close(); return fault }
			}
			if fails("lstat") {
				files.lstat = func(string) (os.FileInfo, error) { return nil, fault }
			}
			files.rename = func(from, to string) error {
				role := "publish-exe"
				switch {
				case from == bundle:
					role = "backup"
				case filepath.Base(from) == "debug.dSYM":
					role = "publish-debug"
				case filepath.Base(from) == "previous.dSYM":
					role = "restore"
				}
				if fails(role) {
					return fault
				}
				return os.Rename(from, to)
			}
			files.removeAll = func(path string) error {
				if path == bundle && fails("remove") {
					return fault
				}
				return os.RemoveAll(path)
			}
			run := func(name string, args ...string) error {
				role := name
				if name == "codesign" && args[0] == "--verify" {
					role = "verify"
				}
				if fails(role) {
					return fault
				}
				if name == "dsymutil" {
					if err := os.Mkdir(args[2], 0o700); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(args[2], "new"), []byte("debug"), 0o600)
				}
				if name == "strip" {
					return os.WriteFile(args[len(args)-1], []byte("stripped"), 0o600)
				}
				return nil
			}
			err := stripNativeSymbolsWithIO(&context{buildConf: conf}, out, nil, run, files)
			wantErr := tt.fail != "" || tt.name == "missing-dsym-output" || tt.bundle == "file"
			if (err != nil) != wantErr {
				t.Fatalf("publication error: %v", err)
			}
			wantExe := "linked"
			if !wantErr || tt.noDebug && tt.fail == "remove" {
				wantExe = "stripped"
			}
			if data, readErr := os.ReadFile(out.Out); readErr != nil || string(data) != wantExe {
				t.Fatalf("published executable: %q, %v", data, readErr)
			}
			switch tt.bundle {
			case "old", "new":
				if _, err := os.Stat(filepath.Join(bundle, tt.bundle)); err != nil {
					t.Fatalf("expected %s bundle: %v", tt.bundle, err)
				}
			case "file":
				if data, err := os.ReadFile(bundle); err != nil || string(data) != "previous" {
					t.Fatalf("overwrote a non-directory output: %q, %v", data, err)
				}
			default:
				if _, err := os.Stat(bundle); !os.IsNotExist(err) {
					t.Fatalf("unexpected published bundle: %v", err)
				}
			}
			if tt.keepBackup {
				backup := filepath.Join(stage, "previous.dSYM")
				if data, readErr := os.ReadFile(filepath.Join(backup, "old")); readErr != nil || string(data) != "previous" || !strings.Contains(err.Error(), backup) {
					t.Fatalf("lost recoverable backup or its reported path: %q, %v, %v", data, readErr, err)
				}
			} else if stage != "" {
				if _, err := os.Stat(stage); !os.IsNotExist(err) {
					t.Fatalf("unneeded staging directory survived: %v", err)
				}
			}
			if tt.noOld && fails("remove") && (!strings.Contains(err.Error(), "newly published dSYM "+bundle) || strings.Contains(err.Error(), "previous dSYM retained")) {
				t.Fatalf("misleading rollback diagnostic: %v", err)
			}
		})
	}
}

func TestSymbolStripValidationAndToolFailures(t *testing.T) {
	ctx := &context{buildConf: &Config{Mode: ModeBuild, Goos: "linux", LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFOmit}}, crossCompile: configurableDebugInfo()}
	ctx.crossCompile.Toolchain.ObjectFormat = crosscompile.ObjectFormatELF
	for _, out := range []*OutFmtDetails{nil, {}, {Out: filepath.Join(t.TempDir(), "missing")}} {
		if err := finalizeNativeSymbols(ctx, out, false); err == nil {
			t.Fatal("accepted missing native output")
		}
		if err := stripNativeSymbolsWith(ctx, out, nil, false, nil); err == nil {
			t.Fatal("accepted missing staged output")
		}
	}
	if err := finalizeNativeSymbols(nil, nil, false); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LLVM_CONFIG", "missing-llvm-config")
	out := &OutFmtDetails{Out: filepath.Join(t.TempDir(), "app")}
	if err := os.WriteFile(out.Out, []byte("linked"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := finalizeNativeSymbols(ctx, out, true); err == nil {
		t.Fatal("ignored missing objcopy")
	}
	ctx.buildConf.Goos = "darwin"
	ctx.buildConf.LinkOptions.DWARF = DWARFPreserve
	ctx.crossCompile.Toolchain.ObjectFormat = crosscompile.ObjectFormatMachO
	if err := finalizeNativeSymbols(ctx, out, false); err == nil || !strings.Contains(err.Error(), "read Darwin exports") {
		t.Fatalf("ignored export-reader failure: %v", err)
	}
	if data, err := os.ReadFile(out.Out); err != nil || string(data) != "linked" {
		t.Fatalf("tool failure changed the linked executable: %q, %v", data, err)
	}
}

func TestDSYMArtifactErrors(t *testing.T) {
	for _, kind := range []string{"missing", "file", "symlink", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			out := &OutFmtDetails{Out: filepath.Join(t.TempDir(), "app")}
			out.DSYM = out.Out + ".dSYM"
			if err := os.WriteFile(out.Out, []byte("exe"), 0o700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "file":
				if err := os.WriteFile(out.DSYM, []byte("not a directory"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink", "unreadable":
				if err := os.Mkdir(out.DSYM, 0o700); err != nil {
					t.Fatal(err)
				}
				if kind == "symlink" {
					if err := os.Symlink(out.Out, filepath.Join(out.DSYM, "resource")); err != nil {
						t.Skip("symlinks unavailable:", err)
					}
				} else {
					if runtime.GOOS == "windows" || os.Geteuid() == 0 {
						t.Skip("requires POSIX directory permissions without root bypass")
					}
					if err := os.Chmod(out.DSYM, 0); err != nil {
						t.Fatal(err)
					}
					defer os.Chmod(out.DSYM, 0o700)
				}
			}
			if _, err := CollectArtifacts(&Config{DebugArtifactMode: DebugArtifactEmbedded}, out); err == nil {
				t.Fatal("accepted an invalid dSYM bundle")
			}
		})
	}
}
