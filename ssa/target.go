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

package ssa

import (
	"runtime"
	"strings"

	"github.com/goplus/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

type Target struct {
	GOOS     string
	GOARCH   string
	GOARM    string // "5", "6", "7" (default)
	Target   string // target name from -target flag (e.g., "esp32", "arm7tdmi", "wasi")
	OptLevel optlevel.Level
}

func (p *Target) targetInfo() (llvm.TargetData, llvm.TargetMachine) {
	spec := p.Spec()
	if spec.Triple == "" {
		spec.Triple = llvm.DefaultTargetTriple()
	}
	t, err := llvm.GetTargetFromTriple(spec.Triple)
	if err != nil {
		panic(err)
	}
	machine := t.CreateTargetMachineWithOptions(
		spec.Triple,
		spec.CPU,
		spec.Features,
		p.codeGenOptLevel(),
		p.targetRelocMode(),
		llvm.CodeModelDefault,
		p.targetMachineOptions(),
	)
	return machine.CreateTargetData(), machine
}

func (p *Target) effectiveOptLevel() optlevel.Level {
	if p != nil && p.OptLevel.IsValid() {
		return p.OptLevel
	}
	if p != nil && p.Target != "" {
		return optlevel.Oz
	}
	return optlevel.O2
}

func (p *Target) codeGenOptLevel() llvm.CodeGenOptLevel {
	switch p.effectiveOptLevel() {
	case optlevel.O0:
		return llvm.CodeGenLevelNone
	case optlevel.O1:
		return llvm.CodeGenLevelLess
	case optlevel.O3:
		return llvm.CodeGenLevelAggressive
	case optlevel.O2, optlevel.Os, optlevel.Oz:
		return llvm.CodeGenLevelDefault
	default:
		return llvm.CodeGenLevelNone
	}
}

func (p *Target) targetRelocMode() llvm.RelocMode {
	if p.useNativeObjectSections() {
		return llvm.RelocPIC
	}
	return llvm.RelocDefault
}

func (p *Target) targetMachineOptions() llvm.TargetMachineOptions {
	if !p.useNativeObjectSections() {
		return llvm.TargetMachineOptions{}
	}
	return llvm.TargetMachineOptions{
		FunctionSections:   true,
		DataSections:       true,
		UniqueSectionNames: true,
	}
}

func (p *Target) useNativeObjectSections() bool {
	goos := p.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := p.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return p.Target == "" && goos == runtime.GOOS && goarch == runtime.GOARCH && goarch != "wasm"
}

type TargetSpec struct {
	Triple   string
	CPU      string
	Features string
}

func (p *Target) Spec() (spec TargetSpec) {
	// Configure based on GOOS/GOARCH environment variables (falling back to
	// runtime.GOOS/runtime.GOARCH), and generate a LLVM target based on it.
	var llvmarch string
	var goarch = p.GOARCH
	var goos = p.GOOS
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goarch {
	case "386":
		llvmarch = "i386"
	case "amd64":
		llvmarch = "x86_64"
	case "arm64":
		llvmarch = "aarch64"
	case "arm":
		switch p.GOARM {
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
		// Use macosx* instead of darwin, otherwise darwin/arm64 will refer
		// to iOS!
		llvmos = "macosx"
		if llvmarch == "aarch64" {
			// Looks like Apple prefers to call this architecture ARM64
			// instead of AArch64.
			llvmarch = "arm64"
			llvmos = "macosx"
		}
		llvmvendor = "apple"
	case "wasip1":
		llvmos = "wasip1"
	}
	// Target triples (which actually have four components, but are called
	// triples for historical reasons) have the form:
	//   arch-vendor-os-environment
	spec.Triple = llvmarch + "-" + llvmvendor + "-" + llvmos
	if llvmos == "windows" {
		spec.Triple += "-gnu"
	} else if goarch == "arm" {
		spec.Triple += "-gnueabihf"
	}
	switch goarch {
	case "386":
		spec.CPU = "pentium4"
		spec.Features = "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"
	case "amd64":
		spec.CPU = "x86-64"
		spec.Features = "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"
	case "arm":
		spec.CPU = "generic"
		switch llvmarch {
		case "armv5":
			spec.Features = "+armv5t,+strict-align,-aes,-bf16,-d32,-dotprod,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fp64,-fpregs,-fullfp16,-mve.fp,-neon,-sha2,-thumb-mode,-vfp2,-vfp2sp,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		case "armv6":
			spec.Features = "+armv6,+dsp,+fp64,+strict-align,+vfp2,+vfp2sp,-aes,-d32,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-neon,-sha2,-thumb-mode,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		case "armv7":
			spec.Features = "+armv7-a,+d32,+dsp,+fp64,+neon,+vfp2,+vfp2sp,+vfp3,+vfp3d16,+vfp3d16sp,+vfp3sp,-aes,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-sha2,-thumb-mode,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		}
	case "arm64":
		spec.CPU = "generic"
		if goos == "darwin" {
			spec.Features = "+neon"
		} else { // windows, linux
			spec.Features = "+neon,-fmv"
		}
	case "wasm":
		spec.CPU = "generic"
		spec.Features = "+bulk-memory,+mutable-globals,+nontrapping-fptoint,+sign-ext"
	}
	return
}

func StripModuleTarget(ir string) string {
	var b strings.Builder
	for _, line := range strings.SplitAfter(ir, "\n") {
		trimmed := strings.TrimSuffix(line, "\n")
		if strings.HasPrefix(trimmed, "target datalayout = ") ||
			strings.HasPrefix(trimmed, "target triple = ") {
			continue
		}
		b.WriteString(line)
	}
	return b.String()
}

// -----------------------------------------------------------------------------
