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
	"fmt"
	"slices"

	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/wasmresume"
	"github.com/xgo-dev/llvm"
)

const wasmResumeBuildTag = "llgo.wasm_resume"

func configureWasmResume(conf *Config, export *crosscompile.Export) error {
	if !IsWasmResumeEnabled() {
		return nil
	}
	if conf == nil || conf.Goarch != "wasm" {
		return fmt.Errorf("%s requires GOARCH=wasm", llgoWasmResume)
	}
	if conf.Goos != "js" && conf.Goos != "wasip1" {
		return fmt.Errorf("%s does not support GOOS=%s", llgoWasmResume, conf.Goos)
	}
	if IsWasiThreadsEnabled() {
		return fmt.Errorf("%s is incompatible with %s", llgoWasmResume, llgoWasiThreads)
	}
	if !slices.Contains(export.BuildTags, wasmResumeBuildTag) {
		export.BuildTags = append(export.BuildTags, wasmResumeBuildTag)
	}
	export.WasmPostLink.Asyncify = false
	export.LDFLAGS = slices.DeleteFunc(export.LDFLAGS, func(flag string) bool {
		return flag == "-sASYNCIFY=1"
	})
	return nil
}

func lowerWasmResumeModule(ctx *context, mod llvm.Module) error {
	if ctx == nil || !ctx.prog.WasmResumeABIEnabled() {
		return nil
	}
	if err := wasmresume.Lower(mod, ctx.prog.TargetData()); err != nil {
		return fmt.Errorf("lower WebAssembly resumable ABI: %w", err)
	}
	return nil
}
