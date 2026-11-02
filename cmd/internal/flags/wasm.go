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

package flags

import (
	"flag"
	"fmt"
	"slices"
)

var WasmStrip string

func addWasmFinalFlags(fs *flag.FlagSet) {
	WasmStrip = ""
	add := func(name, usage string, dst *string, values ...string) {
		fs.Func(name, usage, func(value string) error {
			if !slices.Contains(values, value) {
				return fmt.Errorf("invalid %s mode %q (valid: %v)", name, value, values)
			}
			*dst = value
			return nil
		})
	}
	add("wasm-strip", "Final WebAssembly metadata stripping: none, debug, or all (default none)", &WasmStrip, "none", "debug", "all")
}
