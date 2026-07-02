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

import "testing"

func TestEncodePoolsStringsAndBuildsHash(t *testing.T) {
	table, err := Encode([]Record{
		{Symbol: "example.com/p.a", Name: "example.com/p.A", File: "/src/p/shared.go", Line: 10, Column: 1},
		{Symbol: "example.com/p.b", Name: "example.com/p.B", File: "shared.go", Line: 20, Column: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Records) != 2 {
		t.Fatalf("encoded records = %d, want 2", len(table.Records))
	}
	if table.Records[0].FileRoot == table.Records[1].FileRoot {
		t.Fatalf("distinct file roots should use distinct ids")
	}
	if got := table.File(table.Records[1]); got != "shared.go" {
		t.Fatalf("suffix file string = %q, want shared.go", got)
	}
	if len(table.Hash) == 0 || len(table.Hash)&(len(table.Hash)-1) != 0 {
		t.Fatalf("hash bucket count = %d, want power-of-two non-zero", len(table.Hash))
	}
	if idx, ok := table.LookupSymbol("example.com/p.a"); !ok || idx != 0 {
		t.Fatalf("lookup a = %d, %v; want 0, true", idx, ok)
	}
	if idx, ok := table.LookupSymbol("example.com/p.b"); !ok || idx != 1 {
		t.Fatalf("lookup b = %d, %v; want 1, true", idx, ok)
	}
	if _, ok := table.LookupSymbol("missing"); ok {
		t.Fatalf("lookup missing succeeded")
	}
}

func TestEncodeWithPCLines(t *testing.T) {
	table, err := EncodeWithPCLines(
		[]Record{
			{Symbol: "example.com/p.f", Name: "example.com/p.F", File: "/src/p/f.go", Line: 10, Column: 1},
			{Symbol: "example.com/p.g", Name: "example.com/p.G", File: "/src/p/g.go", Line: 20, Column: 1},
		},
		[]PCLineRecord{
			{ID: 3, Symbol: "missing", File: "missing.go", Line: 30},
			{ID: 2, Symbol: "example.com/p.g", File: "/src/p/call_g.go", Line: 22},
			{ID: 1, Symbol: "example.com/p.f", File: "/src/p/call_f.go", Line: 12},
			{ID: 0, Symbol: "example.com/p.f", File: "zero.go", Line: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.PCLines) != 2 {
		t.Fatalf("encoded pclines = %d, want 2", len(table.PCLines))
	}
	if got := table.PCLines[0]; got.ID != 1 || got.Func != 1 || got.Line != 12 {
		t.Fatalf("first pcline = %+v, want id 1 func 1 line 12", got)
	}
	if got := table.PCLines[1]; got.ID != 2 || got.Func != 2 || got.Line != 22 {
		t.Fatalf("second pcline = %+v, want id 2 func 2 line 22", got)
	}
	if got := table.PCLineFile(table.PCLines[0]); got != "/src/p/call_f.go" {
		t.Fatalf("pcline file = %q, want /src/p/call_f.go", got)
	}
}

func TestEncodeRoundTripsSingleRecord(t *testing.T) {
	table, err := Encode([]Record{{Symbol: "s", Name: "n", File: "f", Line: 1, Column: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(table.Records), 1; got != want {
		t.Fatalf("records = %d, want %d", got, want)
	}
	rec := table.Records[0]
	if got, want := table.Symbol(rec), "s"; got != want {
		t.Fatalf("symbol = %q, want %q", got, want)
	}
	if got, want := table.Name(rec), "n"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := table.File(rec), "f"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if rec.Line != 1 {
		t.Fatalf("source line = %d, want 1", rec.Line)
	}
}

func TestEncodeHashHandlesCollisions(t *testing.T) {
	a, b := collisionPair(t)
	table, err := Encode([]Record{
		{Symbol: a, Name: a, File: "a.go"},
		{Symbol: b, Name: b, File: "b.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if idx, ok := table.LookupSymbol(a); !ok || idx != 0 {
		t.Fatalf("lookup collision a = %d, %v; want 0, true", idx, ok)
	}
	if idx, ok := table.LookupSymbol(b); !ok || idx != 1 {
		t.Fatalf("lookup collision b = %d, %v; want 1, true", idx, ok)
	}
}

func TestEncodeOmitsHashWhenRecordIndexesDoNotFitUint16(t *testing.T) {
	records := make([]Record, 1<<16)
	for i := range records {
		records[i] = Record{Symbol: "example.com/p.f", Name: "example.com/p.F"}
	}
	table, err := Encode(records)
	if err != nil {
		t.Fatal(err)
	}
	if table.Hash != nil {
		t.Fatalf("hash buckets = %d, want nil fallback for oversized table", len(table.Hash))
	}
	if len(table.Records) != len(records) {
		t.Fatalf("records = %d, want %d", len(table.Records), len(records))
	}
}

func TestEncodeSplitsPackageAndFilePrefixes(t *testing.T) {
	records := []Record{
		{Symbol: "example.com/p.alpha", Name: "example.com/p.Alpha", File: "/home/me/mod/p/alpha.go", Line: 10},
		{Symbol: "example.com/p.beta", Name: "example.com/p.Beta", File: "/home/me/mod/p/beta.go", Line: 20},
		{Symbol: "example.com/q.gamma", Name: "example.com/q.Gamma", File: "/home/me/mod/q/gamma.go", Line: 30},
	}
	table, err := Encode(records)
	if err != nil {
		t.Fatal(err)
	}
	for i, rec := range table.Records {
		if got := table.Symbol(rec); got != records[i].Symbol {
			t.Fatalf("record %d symbol = %q, want %q", i, got, records[i].Symbol)
		}
		if got := table.Name(rec); got != records[i].Name {
			t.Fatalf("record %d name = %q, want %q", i, got, records[i].Name)
		}
		if got := table.File(rec); got != records[i].File {
			t.Fatalf("record %d file = %q, want %q", i, got, records[i].File)
		}
	}
	if table.Records[0].SymbolPkg != table.Records[1].SymbolPkg {
		t.Fatalf("same package prefix got different ids: %d vs %d", table.Records[0].SymbolPkg, table.Records[1].SymbolPkg)
	}
	if table.Records[0].FileRoot != table.Records[1].FileRoot {
		t.Fatalf("same file root got different ids: %d vs %d", table.Records[0].FileRoot, table.Records[1].FileRoot)
	}
	if got := table.SizeBytes(); got >= legacySizeBytes(records) {
		t.Fatalf("compressed table size = %d, want below legacy %d", got, legacySizeBytes(records))
	}
}

func TestLookupPCUsesPageIndex(t *testing.T) {
	entries := []uint64{0x1000, 0x1010, 0x2800, 0x4000, 0x4010}
	index := BuildPCIndex(entries)
	tests := []struct {
		pc   uint64
		want int
	}{
		{0xfff, -1},
		{0x1000, 0},
		{0x100f, 0},
		{0x1010, 1},
		{0x27ff, 1},
		{0x2800, 2},
		{0x4018, 4},
	}
	for _, tt := range tests {
		if got := LookupPC(entries, index, tt.pc); got != tt.want {
			t.Fatalf("LookupPC(%#x) = %d, want %d", tt.pc, got, tt.want)
		}
	}
}

func BenchmarkLookupPCRandom(b *testing.B) {
	entries := make([]uint64, 8192)
	for i := range entries {
		entries[i] = 0x100000 + uint64(i)*37
	}
	index := BuildPCIndex(entries)
	var sum int
	for i := 0; i < b.N; i++ {
		pc := entries[(i*1103515245+12345)&(len(entries)-1)] + uint64(i&31)
		sum += LookupPC(entries, index, pc)
	}
	if sum == 0 {
		b.Fatal(sum)
	}
}

func collisionPair(t *testing.T) (string, string) {
	t.Helper()
	const mask = uint32(3)
	seen := make(map[uint32]string)
	for i := 0; i < 100; i++ {
		s := string(rune('a' + i))
		slot := HashString(s) & mask
		if prev, ok := seen[slot]; ok {
			return prev, s
		}
		seen[slot] = s
	}
	t.Fatal("failed to find hash collision")
	return "", ""
}

func legacySizeBytes(records []Record) int {
	seen := make(map[string]bool)
	stringsBytes := 1
	for _, rec := range records {
		for _, s := range []string{rec.Symbol, rec.Name, rec.File} {
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			stringsBytes += len(s) + 1
		}
	}
	buckets := 1
	for buckets*3 < len(records)*4 {
		buckets <<= 1
	}
	return len(records)*20 + stringsBytes + buckets*4
}
