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
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// The linker writes source paths relative to its private staging directory.
// Publishing the map one directory above must preserve those source locations.
func relocateWasmSourceMap(path, destination string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("read Wasm source map: %w", err)
	}
	var root string
	if value, ok := data["sourceRoot"]; ok {
		if err := json.Unmarshal(value, &root); err != nil {
			return err
		}
	}
	relocate := func(source string) (string, error) {
		parsed, err := url.Parse(source)
		if filepath.IsAbs(source) || err == nil && (parsed.IsAbs() || parsed.Host != "") {
			return source, nil
		}
		absolute, err := filepath.Abs(filepath.Join(filepath.Dir(path), filepath.FromSlash(source)))
		if err != nil {
			return "", err
		}
		destination, err := filepath.Abs(destination)
		if err != nil {
			return "", err
		}
		relative, err := filepath.Rel(destination, absolute)
		return filepath.ToSlash(relative), err
	}
	if root != "" {
		root, err = relocate(root)
		if err != nil {
			return err
		}
		data["sourceRoot"], err = json.Marshal(root)
	} else {
		var sources []string
		if err := json.Unmarshal(data["sources"], &sources); err != nil {
			return err
		}
		for i, source := range sources {
			sources[i], err = relocate(source)
			if err != nil {
				return err
			}
		}
		data["sources"], err = json.Marshal(sources)
	}
	if err != nil {
		return err
	}
	raw, err = json.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
