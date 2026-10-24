//go:build !llgo

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

package debugabi

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/runtime/abi"
)

func TestRecordRoundTrip(t *testing.T) {
	want := NewRecord(8, ByteOrderLittle)
	raw, err := want.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != RecordSize || !bytes.Equal(raw[:8], []byte(recordMagic)) {
		t.Fatalf("record bytes = %x", raw)
	}
	got, err := ParseRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ParseRecord() = %+v, want %+v", got, want)
	}
}

func TestRecordValidation(t *testing.T) {
	if _, err := (Record{}).MarshalBinary(); err == nil {
		t.Fatal("MarshalBinary accepted an empty record")
	}
	valid, err := NewRecord(4, ByteOrderLittle).MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		raw  []byte
		want string
	}{
		{name: "short", raw: valid[:15], want: "size"},
		{name: "magic", raw: mutateRecord(valid, 0, 0), want: "magic"},
		{name: "record version", raw: mutateRecord(valid, 8, 2), want: "record version"},
		{name: "schema version", raw: mutateRecord(valid, 9, 2), want: "schema version"},
		{name: "runtime layout", raw: mutateRecord(valid, 10, 1), want: "runtime layout"},
		{name: "LLGo ABI", raw: mutateRecord(valid, 11, 2), want: "LLGo ABI"},
		{name: "reserved ABI", raw: mutateRecord(valid, 12, 3), want: "reserved"},
		{name: "pointer size", raw: mutateRecord(valid, 13, 16), want: "pointer size"},
		{name: "byte order", raw: mutateRecord(valid, 14, 3), want: "byte order"},
		{name: "reserved", raw: mutateRecord(valid, 15, 1), want: "reserved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseRecord(tt.raw); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ParseRecord() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func mutateRecord(raw []byte, offset int, value byte) []byte {
	copy := bytes.Clone(raw)
	copy[offset] = value
	return copy
}

func TestSchemaV1Contract(t *testing.T) {
	schema, err := ParseSchemaV1()
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Record.LegacySymbols) != 0 {
		t.Fatal("legacy marker alone must not authorize runtime layout inspection")
	}
	if len(schema.Record.Fields) != 8 {
		t.Fatalf("record fields = %d, want 8", len(schema.Record.Fields))
	}
	layout, ok := schema.RuntimeLayouts["2"]
	if !ok {
		t.Fatal("runtime layout v2 is missing")
	}
	var categories map[string]json.RawMessage
	if err := json.Unmarshal(layout, &categories); err != nil {
		t.Fatal(err)
	}
	for _, category := range []string{
		"string", "slice", "interface", "runtime_type", "function",
		"map", "channel", "goroutine",
	} {
		if len(categories[category]) == 0 {
			t.Errorf("runtime layout is missing %q", category)
		}
	}
	var mapLayout struct {
		SameSizeGrowFlag    int `json:"same_size_grow_flag"`
		EvacuatedTophashMin int `json:"evacuated_tophash_min"`
		EvacuatedTophashMax int `json:"evacuated_tophash_max"`
		OccupiedTophashMin  int `json:"occupied_tophash_min"`
	}
	if err := json.Unmarshal(categories["map"], &mapLayout); err != nil {
		t.Fatal(err)
	}
	if mapLayout.SameSizeGrowFlag != 8 ||
		mapLayout.EvacuatedTophashMin != 2 ||
		mapLayout.EvacuatedTophashMax != 4 ||
		mapLayout.OccupiedTophashMin != 5 {
		t.Fatalf("map state constants = %+v", mapLayout)
	}

	var runtimeType struct {
		Kind                string              `json:"kind"`
		DirectInterfaceFlag int                 `json:"direct_interface_flag"`
		UncommonFlag        int                 `json:"uncommon_flag"`
		NamedFlag           int                 `json:"named_flag"`
		KindMask            int                 `json:"kind_mask"`
		PointerKind         int                 `json:"pointer_kind"`
		KindTypes           map[abi.Kind]string `json:"kind_types"`
	}
	if err := json.Unmarshal(categories["runtime_type"], &runtimeType); err != nil {
		t.Fatal(err)
	}
	if runtimeType.Kind != "Kind_" || runtimeType.DirectInterfaceFlag != 32 {
		t.Fatalf("direct interface contract = %+v", runtimeType)
	}
	if runtimeType.UncommonFlag != int(abi.TFlagUncommon) || runtimeType.NamedFlag != int(abi.TFlagNamed) ||
		runtimeType.KindMask != abi.KindMask || runtimeType.PointerKind != int(abi.Pointer) {
		t.Fatalf("runtime type identity contract = %+v", runtimeType)
	}
	for kind, name := range map[abi.Kind]string{
		abi.Array: "ArrayType", abi.Chan: "ChanType", abi.Func: "FuncType", abi.Interface: "InterfaceType",
		abi.Map: "MapType", abi.Pointer: "PtrType", abi.Slice: "SliceType", abi.Struct: "StructType",
	} {
		if got := runtimeType.KindTypes[kind]; got != "github.com/xgo-dev/llgo/runtime/abi."+name {
			t.Errorf("runtime kind %d type = %q, want %s", kind, got, name)
		}
	}

	first := SchemaV1()
	first[0] = 0
	if bytes.Equal(first, SchemaV1()) {
		t.Fatal("SchemaV1 returned shared mutable storage")
	}
}
