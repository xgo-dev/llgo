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
	// CarrierBytesRemoved is the number of bytes cut from the metadata carrier
	// before an existing Mach-O code signature is regenerated. It is not the
	// signed output file's net size change.
	CarrierBytesRemoved uint64
}

// Rewrite parses the linked binary's funcinfo site sections, deduplicates
// LTO inline copies against the symbol table, builds the Go-layout prebuilt
// table, stages a compact executable image, and atomically replaces the input.
// The runtime adopts the table when it sees the magic header. Raw-site
// carriers support first-use construction; symbol-only carriers require a
// successful rewrite. Any failure leaves the input byte-for-byte unchanged.
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
	entries, err := entryRecords(path, info)
	if err != nil {
		return st, err
	}
	st.EntryRecords = len(entries)
	if len(entries) == 0 {
		return st, fmt.Errorf("no entry records")
	}
	kept, inline, nosym := dedupe(info, entries, false)
	st.Kept, st.InlineCopies, st.NoSymbol = len(kept), inline, nosym
	if len(kept) == 0 {
		return st, fmt.Errorf("no records survived dedup")
	}
	ftab, buckets, removed, err := writeBack(path, info, kept)
	if err != nil {
		return st, err
	}
	st.FtabEntries, st.Buckets, st.CarrierBytesRemoved = ftab, buckets, removed
	return st, nil
}
