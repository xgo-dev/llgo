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

	"github.com/xgo-dev/llgo/cmd/xtool/env/llvm"
)

func Match(filename, input string) error {
	return MatchWithPrefixes(filename, input)
}

func MatchWithPrefixes(filename, input string, prefixes ...string) error {
	return match(filename, input, false, prefixes...)
}

// MatchWithTargetPrefixes allows architecture and target-specific prefixes to
// be absent while still checking portable CHECK directives.
func MatchWithTargetPrefixes(filename, input string, prefixes ...string) error {
	return match(filename, input, true, prefixes...)
}

func match(filename, input string, allowUnusedPrefixes bool, prefixes ...string) error {
	args := make([]string, 0, len(prefixes)+2)
	if allowUnusedPrefixes {
		args = append(args, "--allow-unused-prefixes")
	}
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

// TargetPrefixes returns CHECK plus the applicable operating-system,
// architecture, and specific compile-target prefixes. A named target produces
// TARGET-<TARGET>; otherwise GOOS and GOARCH produce <GOOS>-<GOARCH>.
func TargetPrefixes(goos, goarch, target string) []string {
	prefixes := []string{"CHECK"}
	if (goos == "") != (goarch == "") {
		panic("filecheck: GOOS and GOARCH must be provided together")
	}
	if goos != "" && goarch != "" {
		prefixes = append(prefixes, strings.ToUpper(goos), ArchitecturePrefix(goarch))
	}
	if target != "" {
		return append(prefixes, "TARGET-"+strings.ToUpper(target))
	}
	if goos != "" && goarch != "" {
		return append(prefixes, strings.ToUpper(goos+"-"+goarch))
	}
	return prefixes
}

// ArchitecturePrefix returns the FileCheck prefix shared by targets with the
// same GOARCH.
func ArchitecturePrefix(goarch string) string {
	return strings.ToUpper(goarch)
}
