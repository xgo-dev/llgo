//go:build !wasm

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

package llgocmd

// These are host command-line compatibility tests: the test process invokes
// Go/LLGo and the produced host executable. A wasm module cannot spawn those
// host tools; wasm target execution is covered by internal/build and dev tests.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func runGoCompiler(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return runCompilerCommand(t, dir, "go", args...)
}

func runCompilerCommand(t *testing.T, dir, compiler string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(compiler, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// Successful output is compared exactly by several compatibility tests.
	// Include stderr only when it helps diagnose a failed command.
	if err != nil {
		stdout.Write(stderr.Bytes())
	}
	return stdout.String(), err
}

func runExecutable(t *testing.T, dir, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func executablePath(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}
