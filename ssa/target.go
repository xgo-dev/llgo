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

	archcfg "github.com/xgo-dev/llgo/internal/goarch"
	"github.com/xgo-dev/llgo/internal/optlevel"
	intllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

type Target struct {
	GOOS                    string
	GOARCH                  string
	GO386                   string // "sse2" (default) or "softfloat"
	GOAMD64                 string // "v1" (default), "v2", "v3", or "v4"
	GOARM                   string // "5", "6", "7" (default), with optional float mode
	GOARM64                 string // "v8.0" (default) through "v9.5", with optional extensions
	Target                  string // target name from -target flag (e.g., "esp32", "arm7tdmi", "wasi")
	LLVMTarget              string // physical LLVM target selected by a target configuration
	BuildTags               string // comma-separated Go build tags used for the target runtime
	OptLevel                optlevel.Level
	SaturatingFloatToUint32 bool
}

func (p *Target) effectiveGOOS() string {
	if p.GOOS == "" {
		return runtime.GOOS
	}
	return p.GOOS
}

func (p *Target) effectiveGOARCH() string {
	if p.GOARCH == "" {
		return runtime.GOARCH
	}
	return p.GOARCH
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
		return optlevel.TargetDefault
	}
	return optlevel.Default
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
	goos := p.effectiveGOOS()
	goarch := p.effectiveGOARCH()
	return p.Target == "" && goos == runtime.GOOS && goarch == runtime.GOARCH && goarch != "wasm"
}

type TargetSpec struct {
	Triple   string
	CPU      string
	Features string
}

func (p *Target) goArchitectureSetting(value string) string {
	if p.Target != "" {
		return ""
	}
	return value
}

func (p *Target) Spec() (spec TargetSpec) {
	// Configure based on GOOS/GOARCH environment variables (falling back to
	// runtime.GOOS/runtime.GOARCH), and generate a LLVM target based on it.
	goarch := p.effectiveGOARCH()
	goos := p.effectiveGOOS()
	goarm := p.goArchitectureSetting(p.GOARM)
	spec.Triple = intllvm.GetTargetTripleWithGOARM(goos, goarch, goarm)
	// Build validates these settings before constructing Target. Spec also
	// accepts hand-built Targets, so it intentionally uses each resolver's
	// documented Go-default fallback when its error cannot be returned here.
	switch goarch {
	case "386":
		spec.CPU = "pentium4"
		go386, _ := archcfg.Resolve386(p.goArchitectureSetting(p.GO386))
		if go386 == "softfloat" {
			spec.Features = "+cx8,+fxsr,+mmx,+soft-float,-sse,-sse2,-x87"
		} else {
			spec.Features = "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"
		}
	case "amd64":
		goamd64, _ := archcfg.ResolveAMD64(p.goArchitectureSetting(p.GOAMD64))
		spec.CPU = "x86-64"
		if goamd64 != "v1" {
			spec.CPU += "-" + goamd64
		}
		spec.Features = "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"
	case "arm":
		spec.CPU = "generic"
		arm, _ := archcfg.ParseARM(goarm)
		switch arm.Version {
		case "5":
			if arm.SoftFloat {
				spec.Features = "+armv5t,+strict-align,-aes,-bf16,-d32,-dotprod,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fp64,-fpregs,-fullfp16,-mve.fp,-neon,-sha2,-thumb-mode,-vfp2,-vfp2sp,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
			} else {
				// GOARM=5,hardfloat explicitly enables VFPv2 without also
				// carrying contradictory disable tokens for the same features.
				spec.Features = "+armv5t,+strict-align,-aes,-bf16,-d32,-dotprod,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,+fp64,+fpregs,-fullfp16,-mve.fp,-neon,-sha2,-thumb-mode,+vfp2,+vfp2sp,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
			}
		case "6":
			spec.Features = "+armv6,+dsp,+fp64,+strict-align,+vfp2,+vfp2sp,-aes,-d32,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-neon,-sha2,-thumb-mode,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		case "7":
			spec.Features = "+armv7-a,+d32,+dsp,+fp64,+neon,+vfp2,+vfp2sp,+vfp3,+vfp3d16,+vfp3d16sp,+vfp3sp,-aes,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-sha2,-thumb-mode,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"
		}
		if arm.SoftFloat {
			spec.Features += ",+soft-float"
		}
	case "arm64":
		spec.CPU = "generic"
		arm64, _ := archcfg.ParseARM64(p.goArchitectureSetting(p.GOARM64))
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
		if goos != "darwin" { // windows, linux
			features = append(features, "-fmv")
		}
		spec.Features = strings.Join(features, ",")
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
