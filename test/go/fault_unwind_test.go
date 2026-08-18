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

// Hardware faults in C code called from Go convert to recoverable Go
// panics, and the unrecovered traceback shows the fault-site chain — C
// frames down through the Go callers — captured from the signal ucontext
// (dynamically-resolved libunwind first, FP chain otherwise).
package gotest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const faultProbeMain = `package main

import (
	"fmt"
	"os"
	_ "unsafe"
)

const (
	LLGoFiles = "wrap/fault.c"
)

//go:linkname cexcSegv C.cexc_segv
func cexcSegv(depth int32)

//go:noinline
func viaGo() { cexcSegv(2) }

func one() {
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok || err.Error() != "runtime error: invalid memory address or nil pointer dereference" {
			panic(r)
		}
	}()
	viaGo()
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "recover" {
		// Three sequential faults: a handler leaving the signal blocked
		// after the longjmp escape survives the first and cores on the
		// second (the SA_NODEFER regression).
		one()
		one()
		one()
		fmt.Println("FAULT_RECOVER_OK")
		return
	}
	viaGo()
	fmt.Println("survived")
}
`

const faultProbeC = `#include <stdint.h>

/* volatile: a bare NULL store is UB and clang would propagate it into
 * "unreachable", turning the recursion into an infinite loop. */
static int32_t *volatile cexc_null;
volatile int32_t cexc_marks;

void cexc_leaf(void) { *cexc_null = 42; }

void cexc_mid(int32_t depth) {
	if (depth > 0) {
		cexc_mid(depth - 1);
		cexc_marks++;
		return;
	}
	cexc_leaf();
	cexc_marks++;
}

void cexc_segv(int32_t depth) {
	cexc_mid(depth);
	cexc_marks++;
}
`

var (
	faultLLGoOnce sync.Once
	faultLLGoBin  string
	faultLLGoErr  string
)

func faultLLGo(t *testing.T) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	if llgo := configuredLLGo(t); llgo != "" {
		return llgo
	}
	faultLLGoOnce.Do(func() {
		tmp, err := os.MkdirTemp("", "llgo-fault-bin")
		if err != nil {
			faultLLGoErr = err.Error()
			return
		}
		bin := filepath.Join(tmp, "llgo")
		build := exec.Command("go", "build", "-o", bin, "./cmd/llgo")
		build.Dir = repoRoot
		if out, berr := build.CombinedOutput(); berr != nil {
			faultLLGoErr = berr.Error() + "\n" + string(out)
			return
		}
		faultLLGoBin = bin
	})
	if faultLLGoErr != "" {
		t.Fatalf("building llgo failed: %s", faultLLGoErr)
	}
	return faultLLGoBin
}

func writeFaultProbe(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"main.go":      faultProbeMain,
		"go.mod":       "module faultprobe\n\ngo 1.21\n",
		"wrap/fault.c": faultProbeC,
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCFaultRecoverable(t *testing.T) {
	bin := faultLLGo(t)
	dir := writeFaultProbe(t)
	cmd := exec.Command(bin, "run", ".", "recover")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fault probe died on the signal: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "FAULT_RECOVER_OK") {
		t.Fatalf("recover marker missing:\n%s", out)
	}
}

func TestCFaultTraceback(t *testing.T) {
	bin := faultLLGo(t)
	dir := writeFaultProbe(t)
	cmd := exec.Command(bin, "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("unrecovered fault probe unexpectedly succeeded:\n%s", out)
	}
	wants := []string{
		"panic: runtime error: invalid memory address or nil pointer dereference",
		"goroutine 1 [running]:",
		"main.go:", // the Go caller frames carry file:line
	}
	if runtime.GOOS == "darwin" {
		// C frame names come from dladdr/libunwind; linux runners may
		// lack a .symtab-reading libunwind flavor.
		wants = append(wants, "cexc_segv")
	}
	for _, want := range wants {
		if !strings.Contains(string(out), want) {
			t.Fatalf("fault traceback missing %q:\n%s", want, out)
		}
	}
}
