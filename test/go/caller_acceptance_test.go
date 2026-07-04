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

// End-user acceptance for caller information: every scenario asserts the
// output an application developer actually sees — log prefixes, slog source
// attributes, testing failure locations, panic tracebacks — with the
// expected values verified against gc (`go run` prints the identical
// file:line for each probe).
package gotest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Scenario: the standard log package reports the caller's file:line
// (Lshortfile/Llongfile), and log/slog's AddSource reports it as a source
// attribute — through both the convenience functions and logger methods.
const loggingAcceptanceProbe = `package main

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

var checked = 0

func expectPrefix(out, want string) {
	if !strings.HasPrefix(out, want) {
		panic("bad location: got " + out + ", want prefix " + want)
	}
	checked++
}

type sink struct{ strings.Builder }

func main() {
	var buf sink
	log.SetOutput(&buf)
	log.SetFlags(log.Lshortfile)
	log.Println("p") // PKGLOG_MARK
	expectPrefix(buf.String(), "main.go:"+strconv.Itoa(PKGLOG_LINE)+":")

	buf.Reset()
	logger := log.New(&buf, "", log.Lshortfile)
	logger.Printf("m") // NEWLOG_MARK
	expectPrefix(buf.String(), "main.go:"+strconv.Itoa(NEWLOG_LINE)+":")

	buf.Reset()
	sl := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{AddSource: true}))
	sl.Info("s") // SLOGTXT_MARK
	if !strings.Contains(buf.String(), "main.go:"+strconv.Itoa(SLOGTXT_LINE)+" ") {
		panic("bad slog source: " + buf.String())
	}
	checked++

	buf.Reset()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{AddSource: true})))
	slog.Warn("w") // SLOGJSN_MARK
	if !strings.Contains(buf.String(), "main.go\",\"line\":"+strconv.Itoa(SLOGJSN_LINE)) {
		panic("bad slog json source: " + buf.String())
	}
	checked++

	if checked != 4 {
		panic("scenario undercount")
	}
	os.Stdout.WriteString("LOGGING_OK\n")
}
`

func TestCallerAcceptanceLogging(t *testing.T) {
	runCallerAcceptanceProbe(t, loggingAcceptanceProbe, "LOGGING_OK",
		"PKGLOG", "NEWLOG", "SLOGTXT", "SLOGJSN")
}

// Scenario: an unrecovered panic prints a Go-style traceback — function
// names plus file:line for the panic site and its callers — and exits 2.
const panicAcceptanceProbe = `package main

import "runtime"

//go:noinline
func boom() {
	panic("acceptance-boom") // PANIC_MARK
}

//go:noinline
func caller() {
	boom() // PANIC_CALLER_MARK
}

func main() {
	_ = runtime.NumCPU() // keep the runtime package linked
	caller() // PANIC_MAIN_MARK
}
`

func TestCallerAcceptancePanicTraceback(t *testing.T) {
	source, dir := prepareCallerAcceptanceProbe(t, panicAcceptanceProbe, "PANIC", "PANIC_CALLER", "PANIC_MAIN")
	out, err := runLLGoProbe(t, dir)
	if err == nil {
		t.Fatalf("panic probe unexpectedly succeeded:\n%s", out)
	}
	for _, want := range []string{
		"panic: acceptance-boom",
		"goroutine 1 [running]:",
		"main.boom(...)",
		"main.go:" + markerLineOf(t, source, "PANIC_MARK"),
		"main.caller(...)",
		"main.go:" + markerLineOf(t, source, "PANIC_CALLER_MARK"),
		"main.main(...)",
		"main.go:" + markerLineOf(t, source, "PANIC_MAIN_MARK"),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("panic traceback missing %q:\n%s", want, out)
		}
	}
}

// Scenario: a failing test reports the t.Errorf call's file:line, exactly
// like `go test` (the whole testing harness runs under llgo).
func TestCallerAcceptanceTestingFailure(t *testing.T) {
	dir := t.TempDir()
	const testSrc = `package tpkg

import "testing"

func TestFail(t *testing.T) {
	t.Errorf("acceptance failure") // TESTING_MARK
}
`
	wantLine := markerLineOf(t, testSrc, "TESTING_MARK")
	if err := os.WriteFile(filepath.Join(dir, "x_test.go"), []byte(testSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tpkg\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runLLGoInModule(t, dir, "test", ".")
	if err == nil {
		t.Fatalf("failing test unexpectedly passed:\n%s", out)
	}
	if !strings.Contains(out, "x_test.go:"+wantLine+": acceptance failure") {
		t.Fatalf("testing failure missing exact location:\n%s", out)
	}
	if !strings.Contains(out, "--- FAIL: TestFail") {
		t.Fatalf("testing failure missing FAIL header:\n%s", out)
	}
}

// Scenario grab-bag verified against gc: caller info from goroutines, from
// init, from deferred functions; FuncForPC names for methods, closures and
// generic instantiations; and the errors-with-stack pattern (capture pcs at
// error creation, symbolize when printing — the zap/sentry idiom).
const introspectionAcceptanceProbe = `package main

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func expectLine(what string, line, want int) {
	if line != want {
		panic("bad " + what + " line: " + strconv.Itoa(line) + " want " + strconv.Itoa(want))
	}
}

func expectFunc(what, got, want string) {
	if got != want {
		panic("bad " + what + ": " + got + " want " + want)
	}
}

var initLine int

func init() {
	_, _, initLine, _ = runtime.Caller(0) // INIT_MARK
}

type receiver struct{}

// Every probe body writes a distinct global: identical bodies would be
// folded to one address by linker ICF and FuncForPC could only report the
// surviving symbol (observed: value/pointer/closure/generic all folding).
var sinkValue, sinkPointer, sinkClosure, sinkGeneric int

//go:noinline
func (receiver) value() uintptr { sinkValue++; pc, _, _, _ := runtime.Caller(0); return pc }

//go:noinline
func (*receiver) pointer() uintptr { sinkPointer++; pc, _, _, _ := runtime.Caller(0); return pc }

//go:noinline
func generic[T any](v T) uintptr { sinkGeneric++; pc, _, _, _ := runtime.Caller(0); return pc }

type stackErr struct {
	msg string
	pcs [8]uintptr
	n   int
}

func (e *stackErr) Error() string { return e.msg }

//go:noinline
func newStackErr(msg string) *stackErr {
	e := &stackErr{msg: msg}
	e.n = runtime.Callers(1, e.pcs[:])
	return e
}

func main() {
	expectLine("init caller", initLine, INIT_LINE)

	var wg sync.WaitGroup
	wg.Add(1)
	var gLine int
	var gFile string
	go func() {
		defer wg.Done()
		_, gFile, gLine, _ = runtime.Caller(0) // GOROUTINE_MARK
	}()
	wg.Wait()
	expectLine("goroutine caller", gLine, GOROUTINE_LINE)
	if !strings.HasSuffix(gFile, "main.go") {
		panic("bad goroutine caller file: " + gFile)
	}

	var dLine int
	func() {
		defer func() {
			_, _, dLine, _ = runtime.Caller(0) // DEFER_MARK
		}()
	}()
	expectLine("deferred caller", dLine, DEFER_LINE)

	var r receiver
	expectFunc("value method", runtime.FuncForPC(r.value()).Name(), "main.receiver.value")
	expectFunc("pointer method", runtime.FuncForPC((&r).pointer()).Name(), "main.(*receiver).pointer")
	closure := func() uintptr { sinkClosure++; pc, _, _, _ := runtime.Caller(0); return pc }
	// LLGo names anonymous functions pkg.fn$N (gc uses pkg.fn.funcN);
	// accept both — the P4 name work decides whether to normalize.
	name := runtime.FuncForPC(closure()).Name()
	if !strings.HasPrefix(name, "main.main.func") && !strings.HasPrefix(name, "main.main$") {
		panic("bad closure name: " + name)
	}
	gname := runtime.FuncForPC(generic(0)).Name()
	if !strings.HasPrefix(gname, "main.generic") {
		panic("bad generic name: " + gname)
	}

	err := newStackErr("wrapped") // STACKERR_MARK
	frames := runtime.CallersFrames(err.pcs[:err.n])
	found := false
	for {
		frame, more := frames.Next()
		if frame.Function == "main.main" {
			expectLine("stack error capture", frame.Line, STACKERR_LINE)
			found = true
		}
		if !more {
			break
		}
	}
	if !found {
		panic("stack error: main.main frame missing")
	}

	os.Stdout.WriteString("INTROSPECTION_OK\n")
}
`

func TestCallerAcceptanceIntrospection(t *testing.T) {
	runCallerAcceptanceProbe(t, introspectionAcceptanceProbe, "INTROSPECTION_OK",
		"INIT", "GOROUTINE", "DEFER", "STACKERR")
}

// --- harness ---

// prepareCallerAcceptanceProbe substitutes NAME_LINE placeholders with the
// line numbers of the corresponding NAME_MARK comments and writes main.go
// into a temp dir. Returns the final source and the dir.
func prepareCallerAcceptanceProbe(t *testing.T, source string, names ...string) (string, string) {
	t.Helper()
	for _, name := range names {
		line := markerLine(source, name+"_MARK")
		if line == 0 {
			t.Fatalf("marker %s_MARK not found", name)
		}
		source = strings.ReplaceAll(source, name+"_LINE", strconv.Itoa(line))
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return source, dir
}

func markerLineOf(t *testing.T, source, marker string) string {
	t.Helper()
	line := markerLine(source, marker)
	if line == 0 {
		t.Fatalf("marker %s not found", marker)
	}
	return strconv.Itoa(line)
}

func runLLGoProbe(t *testing.T, dir string) (string, error) {
	t.Helper()
	repoRoot := findStringConversionRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	cmd := exec.Command("go", "run", "./cmd/llgo", "run", "-a", filepath.Join(dir, "main.go"))
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runCallerAcceptanceProbe(t *testing.T, source, okMarker string, names ...string) {
	t.Helper()
	_, dir := prepareCallerAcceptanceProbe(t, source, names...)
	out, err := runLLGoProbe(t, dir)
	if err != nil {
		t.Fatalf("acceptance probe failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, okMarker) {
		t.Fatalf("acceptance probe missing %s:\n%s", okMarker, out)
	}
}

// Scenario: a module-named main package must still report gc's "main."
// prefix in every runtime name (frame filters match on it); the other
// probes run as command-line-arguments and cannot catch a regression here.
func TestCallerAcceptanceModuleMainNaming(t *testing.T) {
	dir := t.TempDir()
	const src = `package main

import (
	"os"
	"runtime"
	"strings"
)

//go:noinline
func here() (string, bool) {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	return runtime.FuncForPC(pc).Name(), true
}

func main() {
	name, ok := here()
	if !ok || name != "main.here" {
		panic("bad module-main name: " + name)
	}
	var pcs [8]uintptr
	frames := runtime.CallersFrames(pcs[:runtime.Callers(0, pcs[:])])
	foundMain := false
	for {
		f, more := frames.Next()
		if strings.HasPrefix(f.Function, "mymainmod.") {
			panic("module path leaked into frame name: " + f.Function)
		}
		if f.Function == "main.main" {
			foundMain = true
		}
		if !more {
			break
		}
	}
	if !foundMain {
		panic("main.main frame missing")
	}
	os.Stdout.WriteString("MODMAIN_OK\n")
}
`
	writeCallerAcceptanceModule(t, dir, map[string]string{
		"main.go": src,
		"go.mod":  "module mymainmod\n\ngo 1.21\n",
	})
	out, err := runLLGoInModule(t, dir, "run", ".")
	if err != nil {
		t.Fatalf("module-main probe failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "MODMAIN_OK") {
		t.Fatalf("module-main probe missing marker:\n%s", out)
	}
}

// Scenario: a hardware fault inside C code called from Go (NULL store in a
// C helper) converts to a Go panic that recover observes with gc's error
// text; the process must not die on the raw signal.
func TestCallerAcceptanceCFaultRecover(t *testing.T) {
	dir := t.TempDir()
	const src = `package main

import (
	"fmt"
	"os"
	"runtime/debug"
	_ "unsafe"
)

const (
	LLGoFiles = "wrap/fault.c"
)

//go:linkname cexcSegv C.cexc_segv
func cexcSegv(depth int32)

//go:noinline
func viaGo() {
	cexcSegv(2)
}

func one(last bool) {
	defer func() {
		r := recover()
		if r == nil {
			fmt.Println("no panic")
			return
		}
		err, ok := r.(error)
		if !ok || err.Error() != "runtime error: invalid memory address or nil pointer dereference" {
			panic(r)
		}
		if last {
			os.Stdout.WriteString("CFAULT_OK\n")
			os.Stdout.Write(debug.Stack())
		}
	}()
	viaGo()
}

func main() {
	// Three sequential faults: a handler that leaves the signal blocked
	// after the longjmp escape (savemask=0 jmpbufs) survives the first
	// fault and core-dumps on the second — the exact CI failure mode.
	one(false)
	one(false)
	one(true)
}
`
	const csrc = `#include <stdint.h>

/* volatile: a bare NULL store is UB and clang would propagate it into
 * "unreachable", turning the recursion into an infinite loop. */
static int32_t *volatile cexc_null;
volatile int32_t cexc_marks;

/* Non-static: the fault-chain tail cut keeps frames dladdr can name;
 * static helpers have no symbol and would end the visible chain. */
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
	writeCallerAcceptanceModule(t, dir, map[string]string{
		"main.go":      src,
		"go.mod":       "module cfault\n\ngo 1.21\n",
		"wrap/fault.c": csrc,
	})
	out, err := runLLGoInModule(t, dir, "run", ".")
	if err != nil {
		t.Fatalf("C fault probe failed (process died on the signal?): %v\n%s", err, out)
	}
	if !strings.Contains(out, "CFAULT_OK") {
		t.Fatalf("C fault probe missing marker:\n%s", out)
	}
	// The fault-site chain must be visible from the recovered deferred
	// function. C frame names come from dladdr, which on linux only sees
	// dynamic symbols — there the C frames show as raw pcs and only the
	// Go side of the chain is asserted (foreign-symbol naming is the
	// planned link-phase hole-sentinel follow-up).
	wants := []string{"main.viaGo", "main.main"}
	if runtime.GOOS == "darwin" {
		wants = append(wants, "cexc_segv")
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("recovered stack missing fault frame %q:\n%s", want, out)
		}
	}
}

func writeCallerAcceptanceModule(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

var (
	acceptanceLLGoOnce sync.Once
	acceptanceLLGoBin  string
	acceptanceLLGoErr  string
)

// runLLGoInModule builds the llgo binary once per test process and runs it
// with the module directory as the working directory (llgo resolves
// packages relative to the cwd, and `go run` refuses directories outside
// its own module).
func runLLGoInModule(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	repoRoot := findStringConversionRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	acceptanceLLGoOnce.Do(func() {
		tmp, err := os.MkdirTemp("", "llgo-acceptance-bin")
		if err != nil {
			acceptanceLLGoErr = err.Error()
			return
		}
		bin := filepath.Join(tmp, "llgo")
		build := exec.Command("go", "build", "-o", bin, "./cmd/llgo")
		build.Dir = repoRoot
		if bout, berr := build.CombinedOutput(); berr != nil {
			acceptanceLLGoErr = berr.Error() + "\n" + string(bout)
			return
		}
		acceptanceLLGoBin = bin
	})
	if acceptanceLLGoErr != "" {
		t.Fatalf("building llgo failed: %s", acceptanceLLGoErr)
	}
	cmd := exec.Command(acceptanceLLGoBin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
