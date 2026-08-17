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
	"io"
	"log"
	"os"

	"github.com/xgo-dev/llgo/xtool/ar"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ardump xxx.a")
		return
	}

	f, err := os.Open(os.Args[1])
	check(err)
	defer f.Close()

	r, err := ar.NewReader(f)
	check(err)
	for {
		hdr, err := r.Next()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			break
		}
		fmt.Println(hdr.Name)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
