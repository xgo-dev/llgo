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

package target

import (
	"runtime"
	"strings"
)

// IsDarwinTriple reports whether a physical LLVM target uses Apple's ABI.
func IsDarwinTriple(triple string) bool {
	triple = strings.ToLower(triple)
	return strings.Contains(triple, "apple") || strings.Contains(triple, "darwin") || strings.Contains(triple, "macosx")
}

// IsWindowsTriple recognizes native Windows and MinGW triples. Cygwin and MSYS
// use a different ABI.
func IsWindowsTriple(triple string) bool {
	parts := strings.Split(strings.ToLower(triple), "-")
	windows := false
	for _, part := range parts[1:] {
		switch {
		case strings.Contains(part, "cygwin"), part == "cygnus", strings.Contains(part, "msys"):
			return false
		case part == "windows", part == "win32", strings.Contains(part, "mingw"):
			windows = true
		}
	}
	return windows
}

// Triple returns the default LLVM target triple for a Go GOOS/GOARCH pair.
func Triple(goos, goarch string) string {
	return TripleWithGOARM(goos, goarch, "")
}

// TripleWithGOARM returns the LLVM target triple for a Go target, honoring
// GOARM version and softfloat/hardfloat settings when goarch == "arm".
func TripleWithGOARM(goos, goarch, goarm string) string {
	var llvmarch string
	var armConfig ARM
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goarch {
	case "386":
		if goos == "windows" {
			llvmarch = "i686"
		} else {
			llvmarch = "i386"
		}
	case "amd64":
		llvmarch = "x86_64"
	case "arm64":
		llvmarch = "aarch64"
	case "arm":
		armConfig, _ = ParseARM(goarm)
		switch armConfig.Version {
		case "5":
			llvmarch = "armv5"
		case "6":
			llvmarch = "armv6"
		default:
			llvmarch = "armv7"
		}
	case "wasm":
		llvmarch = "wasm32"
	default:
		llvmarch = goarch
	}
	llvmvendor := "unknown"
	llvmos := goos
	switch goos {
	case "darwin":
		llvmos = "macosx"
		if llvmarch == "aarch64" {
			llvmarch = "arm64"
		}
		llvmvendor = "apple"
	case "wasip1":
		llvmos = "wasip1"
	case "windows":
		llvmvendor = "pc"
	}
	triple := llvmarch + "-" + llvmvendor + "-" + llvmos
	if llvmos == "windows" {
		triple += "-msvc"
	} else if goarch == "arm" {
		triple += "-gnueabi"
		if !armConfig.SoftFloat {
			triple += "hf"
		}
	} else if llvmos == "linux" {
		triple += "-gnu"
	}
	return triple
}

// Spec describes an LLGo compilation target independently of LLVM's TargetData.
type Spec struct {
	GOOS         string
	GOARCH       string
	GO386        string
	GOAMD64      string
	GOARM        string
	GOARM64      string
	Name         string // optional named target profile (e.g., "esp32", "wasi")
	LLVMTarget   string // optional physical LLVM triple override
	WasmProfile  string // optional hosted WebAssembly profile ("j32", "j64", "w32")
	WasmProvider string // optional hosted WebAssembly provider ("gojs", "emscripten", "wasi")
	TargetABI    string // optional ABI name (e.g., "lp64d", "ilp32")

	// Resolved ABI / codegen parameters populated by ResolveSpec.
	Triple           string // backend LLVM triple used with host LLVM
	PhysicalTriple   string // physical target triple (prefers LLVMTarget when set, else Triple)
	ArchLevel        string // resolved Go architecture level (e.g. "v1".."v4", "sse2", "softfloat", "5".."7", "v8.0".."v9.5")
	CPU              string
	Features         string
	WordSize         int  // logical Go int/uintptr word size in bytes (4 or 8)
	PtrSize          int  // physical native/C pointer size in bytes (4 or 8)
	MaxAlign         int  // maximum Go struct/scalar alignment in bytes
	HostAlign64      int  // C/host ABI alignment of 64-bit scalars (int64/float64) in bytes
	HostUsesNativeGo bool // true when target uses native host alignment for ordinary Go aggregates
	BigEndian        bool
}

// ArchSetting returns the effective Go architecture environment setting (such
// as GO386, GOAMD64, GOARM, or GOARM64) for a target. Named target profiles
// define their own CPU/feature baseline and therefore ignore Go arch settings.
func ArchSetting(targetName, value string) string {
	if targetName != "" {
		return ""
	}
	return value
}

// UsesWideGoStorage reports whether Go pointers/words are stored in wider slots
// than physical native pointers (e.g. wasm32 hosted profiles where Go word=8, ptr=4).
func (s Spec) UsesWideGoStorage() bool {
	return s.WordSize > s.PtrSize && s.PtrSize > 0
}

// ResolveSpec fills in derived LLVM triple, CPU, target features, word/pointer
// widths, alignment rules, and endianness in pure Go.
//
// Behavioral contracts:
//   - Parse errors in GO386, GOAMD64, GOARM, and GOARM64 are intentionally
//     swallowed and fall back to each resolver's documented Go default. Callers
//     that require strict validation should invoke Resolve386, ResolveAMD64,
//     ParseARM, or ParseARM64 beforehand.
//   - When Name is non-empty (a named target profile), GO386, GOAMD64, GOARM,
//     and GOARM64 are ignored via ArchSetting.
//   - Triple keeps the GOOS/GOARCH-derived backend triple for non-Wasm named
//     targets (because external cross-compilers may support targets such as
//     Xtensa that host LLVM does not register). PhysicalTriple always prefers
//     LLVMTarget when provided and falls back to Triple otherwise.
func ResolveSpec(in Spec) Spec {
	s := in
	if s.GOOS == "" {
		s.GOOS = runtime.GOOS
	}
	if s.GOARCH == "" {
		s.GOARCH = runtime.GOARCH
	}
	goarm := ArchSetting(s.Name, s.GOARM)
	s.Triple = TripleWithGOARM(s.GOOS, s.GOARCH, goarm)
	if s.LLVMTarget != "" && (s.Name == "" || s.WasmProfile != "") {
		s.Triple = s.LLVMTarget
	}
	if s.LLVMTarget != "" {
		s.PhysicalTriple = s.LLVMTarget
	} else {
		s.PhysicalTriple = s.Triple
	}

	switch s.GOARCH {
	case "386":
		s.CPU = "pentium4"
		go386, _ := Resolve386(ArchSetting(s.Name, s.GO386))
		s.ArchLevel = go386
		if go386 == "softfloat" {
			s.Features = "+cx8,+fxsr,+mmx,+soft-float,-sse,-sse2,-x87"
		} else {
			s.Features = "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"
		}
	case "amd64":
		goamd64, _ := ResolveAMD64(ArchSetting(s.Name, s.GOAMD64))
		s.ArchLevel = goamd64
		s.CPU = "x86-64"
		if goamd64 != "v1" {
			s.CPU += "-" + goamd64
		}
		s.Features = "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"
	case "arm":
		s.CPU = "generic"
		arm, _ := ParseARM(goarm)
		s.ArchLevel = arm.Version
		switch arm.Version {
		case "5":
			if arm.SoftFloat {
				s.Features = "+armv5t,+strict-align,-aes,-bf16,-d32,-dotprod,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fp64,-fpregs,-fullfp16,-mve.fp,-neon,-sha2,-thumb-mode,-vfp2,-vfp2sp,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
			} else {
				s.Features = "+armv5t,+strict-align,-aes,-bf16,-d32,-dotprod,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,+fp64,+fpregs,-fullfp16,-mve.fp,-neon,-sha2,-thumb-mode,+vfp2,+vfp2sp,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
			}
		case "6":
			s.Features = "+armv6,+dsp,+fp64,+strict-align,+vfp2,+vfp2sp,-aes,-d32,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-neon,-sha2,-thumb-mode,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		case "7":
			s.Features = "+armv7-a,+d32,+dsp,+fp64,+neon,+vfp2,+vfp2sp,+vfp3,+vfp3d16,+vfp3d16sp,+vfp3sp,-aes,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-sha2,-thumb-mode,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		}
		if arm.SoftFloat {
			s.Features += ",+soft-float"
		}
	case "arm64":
		s.CPU = "generic"
		arm64, _ := ParseARM64(ArchSetting(s.Name, s.GOARM64))
		s.ArchLevel = arm64.Version
		archFeature := arm64.Version + "a"
		if arm64.Version == "v9.0" {
			archFeature = "v9a"
		}
		features := make([]string, 0, 5)
		if arm64.Version != "v8.0" {
			features = append(features, "+"+archFeature)
		}
		features = append(features, "+neon")
		if arm64.LSE {
			features = append(features, "+lse")
		}
		if arm64.Crypto {
			features = append(features, "+crypto")
		}
		if s.GOOS != "darwin" {
			features = append(features, "-fmv")
		}
		s.Features = strings.Join(features, ",")
	case "riscv64":
		if s.Name == "" {
			s.CPU = "generic-rv64"
			s.Features = "+m,+a,+f,+d,+c,+zicsr,+zifencei"
		}
	case "wasm":
		s.CPU = "generic"
		s.Features = "+bulk-memory,+mutable-globals,+nontrapping-fptoint,+sign-ext"
	}

	// Resolve logical Go word size, physical pointer size, and ABI alignment.
	switch s.GOARCH {
	case "386":
		s.WordSize, s.PtrSize, s.MaxAlign, s.HostAlign64 = 4, 4, 4, 4
		if s.GOOS == "windows" {
			s.HostAlign64 = 8
		}
	case "arm", "mips", "mipsle":
		s.WordSize, s.PtrSize, s.MaxAlign, s.HostAlign64 = 4, 4, 4, 8
		s.HostUsesNativeGo = true
	case "wasm":
		s.PtrSize = 4
		s.HostAlign64 = 8
		if s.WasmProfile == "j64" {
			s.PtrSize = 8
			s.WordSize = 8
			s.MaxAlign = 8
		} else if s.WasmProfile == "j32" || s.WasmProfile == "w32" {
			// Hosted Memory32 keeps 64-bit Go words with 32-bit native pointers.
			s.WordSize = 8
			s.MaxAlign = 8
		} else {
			s.WordSize = 4
			s.MaxAlign = 4
		}
	default:
		// 64-bit architectures: amd64, arm64, riscv64, loong64, ppc64, ppc64le, mips64, mips64le, s390x.
		s.WordSize, s.PtrSize, s.MaxAlign, s.HostAlign64 = 8, 8, 8, 8
	}

	switch s.GOARCH {
	case "mips", "mips64", "ppc64", "s390x", "sparc64":
		s.BigEndian = true
	}

	return s
}
