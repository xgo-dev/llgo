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

package wasmdebug

import (
	"errors"
	"fmt"
)

// FunctionNames reads the optional tool-conventions function name map. A nil
// map means no function names are available, not that all functions are dead.
func FunctionNames(module []byte) (map[uint32]string, error) {
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	var names map[uint32]string
	for _, section := range sections {
		if section.id != 0 || section.name != "name" {
			continue
		}
		raw := section.content
		for off := 0; off < len(raw); {
			id := raw[off]
			off++
			size, err := readULEB32(raw, &off)
			if err != nil {
				return nil, err
			}
			if uint64(size) > uint64(len(raw)-off) {
				return nil, errors.New("truncated Wasm name subsection")
			}
			payload := raw[off : off+int(size)]
			off += int(size)
			if id != 1 {
				continue
			}
			if names != nil {
				return nil, errors.New("duplicate Wasm function name map")
			}
			names = make(map[uint32]string)
			pos := 0
			count, err := readULEB32(payload, &pos)
			if err != nil {
				return nil, err
			}
			for i := uint32(0); i < count; i++ {
				index, err := readULEB32(payload, &pos)
				if err != nil {
					return nil, err
				}
				name, err := readName(payload, &pos)
				if err != nil {
					return nil, err
				}
				if _, exists := names[index]; exists {
					return nil, fmt.Errorf("duplicate Wasm function name index %d", index)
				}
				names[index] = name
			}
			if pos != len(payload) {
				return nil, errors.New("trailing Wasm function name bytes")
			}
		}
	}
	return names, nil
}

// FunctionNamesComplete proves that every imported and defined function has a
// nonempty name. Optional sparse maps cannot establish that a PCLN symbol died.
// Import types this reader does not understand conservatively disable pruning.
func FunctionNamesComplete(module []byte, names map[uint32]string) (bool, error) {
	sections, err := parse(module)
	if err != nil {
		return false, err
	}
	var count uint64
	seen := make(map[byte]bool)
	for _, section := range sections {
		if section.id != 2 && section.id != 3 {
			continue
		}
		if seen[section.id] {
			return false, nil
		}
		seen[section.id] = true
		raw, pos := section.content, 0
		n, err := readULEB32(raw, &pos)
		if err != nil {
			return false, nil
		}
		if section.id == 3 {
			count += uint64(n)
			for range n {
				if _, err := readULEB32(raw, &pos); err != nil {
					return false, nil
				}
			}
		} else {
			for range n {
				if _, err := readName(raw, &pos); err != nil {
					return false, nil
				}
				if _, err := readName(raw, &pos); err != nil {
					return false, nil
				}
				if pos == len(raw) {
					return false, nil
				}
				kind := raw[pos]
				pos++
				switch kind {
				case 0: // function type index
					if _, err := readULEB32(raw, &pos); err != nil {
						return false, nil
					}
					count++
				case 1: // reference type and table limits
					if pos == len(raw) || raw[pos] != 0x70 && raw[pos] != 0x6f {
						return false, nil
					}
					pos++
					if !skipNameCoverageLimits(raw, &pos) {
						return false, nil
					}
				case 2: // memory limits
					if !skipNameCoverageLimits(raw, &pos) {
						return false, nil
					}
				case 3: // global value type and mutability
					if len(raw)-pos < 2 || !simpleWasmValueType(raw[pos]) || raw[pos+1] > 1 {
						return false, nil
					}
					pos += 2
				case 4: // exception tag attribute and type index
					if pos == len(raw) || raw[pos] != 0 {
						return false, nil
					}
					pos++
					if _, err := readULEB32(raw, &pos); err != nil {
						return false, nil
					}
				default:
					return false, nil
				}
			}
		}
		if pos != len(raw) {
			return false, nil
		}
	}
	if count == 0 || uint64(len(names)) != count {
		return false, nil
	}
	for index, name := range names {
		if uint64(index) >= count || name == "" {
			return false, nil
		}
	}
	return true, nil
}

func skipNameCoverageLimits(raw []byte, pos *int) bool {
	flags, err := readULEB32(raw, pos)
	// New memory/table encodings need a reader update before proving coverage.
	if err != nil || flags & ^uint32(3) != 0 {
		return false
	}
	if _, err := readULEB32(raw, pos); err != nil {
		return false
	}
	if flags&1 != 0 {
		if _, err := readULEB32(raw, pos); err != nil {
			return false
		}
	}
	return true
}

func simpleWasmValueType(typ byte) bool {
	return typ >= 0x7b && typ <= 0x7f || typ == 0x70 || typ == 0x6f
}
