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

package build

import (
	"bytes"
	stdcontext "context"
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/pclnmap"
)

// The fixture obtains both a function-value entry PC and a target mid-function
// PC. Together those lookups and the stack APIs exercise the full metadata
// contract, including its deliberate absence in pclntab=none.
const pclntabModesFixture = `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

//go:noinline
func pclnTarget() uintptr {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		return 0
	}
	// Keep the target body comfortably larger than the entry-anchor slack.
	// This makes the returned mid-function PC unambiguously belong to the
	// target on both fixed-width arm64 and byte-aligned amd64.
	value := pc
	for i := uintptr(0); i < 64; i++ {
		value = value*33 + i
	}
	pclnTargetSink = value
	return pc
}

var pclnTargetSink uintptr

func pclnState() string {
	pc := reflect.ValueOf(pclnTarget).Pointer()
	return pclnStateForPC(pc)
}

func pclnStateForPC(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "MISS"
	}
	file, line := fn.FileLine(pc)
	if strings.HasSuffix(fn.Name(), ".pclnTarget") && filepath.Base(file) == "main.go" && line > 0 {
		return "FULL"
	}
	return "MISS"
}

func pclnClosurePCState(fn *runtime.Func, pc uintptr) string {
	if fn == nil || !strings.HasSuffix(fn.Name(), ".pclnTarget") {
		return "MISS"
	}
	file, line := fn.FileLine(pc)
	if filepath.Base(file) == "main.go" && line > 0 {
		return "FULL"
	}
	return "MISS"
}

func pclnClosureState() string {
	funcvalPC := reflect.ValueOf(pclnTarget).Pointer()
	targetPC := pclnTarget()
	funcvalFn := runtime.FuncForPC(funcvalPC)
	targetFn := runtime.FuncForPC(targetPC)
	sameEntry := funcvalFn != nil && targetFn != nil &&
		funcvalFn.Name() == targetFn.Name() &&
		funcvalFn.Entry() == funcvalPC && targetFn.Entry() == funcvalFn.Entry() &&
		targetFn.Entry() <= targetPC
	return fmt.Sprintf("target=%s funcval=%s same-entry=%t",
		pclnClosurePCState(targetFn, targetPC), pclnClosurePCState(funcvalFn, funcvalPC), sameEntry)
}

//go:noinline
func pclnCallerState() string {
	_, file, line, ok := runtime.Caller(0)
	if ok && filepath.Base(file) == "main.go" && line > 0 {
		return "FULL"
	}
	return "MISS"
}

//go:noinline
func pclnFramesState() string {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if strings.HasSuffix(frame.Function, ".pclnFramesState") && filepath.Base(frame.File) == "main.go" && frame.Line > 0 {
			return "FULL"
		}
		if !more {
			return "MISS"
		}
	}
}

//go:noinline
func pclnPanic() {
	panic("pcln panic")
}

func recoverHardwareFault() (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	// Use an invalid non-nil address so this exercises the hardware-fault
	// signal path rather than a compiler-inserted nil check. Returning a
	// predicate over the value keeps the load observable under optimization.
	return *(*byte)(unsafe.Pointer(uintptr(1))) == 0
}

func pclnWorker(start <-chan struct{}, results chan<- string, pc uintptr) {
	<-start
	results <- pclnStateForPC(pc)
}

func pclnControlWorker(start <-chan struct{}, results chan<- string) {
	<-start
	results <- "FULL"
}

func concurrentPCLN(control bool) string {
	const workers = 32
	start := make(chan struct{})
	results := make(chan string, workers)
	pc := reflect.ValueOf(pclnTarget).Pointer()
	for i := 0; i < workers; i++ {
		if control {
			go pclnControlWorker(start, results)
		} else {
			go pclnWorker(start, results, pc)
		}
	}
	close(start)
	full := 0
	for i := 0; i < workers; i++ {
		if <-results == "FULL" {
			full++
		}
	}
	return fmt.Sprintf("%d/%d", full, workers)
}

func repairPCLN(source, target string) {
	_ = os.Remove(target)
	if err := os.Rename(source, target); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		panic("missing scenario")
	}
	switch os.Args[1] {
	case "once":
		fmt.Println(pclnState())
	case "chdir-once":
		if err := os.Chdir(os.Args[2]); err != nil {
			panic(err)
		}
		fmt.Println(pclnState())
	case "apis":
		fmt.Printf("caller=%s frames=%s func=%s\n", pclnCallerState(), pclnFramesState(), pclnState())
	case "closure":
		fmt.Println(pclnClosureState())
	case "panic":
		pclnPanic()
	case "success-cache":
		fmt.Println(pclnState())
		if err := os.Remove(os.Args[2]); err != nil {
			panic(err)
		}
		fmt.Println(pclnState())
	case "failure-cache":
		fmt.Println(pclnState())
		repairPCLN(os.Args[2], os.Args[3])
		fmt.Println(pclnState())
	case "fault-no-load":
		fmt.Println(recoverHardwareFault())
		repairPCLN(os.Args[2], os.Args[3])
		fmt.Println(pclnState())
	case "concurrent-control":
		fmt.Println(concurrentPCLN(true))
	case "concurrent-loaded":
		fmt.Println(pclnState())
		fmt.Println(concurrentPCLN(false))
	case "concurrent":
		fmt.Println(concurrentPCLN(false))
	default:
		panic("unknown scenario")
	}
}
`

const pclntabPureCLibraryFixture = `package main

import "github.com/goplus/lib/c"

func main() {
	c.Printf(c.Str("pure-c-pclntab\n"))
}
`

func TestPCLNModeNativeIntegration(t *testing.T) {
	requireNativePCLNSidecars(t)
	setPCLNIntegrationEnv(t)
	source := writePCLNIntegrationFixture(t)

	packagingDir := t.TempDir()
	binaries := make(map[PCLNMode]string)
	t.Run("packaging", func(t *testing.T) {
		tests := []struct {
			name        string
			mode        PCLNMode
			wantRuntime string
			wantAPIs    string
			wantSidecar bool
		}{
			{name: "embedded", mode: PCLNEmbedded, wantRuntime: "FULL\n", wantAPIs: "caller=FULL frames=FULL func=FULL\n"},
			{name: "external", mode: PCLNExternal, wantRuntime: "FULL\n", wantAPIs: "caller=FULL frames=FULL func=FULL\n", wantSidecar: true},
			{name: "none", mode: PCLNNone, wantRuntime: "MISS\n", wantAPIs: "caller=MISS frames=MISS func=MISS\n"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bin := buildPCLNIntegrationBinaryAt(t, source, tt.mode, LinkOptions{}, filepath.Join(packagingDir, "pclntab-modes-"+tt.name), false)
				binaries[tt.mode] = bin
				if tt.mode == PCLNExternal {
					verifyPCLNIntegrationSignature(t, bin)
				}
				sidecar := bin + ".pclntab"
				_, err := os.Stat(sidecar)
				if tt.wantSidecar && err != nil {
					t.Fatalf("external sidecar %s: %v", sidecar, err)
				}
				if !tt.wantSidecar && !os.IsNotExist(err) {
					t.Fatalf("unexpected sidecar %s: stat error %v", sidecar, err)
				}
				if got := runPCLNIntegrationBinary(t, bin, "once"); got != tt.wantRuntime {
					t.Fatalf("runtime metadata state = %q, want %q", got, tt.wantRuntime)
				}
				if got := runPCLNIntegrationBinary(t, bin, "apis"); got != tt.wantAPIs {
					t.Fatalf("runtime API states = %q, want %q", got, tt.wantAPIs)
				}
				panicOut := runPCLNIntegrationBinaryFailure(t, bin, "panic")
				if !strings.Contains(panicOut, "pcln panic") {
					t.Fatalf("panic output does not contain panic value:\n%s", panicOut)
				}
				// Darwin's native fallback currently retains the function name.
				// Linux's pre-existing clite traceback can degrade to raw PCs when
				// the physical FP snapshot is unavailable; PCLN API fidelity is
				// asserted independently above for both platforms.
				if runtime.GOOS == "darwin" && tt.mode != PCLNNone && !strings.Contains(panicOut, ".pclnPanic") {
					t.Fatalf("panic output does not contain pclnPanic frame:\n%s", panicOut)
				}
			})
		}
	})

	// Build the relatively expensive external artifact once. Each scenario
	// gets a byte-for-byte clone, which preserves the embedded identity while
	// allowing destructive sidecar tests to remain isolated.
	base := binaries[PCLNExternal]
	if base == "" {
		// A focused -run selection may skip the packaging subtest.
		base = buildPCLNIntegrationBinary(t, source, PCLNExternal, LinkOptions{})
	}

	t.Run("normal-sidecar", func(t *testing.T) {
		bin, _ := clonePCLNIntegrationArtifact(t, base)
		if got := runPCLNIntegrationBinary(t, bin, "once"); got != "FULL\n" {
			t.Fatalf("runtime metadata state = %q, want FULL", got)
		}
		if got := runPCLNIntegrationBinary(t, bin, "closure"); got != "target=FULL funcval=FULL same-entry=true\n" {
			t.Fatalf("closure target/funcval metadata = %q", got)
		}
	})

	t.Run("relative-launch-then-chdir", func(t *testing.T) {
		bin, _ := clonePCLNIntegrationArtifact(t, base)
		launchDir := filepath.Dir(bin)
		if got := runPCLNIntegrationBinaryInDir(t, launchDir, "./"+filepath.Base(bin), "chdir-once", t.TempDir()); got != "FULL\n" {
			t.Fatalf("runtime metadata after relative launch and chdir = %q, want FULL", got)
		}
	})

	t.Run("successful-load-is-cached", func(t *testing.T) {
		bin, sidecar := clonePCLNIntegrationArtifact(t, base)
		if got := runPCLNIntegrationBinary(t, bin, "success-cache", sidecar); got != "FULL\nFULL\n" {
			t.Fatalf("successful cache result = %q, want two FULL lookups", got)
		}
	})

	t.Run("fault-signal-does-not-load", func(t *testing.T) {
		bin, sidecar := clonePCLNIntegrationArtifact(t, base)
		valid := sidecar + ".valid"
		copyPCLNIntegrationFile(t, sidecar, valid)
		removePCLNIntegrationSidecar(t, sidecar)
		if got := runPCLNIntegrationBinary(t, bin, "fault-no-load", valid, sidecar); got != "true\nFULL\n" {
			t.Fatalf("fault-path loader state = %q, want recovered fault followed by FULL", got)
		}
	})

	t.Run("failed-load-is-cached", func(t *testing.T) {
		failures := []struct {
			name   string
			mutate func(*testing.T, string)
		}{
			{name: "missing", mutate: removePCLNIntegrationSidecar},
			{name: "truncated", mutate: truncatePCLNIntegrationSidecar},
			{name: "permission-denied", mutate: denyPCLNIntegrationPermission},
			{name: "corrupt-payload", mutate: corruptPCLNIntegrationPayload},
			{name: "identity-mismatch", mutate: mismatchPCLNIntegrationIdentity},
			{name: "wrong-version", mutate: mismatchPCLNIntegrationVersion},
			{name: "wrong-ABI", mutate: mismatchPCLNIntegrationABI},
			{name: "wrong-architecture", mutate: mismatchPCLNIntegrationArchitecture},
			{name: "overlapping-sections", mutate: overlapPCLNIntegrationSections},
			{name: "misaligned-pc-site-section", mutate: misalignPCLNIntegrationPCSiteSection},
			{name: "unterminated-string-pool", mutate: unterminatePCLNIntegrationStringPool},
		}
		for _, failure := range failures {
			t.Run(failure.name, func(t *testing.T) {
				bin, sidecar := clonePCLNIntegrationArtifact(t, base)
				valid := sidecar + ".valid"
				copyPCLNIntegrationFile(t, sidecar, valid)
				failure.mutate(t, sidecar)
				got := runPCLNIntegrationBinary(t, bin, "failure-cache", valid, sidecar)
				if got != "MISS\nMISS\n" {
					t.Fatalf("failed cache result = %q, want two MISS lookups", got)
				}
			})
		}
	})

	t.Run("concurrent-first-load", func(t *testing.T) {
		bin, _ := clonePCLNIntegrationArtifact(t, base)
		if got := runPCLNIntegrationBinary(t, bin, "concurrent-control"); got != "32/32\n" {
			t.Fatalf("concurrent scheduler control = %q, want all workers", got)
		}
		bin, _ = clonePCLNIntegrationArtifact(t, base)
		if got := runPCLNIntegrationBinary(t, bin, "concurrent-loaded"); got != "FULL\n32/32\n" {
			t.Fatalf("concurrent loaded lookups = %q, want all lookups to succeed", got)
		}
		bin, _ = clonePCLNIntegrationArtifact(t, base)
		if got := runPCLNIntegrationBinary(t, bin, "concurrent"); got != "32/32\n" {
			t.Fatalf("concurrent first load = %q, want all lookups to succeed", got)
		}
	})
}

func TestPCLNExternalPureCLibraryIdentityRetentionIntegration(t *testing.T) {
	requireNativePCLNSidecars(t)
	setPCLNIntegrationEnv(t)
	source := writePCLNIntegrationSource(t, pclntabPureCLibraryFixture)
	// Exercise the external-sidecar summary as part of a real detach rather
	// than mocking the final publication path.
	bin := buildPCLNIntegrationBinaryAt(
		t, source, PCLNExternal, LinkOptions{}, filepath.Join(t.TempDir(), "pclntab-pure-c"), true,
	)
	verifyPCLNIntegrationSignature(t, bin)
	if got := runPCLNIntegrationBinary(t, bin); got != "pure-c-pclntab\n" {
		t.Fatalf("pure lib/c output = %q", got)
	}
	if pclnIntegrationHasLLGoRuntime(t, bin) {
		t.Fatal("pure lib/c fixture unexpectedly linked the LLGo runtime")
	}
	raw, err := os.ReadFile(bin + ".pclntab")
	if err != nil {
		t.Fatal(err)
	}
	view, err := pclnmap.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := pclnIntegrationBinaryIdentity(t, bin); got != view.Identity {
		t.Fatalf("binary identity = %x, sidecar identity = %x", got, view.Identity)
	}
}

func TestPCLNExternalLinkOptionsIntegration(t *testing.T) {
	requireNativePCLNSidecars(t)
	setPCLNIntegrationEnv(t)
	source := writePCLNIntegrationFixture(t)

	// PR #2113 phase one consumes -s as typed intent and applies its Go
	// semantics (-s implies -w), but deliberately does not remove the native
	// symbol table yet. Keeping wantSymtab explicit makes the single expected
	// change obvious when native symbol stripping is implemented.
	tests := []struct {
		name       string
		options    LinkOptions
		wantDebug  bool
		wantSymtab bool
	}{
		{name: "default", wantDebug: true, wantSymtab: true},
		{name: "w", options: LinkOptions{DWARF: DWARFOmit}, wantSymtab: true},
		{name: "s-implies-w", options: LinkOptions{OmitSymbolTable: true}, wantSymtab: true},
		{name: "s-with-w-false", options: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}, wantDebug: true, wantSymtab: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := buildPCLNIntegrationBinary(t, source, PCLNExternal, tt.options)
			verifyPCLNIntegrationSignature(t, bin)
			if got := runPCLNIntegrationBinary(t, bin, "once"); got != "FULL\n" {
				t.Fatalf("external runtime metadata with LinkOptions %+v = %q, want FULL", tt.options, got)
			}
			if got := pclnIntegrationHasDebugInfo(t, bin); got != tt.wantDebug {
				t.Fatalf("debug metadata with LinkOptions %+v = %v, want %v", tt.options, got, tt.wantDebug)
			}
			if got := pclnIntegrationHasSymbolTable(t, bin); got != tt.wantSymtab {
				t.Fatalf("native symbol table with LinkOptions %+v = %v, want %v", tt.options, got, tt.wantSymtab)
			}
		})
	}
}

func requireNativePCLNSidecars(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("external PCLN sidecars initially support native Mach-O and ELF")
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("external PCLN sidecars initially support 64-bit amd64 and arm64")
	}
}

func verifyPCLNIntegrationSignature(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return
	}
	out, err := exec.Command("codesign", "--verify", "--verbose=4", path).CombinedOutput()
	if err != nil {
		t.Fatalf("verify final external-PCLN signature for %s: %v\n%s", path, err, out)
	}
}

func setPCLNIntegrationEnv(t *testing.T) {
	t.Helper()
	// Explicit typed PCLNMode remains authoritative over the legacy funcinfo
	// environment switches.
	t.Setenv(llgoFuncInfo, "1")
	t.Setenv(llgoFuncInfoSites, "1")
}

func writePCLNIntegrationFixture(t *testing.T) string {
	t.Helper()
	return writePCLNIntegrationSource(t, pclntabModesFixture)
}

func writePCLNIntegrationSource(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	// go/packages' file= query keeps the fixture self-contained without
	// adding another permanent testdata package to the repository.
	return "file=" + path
}

func pclnIntegrationHasLLGoRuntime(t *testing.T, path string) bool {
	t.Helper()
	const runtimeSymbol = "github.com/xgo-dev/llgo/runtime/internal/runtime."
	switch runtime.GOOS {
	case "linux":
		f, err := elf.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		syms, err := f.Symbols()
		if err != nil {
			t.Fatal(err)
		}
		for _, sym := range syms {
			if strings.Contains(sym.Name, runtimeSymbol) {
				return true
			}
		}
	case "darwin":
		f, err := macho.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if f.Symtab == nil {
			t.Fatal("Mach-O has no symbol table")
		}
		for _, sym := range f.Symtab.Syms {
			if strings.Contains(sym.Name, runtimeSymbol) {
				return true
			}
		}
	default:
		panic(fmt.Sprintf("unexpected GOOS %q", runtime.GOOS))
	}
	return false
}

func pclnIntegrationBinaryIdentity(t *testing.T, path string) [32]byte {
	t.Helper()
	var raw []byte
	var err error
	switch runtime.GOOS {
	case "linux":
		f, openErr := elf.Open(path)
		if openErr != nil {
			t.Fatal(openErr)
		}
		defer f.Close()
		section := f.Section("llgo_pclntab_id")
		if section == nil {
			t.Fatal("ELF pclntab identity section is missing")
		}
		raw, err = section.Data()
	case "darwin":
		f, openErr := macho.Open(path)
		if openErr != nil {
			t.Fatal(openErr)
		}
		defer f.Close()
		for _, section := range f.Sections {
			if section.Seg == "__DATA" && section.Name == "__llgo_pid" {
				raw, err = section.Data()
				break
			}
		}
	default:
		panic(fmt.Sprintf("unexpected GOOS %q", runtime.GOOS))
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 32 {
		t.Fatalf("pclntab identity section size = %d, want 32", len(raw))
	}
	var identity [32]byte
	copy(identity[:], raw)
	return identity
}

func buildPCLNIntegrationBinary(t *testing.T, source string, mode PCLNMode, options LinkOptions) string {
	t.Helper()
	return buildPCLNIntegrationBinaryAt(t, source, mode, options, filepath.Join(t.TempDir(), "pclntab-modes"), false)
}

func buildPCLNIntegrationBinaryAt(t *testing.T, source string, mode PCLNMode, options LinkOptions, bin string, verbose bool) string {
	t.Helper()
	started := time.Now()
	conf := &Config{
		Mode:        ModeBuild,
		BuildMode:   BuildModeExe,
		OutFile:     bin,
		PCLNMode:    mode,
		PCLNModeSet: true,
		LinkOptions: options,
		Verbose:     verbose,
	}
	if _, err := Do([]string{source}, conf); err != nil {
		t.Fatalf("build %s PCLN fixture with LinkOptions %+v: %v", mode, options, err)
	}
	t.Logf("built %s PCLN fixture with LinkOptions %+v in %s", mode, options, time.Since(started).Round(time.Millisecond))
	return bin
}

func runPCLNIntegrationBinary(t *testing.T, bin string, args ...string) string {
	t.Helper()
	return runPCLNIntegrationBinaryInDir(t, "", bin, args...)
}

func runPCLNIntegrationBinaryInDir(t *testing.T, dir, bin string, args ...string) string {
	t.Helper()
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("run %s %q: %v\n%s", bin, args, ctx.Err(), out)
	}
	if err != nil {
		t.Fatalf("run %s %q: %v\n%s", bin, args, err, out)
	}
	return string(out)
}

func runPCLNIntegrationBinaryFailure(t *testing.T, bin string, args ...string) string {
	t.Helper()
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("run %s %q: %v\n%s", bin, args, ctx.Err(), out)
	}
	if err == nil {
		t.Fatalf("run %s %q unexpectedly succeeded:\n%s", bin, args, out)
	}
	return string(out)
}

func clonePCLNIntegrationArtifact(t *testing.T, base string) (bin, sidecar string) {
	t.Helper()
	bin = filepath.Join(t.TempDir(), "pclntab-modes")
	sidecar = bin + ".pclntab"
	copyPCLNIntegrationFile(t, base, bin)
	copyPCLNIntegrationFile(t, base+".pclntab", sidecar)
	return bin, sidecar
}

func copyPCLNIntegrationFile(t *testing.T, source, target string) {
	t.Helper()
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, raw, info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
}

func removePCLNIntegrationSidecar(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func truncatePCLNIntegrationSidecar(t *testing.T, path string) {
	t.Helper()
	if err := os.Truncate(path, int64(pclnmap.HeaderSize-1)); err != nil {
		t.Fatal(err)
	}
}

func denyPCLNIntegrationPermission(t *testing.T, path string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root can open a mode-000 sidecar")
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
}

func corruptPCLNIntegrationPayload(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) <= int(pclnmap.HeaderSize) {
		t.Fatalf("sidecar has no payload: %d bytes", len(raw))
	}
	raw[len(raw)-1] ^= 0xff
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mismatchPCLNIntegrationIdentity(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	view, err := pclnmap.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	off := bytes.Index(raw[:pclnmap.HeaderSize], view.Identity[:])
	if off < 0 {
		t.Fatal("sidecar identity is not present in its header")
	}
	raw[off] ^= 0xff
	// Identity is outside the checksummed payload, so the map remains
	// structurally valid and fails specifically at executable identity.
	if _, err := pclnmap.Parse(raw); err != nil {
		t.Fatalf("identity-only mutation invalidated the map: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

const (
	pclnIntegrationHeaderVersion     = 8
	pclnIntegrationHeaderGOARCH      = 19
	pclnIntegrationHeaderABIVersion  = 20
	pclnIntegrationHeaderPayloadHash = 32
	pclnIntegrationHeaderSections    = 96
	pclnIntegrationSectionSize       = 16
	pclnIntegrationDescStrings       = 2
)

func mismatchPCLNIntegrationVersion(t *testing.T, path string) {
	t.Helper()
	mutatePCLNIntegrationHeader(t, path, func(raw []byte) {
		binary.LittleEndian.PutUint32(raw[pclnIntegrationHeaderVersion:], pclnmap.Version+1)
	})
}

func mismatchPCLNIntegrationABI(t *testing.T, path string) {
	t.Helper()
	mutatePCLNIntegrationHeader(t, path, func(raw []byte) {
		binary.LittleEndian.PutUint32(raw[pclnIntegrationHeaderABIVersion:], pclnmap.ABIVersion+1)
	})
}

func mismatchPCLNIntegrationArchitecture(t *testing.T, path string) {
	t.Helper()
	mutatePCLNIntegrationHeader(t, path, func(raw []byte) {
		switch raw[pclnIntegrationHeaderGOARCH] {
		case pclnmap.GOARCHAMD64:
			raw[pclnIntegrationHeaderGOARCH] = pclnmap.GOARCHARM64
		default:
			raw[pclnIntegrationHeaderGOARCH] = pclnmap.GOARCHAMD64
		}
	})
}

func overlapPCLNIntegrationSections(t *testing.T, path string) {
	t.Helper()
	mutatePCLNIntegrationHeader(t, path, func(raw []byte) {
		// Force the pcline section to start at the func-record section. The
		// individual descriptors remain in bounds, but their byte ranges now
		// overlap and must be rejected before any runtime globals are installed.
		records := pclnIntegrationHeaderSections
		pclines := records + pclnIntegrationSectionSize
		off := binary.LittleEndian.Uint64(raw[records:])
		binary.LittleEndian.PutUint64(raw[pclines:], off)
	})
}

func misalignPCLNIntegrationPCSiteSection(t *testing.T, path string) {
	t.Helper()
	mutatePCLNIntegrationHeader(t, path, func(raw []byte) {
		// v4 descriptor order: records, pclines, strings, string offsets,
		// hash, symbol index, entries, pc sites.
		pcSites := pclnIntegrationHeaderSections + 7*pclnIntegrationSectionSize
		off := binary.LittleEndian.Uint64(raw[pcSites:])
		binary.LittleEndian.PutUint64(raw[pcSites:], off+1)
	})
}

func unterminatePCLNIntegrationStringPool(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	view, err := pclnmap.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	stringsSec := view.Sections[pclnIntegrationDescStrings]
	if stringsSec.Count == 0 || stringsSec.Offset+stringsSec.Count > uint64(len(raw)) {
		t.Fatalf("invalid string section: %+v", stringsSec)
	}
	last := stringsSec.Offset + stringsSec.Count - 1
	if raw[last] != 0 {
		t.Fatalf("string pool last byte = %#x, want NUL", raw[last])
	}
	raw[last] = 'x'
	hash := fnv.New64a()
	_, _ = hash.Write(raw[pclnmap.HeaderSize:])
	binary.LittleEndian.PutUint64(raw[pclnIntegrationHeaderPayloadHash:], hash.Sum64())
	// Keep the sidecar structurally valid so the runtime rejects the string
	// pool itself rather than its payload checksum.
	if _, err := pclnmap.Parse(raw); err != nil {
		t.Fatalf("string-only mutation invalidated the map: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mutatePCLNIntegrationHeader(t *testing.T, path string, mutate func([]byte)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < int(pclnmap.HeaderSize) {
		t.Fatalf("sidecar header is truncated: %d bytes", len(raw))
	}
	mutate(raw)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func pclnIntegrationHasDebugInfo(t *testing.T, path string) bool {
	t.Helper()
	if runtime.GOOS == "linux" {
		return elfHasDebugInfo(t, path)
	}
	return machoHasStabs(t, path)
}

func pclnIntegrationHasSymbolTable(t *testing.T, path string) bool {
	t.Helper()
	switch runtime.GOOS {
	case "linux":
		f, err := elf.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		return f.Section(".symtab") != nil
	case "darwin":
		f, err := macho.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		return f.Symtab != nil && len(f.Symtab.Syms) != 0
	default:
		panic(fmt.Sprintf("unexpected GOOS %q", runtime.GOOS))
	}
}
