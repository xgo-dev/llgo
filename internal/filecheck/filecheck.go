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
	"bytes"
	"fmt"
	"strings"

	"github.com/goplus/llgo/xtool/env/llvm"
)

var supportedDirectives = []string{
	"CHECK:",
	"CHECK-NEXT:",
	"CHECK-SAME:",
	"CHECK-NOT:",
	"CHECK-LABEL:",
	"CHECK-EMPTY:",
	"CHECK-DAG:",
	"CHECK-COUNT-",
}

func HasDirectives(text string) (bool, error) {
	for _, line := range splitLines(text) {
		bodyStart, ok := directiveBody(line)
		if !ok {
			continue
		}
		body := line[bodyStart:]
		for _, directive := range supportedDirectives {
			if strings.HasPrefix(body, directive) {
				return true, nil
			}
		}
	}
	return false, nil
}

func Match(filename, input string) error {
	cmd, err := llvm.New("").FileCheck(filename)
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w\n%s", err, strings.TrimRight(stderr.String(), "\n"))
		}
		return err
	}
	return nil
}

func directiveBody(line string) (int, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "//") {
		return 0, false
	}
	indent := len(line) - len(trimmed)
	return indent + 2 + lenLeftTrim(trimmed[2:], " \t"), true
}

func lenLeftTrim(s, cutset string) int {
	return len(s) - len(strings.TrimLeft(s, cutset))
}

func splitLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}
