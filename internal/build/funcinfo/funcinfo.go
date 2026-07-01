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

package funcinfo

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Record struct {
	Symbol string
	Name   string
	File   string
	Line   uint32
	Column uint32
}

type PCLineRecord struct {
	ID     uint64
	Symbol string
	File   string
	Line   uint32
	Column uint32
}

type EncodedRecord struct {
	SymbolPkg  uint16
	SymbolName uint16
	NamePkg    uint16
	NameName   uint16
	FileRoot   uint16
	FileName   uint16
	Line       uint32
}

type EncodedPCLineRecord struct {
	ID   uint64
	Func uint32
	File uint32
	Line uint32
}

type Table struct {
	Records       []EncodedRecord
	PCLines       []EncodedPCLineRecord
	StringOffsets []uint32
	Strings       []byte
	Hash          []uint16
}

func Encode(records []Record) (Table, error) {
	return EncodeWithPCLines(records, nil)
}

func EncodeWithPCLines(records []Record, pcLines []PCLineRecord) (Table, error) {
	funcIndex := make(map[string]uint32, len(records))
	for i, rec := range records {
		if rec.Symbol != "" {
			funcIndex[rec.Symbol] = uint32(i + 1)
		}
	}
	filteredPCLines := make([]PCLineRecord, 0, len(pcLines))
	for _, rec := range pcLines {
		if rec.ID == 0 || funcIndex[rec.Symbol] == 0 {
			continue
		}
		filteredPCLines = append(filteredPCLines, rec)
	}
	if len(records) == 0 && len(filteredPCLines) == 0 {
		return Table{}, nil
	}
	ids, offsets, strings, err := buildStringTable(collectStrings(records, filteredPCLines))
	if err != nil {
		return Table{}, err
	}
	out := Table{
		Records:       make([]EncodedRecord, 0, len(records)),
		StringOffsets: offsets,
		Strings:       strings,
	}
	for _, rec := range records {
		symPkg, symName := splitQualifiedName(rec.Symbol)
		namePkg, nameName := splitQualifiedName(rec.Name)
		fileRoot, fileName := splitFileName(rec.File)
		out.Records = append(out.Records, EncodedRecord{
			SymbolPkg:  ids[symPkg],
			SymbolName: ids[symName],
			NamePkg:    ids[namePkg],
			NameName:   ids[nameName],
			FileRoot:   ids[fileRoot],
			FileName:   ids[fileName],
			Line:       rec.Line,
		})
	}
	out.PCLines = make([]EncodedPCLineRecord, 0, len(filteredPCLines))
	for _, rec := range filteredPCLines {
		idx := funcIndex[rec.Symbol]
		fileRoot, fileName := splitFileName(rec.File)
		out.PCLines = append(out.PCLines, EncodedPCLineRecord{
			ID:   rec.ID,
			Func: idx,
			File: packStringIDs(ids[fileRoot], ids[fileName]),
			Line: rec.Line,
		})
	}
	sort.Slice(out.PCLines, func(i, j int) bool {
		return out.PCLines[i].ID < out.PCLines[j].ID
	})
	out.Hash, err = buildHash(records)
	if err != nil {
		return Table{}, err
	}
	return out, nil
}

func collectStrings(records []Record, pcLines []PCLineRecord) []string {
	seen := make(map[string]bool)
	for _, rec := range records {
		for _, s := range splitRecordStrings(rec) {
			seen[s] = true
		}
	}
	for _, rec := range pcLines {
		fileRoot, fileName := splitFileName(rec.File)
		seen[fileRoot] = true
		seen[fileName] = true
	}
	delete(seen, "")
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) > len(out[j])
		}
		return out[i] < out[j]
	})
	return out
}

func packStringIDs(hi, lo uint16) uint32 {
	return uint32(hi)<<16 | uint32(lo)
}

func splitRecordStrings(rec Record) []string {
	symPkg, symName := splitQualifiedName(rec.Symbol)
	namePkg, nameName := splitQualifiedName(rec.Name)
	fileRoot, fileName := splitFileName(rec.File)
	return []string{symPkg, symName, namePkg, nameName, fileRoot, fileName}
}

func buildStringTable(strings []string) (map[string]uint16, []uint32, []byte, error) {
	ids := map[string]uint16{"": 0}
	values := []string{""}
	for _, s := range strings {
		if _, ok := ids[s]; ok {
			continue
		}
		if len(values) > math.MaxUint16 {
			return nil, nil, nil, fmt.Errorf("funcinfo string id table exceeds 65535 entries")
		}
		ids[s] = uint16(len(values))
		values = append(values, s)
	}
	pool := stringPool{
		offsets: map[string]uint32{"": 0},
		data:    []byte{0},
		text:    "\x00",
	}
	offsets := make([]uint32, len(values))
	for id, s := range values {
		off, err := pool.offset(s)
		if err != nil {
			return nil, nil, nil, err
		}
		offsets[id] = off
	}
	return ids, offsets, pool.data, nil
}

func splitQualifiedName(name string) (pkg, local string) {
	if name == "" {
		return "", ""
	}
	start := strings.LastIndexByte(name, '/')
	if start < 0 {
		start = 0
	} else {
		start++
	}
	if dot := strings.IndexByte(name[start:], '.'); dot >= 0 {
		idx := start + dot
		return name[:idx], name[idx+1:]
	}
	return "", name
}

func splitFileName(file string) (root, name string) {
	if file == "" {
		return "", ""
	}
	if slash := strings.LastIndexByte(file, '/'); slash >= 0 {
		return file[:slash+1], file[slash+1:]
	}
	return "", file
}

type stringPool struct {
	offsets map[string]uint32
	data    []byte
	text    string
}

func (p *stringPool) offset(s string) (uint32, error) {
	if off, ok := p.offsets[s]; ok {
		return off, nil
	}
	if off := strings.Index(p.text, s+"\x00"); off >= 0 {
		uoff := uint32(off)
		p.offsets[s] = uoff
		return uoff, nil
	}
	if len(p.data)+len(s)+1 > math.MaxUint32 {
		return 0, fmt.Errorf("funcinfo string table exceeds 4 GiB")
	}
	off := uint32(len(p.data))
	p.data = append(p.data, s...)
	p.data = append(p.data, 0)
	p.text = string(p.data)
	p.offsets[s] = off
	return off, nil
}

func buildHash(records []Record) ([]uint16, error) {
	if len(records) == 0 {
		return nil, nil
	}
	if len(records) > math.MaxUint16 {
		// Runtime hash slots store 1-based uint16 record indexes. Larger
		// tables remain correct by omitting the hash and using linear lookup.
		return nil, nil
	}
	buckets := 1
	for buckets*3 < len(records)*4 {
		buckets <<= 1
	}
	hash := make([]uint16, buckets)
	for i, rec := range records {
		slot := int(HashString(rec.Symbol) & uint32(buckets-1))
		for hash[slot] != 0 {
			slot = (slot + 1) & (buckets - 1)
		}
		hash[slot] = uint16(i + 1)
	}
	return hash, nil
}

func HashString(s string) uint32 {
	const (
		offset = uint32(2166136261)
		prime  = uint32(16777619)
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	return h
}

func (t Table) String(id uint16) string {
	if int(id) >= len(t.StringOffsets) {
		return ""
	}
	return cstring(t.Strings, t.StringOffsets[id])
}

func (t Table) Symbol(rec EncodedRecord) string {
	return joinQualified(t.String(rec.SymbolPkg), t.String(rec.SymbolName))
}

func (t Table) Name(rec EncodedRecord) string {
	return joinQualified(t.String(rec.NamePkg), t.String(rec.NameName))
}

func (t Table) File(rec EncodedRecord) string {
	return t.String(rec.FileRoot) + t.String(rec.FileName)
}

func (t Table) PCLineFile(rec EncodedPCLineRecord) string {
	return t.String(uint16(rec.File>>16)) + t.String(uint16(rec.File))
}

func (t Table) LookupSymbol(symbol string) (int, bool) {
	if len(t.Hash) == 0 {
		return 0, false
	}
	mask := uint32(len(t.Hash) - 1)
	slot := HashString(symbol) & mask
	for probes := 0; probes < len(t.Hash); probes++ {
		idx := t.Hash[slot]
		if idx == 0 {
			return 0, false
		}
		rec := t.Records[idx-1]
		if t.Symbol(rec) == symbol {
			return int(idx - 1), true
		}
		slot = (slot + 1) & mask
	}
	return 0, false
}

func (t Table) SizeBytes() int {
	return len(t.Records)*16 + len(t.PCLines)*24 + len(t.StringOffsets)*4 + len(t.Strings) + len(t.Hash)*2
}

func joinQualified(pkg, local string) string {
	if pkg == "" {
		return local
	}
	if local == "" {
		return pkg
	}
	return pkg + "." + local
}

func cstring(data []byte, off uint32) string {
	end := int(off)
	for end < len(data) && data[end] != 0 {
		end++
	}
	return string(data[off:end])
}

type PCIndex struct {
	PageShift uint
	Base      uint64
	Pages     []uint32
}

const DefaultPCPageShift = 12

func BuildPCIndex(entries []uint64) PCIndex {
	return BuildPCIndexWithShift(entries, DefaultPCPageShift)
}

func BuildPCIndexWithShift(entries []uint64, shift uint) PCIndex {
	if len(entries) == 0 {
		return PCIndex{PageShift: shift}
	}
	base := entries[0] >> shift
	last := entries[len(entries)-1] >> shift
	pages := make([]uint32, last-base+2)
	next := 0
	for page := range pages {
		limit := (base + uint64(page)) << shift
		for next < len(entries) && entries[next] < limit {
			next++
		}
		pages[page] = uint32(next)
	}
	return PCIndex{
		PageShift: shift,
		Base:      base,
		Pages:     pages,
	}
}

func LookupPC(entries []uint64, index PCIndex, pc uint64) int {
	if len(entries) == 0 {
		return -1
	}
	lo, hi := 0, len(entries)
	page := pc >> index.PageShift
	if len(index.Pages) != 0 && page >= index.Base {
		off := page - index.Base
		if off < uint64(len(index.Pages)) {
			lo = int(index.Pages[off])
			if off+1 < uint64(len(index.Pages)) {
				hi = int(index.Pages[off+1])
			}
			if lo > 0 {
				lo--
			}
			if hi < len(entries) {
				hi++
			}
		}
	}
	i := sort.Search(hi-lo, func(i int) bool {
		return entries[lo+i] > pc
	})
	idx := lo + i - 1
	if idx < 0 {
		return -1
	}
	return idx
}
