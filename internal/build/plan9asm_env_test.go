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
	"testing"

	"github.com/goplus/llgo/internal/cabi"
	"golang.org/x/tools/go/packages"
)

func TestParsePlan9AsmPkgsEnv(t *testing.T) {
	tests := []struct {
		raw  string
		mode plan9asmPkgsEnvMode
		pkgs map[string]bool
	}{
		{raw: "", mode: plan9asmEnvDefaults},
		{raw: " off ", mode: plan9asmEnvNone},
		{raw: "FALSE", mode: plan9asmEnvNone},
		{raw: "*", mode: plan9asmEnvAll},
		{raw: "On", mode: plan9asmEnvAll},
		{raw: "runtime, net ; crypto\t net\n", mode: plan9asmEnvSelected, pkgs: map[string]bool{"runtime": true, "net": true, "crypto": true}},
	}
	for _, tt := range tests {
		got := parsePlan9AsmPkgsEnv(tt.raw)
		if got.mode != tt.mode {
			t.Fatalf("parsePlan9AsmPkgsEnv(%q).mode = %v, want %v", tt.raw, got.mode, tt.mode)
		}
		if len(got.pkgs) != len(tt.pkgs) {
			t.Fatalf("parsePlan9AsmPkgsEnv(%q).pkgs = %v, want %v", tt.raw, got.pkgs, tt.pkgs)
		}
		for pkg := range tt.pkgs {
			if !got.pkgs[pkg] {
				t.Fatalf("parsePlan9AsmPkgsEnv(%q) did not retain %q", tt.raw, pkg)
			}
		}
	}
}

func TestPlan9AsmEnvironmentHelpers(t *testing.T) {
	t.Setenv(llgoPlan9ASMPkgs, "off")
	if !plan9asmDisabledByEnv() {
		t.Fatal("plan9asmDisabledByEnv = false, want true")
	}
	if plan9asmEnabledByEnv("runtime") {
		t.Fatal("plan9asmEnabledByEnv(runtime) = true with disabled environment")
	}

	t.Setenv(llgoPlan9ASMPkgs, "runtime")
	if plan9asmDisabledByEnv() {
		t.Fatal("plan9asmDisabledByEnv = true for selected package")
	}
	if !plan9asmEnabledByEnv("runtime") {
		t.Fatal("plan9asmEnabledByEnv(runtime) = false, want true")
	}
	if plan9asmEnabledByEnv("reflect") {
		t.Fatal("plan9asmEnabledByEnv(reflect) = true, want false")
	}

	t.Setenv(llgoPlan9ASMPkgs, "all")
	if !plan9asmEnabledByEnv("example.com/pkg") {
		t.Fatal("plan9asmEnabledByEnv(example.com/pkg) = false with all enabled")
	}
}

func TestPlan9AsmSignatureCacheBoundaries(t *testing.T) {
	if got, err := plan9asmSigsForPkg(nil, "runtime"); err != nil || got != nil {
		t.Fatalf("plan9asmSigsForPkg(nil) = (%v, %v), want (nil, nil)", got, err)
	}

	disabled := &context{
		buildConf:     &Config{Goarch: "amd64"},
		plan9asmReady: true,
		plan9asmMode:  plan9asmEnvNone,
	}
	got, err := plan9asmSigsForPkg(disabled, "runtime")
	if err != nil || len(got) != 0 {
		t.Fatalf("disabled plan9asmSigsForPkg = (%v, %v), want empty signatures", got, err)
	}
	if cached := disabled.plan9asmSigs["runtime"]; cached == nil {
		t.Fatal("disabled package signatures were not cached")
	}
	if again, err := plan9asmSigsForPkg(disabled, "runtime"); err != nil || len(again) != 0 {
		t.Fatalf("cached plan9asmSigsForPkg = (%v, %v), want empty signatures", again, err)
	}

	missing := &context{
		buildConf:     &Config{Goarch: "amd64"},
		plan9asmReady: true,
		plan9asmMode:  plan9asmEnvAll,
	}
	got, err = plan9asmSigsForPkg(missing, "example.com/missing")
	if err != nil || len(got) != 0 {
		t.Fatalf("missing package plan9asmSigsForPkg = (%v, %v), want empty signatures", got, err)
	}
	if (&context{}).hasAltPkgWithResolvedPlan9("runtime") {
		t.Fatal("context without a build configuration resolved an alternate package")
	}

	resolvedAlt := &context{
		buildConf:     &Config{Goarch: "amd64", AbiMode: cabi.ModeAllFunc},
		plan9asmReady: true,
		plan9asmMode:  plan9asmEnvAll,
	}
	got, err = plan9asmSigsForPkg(resolvedAlt, "runtime")
	if err != nil || len(got) != 0 {
		t.Fatalf("alternate package plan9asmSigsForPkg = (%v, %v), want empty signatures", got, err)
	}
}

func TestPlan9AsmPolicyBoundaries(t *testing.T) {
	additive := &context{buildConf: &Config{Goarch: "amd64", AbiMode: cabi.ModeAllFunc}}
	if !additive.hasAltPkgWithResolvedPlan9("internal/runtime/sys") {
		t.Fatal("additive alternate package was not resolved")
	}

	replacement := &context{buildConf: &Config{Goarch: "amd64", AbiMode: cabi.ModeAllFunc}}
	if !replacement.hasAltPkgWithResolvedPlan9("runtime") {
		t.Fatal("replacement alternate package was not resolved in all-function ABI mode")
	}

	nonAll := &context{
		buildConf:    &Config{Goarch: "amd64"},
		plan9asmMode: plan9asmEnvAll,
	}
	if nonAll.hasAltPkgWithResolvedPlan9("runtime") {
		t.Fatal("explicit all Plan9 asm policy retained the replacement package")
	}
	nonAll.plan9asmMode = plan9asmEnvSelected
	nonAll.plan9asmPkgs = map[string]bool{"runtime": true}
	if nonAll.hasAltPkgWithResolvedPlan9("runtime") {
		t.Fatal("selected Plan9 asm policy retained the replacement package")
	}
	nonAll.plan9asmPkgs = nil
	if !nonAll.hasAltPkgWithResolvedPlan9("runtime") {
		t.Fatal("unselected Plan9 asm policy did not resolve the replacement package")
	}

	t.Setenv(llgoPlan9ASMPkgs, "runtime")
	selected := &context{buildConf: &Config{Goarch: "amd64"}}
	if !selected.plan9asmEnabled("runtime") || selected.plan9asmEnabled("reflect") {
		t.Fatal("selected Plan9 asm policy was not initialized from the environment")
	}

	frozen := &context{sfilesFrozen: true}
	if _, err := pkgSFiles(frozen, &packages.Package{ID: "p", PkgPath: "example.com/p"}); err == nil {
		t.Fatal("frozen assembly file cache accepted an unprepared package")
	}
}
