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
	"path/filepath"
	"strings"
	"testing"
)

const recoverThenDeferredPanicProbe = `package main

func end() {
	if recovered := recover(); recovered != nil {
		defer panic(recovered)
		println("will panic in defer")
	}
	println("end")
}

func main() {
	defer end()
	panic("panic in main")
}
`

const cgoDeferredFreeProbe = `package main

/*
#include <stdlib.h>
*/
import "C"

func main() {
	p := C.malloc(8)
	if p == nil {
		panic("C.malloc returned nil")
	}
	defer C.free(p)
}
`

func TestRecoverThenDeferredPanicIRTerminatesBlocks(t *testing.T) {
	ir := llgoIRFromProbe(t, "recover-then-deferred-panic", recoverThenDeferredPanicProbe)
	assertNoInstructionsAfterUnreachable(t, ir)
}

func TestCgoDeferredFreeReleasesNodeBeforeCall(t *testing.T) {
	if strings.TrimSpace(runGoCmd(t, "", "env", "CGO_ENABLED")) != "1" {
		t.Skip("cgo is disabled")
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang is unavailable")
	}

	ir := llgoIRFromProbe(t, "cgo-deferred-free", cgoDeferredFreeProbe)
	freeDefer := indexLineContaining(ir, "call void @", "FreeDeferNode")
	deferredCall := indexLineContaining(ir, "call void %")
	if freeDefer < 0 {
		t.Fatalf("missing FreeDeferNode call in IR:\n%s", ir)
	}
	if deferredCall < 0 {
		t.Fatalf("missing deferred indirect call in IR:\n%s", ir)
	}
	if freeDefer > deferredCall {
		t.Fatalf("FreeDeferNode must run before deferred call, got FreeDeferNode line %d after call line %d", freeDefer, deferredCall)
	}
}

func llgoIRFromProbe(t *testing.T, name, src string) string {
	t.Helper()

	root := findLLGoRoot(t)
	dir, err := os.MkdirTemp(root, "."+name+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("remove temp probe dir %s: %v", dir, err)
		}
	})

	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, root, "run", "./chore/llgen", filepath.ToSlash(dir))
	data, err := os.ReadFile(filepath.Join(dir, "llgo_autogen.ll"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertNoInstructionsAfterUnreachable(t *testing.T, ir string) {
	t.Helper()

	lines := strings.Split(ir, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "unreachable" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "" {
				continue
			}
			if trimmed == "}" || isLLVMBasicBlockLabel(trimmed) {
				break
			}
			t.Fatalf("instruction after unreachable at IR line %d: %s", j+1, trimmed)
		}
	}
}

func isLLVMBasicBlockLabel(line string) bool {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return false
	}
	return !strings.ContainsAny(line[:colon], " \t")
}

func indexLineContaining(s string, parts ...string) int {
	for i, line := range strings.Split(s, "\n") {
		ok := true
		for _, part := range parts {
			if !strings.Contains(line, part) {
				ok = false
				break
			}
		}
		if ok {
			return i + 1
		}
	}
	return -1
}
