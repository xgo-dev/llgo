package debug_test

import (
	"errors"
	"io"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"testing"
)

func TestStackReportsCaller(t *testing.T) {
	stack := string(debug.Stack())
	if !strings.Contains(stack, "TestStackReportsCaller") {
		t.Fatalf("Stack does not contain the caller: %q", stack)
	}
}

func TestPrintStackReportsCaller(t *testing.T) {
	// Capture through a seekable file: wasm has files but no OS pipe support.
	w, err := os.CreateTemp(t.TempDir(), "stack-*.log")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()
	os.Stderr = w
	debug.PrintStack()
	os.Stderr = oldStderr
	if _, err := w.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	stack, err := io.ReadAll(w)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stack), "TestPrintStackReportsCaller") {
		t.Fatalf("PrintStack does not contain the caller: %q", stack)
	}
}

func TestRuntimeSettings(t *testing.T) {
	var stats debug.GCStats
	runtime.GC()
	debug.ReadGCStats(&stats)
	if stats.NumGC < 0 || stats.PauseTotal < 0 {
		t.Fatalf("invalid GC statistics: %#v", stats)
	}

	previousGC := debug.SetGCPercent(100)
	if got := debug.SetGCPercent(previousGC); got != 100 {
		t.Fatalf("SetGCPercent restore returned %d, want 100", got)
	}
	previousLimit := debug.SetMemoryLimit(1 << 30)
	if got := debug.SetMemoryLimit(previousLimit); got != 1<<30 {
		t.Fatalf("SetMemoryLimit restore returned %d, want %d", got, int64(1<<30))
	}
	previousStack := debug.SetMaxStack(1 << 30)
	if got := debug.SetMaxStack(previousStack); got != 1<<30 {
		t.Fatalf("SetMaxStack restore returned %d, want %d", got, 1<<30)
	}
	previousThreads := debug.SetMaxThreads(10001)
	if got := debug.SetMaxThreads(previousThreads); got != 10001 {
		t.Fatalf("SetMaxThreads restore returned %d, want 10001", got)
	}
	previousPanicOnFault := debug.SetPanicOnFault(true)
	if got := debug.SetPanicOnFault(previousPanicOnFault); !got {
		t.Fatal("SetPanicOnFault did not report the previous enabled state")
	}

	debug.SetTraceback("single")
	debug.FreeOSMemory()
}

func TestPanicOnFaultStateIsGoroutineLocal(t *testing.T) {
	previous := debug.SetPanicOnFault(true)
	defer debug.SetPanicOnFault(previous)

	result := make(chan [2]bool, 1)
	go func() {
		first := debug.SetPanicOnFault(true)
		second := debug.SetPanicOnFault(false)
		result <- [2]bool{first, second}
	}()
	if got := <-result; got != [2]bool{false, true} {
		t.Fatalf("new goroutine SetPanicOnFault states = %v, want [false true]", got)
	}
	if got := debug.SetPanicOnFault(previous); !got {
		t.Fatal("child goroutine changed the parent SetPanicOnFault state")
	}
}

func TestCrashOutputDescriptor(t *testing.T) {
	crashFile, err := os.CreateTemp(t.TempDir(), "crash-*.log")
	if err != nil {
		t.Fatal(err)
	}
	if err := debug.SetCrashOutput(crashFile, debug.CrashOptions{}); runtime.GOARCH == "wasm" {
		// The Go wasm host API cannot duplicate a descriptor for crash output.
		if !errors.Is(err, syscall.ENOSYS) {
			t.Fatalf("SetCrashOutput = %v, want ENOSYS on wasm", err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	if err := debug.SetCrashOutput(nil, debug.CrashOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := crashFile.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildInfoParsingAndReading(t *testing.T) {
	const encoded = "go\t1.26\npath\texample.com/app\nmod\texample.com/app\tv1.2.3\th1:main\ndep\texample.com/dep\tv0.4.5\th1:dep\nbuild\t-compiler=gc\n"
	info, err := debug.ParseBuildInfo(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != "example.com/app" || info.Main.Path != "example.com/app" || info.Main.Version != "v1.2.3" {
		t.Fatalf("parsed main module = %#v, path %q", info.Main, info.Path)
	}
	if len(info.Deps) != 1 || info.Deps[0].Path != "example.com/dep" || info.Deps[0].Version != "v0.4.5" {
		t.Fatalf("parsed dependencies = %#v", info.Deps)
	}
	if len(info.Settings) != 1 || info.Settings[0].Key != "-compiler" || info.Settings[0].Value != "gc" {
		t.Fatalf("parsed settings = %#v", info.Settings)
	}
	roundTrip, err := debug.ParseBuildInfo(info.String())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip, info) {
		t.Fatalf("build info changed after round trip:\nfirst: %#v\nsecond: %#v", info, roundTrip)
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if !strings.HasPrefix(info.GoVersion, "go") {
			t.Fatalf("GoVersion = %q", info.GoVersion)
		}
		if info.String() == "" {
			t.Fatal("BuildInfo.String returned an empty string")
		}
	}
}
