//go:build !llgo

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

package cl

import (
	"sync"
	"testing"

	gossa "golang.org/x/tools/go/ssa"
)

func TestCallerTrackingPrecomputeFreezesConcurrentReads(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t,
		"example.com/dep", `package dep
import "runtime"
func Where() { runtime.Caller(0) }
`,
		"example.com/root", `package root
import "example.com/dep"
func Logs() { dep.Where() }
`)
	tracking := NewCallerTracking()
	tracking.Precompute([]*gossa.Package{root})
	if !tracking.frozen {
		t.Fatal("CallerTracking was not frozen after precomputation")
	}
	if !runtimeCallerBaseSet(tracking, dep)[dep.Func("Where")] {
		t.Fatal("precomputed base set lost runtime caller function")
	}
	if !runtimeCallerFuncSet(tracking, root)[root.Func("Logs")] {
		t.Fatal("precomputed extended set lost cross-package caller")
	}

	var wg sync.WaitGroup
	errs := make(chan struct{}, 32)
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !runtimeCallerBaseSet(tracking, dep)[dep.Func("Where")] || !runtimeCallerFuncSet(tracking, root)[root.Func("Logs")] {
				errs <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(errs)
	if len(errs) != 0 {
		t.Fatal("concurrent read lost precomputed caller tracking data")
	}
}
