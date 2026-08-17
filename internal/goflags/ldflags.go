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

// Package goflags parses Go-compatible build flag syntax into typed LLGo
// configuration. It is shared by command entry points and non-command build
// drivers; internal/build consumes only the typed result.
package goflags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/build"
)

// ParsedLinkFlags contains the Go linker options that LLGo currently handles.
// Ignored retains the remaining linker arguments so their order is available
// as support is added incrementally.
type ParsedLinkFlags struct {
	Options build.LinkOptions
	Ignored []string
	Present bool
}

// ParseLinkFlags extracts the currently supported flags from normalized Go
// build flags. Repeated -ldflags values are evaluated in command-line order;
// each applicable value replaces the previous complete argument list.
func ParseLinkFlags(buildFlags []string) (ParsedLinkFlags, error) {
	var opts ParsedLinkFlags
	for _, buildFlag := range buildFlags {
		if buildFlag == "-ldflags" {
			return ParsedLinkFlags{}, fmt.Errorf("-ldflags requires a value")
		}

		value, ok := strings.CutPrefix(buildFlag, "-ldflags=")
		if !ok {
			continue
		}
		opts = ParsedLinkFlags{Present: true}

		linkFlags, err := splitPerPackageLinkFlags(value)
		if err != nil {
			return ParsedLinkFlags{}, fmt.Errorf("invalid -ldflags value %q: %w", value, err)
		}
		for _, linkFlag := range linkFlags {
			switch {
			case linkFlag == "-s" || linkFlag == "--s":
				opts.Options.OmitSymbolTable = true
			case hasLinkFlagBoolValue(linkFlag, "s"):
				v, err := parseLinkFlagBool("-s", linkFlagBoolValue(linkFlag, "s"))
				if err != nil {
					return ParsedLinkFlags{}, err
				}
				opts.Options.OmitSymbolTable = v
			case linkFlag == "-w" || linkFlag == "--w":
				opts.Options.DWARF = build.DWARFOmit
			case hasLinkFlagBoolValue(linkFlag, "w"):
				v, err := parseLinkFlagBool("-w", linkFlagBoolValue(linkFlag, "w"))
				if err != nil {
					return ParsedLinkFlags{}, err
				}
				if v {
					opts.Options.DWARF = build.DWARFOmit
				} else {
					opts.Options.DWARF = build.DWARFPreserve
				}
			default:
				opts.Ignored = append(opts.Ignored, linkFlag)
			}
		}
	}
	return opts, nil
}

func hasLinkFlagBoolValue(arg, name string) bool {
	return strings.HasPrefix(arg, "-"+name+"=") || strings.HasPrefix(arg, "--"+name+"=")
}

func linkFlagBoolValue(arg, name string) string {
	arg = strings.TrimPrefix(arg, "-")
	arg = strings.TrimPrefix(arg, "-")
	return strings.TrimPrefix(arg, name+"=")
}

func parseLinkFlagBool(name, value string) (bool, error) {
	if value == "" {
		return false, fmt.Errorf("%s requires a boolean value", name)
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid boolean value %q for %s: %w", value, name, err)
	}
	return v, nil
}

// splitPerPackageLinkFlags implements the currently supported subset of Go's
// per-package build flag syntax: [pattern=]arg-list. LLGo can safely evaluate
// unpatterned values and all= before package loading. Other patterns require
// per-root matching and are rejected instead of being applied globally.
func splitPerPackageLinkFlags(value string) ([]string, error) {
	pattern, flags, err := splitPerPackageArgumentList(value)
	if err != nil {
		return nil, err
	}
	if pattern != "" && pattern != "all" {
		return nil, fmt.Errorf("package pattern %q is not supported yet; use an unpatterned value or all=", pattern)
	}
	return flags, nil
}

// splitPerPackageArgumentList separates the optional package pattern from a
// Go build flag's quoted argument list. An empty pattern denotes the
// unpatterned form.
func splitPerPackageArgumentList(value string) (pattern string, flags []string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, nil
	}
	if value[0] != '-' {
		if value[0] == '\'' || value[0] == '"' {
			return "", nil, fmt.Errorf("parameter may not start with quote character %c", value[0])
		}
		i := strings.IndexByte(value, '=')
		if i < 0 {
			return "", nil, fmt.Errorf("missing =<value> in <pattern>=<value>")
		}
		if i == 0 {
			return "", nil, fmt.Errorf("missing <pattern> in <pattern>=<value>")
		}
		pattern = strings.TrimSpace(value[:i])
		value = value[i+1:]
	}
	flags, err = splitQuotedFields(value)
	return pattern, flags, err
}

// splitQuotedFields follows the quoting rules used by the Go command for
// -ldflags: single or double quotes may surround one argument, with no escape
// processing inside the quoted text.
func splitQuotedFields(value string) ([]string, error) {
	var fields []string
	for len(value) > 0 {
		value = strings.TrimLeft(value, " \t\n\r")
		if value == "" {
			break
		}
		if value[0] == '\'' || value[0] == '"' {
			quote := value[0]
			value = value[1:]
			i := strings.IndexByte(value, quote)
			if i < 0 {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			fields = append(fields, value[:i])
			value = value[i+1:]
			continue
		}

		i := strings.IndexAny(value, " \t\n\r")
		if i < 0 {
			fields = append(fields, value)
			break
		}
		fields = append(fields, value[:i])
		value = value[i:]
	}
	return fields, nil
}
