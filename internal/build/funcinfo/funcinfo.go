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

type EncodedRecord struct {
	Symbol uint32
	Name   uint32
	File   uint32
	Line   uint32
	Column uint32
}

type Table struct {
	Records []EncodedRecord
	Strings []byte
	Hash    []uint32
}

func Encode(records []Record) (Table, error) {
	if len(records) == 0 {
		return Table{}, nil
	}
	pool := stringPool{
		offsets: map[string]uint32{"": 0},
		data:    []byte{0},
		text:    "\x00",
	}
	for _, s := range collectStrings(records) {
		if _, err := pool.offset(s); err != nil {
			return Table{}, err
		}
	}
	out := Table{
		Records: make([]EncodedRecord, 0, len(records)),
	}
	for _, rec := range records {
		out.Records = append(out.Records, EncodedRecord{
			Symbol: pool.offsets[rec.Symbol],
			Name:   pool.offsets[rec.Name],
			File:   pool.offsets[rec.File],
			Line:   rec.Line,
			Column: rec.Column,
		})
	}
	out.Strings = pool.data
	out.Hash = buildHash(records)
	return out, nil
}

func collectStrings(records []Record) []string {
	seen := make(map[string]bool)
	for _, rec := range records {
		seen[rec.Symbol] = true
		seen[rec.Name] = true
		seen[rec.File] = true
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

func buildHash(records []Record) []uint32 {
	if len(records) == 0 {
		return nil
	}
	buckets := 1
	for buckets*3 < len(records)*4 {
		buckets <<= 1
	}
	hash := make([]uint32, buckets)
	for i, rec := range records {
		slot := int(HashString(rec.Symbol) & uint32(buckets-1))
		for hash[slot] != 0 {
			slot = (slot + 1) & (buckets - 1)
		}
		hash[slot] = uint32(i + 1)
	}
	return hash
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
