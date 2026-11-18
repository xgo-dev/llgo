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

package goflags

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/build"
)

// parseNativeFlags guards the typed configuration boundary and returns the
// validated compiler options for the frontend. The package loader accepting a
// gc control does not make that control an LLGo feature.
func parseNativeFlags(args []string) (*frontendGCFlags, error) {
	// Go selects the last matching compiler argument list for each package.
	// An all= list supersedes both supported scopes; an unpatterned list does
	// not supersede an earlier all= list for dependencies. Syntax and unsupported
	// package patterns remain errors even when a later list would replace them.
	seenGCScopes := make(map[string]bool)
	var frontend *frontendGCFlags
	for i := len(args) - 1; i >= 0; i-- {
		arg := args[i]
		name, value, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		switch name {
		case "n", "work", "trimpath", "buildvcs", "race", "msan", "asan", "linkshared", "compiler", "asmflags", "gccgoflags", "installsuffix", "pkgdir", "toolexec", "pgo":
			return nil, fmt.Errorf("-%s is unsupported by the LLGo backend", name)
		case "gcflags":
			pattern, fields, err := splitPerPackageArgumentList(value)
			if err != nil {
				return nil, fmt.Errorf("invalid -gcflags: %w", err)
			}
			if pattern != "" && pattern != "all" {
				return nil, fmt.Errorf("invalid -gcflags: package pattern %q is unsupported; use an unpatterned value or all=", pattern)
			}
			if seenGCScopes["all"] || seenGCScopes[pattern] {
				continue
			}
			parsed, err := parseFrontendGCFlags(fields)
			if err != nil {
				return nil, fmt.Errorf("invalid -gcflags: %w", err)
			}
			if frontend == nil {
				frontend = parsed
			}
			if pattern == "all" {
				frontend.dependencyMCount = parsed.mCount
			}
			seenGCScopes[pattern] = true
		}
	}
	return frontend, nil
}

// loaderBuildFlags removes only native -m controls while preserving argument
// scopes and empty lists, which can override an earlier all= list.
func loaderBuildFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		name, value, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name != "gcflags" {
			out = append(out, arg)
			continue
		}
		pattern, fields, _ := splitPerPackageArgumentList(value) // Already validated.
		kept := make([]string, 0, len(fields))
		removed := false
		for _, field := range fields {
			flagName, _, _ := strings.Cut(strings.TrimLeft(field, "-"), "=")
			if strings.HasPrefix(field, "-") && flagName == "m" {
				removed = true
				continue
			}
			if strings.ContainsAny(field, " \t\r\n") {
				quote := "'"
				if strings.Contains(field, quote) {
					quote = "\""
				}
				field = quote + field + quote
			}
			kept = append(kept, field)
		}
		if removed {
			prefix := ""
			if pattern != "" {
				prefix = pattern + "="
			}
			arg = "-gcflags=" + prefix + strings.Join(kept, " ")
		}
		out = append(out, arg)
	}
	return out
}

// applyOverlay reads the same JSON view used by the Go loader into LLGo's
// parser view. Keep the raw -overlay flag for file selection, including deleted
// and added Go files, and carry validated paths and contents as typed data.
func applyOverlay(conf *build.Config, args []string) error {
	var file string
	present := false
	for _, arg := range args {
		if v, ok := strings.CutPrefix(arg, "-overlay="); ok {
			file = v
			present = true
		}
	}
	if !present {
		return nil
	}
	if file == "" {
		conf.SourceOverlay = nil
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read -overlay: %w", err)
	}
	var overlay struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &overlay); err != nil {
		return fmt.Errorf("decode -overlay: %w", err)
	}
	contents := make(map[string]build.SourceOverlayFile, len(overlay.Replace))
	logicalPaths := make(map[string]string, len(overlay.Replace))
	for _, logical := range slices.Sorted(maps.Keys(overlay.Replace)) {
		name, backing := logical, overlay.Replace[logical]
		if filepath.Ext(name) != ".go" {
			return fmt.Errorf("-overlay: non-Go replacement %q is unsupported by LLGo", name)
		}
		name, err = filepath.Abs(name)
		if err != nil {
			return err
		}
		if previous, ok := logicalPaths[name]; ok {
			return fmt.Errorf("-overlay: duplicate paths %q and %q in overlay map", previous, logical)
		}
		logicalPaths[name] = logical
		var source []byte
		if backing != "" {
			backing, err = filepath.Abs(backing)
			if err != nil {
				return err
			}
			source, err = os.ReadFile(backing)
			if err != nil {
				return fmt.Errorf("read -overlay backing file %q: %w", backing, err)
			}
		}
		contents[name] = build.SourceOverlayFile{BackingFile: backing, Contents: source}
	}
	conf.SourceOverlay = contents
	return nil
}
