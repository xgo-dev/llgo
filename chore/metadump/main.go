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

	"github.com/goplus/llgo/internal/metadata"
)

func main() {
	var out string
	flag.StringVar(&out, "o", "", "write decoded metadata to file")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: metadump [-o output] file.meta\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(flag.Arg(0), out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input, output string) error {
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open %s: %w", input, err)
	}
	defer f.Close()

	pm, err := metadata.ReadMeta(f)
	if err != nil {
		return fmt.Errorf("read %s: %w", input, err)
	}
	text := metadata.MetaString(pm)

	if output == "" {
		fmt.Print(text)
		return nil
	}
	if err := os.WriteFile(output, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}
	return nil
}
