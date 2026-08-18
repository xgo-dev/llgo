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

package gotest

import (
	"os"
	"os/exec"
	"testing"
)

func configuredLLGo(t *testing.T) string {
	t.Helper()
	name := os.Getenv("LLGO_TEST_LLGO")
	if name == "" {
		name = os.Getenv("LLGO")
	}
	return configuredTestTool(t, name)
}

func configuredTestTool(t *testing.T, name string) string {
	t.Helper()
	if name == "" {
		return ""
	}
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("resolve configured test tool %q: %v", name, err)
	}
	return path
}

// commandForTest reuses repository tools built by the build toolchain when a
// versioned test would otherwise rebuild them with its target Go toolchain.
// Ordinary non-versioned tests retain the original go-run path.
func commandForTest(t *testing.T, dir, name string, args ...string) *exec.Cmd {
	t.Helper()
	modfile := os.Getenv("LLGO_TEST_MODFILE")
	if name == "go" && len(args) >= 2 && args[0] == "run" && args[1] == "./cmd/llgo" {
		if llgo := configuredLLGo(t); llgo != "" {
			name = llgo
			args = args[2:]
			if modfile != "" && len(args) != 0 {
				args = append([]string{args[0], "-modfile=" + modfile}, args[1:]...)
			}
		}
	} else if name == "go" && len(args) >= 2 && args[0] == "run" && args[1] == "./chore/llgen" {
		if llgen := configuredTestTool(t, os.Getenv("LLGO_TEST_LLGEN")); llgen != "" {
			name = llgen
			args = args[2:]
			if len(args) != 0 {
				dir = args[len(args)-1]
				args[len(args)-1] = "."
			}
		}
	}
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}
