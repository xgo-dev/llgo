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

// Command pclnpost rewrites a linked LLGo binary's funcinfo entry section
// with the link-phase prebuilt ftab/findfunctab (see
// doc/design/pclntab-linkphase.md and internal/pclnpost). llgo build runs
// the same rewrite automatically; this command exists for manual inspection
// and re-processing.
package main

import (
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/internal/pclnpost"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: pclnpost <linked-binary>")
		os.Exit(2)
	}
	st, err := pclnpost.Rewrite(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "pclnpost:", err)
		os.Exit(1)
	}
	fmt.Printf("%s: entry=%d kept=%d inlineCopies=%d noSymbol=%d -> ftab=%d buckets=%d carrierRemoved=%d\n",
		st.Format, st.EntryRecords, st.Kept, st.InlineCopies, st.NoSymbol, st.FtabEntries, st.Buckets, st.CarrierBytesRemoved)
}
