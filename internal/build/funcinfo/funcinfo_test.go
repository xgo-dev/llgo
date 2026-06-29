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
	if table.Records[0].File == table.Records[1].File {
		t.Fatalf("suffix sharing should not collapse distinct file strings to the same offset")
	}
	if got := cstring(table.Strings, table.Records[1].File); got != "shared.go" {
		t.Fatalf("suffix file string = %q, want shared.go", got)
	}
	if len(table.Hash) == 0 || len(table.Hash)&(len(table.Hash)-1) != 0 {
		t.Fatalf("hash bucket count = %d, want power-of-two non-zero", len(table.Hash))
	}
	if idx, ok := lookup(table, "example.com/p.a"); !ok || idx != 0 {
		t.Fatalf("lookup a = %d, %v; want 0, true", idx, ok)
	}
	if idx, ok := lookup(table, "example.com/p.b"); !ok || idx != 1 {
		t.Fatalf("lookup b = %d, %v; want 1, true", idx, ok)
	}
	if _, ok := lookup(table, "missing"); ok {
		t.Fatalf("lookup missing succeeded")
	}
}

func TestEncodeUsesUint32Records(t *testing.T) {
	table, err := Encode([]Record{{Symbol: "s", Name: "n", File: "f", Line: 1, Column: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(table.Records), 1; got != want {
		t.Fatalf("records = %d, want %d", got, want)
	}
	rec := table.Records[0]
	if got, want := cstring(table.Strings, rec.Symbol), "s"; got != want {
		t.Fatalf("symbol = %q, want %q", got, want)
	}
	if got, want := cstring(table.Strings, rec.Name), "n"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := cstring(table.Strings, rec.File), "f"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if rec.Line != 1 || rec.Column != 2 {
		t.Fatalf("source position = %d:%d, want 1:2", rec.Line, rec.Column)
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
	if idx, ok := lookup(table, a); !ok || idx != 0 {
		t.Fatalf("lookup collision a = %d, %v; want 0, true", idx, ok)
	}
	if idx, ok := lookup(table, b); !ok || idx != 1 {
		t.Fatalf("lookup collision b = %d, %v; want 1, true", idx, ok)
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

func cstring(data []byte, off uint32) string {
	end := int(off)
	for end < len(data) && data[end] != 0 {
		end++
	}
	return string(data[off:end])
}

func lookup(table Table, symbol string) (int, bool) {
	if len(table.Hash) == 0 {
		return 0, false
	}
	mask := uint32(len(table.Hash) - 1)
	slot := HashString(symbol) & mask
	for probes := 0; probes < len(table.Hash); probes++ {
		idx := table.Hash[slot]
		if idx == 0 {
			return 0, false
		}
		rec := table.Records[idx-1]
		if cstring(table.Strings, rec.Symbol) == symbol {
			return int(idx - 1), true
		}
		slot = (slot + 1) & mask
	}
	return 0, false
}
