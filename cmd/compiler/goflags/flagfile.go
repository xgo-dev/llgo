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

package goflags

import (
	"fmt"
	"strings"
	"unicode"
)

var argumentListFlagNames = [...]string{
	"asmflags",
	"gcflags",
	"gccgoflags",
	"ldflags",
	"toolexec",
}

// ParseFlagFile parses a flags.txt-style file. Blank lines and full-line
// comments are ignored. A line beginning with a Go flag whose value is itself
// an argument list is kept intact, so unquoted values such as
// "-ldflags=-s -w=false" work. Other lines accept shell-style whitespace,
// quoting, and backslash escaping and may contain multiple flags.
func ParseFlagFile(data string) ([]string, error) {
	var flags []string
	for lineNo, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var err error
		line, err = trimFlagComment(line)
		if err != nil {
			return nil, fmt.Errorf("flags line %d: %w", lineNo+1, err)
		}
		if flag, ok := wholeLineValueFlag(line); ok {
			flags = append(flags, flag)
			continue
		}
		fields := splitFlagFields(line)
		fields, err = normalizeBuildFlags(fields)
		if err != nil {
			return nil, fmt.Errorf("flags line %d: %w", lineNo+1, err)
		}
		flags = append(flags, fields...)
	}
	return flags, nil
}

func wholeLineValueFlag(line string) (flag string, ok bool) {
	for _, name := range argumentListFlagNames {
		for _, prefix := range []string{"-" + name + "=", "--" + name + "="} {
			value, found := strings.CutPrefix(line, prefix)
			if !found {
				continue
			}
			if len(value) != 0 && (value[0] == '\'' || value[0] == '"') {
				// Let the regular lexer handle quoted values. This also permits
				// ordinary build flags after the quoted argument list.
				return "", false
			}
			return "-" + name + "=" + value, true
		}
	}
	return "", false
}

func trimFlagComment(line string) (string, error) {
	var quote rune
	escaped := false
	inField := false
	for i, r := range line {
		if escaped {
			escaped = false
			inField = true
			continue
		}
		if quote != 0 {
			switch {
			case r == quote:
				quote = 0
			case r == '\\' && quote == '"':
				escaped = true
			}
			inField = true
			continue
		}
		switch {
		case r == '\\':
			escaped = true
			inField = true
		case r == '\'' || r == '"':
			quote = r
			inField = true
		case unicode.IsSpace(r):
			inField = false
		case r == '#' && !inField:
			return strings.TrimSpace(line[:i]), nil
		default:
			inField = true
		}
	}
	if escaped {
		return "", fmt.Errorf("unfinished escape")
	}
	if quote != 0 {
		return "", fmt.Errorf("unterminated %c quote", quote)
	}
	return line, nil
}

// splitFlagFields lexes a line already validated by trimFlagComment.
func splitFlagFields(line string) []string {
	var fields []string
	var field strings.Builder
	var quote rune
	escaped := false
	inField := false
	flush := func() {
		if inField {
			fields = append(fields, field.String())
			field.Reset()
			inField = false
		}
	}
	for _, r := range line {
		if escaped {
			field.WriteRune(r)
			inField = true
			escaped = false
			continue
		}
		if quote != 0 {
			switch {
			case r == quote:
				quote = 0
			case r == '\\' && quote == '"':
				escaped = true
			default:
				field.WriteRune(r)
			}
			inField = true
			continue
		}
		switch {
		case r == '\\':
			escaped = true
			inField = true
		case r == '\'' || r == '"':
			quote = r
			inField = true
		case unicode.IsSpace(r):
			flush()
		default:
			field.WriteRune(r)
			inField = true
		}
	}
	flush()
	return fields
}
