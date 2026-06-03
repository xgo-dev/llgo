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

package littest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/llgo/internal/filecheck"
)

type Mode int

const (
	ModeSkip Mode = iota
	ModeLiteral
	ModeFileCheck
)

type Spec struct {
	Path string
	Text string
	Mode Mode
}

const Marker = "LITTEST"

func LoadSpec(pkgDir string) (Spec, error) {
	if spec, ok, err := loadSourceSpec(pkgDir); err != nil {
		return Spec{}, err
	} else if ok {
		return spec, nil
	}

	path := filepath.Join(pkgDir, "out.ll")
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, err
	}
	if bytes.Equal(data, []byte{';'}) {
		return Spec{Path: path, Mode: ModeSkip}, nil
	}
	return Spec{Path: path, Text: string(data), Mode: ModeLiteral}, nil
}

func Check(spec Spec, actual string) error {
	switch spec.Mode {
	case ModeSkip:
		return nil
	case ModeFileCheck:
		return filecheck.Match(spec.Path, spec.Text, actual)
	case ModeLiteral:
		if actual != spec.Text {
			return fmt.Errorf("%s: literal LLVM IR mismatch", spec.Path)
		}
		return nil
	default:
		return errors.New("unknown lit spec mode")
	}
}

func loadSourceSpec(pkgDir string) (Spec, bool, error) {
	marked, ok, err := FindMarkedSourceFile(pkgDir)
	if err != nil {
		return Spec{}, false, err
	}
	if !ok {
		return Spec{}, false, nil
	}
	if marked == "" {
		return Spec{}, false, nil
	}
	data, err := os.ReadFile(marked)
	if err != nil {
		return Spec{}, false, err
	}
	text := string(data)
	ok, err = filecheck.HasDirectives(text)
	if err != nil {
		return Spec{}, false, err
	}
	if !ok {
		return Spec{}, false, fmt.Errorf("%s: %s is marked %s but has no FileCheck directives", pkgDir, filepath.Base(marked), Marker)
	}
	return Spec{
		Path: marked,
		Text: text,
		Mode: ModeFileCheck,
	}, true, nil
}

func FindMarkedSourceFile(dir string) (string, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false, err
	}
	var marked string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !IsSourceSpecFile(name) {
			continue
		}
		path := filepath.Join(dir, name)
		ok, err := HasMarker(path)
		if err != nil {
			return "", false, err
		}
		if !ok {
			continue
		}
		if marked != "" {
			return "", false, fmt.Errorf("%s: multiple source lit specs found: %s, %s", dir, filepath.Base(marked), filepath.Base(path))
		}
		marked = path
	}
	if marked == "" {
		return "", false, nil
	}
	return marked, true, nil
}

func HasMarker(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	line := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(line, "//") {
		return false, nil
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "//")) == Marker, nil
}

func IsSourceSpecFile(name string) bool {
	return filepath.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go")
}
