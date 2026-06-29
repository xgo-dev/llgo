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

func Match(filename, input string) error {
	return MatchWithPrefixes(filename, input)
}

func MatchWithPrefixes(filename, input string, prefixes ...string) error {
	args := make([]string, 0, len(prefixes)+1)
	for _, prefix := range prefixes {
		args = append(args, "--check-prefix="+prefix)
	}
	args = append(args, filename)
	cmd, err := llvm.New("").FileCheck(args...)
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
