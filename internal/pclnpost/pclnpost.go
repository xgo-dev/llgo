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

package pclnpost

import (
	"encoding/binary"
	"fmt"
)

// Stats summarizes one rewrite.
type Stats struct {
	Format       string
	EntryRecords int
	Kept         int
	InlineCopies int
	NoSymbol     int
	FtabEntries  int
	Buckets      int
}

// Rewrite parses the linked binary's funcinfo site sections, deduplicates
// LTO inline copies against the symbol table, builds the Go-layout prebuilt
// table and rewrites the entry section in place.
// The runtime adopts the table when it sees the magic header and falls back
// to first-use construction otherwise, so failures here leave a fully
// functional binary.
func Rewrite(path string) (Stats, error) {
	var st Stats
	info, err := load(path)
	if err != nil {
		return st, err
	}
	st.Format = info.format
	if len(info.entrySec) >= 8 {
		if m := binary.LittleEndian.Uint64(info.entrySec); m == prebuiltMagic {
			return st, fmt.Errorf("already rewritten")
		}
	}
	entries := parseRecords(info, info.entrySec)
	st.EntryRecords = len(entries)
	if len(entries) == 0 {
		return st, fmt.Errorf("no entry records")
	}
	kept, inline, nosym := dedupe(info, entries, false)
	st.Kept, st.InlineCopies, st.NoSymbol = len(kept), inline, nosym
	if len(kept) == 0 {
		return st, fmt.Errorf("no records survived dedup")
	}
	ftab, buckets, err := writeBack(path, info, kept)
	if err != nil {
		return st, err
	}
	st.FtabEntries, st.Buckets = ftab, buckets
	return st, nil
}
