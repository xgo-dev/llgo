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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/llgen"
	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

var (
	abi = flag.Int("abi", 0, "ABI mode (default 0). 0 = none, 1 = cfunc, 2 = allfunc.")
)

func main() {
	llvm.SetupPath()
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: llgen [flags] <pkg>")
		return
	}
	llgen.SmartDoFileEx(flag.Args()[0], build.AbiMode(*abi))
}
