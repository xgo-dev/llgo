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
	"runtime"
	"testing"
)

const numGoroutineChildEnv = "LLGO_TEST_NUM_GOROUTINE_CHILD"

func TestRuntimeNumGoroutineIncludesNewProc(t *testing.T) {
	if runtime.GOARCH != "wasm" && os.Getenv(numGoroutineChildEnv) == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeNumGoroutineIncludesNewProc$")
		cmd.Env = append(os.Environ(), numGoroutineChildEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("NumGoroutine child failed: %v\n%s", err, output)
		}
		return
	}

	before := runtime.NumGoroutine()
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		<-release
		close(done)
	}()
	<-started
	during := runtime.NumGoroutine()
	if during != before+1 {
		t.Fatalf("NumGoroutine: before=%d during=%d, want %d", before, during, before+1)
	}
	close(release)
	<-done
}
