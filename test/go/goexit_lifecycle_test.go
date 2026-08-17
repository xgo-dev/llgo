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
	"strings"
	"testing"
)

const mainGoexitLifecycleProbe = `package main

import (
	"fmt"
	"runtime"
	_ "unsafe"
)

//go:linkname runtimeGStateForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GStateForTesting
func runtimeGStateForTesting() (count uint64, mainExited bool)

func init() {
	done := make(chan int, 1)
	defer func() { done <- 0 }()
	go func() {
		<-done
		for {
			count, mainExited := runtimeGStateForTesting()
			if count == 1 && mainExited {
				break
			}
		}
		fmt.Println("WORKER_RETURNING")
	}()
	runtime.Goexit()
}

func main() {}
`

func TestMainGoexitLifecycleReleasedOnce(t *testing.T) {
	_, dir := prepareCallerAcceptanceProbe(t, mainGoexitLifecycleProbe)
	out, err := runLLGoProbe(t, dir)
	if err == nil {
		t.Fatalf("main Goexit probe unexpectedly succeeded:\\n%s", out)
	}
	worker := strings.Index(out, "WORKER_RETURNING")
	deadlock := strings.Index(out, "no goroutines (main called runtime.Goexit) - deadlock!")
	if worker < 0 || deadlock < 0 || worker > deadlock {
		t.Fatalf("worker must return before the last-goroutine deadlock:\\n%s", out)
	}
}
