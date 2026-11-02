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

// Package wasmdebug implements the WebAssembly tool-conventions packaging for
// embedded and external DWARF custom sections.
package wasmdebug

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xgo-dev/llgo/internal/debugabi"
)

const externalDebugInfo = "external_debug_info"
const buildIDSection = "build_id"

var wasmHeader = []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

type section struct {
	raw     []byte
	id      byte
	name    string
	content []byte
}

func readULEB32(raw []byte, off *int) (uint32, error) {
	var value uint32
	for shift := uint(0); shift < 35; shift += 7 {
		if *off >= len(raw) {
			return 0, errors.New("truncated WebAssembly varuint32")
		}
		b := raw[*off]
		(*off)++
		if shift == 28 && b > 0x0f {
			return 0, errors.New("WebAssembly varuint32 overflows")
		}
		value |= uint32(b&0x7f) << shift
		if b&0x80 == 0 {
			return value, nil
		}
	}
	return 0, errors.New("invalid WebAssembly varuint32")
}

func appendULEB32(dst []byte, value uint32) []byte {
	for {
		b := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		dst = append(dst, b)
		if value == 0 {
			return dst
		}
	}
}

func readName(raw []byte, off *int) (string, error) {
	size, err := readULEB32(raw, off)
	if err != nil {
		return "", err
	}
	if uint64(size) > uint64(len(raw)-*off) {
		return "", errors.New("truncated WebAssembly name")
	}
	name := raw[*off : *off+int(size)]
	*off += int(size)
	if !utf8.Valid(name) {
		return "", errors.New("invalid UTF-8 WebAssembly name")
	}
	return string(name), nil
}

func parse(raw []byte) ([]section, error) {
	if len(raw) < len(wasmHeader) || !bytes.Equal(raw[:len(wasmHeader)], wasmHeader) {
		return nil, errors.New("invalid WebAssembly header")
	}
	var sections []section
	for off := len(wasmHeader); off < len(raw); {
		start := off
		id := raw[off]
		off++
		size, err := readULEB32(raw, &off)
		if err != nil {
			return nil, err
		}
		if uint64(size) > uint64(len(raw)-off) {
			return nil, errors.New("truncated WebAssembly section")
		}
		end := off + int(size)
		entry := section{raw: raw[start:end], id: id, content: raw[off:end]}
		if id == 0 {
			payloadOff := off
			entry.name, err = readName(raw[:end], &payloadOff)
			if err != nil {
				return nil, fmt.Errorf("invalid WebAssembly custom section: %w", err)
			}
			entry.content = raw[payloadOff:end]
		}
		sections = append(sections, entry)
		off = end
	}
	return sections, nil
}

func isDWARFSection(name string) bool {
	return strings.HasPrefix(name, ".debug_") ||
		strings.HasPrefix(name, ".zdebug_") ||
		strings.HasPrefix(name, "reloc..debug_") ||
		strings.HasPrefix(name, "reloc..zdebug_")
}

func appendCustomSection(dst []byte, name string, content []byte) []byte {
	payload := appendULEB32(nil, uint32(len(name)))
	payload = append(payload, name...)
	payload = append(payload, content...)
	dst = append(dst, 0)
	dst = appendULEB32(dst, uint32(len(payload)))
	return append(dst, payload...)
}

// BuildID returns the unique WebAssembly tool-conventions build_id, if
// present. Build IDs are arbitrary bytes and are not interpreted as UTF-8.
func BuildID(module []byte) ([]byte, bool, error) {
	sections, err := parse(module)
	if err != nil {
		return nil, false, err
	}
	var id []byte
	found := false
	for _, section := range sections {
		if section.id != 0 || section.name != buildIDSection {
			continue
		}
		if found {
			return nil, false, errors.New("multiple WebAssembly build_id sections")
		}
		off := 0
		size, err := readULEB32(section.content, &off)
		if err != nil {
			return nil, false, fmt.Errorf("invalid WebAssembly build_id section: %w", err)
		}
		if int(size) != len(section.content)-off {
			return nil, false, errors.New("invalid WebAssembly build_id length")
		}
		id = bytes.Clone(section.content[off:])
		found = true
	}
	return id, found, nil
}

// SetBuildID replaces the WebAssembly tool-conventions build_id section.
func SetBuildID(module, id []byte) ([]byte, error) {
	if len(id) == 0 {
		return nil, errors.New("WebAssembly build ID must not be empty")
	}
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), wasmHeader...)
	for _, section := range sections {
		if section.id == 0 && section.name == buildIDSection {
			continue
		}
		out = append(out, section.raw...)
	}
	payload := appendULEB32(nil, uint32(len(id)))
	payload = append(payload, id...)
	return appendCustomSection(out, buildIDSection, payload), nil
}

// EnsureBuildID preserves an existing valid build ID or installs a
// deterministic SHA-256 ID over the complete module without a build_id
// section. Calling it before externalization gives the main module and DWARF
// sidecar the same identity for stale-sidecar checks.
func EnsureBuildID(module []byte) ([]byte, []byte, error) {
	if id, ok, err := BuildID(module); err != nil {
		return nil, nil, err
	} else if ok {
		return bytes.Clone(module), id, nil
	}
	digest := sha256.Sum256(module)
	result, err := SetBuildID(module, digest[:])
	if err != nil {
		return nil, nil, err
	}
	return result, bytes.Clone(digest[:]), nil
}

// RefreshBuildID computes a content identity after debugger metadata is added.
// It validates an existing ID, then hashes every section except build_id so
// repeated packaging is deterministic and ABI edits invalidate old sidecars.
// Call it before externalization to give both artifacts the same identity.
func RefreshBuildID(module []byte) ([]byte, []byte, error) {
	if _, _, err := BuildID(module); err != nil {
		return nil, nil, err
	}
	sections, err := parse(module)
	if err != nil {
		return nil, nil, err
	}
	content := append([]byte(nil), wasmHeader...)
	for _, section := range sections {
		if section.id != 0 || section.name != buildIDSection {
			content = append(content, section.raw...)
		}
	}
	digest := sha256.Sum256(content)
	result, err := SetBuildID(content, digest[:])
	if err != nil {
		return nil, nil, err
	}
	return result, bytes.Clone(digest[:]), nil
}

// HasDWARF reports whether module contains at least one DWARF custom section.
func HasDWARF(module []byte) (bool, error) {
	sections, err := parse(module)
	if err != nil {
		return false, err
	}
	for _, section := range sections {
		if section.id == 0 && isDWARFSection(section.name) {
			return true, nil
		}
	}
	return false, nil
}

// SetDebuggerRecord replaces the LLGo debugger ABI custom section with the
// canonical encoding of record. Other custom and standard sections retain
// their original bytes and order.
func SetDebuggerRecord(module []byte, record debugabi.Record) ([]byte, error) {
	raw, err := record.MarshalBinary()
	if err != nil {
		return nil, err
	}
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), wasmHeader...)
	for _, section := range sections {
		if section.id == 0 && section.name == debugabi.WasmSectionName {
			continue
		}
		out = append(out, section.raw...)
	}
	return appendCustomSection(out, debugabi.WasmSectionName, raw), nil
}

// DebuggerRecord returns the unique LLGo debugger ABI record, if present.
func DebuggerRecord(module []byte) (debugabi.Record, bool, error) {
	sections, err := parse(module)
	if err != nil {
		return debugabi.Record{}, false, err
	}
	var record debugabi.Record
	found := false
	for _, section := range sections {
		if section.id != 0 || section.name != debugabi.WasmSectionName {
			continue
		}
		if found {
			return debugabi.Record{}, false, errors.New("multiple LLGo debugger ABI sections")
		}
		record, err = debugabi.ParseRecord(section.content)
		if err != nil {
			return debugabi.Record{}, false, fmt.Errorf("invalid LLGo debugger ABI section: %w", err)
		}
		found = true
	}
	return record, found, nil
}

// Externalize removes embedded DWARF custom sections and appends the standard
// external_debug_info URL record. The original module is suitable for use as
// the sidecar because the convention permits it to retain code and data.
func Externalize(module []byte, url string) ([]byte, error) {
	if url == "" || !utf8.ValidString(url) {
		return nil, errors.New("external DWARF URL must be non-empty UTF-8")
	}
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), wasmHeader...)
	foundDWARF := false
	for _, section := range sections {
		if section.id == 0 {
			if isDWARFSection(section.name) {
				foundDWARF = true
				continue
			}
			if section.name == externalDebugInfo {
				continue
			}
		}
		out = append(out, section.raw...)
	}
	if !foundDWARF {
		return nil, errors.New("WebAssembly module contains no DWARF sections")
	}
	content := appendULEB32(nil, uint32(len(url)))
	content = append(content, url...)
	return appendCustomSection(out, externalDebugInfo, content), nil
}

// ExternalURL returns the external_debug_info URL, if present.
func ExternalURL(module []byte) (string, bool, error) {
	sections, err := parse(module)
	if err != nil {
		return "", false, err
	}
	var url string
	found := false
	for _, section := range sections {
		if section.id != 0 || section.name != externalDebugInfo {
			continue
		}
		if found {
			return "", false, errors.New("multiple external_debug_info sections")
		}
		off := 0
		url, err = readName(section.content, &off)
		if err != nil {
			return "", false, fmt.Errorf("invalid external_debug_info section: %w", err)
		}
		if off != len(section.content) {
			return "", false, errors.New("external_debug_info section has trailing data")
		}
		found = true
	}
	return url, found, nil
}

// DWARFSections returns the unique embedded DWARF custom sections keyed by
// their standard section names. Returned contents do not alias module.
func DWARFSections(module []byte) (map[string][]byte, error) {
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]byte)
	for _, section := range sections {
		if section.id != 0 || !isDWARFSection(section.name) {
			continue
		}
		if _, exists := result[section.name]; exists {
			return nil, fmt.Errorf("multiple %s WebAssembly custom sections", section.name)
		}
		result[section.name] = bytes.Clone(section.content)
	}
	return result, nil
}
