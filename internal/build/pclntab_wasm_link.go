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
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/pclnpost"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func canPruneWasmPCLN(ctx *context) bool {
	// WASI links the module directly. Emscripten owns a group of companion
	// artifacts, which must be staged and published together when relinking.
	return ctx.buildConf.Goarch == "wasm" && ctx.prog != nil &&
		ctx.prog.Target().WasmProvider == "wasi" && !ctx.prog.Target().WasmFuncInfoEntries
}

// analyzeWasmPCLN uses the linked function names, never Wasm function indexes as
// stable metadata identities. Wasm has no native PC-line anchors; inline shadow
// frames carry their own locations and retain their function's address.
func analyzeWasmPCLN(path string, original pclnmap.Data) (pclnpost.ExternalAnalysis, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return pclnpost.ExternalAnalysis{}, err
	}
	names, err := wasmdebug.FunctionNames(raw)
	if err != nil {
		return pclnpost.ExternalAnalysis{}, err
	}
	complete, err := wasmdebug.FunctionNamesComplete(raw, names)
	if err != nil {
		return pclnpost.ExternalAnalysis{}, err
	}
	live := make(map[string]bool, len(names))
	for _, name := range names {
		live[name] = true
		// LLVM may promote or clone a symbol during LTO. Retaining its
		// source record is conservative even if another suffix is incidental.
		for at := strings.LastIndexByte(name, '.'); at >= 0; at = strings.LastIndexByte(name, '.') {
			name = name[:at]
			live[name] = true
		}
	}
	analysis := pclnpost.ExternalAnalysis{PointerSize: original.PointerSize}
	for i, rec := range original.Table.Records {
		symbol := original.Table.Symbol(rec)
		// Missing, empty or sparse maps provide no complete liveness evidence.
		if !complete || live[symbol] {
			analysis.EntrySites = append(analysis.EntrySites, pclnpost.ExternalSite{
				PCOffset: uint64(i + 1), ID: funcInfoSymbolID(symbol),
			})
		}
	}
	return analysis, nil
}
