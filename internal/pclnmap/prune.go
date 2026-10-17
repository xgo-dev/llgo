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

package pclnmap

import (
	"fmt"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
)

// Prune rebuilds an external map from its final, ownership-checked entry and
// PC-line sites. Compile-time records for functions removed by the linker, and
// PC-line records with no surviving site, do not belong in the sidecar. Reusing
// the funcinfo encoder also rebuilds the string pool and symbol hash table.
// The input is not modified; executable identity and final PCs are preserved.
func Prune(data Data) (Data, error) {
	table := data.Table
	symbols := make(map[uint64]uint32, len(data.SymbolIndex))
	for _, entry := range data.SymbolIndex {
		if entry.FuncIndex == 0 || uint64(entry.FuncIndex) > uint64(len(table.Records)) {
			return Data{}, fmt.Errorf("symbol %#x has invalid function index %d", entry.SymbolID, entry.FuncIndex)
		}
		symbols[entry.SymbolID] = entry.FuncIndex
	}
	live := make([]bool, len(table.Records)+1)
	for _, site := range data.EntrySites {
		index := symbols[site.ID]
		if index == 0 {
			return Data{}, fmt.Errorf("entry site %#x has no function record", site.ID)
		}
		live[index] = true
	}
	livePC := make(map[uint64]bool, len(data.PCSites))
	for _, site := range data.PCSites {
		livePC[site.ID] = false // requested, not yet matched to a unique record
	}
	pcLines := make([]funcinfo.PCLineRecord, 0, len(livePC))
	for _, rec := range table.PCLines {
		seen, needed := livePC[rec.ID]
		if !needed {
			continue
		}
		if seen {
			return Data{}, fmt.Errorf("duplicate live PC-line record %#x", rec.ID)
		}
		if rec.Func == 0 || uint64(rec.Func) > uint64(len(table.Records)) {
			return Data{}, fmt.Errorf("PC-line %#x has invalid function index %d", rec.ID, rec.Func)
		}
		livePC[rec.ID] = true
		live[rec.Func] = true
		pcLines = append(pcLines, funcinfo.PCLineRecord{
			ID: rec.ID, Symbol: table.Symbol(table.Records[rec.Func-1]),
			File: table.PCLineFile(rec), Line: rec.Line,
		})
	}
	if len(pcLines) != len(livePC) {
		return Data{}, fmt.Errorf("PC-line sites have %d IDs but %d matching records", len(livePC), len(pcLines))
	}

	liveCount := 0
	for _, used := range live {
		if used {
			liveCount++
		}
	}
	remap := make([]uint32, len(live))
	records := make([]funcinfo.Record, 0, liveCount)
	symbolNames := make(map[string]bool, liveCount)
	for i, rec := range table.Records {
		if !live[i+1] {
			continue
		}
		// EncodeWithPCLines joins by symbol string. Check the compiler's
		// unique, non-empty symbol contract before it can drop or misjoin a
		// retained PC line. Equal display names remain valid for wrappers.
		symbol := table.Symbol(rec)
		if symbol == "" || symbolNames[symbol] {
			return Data{}, fmt.Errorf("live function record %d has empty or duplicate symbol %q", i+1, symbol)
		}
		symbolNames[symbol] = true
		flags := uint32(0)
		if rec.Line&funcinfo.EncodedLineWrapper != 0 {
			flags = funcinfo.RecordFlagWrapper
		}
		records = append(records, funcinfo.Record{
			Symbol: symbol, Name: table.Name(rec), File: table.File(rec),
			Line: rec.Line & funcinfo.EncodedLineMask, Flags: flags,
		})
		remap[i+1] = uint32(len(records))
	}
	pruned, err := funcinfo.EncodeWithPCLines(records, pcLines)
	if err != nil {
		return Data{}, err
	}
	index := make([]SymbolIndexEntry, 0, len(records))
	for _, entry := range data.SymbolIndex {
		if mapped := remap[entry.FuncIndex]; mapped != 0 {
			index = append(index, SymbolIndexEntry{SymbolID: entry.SymbolID, FuncIndex: mapped})
		}
	}
	data.Table = pruned
	data.SymbolIndex = index
	return data, nil
}
