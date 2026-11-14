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

// Package debugabi defines the debugger-independent LLGo runtime contract.
package debugabi

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	SchemaVersion        uint8 = 1
	RuntimeLayoutVersion uint8 = 2
	LLGoABIVersion       uint8 = 1
	RecordVersion        uint8 = 1
	RecordSize                 = 16

	NativeRecordSymbol = "__llgo_debugger_abi_v1"
	LegacyMarkerSymbol = "__llgo_debugger_marker_v1"
	WasmSectionName    = "llgo.debugger"

	WasmMemoryAnchorVariable = "__llgo_wasm_memory"
	WasmMemoryAnchorOffset   = 1
)

const recordMagic = "LLGODBG\x00"

// ByteOrder is the target byte order recorded independently of the fixed
// byte-oriented debugger record encoding.
type ByteOrder uint8

const (
	ByteOrderUnknown ByteOrder = iota
	ByteOrderLittle
	ByteOrderBig
)

// Record identifies the schema and the target ABI needed by debugger
// frontends. Every field is one byte so the record has the same encoding on
// native targets and in a WebAssembly custom section.
type Record struct {
	RecordVersion        uint8
	SchemaVersion        uint8
	RuntimeLayoutVersion uint8
	LLGoABIVersion       uint8
	PointerSize          uint8
	ByteOrder            ByteOrder
}

// NewRecord returns the current debugger ABI record for a target.
func NewRecord(pointerSize uint8, byteOrder ByteOrder) Record {
	return Record{
		RecordVersion:        RecordVersion,
		SchemaVersion:        SchemaVersion,
		RuntimeLayoutVersion: RuntimeLayoutVersion,
		LLGoABIVersion:       LLGoABIVersion,
		PointerSize:          pointerSize,
		ByteOrder:            byteOrder,
	}
}

// MarshalBinary serializes the record using the stable v1 byte layout.
func (r Record) MarshalBinary() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	out := make([]byte, RecordSize)
	copy(out, recordMagic)
	out[8] = r.RecordVersion
	out[9] = r.SchemaVersion
	out[10] = r.RuntimeLayoutVersion
	out[11] = r.LLGoABIVersion
	out[13] = r.PointerSize
	out[14] = byte(r.ByteOrder)
	return out, nil
}

// ParseRecord validates and decodes a v1 debugger ABI record.
func ParseRecord(raw []byte) (Record, error) {
	if len(raw) != RecordSize {
		return Record{}, fmt.Errorf("debugger ABI record size is %d, want %d", len(raw), RecordSize)
	}
	if !bytes.Equal(raw[:len(recordMagic)], []byte(recordMagic)) {
		return Record{}, errors.New("invalid debugger ABI record magic")
	}
	if raw[12] != 0 || raw[15] != 0 {
		return Record{}, errors.New("debugger ABI record reserved byte is non-zero")
	}
	r := Record{
		RecordVersion:        raw[8],
		SchemaVersion:        raw[9],
		RuntimeLayoutVersion: raw[10],
		LLGoABIVersion:       raw[11],
		PointerSize:          raw[13],
		ByteOrder:            ByteOrder(raw[14]),
	}
	if err := r.Validate(); err != nil {
		return Record{}, err
	}
	return r, nil
}

// Validate rejects records that this schema cannot describe.
func (r Record) Validate() error {
	if r.RecordVersion != RecordVersion {
		return fmt.Errorf("unsupported debugger ABI record version %d", r.RecordVersion)
	}
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported debugger schema version %d", r.SchemaVersion)
	}
	if r.RuntimeLayoutVersion != RuntimeLayoutVersion {
		return fmt.Errorf("unsupported runtime layout version %d", r.RuntimeLayoutVersion)
	}
	if r.LLGoABIVersion != LLGoABIVersion {
		return fmt.Errorf("unsupported LLGo ABI version %d", r.LLGoABIVersion)
	}
	if r.PointerSize != 4 && r.PointerSize != 8 {
		return fmt.Errorf("invalid pointer size %d", r.PointerSize)
	}
	if r.ByteOrder != ByteOrderLittle && r.ByteOrder != ByteOrderBig {
		return fmt.Errorf("invalid byte order %d", r.ByteOrder)
	}
	return nil
}

// Schema is the stable, language-neutral description consumed by LLDB, GDB,
// and browser adapters.
type Schema struct {
	Contract             string                     `json:"contract"`
	SchemaVersion        uint8                      `json:"schema_version"`
	RuntimeLayoutVersion uint8                      `json:"runtime_layout_version"`
	LLGoABIVersion       uint8                      `json:"llgo_abi_version"`
	Record               SchemaRecord               `json:"record"`
	ByteOrders           map[string]string          `json:"byte_orders"`
	RuntimeLayouts       map[string]json.RawMessage `json:"runtime_layouts"`
}

// SchemaRecord describes how frontends locate and decode Record.
type SchemaRecord struct {
	Version           uint8                     `json:"version"`
	Size              int                       `json:"size"`
	MagicHex          string                    `json:"magic_hex"`
	NativeSymbol      string                    `json:"native_symbol"`
	LegacySymbols     map[string]LegacyContract `json:"legacy_symbols"`
	WasmCustomSection string                    `json:"wasm_custom_section"`
	Fields            []SchemaField             `json:"fields"`
}

// LegacyContract maps the original symbol-only marker to explicit versions.
type LegacyContract struct {
	SchemaVersion        uint8 `json:"schema_version"`
	RuntimeLayoutVersion uint8 `json:"runtime_layout_version"`
	LLGoABIVersion       uint8 `json:"llgo_abi_version"`
}

// SchemaField describes one byte range in Record.
type SchemaField struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
}

//go:embed schema_v1.json
var schemaV1 []byte

// SchemaV1 returns an independent copy of the canonical schema document.
func SchemaV1() []byte {
	return bytes.Clone(schemaV1)
}

// ParseSchemaV1 validates the canonical schema metadata and returns it.
func ParseSchemaV1() (Schema, error) {
	var schema Schema
	if err := json.Unmarshal(schemaV1, &schema); err != nil {
		return Schema{}, err
	}
	magic, err := hex.DecodeString(schema.Record.MagicHex)
	if err != nil {
		return Schema{}, fmt.Errorf("decode debugger ABI magic: %w", err)
	}
	if schema.Contract != "llgo.debugger" ||
		schema.SchemaVersion != SchemaVersion ||
		schema.RuntimeLayoutVersion != RuntimeLayoutVersion ||
		schema.LLGoABIVersion != LLGoABIVersion ||
		schema.Record.Version != RecordVersion ||
		schema.Record.Size != RecordSize ||
		!bytes.Equal(magic, []byte(recordMagic)) ||
		schema.Record.NativeSymbol != NativeRecordSymbol ||
		schema.Record.WasmCustomSection != WasmSectionName {
		return Schema{}, errors.New("canonical debugger schema metadata does not match the v1 contract")
	}
	return schema, nil
}
