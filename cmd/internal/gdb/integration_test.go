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

package gdb

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/quoted"
)

func TestGDBIntegration(t *testing.T) {
	gdbPath, fixtureDir, executable := integrationBuildFixture(t)
	source := filepath.Join(fixtureDir, "main.go")

	runtimeLine := integrationMarkerLine(t, source, "LLDB_BREAK: runtime_values")
	interfaceLine := integrationMarkerLine(t, source, "LLDB_BREAK: interface_values")
	functionLine := integrationMarkerLine(t, source, "LLDB_BREAK: function_values")
	containerLine := integrationMarkerLine(t, source, "LLDB_BREAK: container_values")
	args := integrationGDBArgs(executable)
	args = append(args,
		"-ex", integrationBreakpoint(source, runtimeLine),
		"-ex", integrationBreakpoint(source, interfaceLine),
		"-ex", integrationBreakpoint(source, functionLine),
		"-ex", integrationBreakpoint(source, containerLine),
		"-ex", "break main.InspectGoroutineValues",
		"-ex", "run",
		"-ex", "echo LLGO_CASE=runtime\\n",
		"-ex", "llgo status",
		"-ex", "p text",
		"-ex", "p binary",
		"-ex", "p ints",
		"-ex", "p namedInts",
		"-ex", "continue",
		"-ex", "echo LLGO_CASE=interface\\n",
		"-ex", "p nilAny",
		"-ex", "p anyInt",
		"-ex", "p anyText",
		"-ex", "p nilFoo",
		"-ex", "p foo",
		"-ex", "p err",
		"-ex", "continue",
		"-ex", "echo LLGO_CASE=function\\n",
		"-ex", "p plain",
		"-ex", "p closure",
		"-ex", "p bound",
		"-ex", "p nilFunc",
		"-ex", "continue",
		"-ex", "echo LLGO_CASE=container\\n",
		"-ex", "set print elements 64",
		"-ex", "p nilMap",
		"-ex", "p single",
		"-ex", "p named",
		"-ex", "p many",
		"-ex", "p queued",
		"-ex", "p namedChannel",
		"-ex", "p closedChannel",
		"-ex", "continue",
		"-ex", "echo LLGO_CASE=goroutine\\n",
		"-ex", "llgo goroutines",
		"-ex", gdbSourceCommand(filepath.Join(fixtureDir, "gdb_registry.py")),
		"-ex", "llgo goroutine 1 bt 3",
	)
	output := integrationRunGDB(t, gdbPath, fixtureDir, args...)
	for _, expected := range []string{
		"LLGO_CASE=runtime",
		"LLGo debugger schema v1 (runtime layout v2)",
		`= "hello"`,
		`= "a\000b"`,
		"len=2 cap=4 = {7, 8}",
		"len=4 cap=4 = {11, 12, 13, 14}",
		"LLGO_CASE=interface",
		"= nil",
		"= type=int",
		"= type=string",
		"= type=*main.Struct",
		"= type=*errors.errorString",
		"LLGO_CASE=function",
		"= main.Plain",
		"main.RuntimeFunctionValues$1 (closure)",
		"main.(*Counter).Add$bound (bound method)",
		"LLGO_CASE=container",
		`len=1 = {["answer"] = 42}`,
		`len=1 = {["named"] = 17}`,
		"len=24 = {",
		"len=2 cap=4 = {8, 9}",
		"len=1 cap=2 = {31}",
		`len=1 cap=2 closed = {"remaining"}`,
		"LLGO_CASE=goroutine",
		"main.InspectGoroutineValues",
		"main.RuntimeGoroutineValues",
		"LLGO_THREAD_PRESERVED=True",
		"LLGO_GOROUTINE_REGISTRY=root+2workers",
		"goroutine 1 [running] parent=0",
		"tid=",
		"main.InspectGoroutineValues",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("GDB output missing %q:\n%s", expected, output)
		}
	}

	integrationTestFallback(t, gdbPath)
}

// TestGDBCompleteWorkerUnwind deliberately remains strict on every host. The
// native-values test is a separate capability: stock Darwin and Windows ARM64
// GDB currently cannot unwind blocked system frames. Darwin also reproduces
// this limitation with a plain C program.
// Their CI lanes run TestGDBIntegration and document that boundary instead of
// weakening this test or treating partial worker stacks as a pass.
func TestGDBCompleteWorkerUnwind(t *testing.T) {
	gdbPath, fixtureDir, executable := integrationBuildFixture(t)
	args := append(integrationGDBArgs(executable),
		"-ex", "break main.InspectGoroutineValues",
		"-ex", "run",
		"-ex", "llgo goroutines",
		"-ex", gdbSourceCommand(filepath.Join(fixtureDir, "gdb_goroutines.py")),
	)
	output := integrationRunGDB(t, gdbPath, fixtureDir, args...)
	for _, expected := range []string{
		"LLGO_THREAD_PRESERVED=True", "LLGO_GOROUTINE_STACKS=root+2workers",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("complete worker unwind missing %q:\n%s", expected, output)
		}
	}
}

func integrationBuildFixture(t *testing.T) (gdbPath, fixtureDir, executable string) {
	t.Helper()
	if os.Getenv("LLGO_GDB_INTEGRATION") == "" {
		t.Skip("set LLGO_GDB_INTEGRATION=1 to run the native GDB acceptance test")
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		t.Fatal("GDB does not support native Apple Silicon processes; use native LLDB or GDB with a remote target")
	}

	gdbPath = integrationConfiguredTool(t, "LLGO_GDB", "gdb-multiarch", "gdb")
	llgoPath := integrationConfiguredTool(t, "LLGO", "llgo")
	root := integrationRepoRoot(t)
	fixtureDir = filepath.Join(root, "test", "debug", "runtime")
	executable = filepath.Join(t.TempDir(), integrationExecutable("debug"))

	build := exec.Command(
		llgoPath, "build", "-O0", "-ldflags=-w=false",
		"-o", executable, ".",
	)
	build.Dir = fixtureDir
	build.Env = append(os.Environ(), "LLGO_ROOT="+root)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build GDB fixture: %v\n%s", err, output)
	}

	return gdbPath, fixtureDir, executable
}

func integrationGDBArgs(executable string) []string {
	args := []string{
		"--nx", "--quiet", "--batch", executable,
		"-ex", "set trace-commands on",
		"-ex", "set debuginfod enabled off",
		"-ex", "python _llgo_last_stop = []; gdb.events.stop.connect(lambda event: _llgo_last_stop.__setitem__(slice(None), [event]))",
	}
	if runtime.GOOS != "windows" {
		// GDB's Windows-native build has no startup-with-shell parameter.
		args = append(args, "-ex", "set startup-with-shell off")
	}
	if runtime.GOOS == "linux" {
		// Only Linux uses these Boehm handshake signals. Do not hide other
		// signals, access violations, or an unexpected platform-specific stop.
		args = append(args, "-ex", "handle SIGPWR SIGXCPU nostop noprint pass")
	}
	return args
}

func integrationTestFallback(t *testing.T, gdbPath string) {
	cc, err := quoted.Split(os.Getenv("CC"))
	if err != nil {
		t.Fatalf("parse target CC: %v", err)
	}
	if len(cc) == 0 {
		cc = []string{integrationTool(t, "cc", "clang")}
	}
	compileFixture := func(source, executable string, defines ...string) {
		t.Helper()
		// Clang's MSVC target defaults to CodeView; GDB needs DWARF. Keep
		// target/CRT/linker arguments from CC, including a quoted compiler path.
		args := append(append([]string(nil), cc[1:]...), "-gdwarf-4", "-O0")
		args = append(args, defines...)
		args = append(args, "-o", executable, source)
		if output, err := exec.Command(cc[0], args...).CombinedOutput(); err != nil {
			t.Fatalf("compile fallback fixture: %v\n%s", err, output)
		}
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "fallback.c")
	executable := filepath.Join(dir, integrationExecutable("fallback"))
	code := `
#ifdef _WIN32
#define EXPORTED __attribute__((used, dllexport))
#else
#define EXPORTED __attribute__((used, visibility("hidden")))
#endif
typedef struct { const char *data; __SIZE_TYPE__ len; } string;
string cstring = {"raw", 3};
#ifdef LLGO_MARKER_V2
EXPORTED int __llgo_debugger_marker_v2 = 2;
#endif
#ifdef LLGO_BAD_RECORD
EXPORTED int __llgo_debugger_marker_v1 = 1;
EXPORTED unsigned char __llgo_debugger_abi_v1[16] = {
	0x4c, 0x4c, 0x47, 0x4f, 0x44, 0x42, 0x47, 0,
	1, 2, 2, 1, 0, sizeof(void *), 1, 0
};
#endif
int main(void) { return 0; }
`
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	compileFixture(source, executable)
	output := integrationRunGDB(
		t, gdbPath, dir,
		"--nx", "--quiet", "--batch", executable,
		"-ex", "llgo status",
		"-ex", "p cstring",
		"-ex", "p 1+1",
	)
	for _, expected := range []string{
		"Not an LLGo target; raw GDB debugging remains available.",
		`= {data = 0x`,
		`len = 3}`,
		`= 2`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("non-LLGo fallback output missing %q:\n%s", expected, output)
		}
	}

	compileFixture(source, executable, "-DLLGO_MARKER_V2")
	output = integrationRunGDB(
		t, gdbPath, dir,
		"--nx", "--quiet", "--batch", executable,
		"-ex", "llgo status",
		"-ex", "p 1+1",
	)
	for _, expected := range []string{
		"Unsupported LLGo debugger marker version(s): v2",
		"raw GDB debugging remains available.",
		`= 2`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("unsupported marker output missing %q:\n%s", expected, output)
		}
	}

	compileFixture(source, executable, "-DLLGO_BAD_RECORD")
	output = integrationRunGDB(
		t, gdbPath, dir,
		"--nx", "--quiet", "--batch", executable,
		"-ex", "llgo status",
		"-ex", "p 1+1",
	)
	for _, expected := range []string{
		"Unsupported LLGo debugger ABI",
		"unsupported record/schema/runtime/ABI versions",
		"raw GDB debugging remains available.",
		`= 2`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("unsupported record output missing %q:\n%s", expected, output)
		}
	}
}

func integrationRunGDB(t *testing.T, gdbPath, dir string, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()
	// Stream diagnostics before Run returns so a stuck debugger leaves its
	// last command and stop visible when the enclosing Go test times out.
	if err := Run(gdbPath, nil, args, strings.NewReader(""),
		io.MultiWriter(&stdout, os.Stdout), io.MultiWriter(&stderr, os.Stderr)); err != nil {
		t.Fatalf("run GDB: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String() + stderr.String()
	// Keep actual frames and debugger diagnostics in successful CI logs too;
	// a PASS line alone is insufficient evidence for a new host/architecture.
	t.Logf("GDB output:\n%s", output)
	return output
}

func integrationMarkerLine(t *testing.T, path, marker string) int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		if strings.Contains(scanner.Text(), marker) {
			return line
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("marker %q not found in %s", marker, path)
	return 0
}

func integrationConfiguredTool(t *testing.T, environment string, candidates ...string) string {
	t.Helper()
	if configured := os.Getenv(environment); configured != "" {
		// Never silently test a different debugger when an explicit path is
		// unavailable (for example, an incorrectly installed native GDB).
		return integrationTool(t, configured)
	}
	return integrationTool(t, candidates...)
}

func integrationTool(t *testing.T, candidates ...string) string {
	t.Helper()
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	t.Fatalf("required integration tool not found: %s", strings.Join(candidates, ", "))
	return ""
}

func integrationRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func integrationExecutable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func integrationBreakpoint(source string, line int) string {
	// Explicit locations avoid ambiguity between drive-letter colons and line
	// numbers, and quoted forward-slash paths handle spaces on every host.
	return fmt.Sprintf("break -source %s -line %d", strconv.Quote(filepath.ToSlash(source)), line)
}
