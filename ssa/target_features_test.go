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

package ssa

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestTargetArchitectureFeatures(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		wantCPU  string
		wantFeat string
	}{
		{name: "386 default", target: Target{GOOS: "windows", GOARCH: "386"}, wantCPU: "pentium4", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "386 softfloat", target: Target{GOOS: "windows", GOARCH: "386", GO386: "softfloat"}, wantCPU: "pentium4", wantFeat: "+cx8,+fxsr,+mmx,+soft-float,-sse,-sse2,-x87"},
		{name: "amd64 v1", target: Target{GOOS: "windows", GOARCH: "amd64"}, wantCPU: "x86-64", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "amd64 v2", target: Target{GOOS: "windows", GOARCH: "amd64", GOAMD64: "v2"}, wantCPU: "x86-64-v2", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "amd64 v3", target: Target{GOOS: "windows", GOARCH: "amd64", GOAMD64: "v3"}, wantCPU: "x86-64-v3", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "amd64 v4", target: Target{GOOS: "windows", GOARCH: "amd64", GOAMD64: "v4"}, wantCPU: "x86-64-v4", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "arm default", target: Target{GOOS: "linux", GOARCH: "arm"}, wantCPU: "generic", wantFeat: "+armv7-a,+d32,+dsp,+fp64,+neon,+vfp2,+vfp2sp,+vfp3,+vfp3d16,+vfp3d16sp,+vfp3sp,-aes,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-sha2,-thumb-mode,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"},
		{name: "arm6 softfloat", target: Target{GOOS: "linux", GOARCH: "arm", GOARM: "6,softfloat"}, wantCPU: "generic", wantFeat: "+armv6,+dsp,+fp64,+strict-align,+vfp2,+vfp2sp,-aes,-d32,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-neon,-sha2,-thumb-mode,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp,+soft-float"},
		{name: "arm5 hardfloat", target: Target{GOOS: "linux", GOARCH: "arm", GOARM: "5,hardfloat"}, wantCPU: "generic", wantFeat: "+armv5t,+strict-align,-aes,-bf16,-d32,-dotprod,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,+fp64,+fpregs,-fullfp16,-mve.fp,-neon,-sha2,-thumb-mode,+vfp2,+vfp2sp,-vfp3,-vfp3d16,-vfp3d16sp,-vfp3sp,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"},
		{name: "arm64 default", target: Target{GOOS: "windows", GOARCH: "arm64"}, wantCPU: "generic", wantFeat: "+neon,-fmv"},
		{name: "arm64 v8.1", target: Target{GOOS: "windows", GOARCH: "arm64", GOARM64: "v8.1"}, wantCPU: "generic", wantFeat: "+v8.1a,+neon,+lse,-fmv"},
		{name: "arm64 v9 crypto", target: Target{GOOS: "windows", GOARCH: "arm64", GOARM64: "v9.0,crypto"}, wantCPU: "generic", wantFeat: "+v9a,+neon,+lse,+crypto,-fmv"},
		{name: "darwin arm64", target: Target{GOOS: "darwin", GOARCH: "arm64", GOARM64: "v8.2"}, wantCPU: "generic", wantFeat: "+v8.2a,+neon,+lse"},
		{name: "named 386 target unchanged", target: Target{GOOS: "linux", GOARCH: "386", GO386: "softfloat", Target: "custom"}, wantCPU: "pentium4", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "named amd64 target unchanged", target: Target{GOOS: "linux", GOARCH: "amd64", GOAMD64: "v4", Target: "custom"}, wantCPU: "x86-64", wantFeat: "+cx8,+fxsr,+mmx,+sse,+sse2,+x87"},
		{name: "named arm target unchanged", target: Target{GOOS: "linux", GOARCH: "arm", GOARM: "5,hardfloat", Target: "custom"}, wantCPU: "generic", wantFeat: "+armv7-a,+d32,+dsp,+fp64,+neon,+vfp2,+vfp2sp,+vfp3,+vfp3d16,+vfp3d16sp,+vfp3sp,-aes,-fp-armv8,-fp-armv8d16,-fp-armv8d16sp,-fp-armv8sp,-fp16,-fp16fml,-fullfp16,-sha2,-thumb-mode,-vfp4,-vfp4d16,-vfp4d16sp,-vfp4sp"},
		{name: "named arm64 target unchanged", target: Target{GOOS: "linux", GOARCH: "arm64", GOARM64: "v9.5,crypto", Target: "custom"}, wantCPU: "generic", wantFeat: "+neon,-fmv"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := test.target.Spec()
			if spec.CPU != test.wantCPU || spec.Features != test.wantFeat {
				t.Fatalf("target spec = CPU %q, features %q; want CPU %q, features %q", spec.CPU, spec.Features, test.wantCPU, test.wantFeat)
			}
		})
	}
}

func TestTargetSpecUsesResolvedLLVMTarget(t *testing.T) {
	target := &Target{
		GOOS:       "windows",
		GOARCH:     "amd64",
		LLVMTarget: "x86_64-w64-windows-gnu",
	}
	if got := target.Spec().Triple; got != target.LLVMTarget {
		t.Fatalf("target triple = %q, want resolved LLVM target %q", got, target.LLVMTarget)
	}

	prog := NewProgram(target)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")
	if got := pkg.Module().Target(); got != target.LLVMTarget {
		t.Fatalf("module target = %q, want %q", got, target.LLVMTarget)
	}
}

func TestNamedWasmProfileUsesResolvedLLVMTarget(t *testing.T) {
	target := &Target{
		GOOS:        "js",
		GOARCH:      "wasm",
		Target:      "emscripten-memory64",
		LLVMTarget:  "wasm64-unknown-emscripten",
		WasmProfile: "j64",
	}
	if got := target.Spec().Triple; got != target.LLVMTarget {
		t.Fatalf("target triple = %q, want resolved wasm profile target %q", got, target.LLVMTarget)
	}

	// Non-wasm named embedded targets intentionally retain their established
	// backend selection behavior.
	embedded := &Target{GOOS: "linux", GOARCH: "arm", Target: "custom", LLVMTarget: "thumbv6m-unknown-unknown-eabi"}
	if got := embedded.Spec().Triple; got == embedded.LLVMTarget {
		t.Fatalf("generic named target unexpectedly used physical LLVM target %q", got)
	}

}

func TestArchitectureFeatureTargetMachines(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "windows", GOARCH: "386", GO386: "softfloat"},
		{GOOS: "windows", GOARCH: "amd64", GOAMD64: "v4"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "6,softfloat"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5,hardfloat"},
		{GOOS: "windows", GOARCH: "arm64", GOARM64: "v9.5,crypto"},
	} {
		prog := NewProgram(target)
		pkg := prog.NewPackage("p", "example.com/p")
		pkg.NewFunc("example.com/p.f", NoArgsNoRet, InGo).MakeBody(1).Return()
		buf, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.ObjectFile)
		if err != nil {
			prog.Dispose()
			t.Fatalf("emit %s/%s with CPU/features %q/%q: %v", target.GOOS, target.GOARCH, target.Spec().CPU, target.Spec().Features, err)
		}
		buf.Dispose()
		prog.Dispose()
	}
}
