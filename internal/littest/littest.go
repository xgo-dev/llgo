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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/goplus/llgo/internal/filecheck"
)

type Mode int

const (
	ModeFileCheck Mode = iota
)

type Spec struct {
	Path string
	Mode Mode
}

const Marker = "LITTEST"

func LoadSpec(pkgDir string) (Spec, error) {
	marked, ok, err := FindMarkedSourceFile(pkgDir)
	if err != nil {
		return Spec{}, err
	}
	if !ok || marked == "" {
		return Spec{}, fmt.Errorf("%s: missing // %s source lit spec", pkgDir, Marker)
	}
	return Spec{Path: marked, Mode: ModeFileCheck}, nil
}

func Check(spec Spec, actual string, prefixes ...string) error {
	switch spec.Mode {
	case ModeFileCheck:
		if sections, split, err := functionCheckSections(spec.Path, prefixes); err != nil {
			return err
		} else if split {
			for i, section := range sections {
				if err := filecheck.MatchTextWithPrefixes(section, actual, prefixes...); err != nil {
					return fmt.Errorf("%s: function check section %d: %w", spec.Path, i+1, err)
				}
			}
			return nil
		}
		return filecheck.MatchWithPrefixes(spec.Path, actual, prefixes...)
	default:
		return errors.New("unknown lit spec mode")
	}
}

var fileCheckSuffixes = []string{
	"-LABEL:", "-NEXT:", "-SAME:", "-DAG:", "-NOT:", "-EMPTY:", ":",
}

var (
	fileCheckVarDefRE = regexp.MustCompile(`\[\[([A-Za-z_][A-Za-z0-9_]*):`)
	fileCheckVarUseRE = regexp.MustCompile(`\[\[([A-Za-z_][A-Za-z0-9_]*)\]\]`)
)

// functionCheckSections splits a source fixture with multiple function LABELs
// into independent checks. Each section repeats the global directives and
// retains one complete LABEL/SAME/NEXT block, so local and global value
// relationships stay strict while LLVM function-list order remains irrelevant.
func functionCheckSections(path string, prefixes []string) ([]string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	active := append([]string(nil), prefixes...)
	if len(active) == 0 {
		active = []string{"CHECK"}
	}
	// Prefer the most specific prefix when one is a prefix of another (for
	// example DARWIN and DARWIN-ARM64).
	slices.SortFunc(active, func(a, b string) int {
		return len(b) - len(a)
	})

	var globals []string
	var current []string
	var sections []string
	seenLabel := false
	labels := 0
	for _, line := range strings.Split(string(data), "\n") {
		kind, ok := activeDirectiveKind(line, active)
		if !ok {
			continue
		}
		line += "\n"
		if kind == "LABEL" {
			labels++
			if len(current) != 0 {
				sections = append(sections, boundedFunctionSection(globals, current, active))
			}
			current = []string{line}
			seenLabel = true
			continue
		}
		if seenLabel {
			current = append(current, line)
		} else {
			globals = append(globals, line)
		}
	}
	if len(current) != 0 {
		sections = append(sections, boundedFunctionSection(globals, current, active))
	}
	if labels < 2 {
		return nil, false, nil
	}
	if sectionsShareVariables(globals, sections) {
		// A few hand-written fixtures intentionally bind the same type symbol
		// across two functions. Keep their single FileCheck invocation because
		// that cross-function identity is part of the test's contract.
		return nil, false, nil
	}
	return sections, true, nil
}

func boundedFunctionSection(globals, current, active []string) string {
	for _, line := range current {
		body := strings.TrimSpace(line)
		if strings.HasSuffix(body, ": }") || strings.HasSuffix(body, ": {{^}$}}") {
			return strings.Join(append(append([]string(nil), globals...), current...), "")
		}
	}
	prefix := active[0]
	if slices.Contains(active, "CHECK") {
		prefix = "CHECK"
	}
	lines := append(append([]string(nil), globals...), current...)
	lines = append(lines, "// "+prefix+"-LABEL: {{^}$}}\n")
	return strings.Join(lines, "")
}

func sectionsShareVariables(globals []string, sections []string) bool {
	globalDefs := fileCheckVarDefs(strings.Join(globals, ""))
	sectionDefs := make([]map[string]bool, len(sections))
	allSectionDefs := make(map[string]bool)
	for i, section := range sections {
		sectionDefs[i] = fileCheckVarDefs(section)
		for name := range sectionDefs[i] {
			allSectionDefs[name] = true
		}
	}
	for i, section := range sections {
		for _, match := range fileCheckVarUseRE.FindAllStringSubmatch(section, -1) {
			name := match[1]
			if !globalDefs[name] && !sectionDefs[i][name] && allSectionDefs[name] {
				return true
			}
		}
	}
	return false
}

func fileCheckVarDefs(text string) map[string]bool {
	defs := make(map[string]bool)
	for _, match := range fileCheckVarDefRE.FindAllStringSubmatch(text, -1) {
		defs[match[1]] = true
	}
	return defs
}

func activeDirectiveKind(line string, prefixes []string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "//") {
		return "", false
	}
	body := strings.TrimSpace(strings.TrimPrefix(line, "//"))
	for _, prefix := range prefixes {
		if !strings.HasPrefix(body, prefix) {
			continue
		}
		rest := strings.TrimPrefix(body, prefix)
		for _, suffix := range fileCheckSuffixes {
			if strings.HasPrefix(rest, suffix) {
				return strings.TrimSuffix(strings.TrimPrefix(suffix, "-"), ":"), true
			}
		}
	}
	return "", false
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
