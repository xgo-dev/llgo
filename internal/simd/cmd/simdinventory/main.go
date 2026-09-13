/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Command simdinventory audits selected Go 1.27 archsimd source declarations.
//
// Generate a complete audit (not normally committed):
//
//	go run ./internal/simd/cmd/simdinventory -goroot /path/to/go -out inventory.json
//
// Check the compact pinned baseline:
//
//	go run ./internal/simd/cmd/simdinventory -goroot /path/to/go \
//	  -check-summary internal/simd/testdata/go1.27-inventory-summary.json
//
// Compare a new source tree with a complete previous audit. Changes report
// added/removed keys and changed signatures, independently of source hashes:
//
//	go run ./internal/simd/cmd/simdinventory -goroot /path/to/new/go -check inventory.json
//
// Regenerate the compact baseline only from the reviewed reference source:
//
//	go run ./internal/simd/cmd/simdinventory -goroot /path/to/pinned/go \
//	  -summary -out internal/simd/testdata/go1.27-inventory-summary.json
//
// reference_revision identifies the review baseline, not the actual GOROOT
// revision. Source content hashes make a modified tree auditable. This command
// neither invokes an LLVM compiler nor establishes SIMD lowering support.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/xgo-dev/llgo/internal/simd"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("simdinventory", flag.ContinueOnError)
	goroot := flags.String("goroot", runtime.GOROOT(), "selected Go 1.27 source root")
	out := flags.String("out", "", "output JSON path (default stdout)")
	summary := flags.Bool("summary", false, "emit compact source/API digests and counts")
	check := flags.String("check", "", "compare against an earlier complete inventory")
	checkSummary := flags.String("check-summary", "", "compare against a compact inventory baseline")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("simdinventory: unexpected positional arguments")
	}
	if *check != "" && *checkSummary != "" {
		return fmt.Errorf("simdinventory: use only one of -check and -check-summary")
	}
	if (*check != "" || *checkSummary != "") && (*summary || *out != "") {
		return fmt.Errorf("simdinventory: check mode cannot also write an inventory")
	}
	inv, err := simd.ScanInventory(*goroot)
	if err != nil {
		return err
	}
	var changes []simd.InventoryChange
	switch {
	case *check != "":
		var before simd.Inventory
		if err := readJSON(*check, &before); err != nil {
			return err
		}
		changes = simd.CompareInventories(&before, inv)
	case *checkSummary != "":
		var before simd.InventorySummary
		if err := readJSON(*checkSummary, &before); err != nil {
			return err
		}
		changes = simd.CompareInventorySummaries(before, inv.Summary())
	}
	if *check != "" || *checkSummary != "" {
		if len(changes) == 0 {
			_, err := fmt.Fprintln(stdout, "SIMD source inventory matches; no LLVM support is implied.")
			return err
		}
		if err := writeJSON(stdout, changes); err != nil {
			return err
		}
		return fmt.Errorf("simdinventory: %d source/API changes; review before regenerating the baseline", len(changes))
	}
	var value any = inv
	if *summary {
		value = inv.Summary()
	}
	if *out == "" {
		return writeJSON(stdout, value)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(data, '\n'), 0o644)
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func readJSON(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("simdinventory: %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("simdinventory: %s: expected a single JSON document", path)
	}
	return nil
}
