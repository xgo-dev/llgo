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

import "fmt"

// StripMetadata removes optional debugger metadata without touching code/data,
// runtime PCLN, external DWARF references, build identity or unknown extensions.
// all additionally removes producer metadata. target_features is retained so
// consumers can validate the module's required capabilities.
func StripMetadata(module []byte, all bool) ([]byte, error) {
	return stripMetadata(module, all, false)
}

// StripNames implements -s -w=0: remove names while preserving DWARF and maps.
func StripNames(module []byte) ([]byte, error) {
	return stripMetadata(module, false, true)
}

func stripMetadata(module []byte, all, namesOnly bool) ([]byte, error) {
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	out := append(make([]byte, 0, len(module)), wasmHeader...)
	for _, s := range sections {
		if s.id == 0 && (s.name == "name" || !namesOnly && (isDWARFSection(s.name) || s.name == "sourceMappingURL" || all && s.name == "producers")) {
			continue
		}
		out = append(out, s.raw...)
	}
	return out, nil
}

// TargetFeatures reads required/used features from the tool-conventions
// target_features custom section. Disallowed features ('-') are not required.
func TargetFeatures(module []byte) ([]string, error) {
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	var features []string
	found := false
	for _, s := range sections {
		if s.id != 0 || s.name != "target_features" {
			continue
		}
		if found {
			return nil, fmt.Errorf("duplicate WebAssembly target_features")
		}
		found = true
		off := 0
		n, err := readULEB32(s.content, &off)
		if err != nil {
			return nil, err
		}
		for ; n != 0; n-- {
			if off >= len(s.content) {
				return nil, fmt.Errorf("truncated WebAssembly target feature")
			}
			prefix := s.content[off]
			off++
			if prefix != '+' && prefix != '-' {
				return nil, fmt.Errorf("invalid WebAssembly target feature prefix %q", prefix)
			}
			name, err := readName(s.content, &off)
			if err != nil {
				return nil, err
			}
			if prefix == '+' {
				features = append(features, name)
			}
		}
		if off != len(s.content) {
			return nil, fmt.Errorf("trailing WebAssembly target feature data")
		}
	}
	return features, nil
}

// SourceMappingURL reads the optional debugger URL without interpreting it as a
// filesystem path. The accompanying local map is supplied by the linker.
func SourceMappingURL(module []byte) (string, error) {
	sections, err := parse(module)
	if err != nil {
		return "", err
	}
	var url string
	found := false
	for _, s := range sections {
		if s.id != 0 || s.name != "sourceMappingURL" {
			continue
		}
		if found {
			return "", fmt.Errorf("duplicate WebAssembly sourceMappingURL")
		}
		found = true
		off := 0
		url, err = readName(s.content, &off)
		if err != nil {
			return "", err
		}
		if off != len(s.content) {
			return "", fmt.Errorf("trailing WebAssembly sourceMappingURL data")
		}
	}
	return url, nil
}
