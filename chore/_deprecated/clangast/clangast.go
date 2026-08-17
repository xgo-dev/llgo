/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/_xtool/clang/parser"
)

var (
	dump = flag.Bool("dump", false, "dump AST")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: clangast [-dump] source.i\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		return
	}
	var file = flag.Arg(0)
	var err error
	if *dump {
		doc, _, e := parser.DumpAST(file, nil)
		if e == nil {
			os.Stdout.Write(doc)
			return
		}
		err = e
	} else {
		doc, _, e := parser.ParseFile(file, 0)
		if e == nil {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			check(enc.Encode(doc))
			return
		}
		err = e
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
