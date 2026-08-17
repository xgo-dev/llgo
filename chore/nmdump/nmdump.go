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
	"fmt"
	"log"
	"os"

	"github.com/xgo-dev/llgo/xtool/env/llvm"
	nmtool "github.com/xgo-dev/llgo/xtool/nm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: nmdump [flags] libfile")
		return
	}

	nm := llvm.New("").Nm()

	var flags []string
	libfile := os.Args[len(os.Args)-1]
	if len(os.Args) > 2 {
		flags = os.Args[1 : len(os.Args)-1]
	}

	items, err := nm.List(libfile, flags...)

	for _, item := range items {
		if item.File != "" {
			fmt.Printf("\n%s:\n", item.File)
		}
		for _, sym := range item.Symbols {
			var versionInfo string
			switch sym.VersionType {
			case nmtool.VersionSpecific:
				versionInfo = fmt.Sprintf("@%s", sym.Version)
			case nmtool.VersionDefault:
				versionInfo = fmt.Sprintf("@@%s", sym.Version)
			}
			if sym.FAddr {
				fmt.Printf("%016x %c %s%s\n", sym.Addr, sym.Type, sym.Name, versionInfo)
			} else {
				fmt.Printf("%16s %c %s%s\n", "", sym.Type, sym.Name, versionInfo)
			}
		}
	}
	if err != nil {
		log.Println(err)
	}
}
