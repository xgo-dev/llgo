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

package filecheck

import (
	"fmt"
	"regexp"
	"strings"
)

type kind string

const (
	kindCheck kind = "CHECK"
	kindLine  kind = "CHECK-LINE"
	kindNext  kind = "CHECK-NEXT"
	kindSame  kind = "CHECK-SAME"
	kindNot   kind = "CHECK-NOT"
	kindLabel kind = "CHECK-LABEL"
	kindEmpty kind = "CHECK-EMPTY"
)

type directive struct {
	kind    kind
	pattern string
	re      *regexp.Regexp
	line    int
}

type pos struct {
	line int
	col  int
}

type match struct {
	start pos
	end   pos
}

func HasDirectives(text string) (bool, error) {
	lines := splitLines(text)
	for _, line := range lines {
		_, hasDirective, ok, err := parseDirectiveLine(line)
		if err != nil {
			return false, err
		}
		if hasDirective && ok {
			return true, nil
		}
	}
	return false, nil
}

func Match(filename, checks, input string) error {
	directives, err := parseDirectives(filename, checks)
	if err != nil {
		return err
	}
	lines := splitLines(input)
	if len(lines) == 0 {
		lines = []string{""}
	}
	cur := pos{line: 0, col: 0}
	prev := match{}
	hasPrev := false
	var pendingNot []directive
	for _, dir := range directives {
		switch dir.kind {
		case kindNot:
			pendingNot = append(pendingNot, dir)
			continue
		case kindEmpty:
			if !hasPrev {
				return fmt.Errorf("%s:%d: CHECK-EMPTY requires a prior positive match", filename, dir.line)
			}
			target := pos{line: prev.start.line + 1, col: 0}
			if target.line >= len(lines) {
				return fmt.Errorf("%s:%d: CHECK-EMPTY expected an empty line after input:%d", filename, dir.line, prev.start.line+1)
			}
			if err := checkPendingNot(filename, pendingNot, lines, cur, target); err != nil {
				return err
			}
			if lines[target.line] != "" {
				return fmt.Errorf("%s:%d: CHECK-EMPTY expected input:%d to be empty", filename, dir.line, target.line+1)
			}
			pendingNot = nil
			prev = match{start: target, end: target}
			cur = target
			hasPrev = true
			continue
		}

		var found match
		var ok bool
		switch dir.kind {
		case kindCheck:
			found, ok = findForward(lines, cur, dir.re)
		case kindLine, kindLabel:
			found, ok = findForwardFullLine(lines, cur, dir.re)
		case kindNext:
			if !hasPrev {
				return fmt.Errorf("%s:%d: CHECK-NEXT requires a prior positive match", filename, dir.line)
			}
			found, ok = findLine(lines, prev.start.line+1, 0, dir.re)
		case kindSame:
			if !hasPrev {
				return fmt.Errorf("%s:%d: CHECK-SAME requires a prior positive match", filename, dir.line)
			}
			found, ok = findLine(lines, prev.start.line, prev.end.col, dir.re)
		default:
			return fmt.Errorf("%s:%d: unsupported directive %s", filename, dir.line, dir.kind)
		}
		if !ok {
			switch dir.kind {
			case kindNext:
				return fmt.Errorf("%s:%d: CHECK-NEXT %q did not match input:%d, input: %s", filename, dir.line, dir.pattern, prev.start.line+2, input)
			case kindSame:
				return fmt.Errorf("%s:%d: CHECK-SAME %q did not match input:%d, input: %s", filename, dir.line, dir.pattern, prev.start.line+1, input)
			default:
				return fmt.Errorf("%s:%d: %s %q was not found", filename, dir.line, dir.kind, dir.pattern)
			}
		}
		if err := checkPendingNot(filename, pendingNot, lines, cur, found.start); err != nil {
			return err
		}
		pendingNot = nil
		prev = found
		cur = found.end
		if dir.kind == kindLabel {
			cur = pos{line: min(found.start.line+1, len(lines)-1), col: 0}
		}
		hasPrev = true
	}

	end := pos{line: len(lines) - 1, col: len(lines[len(lines)-1])}
	if err := checkPendingNot(filename, pendingNot, lines, cur, end); err != nil {
		return err
	}
	return nil
}

func parseDirectives(filename, checks string) ([]directive, error) {
	lines := splitLines(checks)
	var directives []directive
	for idx, line := range lines {
		dir, hasDirective, ok, err := parseDirectiveLine(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", filename, idx+1, err)
		}
		if hasDirective && ok {
			dir.line = idx + 1
			directives = append(directives, dir)
		}
	}
	if len(directives) == 0 {
		return nil, fmt.Errorf("%s: no FileCheck directives found", filename)
	}
	return directives, nil
}

func parseDirectiveLine(line string) (directive, bool, bool, error) {
	rest := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(rest, "//") {
		return directive{}, false, false, nil
	}
	rest = strings.TrimLeft(rest[2:], " \t")
	if strings.HasPrefix(rest, "CHECK:") {
		pattern := trimDirectivePattern(strings.TrimPrefix(rest, "CHECK:"))
		re, err := compilePattern(pattern)
		if err != nil {
			return directive{}, true, false, err
		}
		return directive{kind: kindCheck, pattern: pattern, re: re}, true, true, nil
	}
	if !strings.HasPrefix(rest, "CHECK-") {
		return directive{}, false, false, nil
	}
	suffixRest := strings.TrimPrefix(rest, "CHECK-")
	colon := strings.IndexByte(suffixRest, ':')
	if colon < 0 {
		return directive{}, true, false, fmt.Errorf("missing ':' in directive")
	}
	suffix := suffixRest[:colon]
	pattern := trimDirectivePattern(suffixRest[colon+1:])
	k, ok := parseKind(suffix)
	if !ok {
		return directive{}, true, false, fmt.Errorf("unknown directive CHECK-%s", suffix)
	}
	if k == kindEmpty {
		if pattern != "" {
			return directive{}, true, false, fmt.Errorf("CHECK-EMPTY must not have a pattern")
		}
		return directive{kind: k}, true, true, nil
	}
	re, err := compilePattern(pattern)
	if err != nil {
		return directive{}, true, false, err
	}
	return directive{kind: k, pattern: pattern, re: re}, true, true, nil
}

func parseKind(suffix string) (kind, bool) {
	switch suffix {
	case "LINE":
		return kindLine, true
	case "NEXT":
		return kindNext, true
	case "SAME":
		return kindSame, true
	case "NOT":
		return kindNot, true
	case "LABEL":
		return kindLabel, true
	case "EMPTY":
		return kindEmpty, true
	default:
		return "", false
	}
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty pattern")
	}
	var b strings.Builder
	for {
		idx := strings.Index(pattern, "{{")
		if idx < 0 {
			b.WriteString(regexp.QuoteMeta(pattern))
			break
		}
		b.WriteString(regexp.QuoteMeta(pattern[:idx]))
		pattern = pattern[idx+2:]
		end := strings.Index(pattern, "}}")
		if end < 0 {
			return nil, fmt.Errorf("unterminated '{{' in pattern")
		}
		b.WriteString("(?:")
		b.WriteString(pattern[:end])
		b.WriteString(")")
		pattern = pattern[end+2:]
	}
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, err
	}
	return re, nil
}

func findForward(lines []string, start pos, re *regexp.Regexp) (match, bool) {
	for lineIdx := start.line; lineIdx < len(lines); lineIdx++ {
		col := 0
		if lineIdx == start.line {
			col = start.col
		}
		if col > len(lines[lineIdx]) {
			col = len(lines[lineIdx])
		}
		if loc := re.FindStringIndex(lines[lineIdx][col:]); loc != nil {
			return match{
				start: pos{line: lineIdx, col: col + loc[0]},
				end:   pos{line: lineIdx, col: col + loc[1]},
			}, true
		}
	}
	return match{}, false
}

func findLine(lines []string, lineIdx, startCol int, re *regexp.Regexp) (match, bool) {
	if lineIdx < 0 || lineIdx >= len(lines) {
		return match{}, false
	}
	if startCol > len(lines[lineIdx]) {
		startCol = len(lines[lineIdx])
	}
	if loc := re.FindStringIndex(lines[lineIdx][startCol:]); loc != nil {
		return match{
			start: pos{line: lineIdx, col: startCol + loc[0]},
			end:   pos{line: lineIdx, col: startCol + loc[1]},
		}, true
	}
	return match{}, false
}

func findForwardFullLine(lines []string, start pos, re *regexp.Regexp) (match, bool) {
	lineIdx := start.line
	if start.col > 0 {
		lineIdx++
	}
	for ; lineIdx < len(lines); lineIdx++ {
		if loc := re.FindStringIndex(lines[lineIdx]); loc != nil && loc[0] == 0 && loc[1] == len(lines[lineIdx]) {
			return match{
				start: pos{line: lineIdx, col: 0},
				end:   pos{line: lineIdx, col: len(lines[lineIdx])},
			}, true
		}
	}
	return match{}, false
}

func checkPendingNot(filename string, pending []directive, lines []string, from, to pos) error {
	for _, dir := range pending {
		if at, text, ok := findForbidden(lines, from, to, dir.re); ok {
			return fmt.Errorf("%s:%d: %s %q matched forbidden text %q at input:%d", filename, dir.line, dir.kind, dir.pattern, text, at.line+1)
		}
	}
	return nil
}

func findForbidden(lines []string, from, to pos, re *regexp.Regexp) (pos, string, bool) {
	if before(to, from) {
		return pos{}, "", false
	}
	for lineIdx := from.line; lineIdx <= to.line && lineIdx < len(lines); lineIdx++ {
		startCol := 0
		endCol := len(lines[lineIdx])
		if lineIdx == from.line {
			startCol = min(startCol+from.col, len(lines[lineIdx]))
		}
		if lineIdx == to.line {
			endCol = min(endCol, to.col)
		}
		if startCol > endCol {
			continue
		}
		segment := lines[lineIdx][startCol:endCol]
		if loc := re.FindStringIndex(segment); loc != nil {
			return pos{line: lineIdx, col: startCol + loc[0]}, segment[loc[0]:loc[1]], true
		}
	}
	return pos{}, "", false
}

func before(a, b pos) bool {
	if a.line != b.line {
		return a.line < b.line
	}
	return a.col < b.col
}

func splitLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

func trimDirectivePattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	if pattern[0] == ' ' || pattern[0] == '\t' {
		return pattern[1:]
	}
	return pattern
}
