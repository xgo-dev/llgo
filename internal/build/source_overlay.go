/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
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
	"os"
	"slices"
)

// SourceOverlayFile holds one validated replacement. An empty BackingFile
// deletes the logical file from package selection.
type SourceOverlayFile struct {
	BackingFile string
	Contents    []byte
}

// prepareSourceOverlay materializes an absolute Go loader view. Runtime loads
// run from the compiler module, so the caller's relative JSON paths cannot be
// forwarded unchanged. The owning Build removes the file after all child
// groups and failure recovery finish; children reuse the prepared parser view.
func prepareSourceOverlay(conf *Config) (string, error) {
	return prepareSourceOverlayWithIO(conf, os.CreateTemp, (*os.File).Close)
}

func prepareSourceOverlayWithIO(conf *Config, createTemp func(string, string) (*os.File, error), closeFile func(*os.File) error) (string, error) {
	if conf.sourceOverlayFile != "" {
		return conf.sourceOverlayFile, nil
	}
	// A non-nil empty map still represents an explicit overlay. Materialize it
	// so runtime package loads do not inherit the caller's relative JSON path.
	if conf.SourceOverlay == nil {
		return "", nil
	}
	replacements := make(map[string]string, len(conf.SourceOverlay))
	contents := cloneOverlay(conf.Overlay)
	if contents == nil {
		contents = make(map[string][]byte)
	}
	for name, file := range conf.SourceOverlay {
		replacements[name] = file.BackingFile
		contents[name] = file.Contents
	}
	file, err := createTemp("", "llgo-overlay-*.json")
	if err != nil {
		return "", fmt.Errorf("create source overlay: %w", err)
	}
	name := file.Name()
	err = json.NewEncoder(file).Encode(struct{ Replace map[string]string }{replacements})
	closeErr := closeFile(file)
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(name)
		return "", fmt.Errorf("write source overlay: %w", err)
	}
	conf.Overlay = contents
	conf.sourceOverlayFile = name
	return name, nil
}

func cloneSourceOverlay(input map[string]SourceOverlayFile) map[string]SourceOverlayFile {
	if input == nil {
		return nil
	}
	out := make(map[string]SourceOverlayFile, len(input))
	for name, file := range input {
		file.Contents = slices.Clone(file.Contents)
		out[name] = file
	}
	return out
}
