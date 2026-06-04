//go:build !llgo
// +build !llgo

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package cl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/llgo/cl/cltest"
	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/llgen"
)

func testCompile(t *testing.T, src, expected string) {
	t.Helper()
	cltest.TestCompileEx(t, src, "foo.go", expected, false)
}

func requireEmbedTest(t *testing.T) {
	t.Helper()
	if os.Getenv("LLGO_EMBED_TESTS") != "1" {
		t.Skip("Skipping embedded emulator tests; set LLGO_EMBED_TESTS=1 to run")
	}
}

type embedTestSuite struct {
	name   string
	relDir string
}

type embedTargetConfig struct {
	target      string
	ignoreByDir map[string][]string
}

var embedTestSuites = []embedTestSuite{
	{name: "testgo", relDir: "./_testgo"},
	{name: "testlibc", relDir: "./_testlibc"},
	{name: "testrt", relDir: "./_testrt"},
	{name: "testdata", relDir: "./_testdata"},
}

var embedTargetConfigs = []embedTargetConfig{
	{
		target: "esp32c3-basic",
		ignoreByDir: map[string][]string{
			"./_testgo": {
				"./_testgo/abimethod",   // llgo panic: unsatisfied import internal/runtime/sys
				"./_testgo/cgobasic",    // fast fail: build constraints exclude all Go files (cgo)
				"./_testgo/cgocfiles",   // fast fail: build constraints exclude all Go files (cgo)
				"./_testgo/cgodefer",    // fast fail: build constraints exclude all Go files (cgo)
				"./_testgo/cgofull",     // fast fail: build constraints exclude all Go files (cgo)
				"./_testgo/cgomacro",    // fast fail: build constraints exclude all Go files (cgo)
				"./_testgo/cgopython",   // fast fail: build constraints exclude all Go files (cgo)
				"./_testgo/chan",        // timeout: emulator did not auto-exit
				"./_testgo/cursor",      // panic: internal/bytealg: selected .s files require plan9asm translation
				"./_testgo/defer4",      // unexpected output: got "fatal error", expected "recover: panic message"
				"./_testgo/goexit",      // llgo panic: unsatisfied import internal/runtime/sys
				"./_testgo/indexerr",    // unexpected output: len(dst)=12, len(src)=0 (got "fatal error")
				"./_testgo/makeslice",   // unexpected output: len(dst)=23, len(src)=0 (got "fatal error\\nmust error")
				"./_testgo/reflect",     // llgo panic: unsatisfied import internal/runtime/sys
				"./_testgo/reflectconv", // llgo panic: unsatisfied import internal/sync
				"./_testgo/reflectfn",   // llgo panic: unsatisfied import internal/runtime/sys
				"./_testgo/reflectmkfn", // llgo panic: unsatisfied import internal/runtime/sys
				"./_testgo/rewrite",     // llgo panic: unsatisfied import internal/sync
				"./_testgo/select",      // timeout: emulator did not auto-exit
				"./_testgo/selects",     // timeout: emulator did not auto-exit
				"./_testgo/sigsegv",     // unexpected output: got "0/main", expected recover nil-pointer message
				"./_testgo/syncmap",     // llgo panic: unsatisfied import internal/runtime/sys
			},
			"./_testlibc": {
				"./_testlibc/argv",     // timeout: emulator panic (Load access fault), no auto-exit
				"./_testlibc/atomic",   // link error: ld.lld: error: undefined symbol: __atomic_store
				"./_testlibc/complex",  // link error: ld.lld: error: undefined symbol: cabsf
				"./_testlibc/demangle", // link error: ld.lld: error: unknown argument '-Wl,-search_paths_first'
				"./_testlibc/once",     // fast fail: build constraints exclude all Go files (pthread/sync)
				"./_testlibc/setjmp",   // link error: ld.lld: error: undefined symbol: stderr
				"./_testlibc/sqlite",   // link error: ld.lld: error: unable to find library -lsqlite3
			},
			"./_testrt": {
				"./_testrt/asmfull",     // compile/asm error: unrecognized instruction mnemonic
				"./_testrt/fprintf",     // link error: ld.lld: error: undefined symbol: __stderrp
				"./_testrt/hello",       // fast fail: build constraints exclude all Go files
				"./_testrt/linkname",    // unexpected output: line order mismatch ("hello" appears first)
				"./_testrt/makemap",     // link error: ld.lld: error: undefined symbol: __atomic_fetch_or_4
				"./_testrt/strlen",      // fast fail: build constraints exclude all Go files
				"./_testrt/struct",      // fast fail: build constraints exclude all Go files
				"./_testrt/tpfunc",      // unexpected output: type size mismatch (got 8 4 4, expected 16 8 8)
				"./_testrt/typalias",    // fast fail: build constraints exclude all Go files
				"./_testrt/unreachable", // timeout: emulator panic (Instruction access fault), no auto-exit
			},
			"./_testdata": {
				"./_testdata/debug", // llgo panic: unsatisfied import internal/runtime/sys
			},
		},
	},
	{
		target: "esp32",
		ignoreByDir: map[string][]string{
			"./_testgo": {
				"./_testgo/abimethod", // panic: internal/bytealg selected .s files require plan9asm translation
				"./_testgo/alias",     // unexpected output
				"./_testgo/cgodefer",  // panic: cannot build SSA for packages
				"./_testgo/cgopython", // panic: cannot build SSA for packages
				"./_testgo/cursor",    // panic: internal/bytealg: selected .s files require plan9asm translation
				"./_testgo/defer4",    // runtime output: fatal error
				"./_testgo/indexerr",  // runtime output: fatal error
				"./_testgo/invoke",    // unexpected output
				"./_testgo/makeslice", // runtime output: fatal error
				"./_testgo/multiret",  // unexpected output
				"./_testgo/select",    // timeout: emulator did not auto-exit
				"./_testgo/sigsegv",   // unexpected output
				"./_testgo/struczero", // timeout: emulator did not auto-exit
			},
			"./_testlibc": {
				"./_testlibc/atomic",   // unexpected output
				"./_testlibc/demangle", // link error: ld.lld unknown argument -Wl,-search_paths_first
				"./_testlibc/once",     // panic: cannot build SSA for packages
				"./_testlibc/setjmp",   // link error: ld.lld undefined symbol stderr
				"./_testlibc/sqlite",   // link error: ld.lld unable to find library -lsqlite3
			},
			"./_testrt": {
				"./_testrt/asmfull",  // unexpected output
				"./_testrt/cast",     // timeout: emulator did not auto-exit
				"./_testrt/complex",  // unexpected output
				"./_testrt/fprintf",  // link error: ld.lld undefined symbol __stderrp
				"./_testrt/hello",    // panic: cannot build SSA for packages
				"./_testrt/linkname", // unexpected output
				"./_testrt/strlen",   // panic: runtime index out of range
				"./_testrt/struct",   // panic: runtime index out of range
				"./_testrt/tpfunc",   // unexpected output
				"./_testrt/typalias", // panic: runtime index out of range
			},
			"./_testdata": {
				"./_testdata/cpkgimp", // unexpected output
			},
		},
	},
}

func runEmbedTargetSuite(t *testing.T, target, relDir string, ignore []string) {
	t.Helper()
	conf := build.NewDefaultConf(build.ModeRun)
	conf.Target = target
	conf.Emulator = true
	cltest.RunAndTestFromDir(t, "", relDir, ignore,
		cltest.WithRunConfig(conf),
		cltest.WithOutputFilter(cltest.FilterEmulatorOutput),
		cltest.WithIRCheck(false),
	)
}

func TestRunAndTestFromTestgo(t *testing.T) {
	cltest.RunAndTestFromDir(t, "", "./_testgo", nil)
}

func TestFilterEmulatorOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "ESP32C3 output",
			input: `Adding SPI flash device
ESP-ROM:esp32c3-api1-20210207
Build:Feb  7 2021
rst:0x1 (POWERON),boot:0x8 (SPI_FAST_FLASH_BOOT)
SPIWP:0xee
mode:DIO, clock div:1
load:0x3fc855b0,len:0xfc
load:0x3fc856ac,len:0x4
load:0x3fc856b0,len:0x44
load:0x40380000,len:0x1548
load:0x40381548,len:0x68
entry 0x40380000
Hello World!
`,
			expected: `Hello World!
`,
		},
		{
			name: "ESP32 output",
			input: `Adding SPI flash device
ESP-ROM:esp32-xxxx
entry 0x40080000
Hello World!
`,
			expected: `Hello World!
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cltest.FilterEmulatorOutput(tt.input)
			if got != tt.expected {
				t.Fatalf("filterEmulatorOutput() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRunEmbedEmulator(t *testing.T) {
	requireEmbedTest(t)
	for _, targetConf := range embedTargetConfigs {
		targetConf := targetConf
		t.Run(targetConf.target, func(t *testing.T) {
			for _, suite := range embedTestSuites {
				suite := suite
				t.Run(suite.name, func(t *testing.T) {
					runEmbedTargetSuite(t, targetConf.target, suite.relDir, targetConf.ignoreByDir[suite.relDir])
				})
			}
		})
	}
}

func TestRunFromTestgoSelectAllowsKnownInterleavings(t *testing.T) {
	output, err := cltest.RunAndCapture("./_testgo/select", "")
	if err != nil {
		t.Fatalf("run failed: %v\noutput: %s", err, string(output))
	}
	lines := selectOutputLines(string(output))
	if len(lines) != 3 {
		t.Fatalf("unexpected select output lines %q from:\n%s", lines, output)
	}
	if lines[0] != "100" && lines[0] != "200" {
		t.Fatalf("unexpected select send output %q from:\n%s", lines[0], output)
	}
	for _, line := range lines[1:] {
		switch line {
		case "ch1", "ch2", "exit":
		default:
			t.Fatalf("unexpected select recv output %q from:\n%s", line, output)
		}
	}
}

func selectOutputLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "100", "200", "ch1", "ch2", "exit":
			lines = append(lines, line)
		}
	}
	return lines
}

func TestRunAndTestFromTestpy(t *testing.T) {
	cltest.RunAndTestFromDir(t, "", "./_testpy", nil)
}

func TestRunAndTestFromTestlibgo(t *testing.T) {
	cltest.RunAndTestFromDir(t, "", "./_testlibgo", nil)
}

func TestRunAndTestFromTestlibc(t *testing.T) {
	var ignore []string
	if runtime.GOOS == "linux" {
		ignore = []string{
			"./_testlibc/demangle", // Linux demangle symbol differs (itaniumDemangle linkage mismatch).
		}
	}
	cltest.RunAndTestFromDir(t, "", "./_testlibc", ignore)
}

func TestRunAndTestFromTestrt(t *testing.T) {
	var ignore []string
	if runtime.GOOS == "linux" {
		ignore = []string{
			"./_testrt/asmfull", // Output is macOS-specific.
			"./_testrt/fprintf", // Linux uses different stderr symbol (no __stderrp).
		}
	}
	cltest.RunAndTestFromDir(t, "", "./_testrt", ignore)
}

func TestRunAndTestFromTestdata(t *testing.T) {
	cltest.RunAndTestFromDir(t, "", "./_testdata", nil)
}

func TestCgofullGeneratesC2func(t *testing.T) {
	ir := llgen.GenFrom("./_testgo/cgofull")
	if !strings.Contains(ir, "_C2func_test_structs") {
		t.Fatal("missing _C2func_test_structs in cgofull IR")
	}
	if !strings.Contains(ir, "cliteErrno") {
		t.Fatal("missing cliteErrno call in cgofull IR")
	}
}

func TestGoPkgMath(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeInstall)
	_, err := build.Do([]string{"math"}, conf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildGenericMethodTableSymbols(t *testing.T) {
	buildTempModule(t, map[string]string{
		"go.mod": `module repro.methods

go 1.24
`,
		"lib/lib.go": `package lib

type Typed[T comparable] struct{}

func (q *Typed[T]) Add(item T) {}
func (q *Typed[T]) Len() int { return 0 }

type Type = Typed[any]

func NewQueue() *Type { return &Typed[any]{} }

type inner[T any] struct{}

func (i *inner[T]) M() {}
func (i *inner[T]) N() int { return 1 }

type Outer struct{ *inner[int] }

func NewOuter() *Outer { return &Outer{inner: &inner[int]{}} }
`,
		"app/main.go": `package main

import "repro.methods/lib"

type Queue interface {
	Add(any)
	Len() int
}

type Promoted interface {
	M()
	N() int
}

func main() {
	var q Queue = lib.NewQueue()
	var p Promoted = lib.NewOuter()
	_, _ = q, p
}
`,
	})
}

func TestBuildReflectPrivateLinknames(t *testing.T) {
	buildTempModule(t, map[string]string{
		"go.mod": `module repro.reflectlink

go 1.24
`,
		"app/main.go": `package main

import (
	"unsafe"
	_ "unsafe"
)

//go:linkname unsafeNew reflect.unsafe_New
func unsafeNew(unsafe.Pointer) unsafe.Pointer

//go:linkname typedmemmove reflect.typedmemmove
func typedmemmove(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer)

//go:linkname unsafeNewArray reflect.unsafe_NewArray
func unsafeNewArray(unsafe.Pointer, int) unsafe.Pointer

//go:linkname makemap reflect.makemap
func makemap(unsafe.Pointer, int) unsafe.Pointer

//go:linkname mapaccess reflect.mapaccess
func mapaccess(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) unsafe.Pointer

//go:linkname mapiterinit reflect.mapiterinit
func mapiterinit(unsafe.Pointer, unsafe.Pointer, unsafe.Pointer)

//go:linkname mapiternext reflect.mapiternext
func mapiternext(unsafe.Pointer)

//go:linkname ifaceE2I reflect.ifaceE2I
func ifaceE2I(unsafe.Pointer, any, unsafe.Pointer)

func main() {
	_ = unsafeNew(nil)
	typedmemmove(nil, nil, nil)
	_ = unsafeNewArray(nil, 0)
	_ = makemap(nil, 0)
	_ = mapaccess(nil, nil, nil)
	mapiterinit(nil, nil, nil)
	mapiternext(nil)
	ifaceE2I(nil, nil, nil)
}
`,
	})
}

func TestBuildReflectValueGo126Symbols(t *testing.T) {
	buildTempModule(t, map[string]string{
		"go.mod": `module repro.reflectvalue

go 1.24
`,
		"app/main.go": `package main

import (
	"iter"
	"reflect"
	"runtime"
	"unsafe"
	_ "unsafe"
)

type addressableValue struct {
	reflect.Value
	forcedAddr bool
}

//go:linkname valueAbiType reflect.Value.abiType
func valueAbiType(reflect.Value) unsafe.Pointer

//go:linkname valueAbiTypeSlow reflect.Value.abiTypeSlow
func valueAbiTypeSlow(reflect.Value) unsafe.Pointer

var _ interface {
	Fields() iter.Seq2[reflect.StructField, reflect.Value]
	Methods() iter.Seq2[reflect.Method, reflect.Value]
} = addressableValue{}

func main() {
	v := reflect.ValueOf(struct{ A int }{A: 1})
	_, _ = valueAbiType(v), valueAbiTypeSlow(v)
	_ = reflect.TypeOf(addressableValue{Value: v})
	if f := runtime.FuncForPC(0); f != nil {
		_, _ = f.FileLine(0)
	}
}
`,
	})
}

func TestBuildGenericFunctionInstanceLinkOnce(t *testing.T) {
	buildTempModule(t, map[string]string{
		"go.mod": `module repro.genericfn

go 1.24
`,
		"lib/lib.go": `package lib

func Identity[T any](v T) T { return v }
`,
		"a/a.go": `package a

import "repro.genericfn/lib"

func A() string { return lib.Identity("a") }
`,
		"b/b.go": `package b

import "repro.genericfn/lib"

func B() string { return lib.Identity("b") }
`,
		"app/main.go": `package main

import (
	"repro.genericfn/a"
	"repro.genericfn/b"
)

func main() {
	_, _ = a.A(), b.B()
}
`,
	})
}

func TestBuildGenericFunctionClosureLinkOnce(t *testing.T) {
	buildTempModule(t, map[string]string{
		"go.mod": `module repro.genericclosure

go 1.24
`,
		"lib/lib.go": `package lib

func WithClosure[T any](v T) T {
	fn := func() T {
		return v
	}
	return fn()
}
`,
		"a/a.go": `package a

import "repro.genericclosure/lib"

func A() string { return lib.WithClosure("a") }
`,
		"b/b.go": `package b

import "repro.genericclosure/lib"

func B() string { return lib.WithClosure("b") }
`,
		"app/main.go": `package main

import (
	"repro.genericclosure/a"
	"repro.genericclosure/b"
)

func main() {
	_, _ = a.A(), b.B()
}
`,
	})
}

func TestBuildXSysUnixRawSyscallNoError(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("RawSyscallNoError repro uses linux/amd64 x/sys/unix asm")
	}
	buildTempModulePatterns(t, build.ModeInstall, []string{"golang.org/x/sys/unix"}, map[string]string{
		"go.mod": `module repro.xsys

go 1.24

require golang.org/x/sys v0.0.0

replace golang.org/x/sys => ./xsys
`,
		"xsys/go.mod": `module golang.org/x/sys

go 1.24
`,
		"xsys/unix/syscall_linux_gc.go": `//go:build linux && gc

package unix

func RawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr)
func SyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr)

func Gettid() int {
	r1, _ := RawSyscallNoError(186, 0, 0, 0)
	return int(r1)
}

func Sync() {
	SyscallNoError(162, 0, 0, 0)
}
`,
		"xsys/unix/asm_linux_amd64.s": `//go:build linux && gc && amd64

#include "textflag.h"

TEXT ·SyscallNoError(SB),NOSPLIT,$0-48
	RET

TEXT ·RawSyscallNoError(SB),NOSPLIT,$0-48
	RET
`,
	})
}

func buildTempModule(t *testing.T, files map[string]string) {
	t.Helper()
	buildTempModulePatterns(t, build.ModeBuild, []string{"./app"}, files)
}

func buildTempModulePatterns(t *testing.T, mode build.Mode, patterns []string, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Dir(oldWD)
	t.Setenv("LLGO_ROOT", repoRoot)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	out := filepath.Join(t.TempDir(), "app")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	conf := build.NewDefaultConf(mode)
	if mode == build.ModeBuild {
		conf.OutFile = out
	}
	conf.ForceRebuild = true
	if _, err := build.Do(patterns, conf); err != nil {
		t.Fatal(err)
	}
}

func TestVar(t *testing.T) {
	testCompile(t, `package foo

var a int
`, `; ModuleID = 'foo'
source_filename = "foo"

@foo.a = global i64 0, align 8
@"foo.init$guard" = global i1 false, align 1

; Function Attrs: null_pointer_is_valid
define void @foo.init() #0 {
_llgo_0:
  %0 = load i1, ptr @"foo.init$guard", align 1
  br i1 %0, label %_llgo_2, label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_0
  store i1 true, ptr @"foo.init$guard", align 1
  br label %_llgo_2

_llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
  ret void
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestBasicFunc(t *testing.T) {
	testCompile(t, `package foo

func fn(a int, b float64) int {
	return 1
}
`, `; ModuleID = 'foo'
source_filename = "foo"

@"foo.init$guard" = global i1 false, align 1

; Function Attrs: null_pointer_is_valid
define i64 @foo.fn(i64 %0, double %1) #0 {
_llgo_0:
  ret i64 1
}

; Function Attrs: null_pointer_is_valid
define void @foo.init() #0 {
_llgo_0:
  %0 = load i1, ptr @"foo.init$guard", align 1
  br i1 %0, label %_llgo_2, label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_0
  store i1 true, ptr @"foo.init$guard", align 1
  br label %_llgo_2

_llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
  ret void
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestIntrinsicBoolToUint8(t *testing.T) {
	testCompile(t, `package foo

import _ "unsafe"

//go:linkname boolToUint8 llgo.boolToUint8
func boolToUint8(b bool) uint8

func use(b bool) uint8 {
	return boolToUint8(b)
}
`, `; ModuleID = 'foo'
source_filename = "foo"

@"foo.init$guard" = global i1 false, align 1

; Function Attrs: null_pointer_is_valid
define void @foo.init() #0 {
_llgo_0:
  %0 = load i1, ptr @"foo.init$guard", align 1
  br i1 %0, label %_llgo_2, label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_0
  store i1 true, ptr @"foo.init$guard", align 1
  br label %_llgo_2

_llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
  ret void
}

; Function Attrs: null_pointer_is_valid
define i8 @foo.use(i1 %0) #0 {
_llgo_0:
  %1 = select i1 %0, i8 1, i8 0
  ret i8 %1
}

attributes #0 = { null_pointer_is_valid }
`)
}
