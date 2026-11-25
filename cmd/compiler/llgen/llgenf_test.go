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

package llgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
)

func TestApplyFlagsFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.txt")
	if err := applyFlagsFile(new(build.Config), missing); err != nil {
		t.Fatalf("missing flags file: %v", err)
	}

	path := filepath.Join(dir, "flags.txt")
	data := "-target=wasm\n--gcflags 'all=-N -l'\n-ldflags=--s --w=false # keep DWARF\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := new(build.Config)
	if err := applyFlagsFile(conf, path); err != nil {
		t.Fatal(err)
	}
	if conf.Goos != "js" || conf.Goarch != "wasm" || conf.Target != "wasm" {
		t.Fatalf("target config = %s/%s, target %q", conf.Goos, conf.Goarch, conf.Target)
	}
	if conf.OptLevel != optlevel.O0 {
		t.Fatalf("OptLevel = %v, want O0", conf.OptLevel)
	}
	if !conf.LinkOptions.OmitSymbolTable || conf.LinkOptions.EffectiveOmitDWARF() {
		t.Fatalf("LinkOptions = %+v, want -s with explicit -w=false", conf.LinkOptions)
	}
}

func TestApplyFlagsFileErrorIncludesPath(t *testing.T) {
	tests := []string{
		"-ldflags='unterminated\n",
		"-ldflags=-w=invalid\n",
		"-target\n",
		"-target=does-not-exist\n",
	}
	for _, data := range tests {
		t.Run(strings.TrimSpace(data), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "flags.txt")
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			conf := &build.Config{Target: "original"}
			err := applyFlagsFile(conf, path)
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Fatalf("applyFlagsFile(%q) error = %v, want path", data, err)
			}
			if conf.Target != "original" {
				t.Fatalf("failed application changed target to %q", conf.Target)
			}
		})
	}
}

func TestApplyFlagsFileTargetForms(t *testing.T) {
	for _, data := range []string{"-target wasi", "--target wasi", "--target=wasi"} {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "flags.txt")
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			conf := new(build.Config)
			if err := applyFlagsFile(conf, path); err != nil {
				t.Fatal(err)
			}
			if conf.Target != "wasi" {
				t.Fatalf("Target = %q, want wasi", conf.Target)
			}
		})
	}
}

func TestGeneratePostABIIsExplicit(t *testing.T) {
	t.Chdir("../../../runtime")
	const pkg = "./_test/codegen/localitycodegen"
	preABI := GenFrom(pkg)
	postABI := GeneratePostABI(pkg)
	if preABI == postABI.Text {
		t.Fatal("post-ABI generation did not change the module")
	}
	if !strings.Contains(preABI, "define { i64, ptr, ptr } @main.values()") {
		t.Fatal("GenFrom no longer exposes the historical pre-target-ABI signature")
	}
	if !strings.Contains(postABI.Text, "define void @main.values(ptr sret({ i64, ptr, ptr })") {
		t.Fatal("GeneratePostABI did not expose the lowered sret signature")
	}
	if postABI.GOOS == "" || postABI.GOARCH == "" {
		t.Fatalf("missing effective target: %+v", postABI)
	}
}

func TestGenerationTargetMatchesOnlyRequestedPackage(t *testing.T) {
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(targetFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dependencyFile := filepath.Join(t.TempDir(), "dep.go")
	if err := os.WriteFile(dependencyFile, []byte("package dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	directoryTarget := resolveGenerationTarget(targetDir)
	if !directoryTarget.matches("example/main", []string{targetFile}) {
		t.Fatal("directory target did not match its package source")
	}
	if directoryTarget.matches("example/dep", []string{dependencyFile}) {
		t.Fatal("directory target matched a dependency")
	}

	fileTarget := resolveGenerationTarget(targetFile)
	if !fileTarget.matches("command-line-arguments", []string{targetFile}) {
		t.Fatal("file target did not match its package source")
	}

	importTarget := resolveGenerationTarget("example.com/project/pkg")
	if !importTarget.matches("example.com/project/pkg", []string{dependencyFile}) {
		t.Fatal("import target did not match its package path")
	}
}
