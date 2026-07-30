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

import "testing"

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
