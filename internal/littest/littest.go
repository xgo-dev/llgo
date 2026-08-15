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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/filecheck"
)

type Spec struct {
	Path    string
	PostABI bool
	Targets []Target
}

// Target selects one GOOS/GOARCH pair for an IR check.
type Target struct {
	GOOS   string
	GOARCH string
}

func (t Target) String() string {
	return t.GOOS + "/" + t.GOARCH
}

const (
	Marker        = "LITTEST"
	PostABIMarker = "LITTEST: POST-ABI"
)

func LoadSpec(pkgDir string) (Spec, error) {
	spec, ok, err := FindSpec(pkgDir)
	if err != nil {
		return Spec{}, err
	}
	if !ok {
		return Spec{}, fmt.Errorf("%s: missing // %s source lit spec", pkgDir, Marker)
	}
	return spec, nil
}

func Check(spec Spec, actual string, targetPrefixes ...string) error {
	actual = CanonicalizeLLVMIR(actual)
	if len(targetPrefixes) == 0 {
		return filecheck.Match(spec.Path, actual)
	}
	return filecheck.MatchWithTargetPrefixes(spec.Path, actual, targetPrefixes...)
}

// CanonicalizeLLVMIR removes LLVM-version-specific spellings that are not the
// semantic contract of LLGo's IR tests. Verifier, object, link, and runtime
// tests still consume the original IR.
func CanonicalizeLLVMIR(ir string) string {
	ir = strings.ReplaceAll(ir, "getelementptr inbounds nuw ", "getelementptr inbounds ")
	ir = strings.ReplaceAll(ir, " captures(none)", " nocapture")
	return ir
}

func FindMarkedSourceFile(dir string) (string, bool, error) {
	spec, ok, err := FindSpec(dir)
	return spec.Path, ok, err
}

// FindSpec finds the source-embedded IR check in dir without requiring one.
func FindSpec(dir string) (Spec, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Spec{}, false, err
	}
	var spec Spec
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !IsSourceSpecFile(name) {
			continue
		}
		path := filepath.Join(dir, name)
		candidate, ok, err := readMarker(path)
		if err != nil {
			return Spec{}, false, err
		}
		if !ok {
			continue
		}
		if spec.Path != "" {
			return Spec{}, false, fmt.Errorf("%s: multiple source lit specs found: %s, %s", dir, filepath.Base(spec.Path), filepath.Base(path))
		}
		candidate.Path = path
		spec = candidate
	}
	if spec.Path == "" {
		return Spec{}, false, nil
	}
	return spec, true, nil
}

func HasMarker(path string) (bool, error) {
	_, ok, err := ReadMarker(path)
	return ok, err
}

// ReadMarker reports whether the source's first-line marker selects post-ABI IR.
// Plain // LITTEST retains the existing check behavior and may carry a
// space-separated GOOS/GOARCH target matrix. POST-ABI explicitly selects the
// target-ABI-lowered stage and may also carry a matrix.
func ReadMarker(path string) (postABI, found bool, err error) {
	spec, found, err := readMarker(path)
	return spec.PostABI, found, err
}

func readMarker(path string) (spec Spec, found bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return Spec{}, false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return Spec{}, false, scanner.Err()
	}
	line := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(line, "//") {
		return Spec{}, false, nil
	}
	marker := strings.TrimSpace(strings.TrimPrefix(line, "//"))
	switch marker {
	case Marker:
		return Spec{}, true, nil
	case PostABIMarker:
		return Spec{PostABI: true}, true, nil
	}
	postABI := false
	markerName := Marker
	switch {
	case strings.HasPrefix(marker, Marker+" "):
	case strings.HasPrefix(marker, PostABIMarker+" "):
		postABI = true
		markerName = PostABIMarker
	default:
		return Spec{}, false, nil
	}

	targets, err := parseTargets(strings.TrimSpace(strings.TrimPrefix(marker, markerName)))
	if err != nil {
		return Spec{}, false, fmt.Errorf("%s: %w", path, err)
	}
	return Spec{PostABI: postABI, Targets: targets}, true, nil
}

func parseTargets(text string) ([]Target, error) {
	fields := strings.Fields(text)
	targets := make([]Target, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		goos, goarch, ok := strings.Cut(field, "/")
		if !ok || goos == "" || goarch == "" || strings.Contains(goarch, "/") {
			return nil, fmt.Errorf("invalid LITTEST target %q; want GOOS/GOARCH", field)
		}
		if _, ok := seen[field]; ok {
			return nil, fmt.Errorf("duplicate LITTEST target %q", field)
		}
		seen[field] = struct{}{}
		targets = append(targets, Target{GOOS: goos, GOARCH: goarch})
	}
	return targets, nil
}

func IsSourceSpecFile(name string) bool {
	return filepath.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go")
}
