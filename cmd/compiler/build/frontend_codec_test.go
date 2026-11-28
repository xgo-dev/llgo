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

package build

import (
	"crypto/sha256"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
)

func TestFrontendCodecRoundTrip(t *testing.T) {
	export := &frontendExport{
		Version: 1,
		Input:   "pkg_input_key",
		Archive: "pkg_archive_key",
		Functions: []frontendFunction{
			{
				ID:       "pkg.Func1",
				CacheKey: "func1_cache_key",
				Facts: cl.FrontendFacts{
					CallerBase:       true,
					CallerFrame:      true,
					RecoverPanicSite: true,
					UsesRecover:      true,
					RecoverScope:     true,
					InlineCost:       42,
				},
				Escape: &goescape.Summary{
					Parameters: []goescape.Parameter{
						{
							Heap:    1,
							Mutator: 2,
							Callee:  3,
							Results: []goescape.Effect{4, 5},
						},
					},
				},
			},
			{
				ID:       "pkg.Func2",
				CacheKey: "func2_cache_key",
				Facts: cl.FrontendFacts{
					InlineCost: -1,
				},
			},
		},
		Types: frontcache.TypeGraph{
			Nodes: []frontcache.TypeNode{
				{Kind: "basic", Basic: 1, Object: frontcache.Object{Package: "builtin", Name: "int"}, Universe: true},
				{Kind: "pointer", Elem: 0},
				{Kind: "slice", Elem: 0},
				{Kind: "array", Length: 10, Elem: 0, NativeLayout: true},
				{Kind: "chan", Direction: 3, Elem: 0},
				{Kind: "map", Base: 0, Elem: 1},
				{
					Kind:         "struct",
					NativeLayout: true,
					Fields: []frontcache.Object{
						{Name: "Field1", Package: "pkg", Tag: []byte("tag1"), Type: 0},
						{Name: "field2", Tag: []byte("tag2"), Type: 1, Embedded: true},
					},
				},
				{Kind: "tuple", Types: []int{0, 1}, Elements: []frontcache.Object{{Name: "a", Position: "p.go:10"}, {Name: "b"}}},
				{
					Kind:       "func",
					CallConv:   1,
					Variadic:   true,
					RecvParams: []int{0},
					TypeParams: []int{1},
					Params:     []int{0},
					Parameters: []frontcache.Object{{Name: "x"}},
					Results:    []int{1},
					Returns:    []frontcache.Object{{Name: "y"}},
					Receiver:   &frontcache.Object{Name: "r", Package: "pkg", Type: 6},
				},
				{
					Kind:   "named",
					Object: frontcache.Object{Name: "NamedType", Package: "pkg"},
					Elem:   6,
					Types:  []int{0},
					Methods: []frontcache.Object{
						{Name: "Method1", Type: 8},
					},
				},
				{
					Kind:       "alias",
					Object:     frontcache.Object{Name: "AliasType", Package: "pkg"},
					Elem:       0,
					CallConv:   2,
					TypeParams: []int{0},
				},
				{
					Kind:     "interface",
					Implicit: true,
					Types:    []int{0},
					Methods:  []frontcache.Object{{Name: "Close", Type: 8}},
				},
				{
					Kind:  "union",
					Types: []int{0, 1},
					Tilde: []bool{true, false},
				},
				{
					Kind:   "simdvec",
					Basic:  1,
					Length: 4,
					IsMask: true,
				},
				{
					Kind:   "parameter",
					Object: frontcache.Object{Name: "T", Package: "pkg"},
					Elem:   0,
				},
			},
		},
		RuntimeTypes: []int{0, 6, 9},
		Link: &frontendLinkExport{
			NeedABI:       2,
			LocalExports:  true,
			MethodIndexes: []int{1, 2, 3},
			MethodNames:   []string{"M1", "M2"},
			ABISymbols:    []string{"sym1", "sym2"},
			Types: frontcache.TypeGraph{
				Nodes: []frontcache.TypeNode{
					{Kind: "basic", Basic: 1, Object: frontcache.Object{Name: "int"}},
				},
			},
			ABI: []frontendABIType{
				{Name: "type.int", Type: 0},
			},
			CExports: []frontendCExport{
				{
					GoName:    "c_func",
					CName:     "llgo_c_func",
					Signature: 0,
				},
			},
		},
	}

	data, err := marshalFrontendExport(export)
	if err != nil {
		t.Fatalf("marshalFrontendExport failed: %v", err)
	}

	decoded, err := unmarshalFrontendExport(data)
	if err != nil {
		t.Fatalf("unmarshalFrontendExport failed: %v", err)
	}

	if decoded.Version != export.Version {
		t.Errorf("version = %d, want %d", decoded.Version, export.Version)
	}
	if decoded.Input != export.Input {
		t.Errorf("input = %q, want %q", decoded.Input, export.Input)
	}
	if decoded.Archive != export.Archive {
		t.Errorf("archive = %q, want %q", decoded.Archive, export.Archive)
	}
	if len(decoded.Functions) != len(export.Functions) {
		t.Fatalf("functions len = %d, want %d", len(decoded.Functions), len(export.Functions))
	}
	for i := range export.Functions {
		want := export.Functions[i]
		got := decoded.Functions[i]
		if got.ID != want.ID || got.CacheKey != want.CacheKey || got.Facts != want.Facts {
			t.Errorf("function %d mismatch: got %+v, want %+v", i, got, want)
		}
		if (want.Escape == nil) != (got.Escape == nil) {
			t.Fatalf("function %d escape nil mismatch", i)
		}
		if want.Escape != nil && !reflect.DeepEqual(want.Escape, got.Escape) {
			t.Errorf("function %d escape mismatch: got %+v, want %+v", i, got.Escape, want.Escape)
		}
	}

	if len(decoded.Types.Nodes) != len(export.Types.Nodes) {
		t.Fatalf("type nodes count: got %d, want %d", len(decoded.Types.Nodes), len(export.Types.Nodes))
	}
	for i := range export.Types.Nodes {
		want := export.Types.Nodes[i]
		got := decoded.Types.Nodes[i]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("node %d (%s) mismatch:\ngot  %+v\nwant %+v", i, want.Kind, got, want)
		}
	}

	if !reflect.DeepEqual(decoded.RuntimeTypes, export.RuntimeTypes) {
		t.Errorf("runtimeTypes mismatch: got %v, want %v", decoded.RuntimeTypes, export.RuntimeTypes)
	}

	if (export.Link == nil) != (decoded.Link == nil) {
		t.Fatalf("link nil mismatch")
	}
	if export.Link != nil {
		if !reflect.DeepEqual(export.Link, decoded.Link) {
			t.Errorf("link export mismatch:\ngot  %+v\nwant %+v", decoded.Link, export.Link)
		}
	}
}

func TestFrontendCodecCorruptionAndErrors(t *testing.T) {
	// 1. Nil export
	if _, err := marshalFrontendExport(nil); err == nil {
		t.Error("expected error marshaling nil export")
	}

	// 2. Short header
	if _, err := unmarshalFrontendExport([]byte("short")); err == nil {
		t.Error("expected error unmarshaling short header")
	}

	// 3. Bad magic
	badMagic := make([]byte, frontendExportHeaderSize+10)
	copy(badMagic, "WRONGMAG")
	if _, err := unmarshalFrontendExport(badMagic); err == nil {
		t.Error("expected error unmarshaling bad magic")
	}

	// 4. Valid payload with corrupted checksum
	validExport := &frontendExport{
		Version: 1,
		Input:   "valid",
	}
	validData, err := marshalFrontendExport(validExport)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := make([]byte, len(validData))
	copy(corrupted, validData)
	corrupted[len(corrupted)-1] ^= 0xff
	if _, err := unmarshalFrontendExport(corrupted); err == nil {
		t.Error("expected checksum mismatch error on tampered data")
	}

	// 5. Truncated valid payload with valid header checksum of original
	truncated := corrupted[:frontendExportHeaderSize+2]
	if _, err := unmarshalFrontendExport(truncated); err == nil {
		t.Error("expected error on truncated payload")
	}

	// 6. Unknown node kind in TypeGraph
	badNodeData := make([]byte, frontendExportHeaderSize)
	copy(badNodeData, frontendExportMagic)
	var badPayload []byte
	badPayload = appendVarint(badPayload, 1)
	badPayload = appendStr(badPayload, "in")
	badPayload = appendStr(badPayload, "ar")
	badPayload = appendUvarint(badPayload, 0)
	badPayload = appendUvarint(badPayload, 1) // 1 node
	badPayload = appendStr(badPayload, "unknown_kind")
	badPayload = appendStr(badPayload, "key")
	badPayload = append(badPayload, 0) // universe
	// checksum
	sum := sha256.Sum256(badPayload)
	copy(badNodeData[len(frontendExportMagic):], sum[:])
	badNodeData = append(badNodeData, badPayload...)

	if _, err := unmarshalFrontendExport(badNodeData); err == nil {
		t.Error("expected error on unknown type node kind")
	}
}

func appendVarint(buf []byte, v int64) []byte {
	var tmp [10]byte
	n := binaryPutVarint(tmp[:], v)
	return append(buf, tmp[:n]...)
}

func appendUvarint(buf []byte, v uint64) []byte {
	var tmp [10]byte
	n := binaryPutUvarint(tmp[:], v)
	return append(buf, tmp[:n]...)
}

func appendStr(buf []byte, s string) []byte {
	buf = appendUvarint(buf, uint64(len(s)))
	return append(buf, s...)
}

func binaryPutVarint(buf []byte, x int64) int {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return binaryPutUvarint(buf, ux)
}

func binaryPutUvarint(buf []byte, x uint64) int {
	i := 0
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return i + 1
}
