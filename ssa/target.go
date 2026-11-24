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

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

type Target struct {
	GOOS                 string
	GOARCH               string
	GO386                string // "sse2" (default) or "softfloat"
	GOAMD64              string // "v1" (default), "v2", "v3", or "v4"
	GOARM                string // "5", "6", "7" (default), with optional float mode
	GOARM64              string // "v8.0" (default) through "v9.5", with optional extensions
	Target               string // target name from -target flag (e.g., "esp32", "arm7tdmi", "wasi")
	LLVMTarget           string // physical LLVM target selected by a target configuration
	WasmProfile          string // logical hosted WebAssembly profile (j32, j64, or w32)
	WasmProvider         string // hosted WebAssembly provider (gojs, emscripten, or wasi)
	WasmReflectBridges   bool   // emit statically typed reflection bridges for this program
	WasmFuncInfoEntries  bool   // emit table-index symbolization records for this program
	OptLevel             optlevel.Level
	SaturatingFloatToInt bool // enable Go's experimental converthash behavior
}

// IsRuntimeSupportPackage reports whether path is part of the selected Go or
// LLGo runtime implementation rather than an ordinary user/library package.
func IsRuntimeSupportPackage(path string) bool {
	if path == "runtime" || strings.HasPrefix(path, "runtime/internal/") || strings.HasPrefix(path, "internal/runtime/") {
		return true
	}
	runtimeModule := strings.TrimSuffix(PkgRuntime, "/internal/runtime")
	return path == runtimeModule || strings.HasPrefix(path, runtimeModule+"/")
}

func (p *Target) usesWasmReflectBridges() bool {
	if p == nil || p.effectiveGOARCH() != "wasm" {
		return false
	}
	return p.WasmReflectBridges && (p.WasmProvider == "wasi" || p.effectiveGOOS() == "js")
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
	return target.ArchSetting(p.Target, value)
}

// Spec resolves the backend LLVM triple, CPU, and feature list.
// Build validates Go arch settings before constructing Target; Spec also
// accepts hand-built Targets and relies on target.ResolveSpec's documented
// Go-default fallback when invalid settings cannot return an error here.
func (p *Target) Spec() (spec TargetSpec) {
	resolved := target.ResolveSpec(target.Spec{
		GOOS:         p.GOOS,
		GOARCH:       p.GOARCH,
		GO386:        p.GO386,
		GOAMD64:      p.GOAMD64,
		GOARM:        p.GOARM,
		GOARM64:      p.GOARM64,
		Name:         p.Target,
		LLVMTarget:   p.LLVMTarget,
		WasmProfile:  p.WasmProfile,
		WasmProvider: p.WasmProvider,
	})
	return TargetSpec{
		Triple:   resolved.Triple,
		CPU:      resolved.CPU,
		Features: resolved.Features,
	}
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

// ApplyTargetBaseline records an explicit Go architecture level on definitions
// before separate IR consumers (Clang, LTO and the pclntab relink) see them.
func (pkg Package) ApplyTargetBaseline() {
	t := pkg.Prog.Target()
	if t.Target != "" {
		return // named toolchains provide their own baseline
	}
	var cpu, features string
	switch t.effectiveGOARCH() {
	case "amd64":
		if t.GOAMD64 == "" || t.GOAMD64 == "v1" {
			return
		}
		cpu = t.Spec().CPU
	case "arm64":
		if t.GOARM64 == "" || t.GOARM64 == "v8.0" {
			return
		}
		features = t.Spec().Features
	default:
		return
	}
	mod := pkg.Module()
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() {
			continue
		}
		if cpu != "" && fn.GetStringAttributeAtIndex(-1, "target-cpu").IsNil() {
			fn.AddTargetDependentFunctionAttr("target-cpu", cpu)
		}
		if features != "" {
			value := features
			if local := fn.GetStringAttributeAtIndex(-1, "target-features"); !local.IsNil() {
				// LLVM applies features in order. Preserve an operation's
				// explicit overrides after the architecture baseline.
				old := local.GetStringValue()
				if old == features || strings.HasPrefix(old, features+",") {
					continue
				}
				if old != "" {
					value += "," + old
				}
			}
			fn.AddTargetDependentFunctionAttr("target-features", value)
		}
	}
}
