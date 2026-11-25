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

package types

import (
	"strings"

	"github.com/xgo-dev/llgo/target"
)

// NativeIntExt describes how a target's native C ABI requires narrow integer
// arguments to carry signext / zeroext attributes at call boundaries.
type NativeIntExt uint8

const (
	NativeIntNoExt     NativeIntExt = iota
	NativeIntNarrowExt              // extend integers narrower than 32 bits (Darwin arm64, SysV amd64, x86, wasm, arm, riscv32)
	NativeIntBoolExt                // zeroext bool only (Win64 amd64)
	NativeIntRV64Ext                // signext both signed and unsigned 32-bit ints + narrow ints (riscv64)
)

// NativeIntExtForSpec determines the C ABI narrow-integer extension policy for
// spec without inspecting LLVM IR. Named embedded targets may use GOARCH only
// for package selection (for example, RISC-V profiles use arm), so PhysicalTriple
// / LLVMTarget / Triple takes precedence over GOARCH/GOOS when present.
func NativeIntExtForSpec(spec target.Spec) NativeIntExt {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	triple := spec.PhysicalTriple
	if triple == "" {
		triple = spec.LLVMTarget
	}
	if triple == "" {
		triple = spec.Triple
	}
	if triple != "" {
		triple = strings.ToLower(triple)
		arch, _, _ := strings.Cut(triple, "-")
		switch {
		case arch == "arm64", arch == "arm64_32", arch == "aarch64", arch == "aarch64_be":
			if target.IsDarwinTriple(triple) {
				return NativeIntNarrowExt
			}
			return NativeIntNoExt
		case arch == "x86_64", arch == "amd64":
			if target.IsWindowsTriple(triple) {
				return NativeIntBoolExt
			}
			return NativeIntNarrowExt
		case arch == "riscv64":
			return NativeIntRV64Ext
		case arch == "riscv32", arch == "wasm32", arch == "wasm64",
			arch == "x86", arch == "386", arch == "i386", arch == "i486", arch == "i586", arch == "i686",
			strings.HasPrefix(arch, "arm"), strings.HasPrefix(arch, "thumb"):
			return NativeIntNarrowExt
		default:
			return NativeIntNoExt
		}
	}
	arch := strings.ToLower(spec.GOARCH)
	goos := strings.ToLower(spec.GOOS)
	switch arch {
	case "arm64":
		if goos == "darwin" || goos == "ios" {
			return NativeIntNarrowExt
		}
		return NativeIntNoExt
	case "amd64":
		if goos == "windows" {
			return NativeIntBoolExt
		}
		return NativeIntNarrowExt
	case "riscv64":
		return NativeIntRV64Ext
	case "riscv32", "wasm", "386", "arm":
		return NativeIntNarrowExt
	}
	return NativeIntNoExt
}
