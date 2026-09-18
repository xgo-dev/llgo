package testing_test

import (
	"fmt"
	"strings"
	"testing"
)

// Test basic T methods
func TestTBasicMethods(t *testing.T) {
	// Test Name
	if !strings.Contains(t.Name(), "TestTBasicMethods") {
		t.Errorf("Name() = %q", t.Name())
	}

	// Test Log and Logf
	t.Log("This is a log message")
	t.Logf("This is a formatted log: %d", 42)

	// Test Helper (just call it, shouldn't panic)
	helperFunc := func(t *testing.T) {
		t.Helper()
		t.Log("Called from helper")
	}
	helperFunc(t)

	// Test Setenv
	t.Setenv("TEST_VAR", "test_value")

	// Test TempDir
	dir := t.TempDir()
	if dir == "" {
		t.Error("TempDir returned empty")
	}

	// Test Cleanup
	t.Cleanup(func() {
		t.Log("Cleanup called")
	})

	// Test Context
	ctx := t.Context()
	if ctx == nil {
		t.Fatal("Context returned nil")
	}

	// Test Chdir
	origDir := t.TempDir()
	testChdir(t, origDir, func() { t.Chdir(origDir) })
}

func TestTFailureMethods(t *testing.T) {
	// Test Error and Errorf in a way that doesn't fail the test
	t.Run("ErrorMethods", func(t *testing.T) {
		// Capture using Failed()
		if t.Failed() {
			t.Log("Already failed")
		}
		// We test these exist, but in passing subtests
	})

	// Test Fatal and Fatalf existence
	t.Run("FatalExists", func(t *testing.T) {
		// Don't call Fatal, just verify it exists via type
		var _ func(...any) = t.Fatal
		var _ func(string, ...any) = t.Fatalf
	})

	// Test Fail and FailNow existence
	t.Run("FailExists", func(t *testing.T) {
		var _ func() = t.Fail
		var _ func() = t.FailNow
	})
}

func TestTSkip(t *testing.T) {
	// Test Skip methods existence
	t.Run("Skip", func(t *testing.T) {
		if t.Skipped() {
			t.Log("Already skipped")
		}
		var _ func(...any) = t.Skip
		var _ func(string, ...any) = t.Skipf
		var _ func() = t.SkipNow
	})

	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
	t.Log("Not in short mode")
}

func TestShortAndVerbose(t *testing.T) {
	short := testing.Short()
	verbose := testing.Verbose()
	if short {
		t.Log("short mode enabled")
	}
	if verbose {
		t.Log("verbose mode enabled")
	}

	// Test Testing() - should return true since we're in a test
	if !testing.Testing() {
		t.Fatal("Testing() should return true")
	}
}

func TestTempDir(t *testing.T) {
	dir := t.TempDir()
	if dir == "" {
		t.Fatal("TempDir returned empty string")
	}
	t.Logf("TempDir: %s", dir)
}

func TestTCleanup(t *testing.T) {
	cleaned := false
	t.Cleanup(func() {
		cleaned = true
	})
	// Cleanup will run after test finishes
	if cleaned {
		t.Fatal("Cleanup ran too early")
	}
}

func TestTDeadline(t *testing.T) {
	deadline, ok := t.Deadline()
	if ok {
		t.Logf("Test has deadline: %v", deadline)
	} else {
		t.Log("Test has no deadline")
	}
}

// Test Benchmark types
func BenchmarkExample(b *testing.B) {
	// Test B.N
	for i := 0; i < b.N; i++ {
		_ = i * 2
	}
}

func BenchmarkBMethods(b *testing.B) {
	// Test B methods
	b.Log("Log message")
	b.Logf("Formatted: %d", 42)
	b.Helper()

	name := b.Name()
	if name == "" {
		b.Error("Name is empty")
	}

	if elapsed := b.Elapsed(); elapsed < 0 {
		b.Fatalf("Elapsed should be non-negative, got %v", elapsed)
	}

	if b.Failed() {
		b.Log("Benchmark failed")
	}

	if b.Skipped() {
		b.Log("Benchmark skipped")
	}

	dir := b.TempDir()
	if dir != "" {
		testChdir(b, dir, func() { b.Chdir(dir) })
	}

	b.Setenv("BENCH_VAR", "value")
	b.Cleanup(func() {
		b.Log("Cleanup")
	})

	ctx := b.Context()
	if ctx == nil {
		b.Fatal("Context is nil")
	}

	// Test method existence
	var _ func(...any) = b.Error
	var _ func(string, ...any) = b.Errorf
	var _ func() = b.Fail
	var _ func() = b.FailNow
	var _ func(...any) = b.Fatal
	var _ func(string, ...any) = b.Fatalf
	var _ func(...any) = b.Skip
	var _ func(string, ...any) = b.Skipf
	var _ func() = b.SkipNow

	b.SetParallelism(2)

	for i := 0; i < b.N; i++ {
		_ = i
	}
}

func BenchmarkResetTimer(b *testing.B) {
	// Do some setup
	_ = make([]int, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = i * 2
	}
}

func BenchmarkStartStopTimer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Do some work that shouldn't be timed
		_ = make([]int, 10)
		b.StartTimer()

		_ = i * 2
	}
}

func BenchmarkReportAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make([]byte, 10)
	}
}

func BenchmarkReportMetric(b *testing.B) {
	b.ReportMetric(123.45, "custom/op")
	for i := 0; i < b.N; i++ {
		_ = i * 2
	}
}

func BenchmarkSetBytes(b *testing.B) {
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		_ = make([]byte, 1024)
	}
}

func BenchmarkRunParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = 1 + 1
		}
	})
}

func BenchmarkLoop(b *testing.B) {
	// Test B.Loop method
	for b.Loop() {
		_ = 1 + 1
	}
}

func BenchmarkRun(b *testing.B) {
	// Test B.Run method
	b.Run("SubBenchmark", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = i * 2
		}
	})
}

// Test AllocsPerRun
func TestAllocsPerRun(t *testing.T) {
	avg := testing.AllocsPerRun(100, func() {
		_ = make([]int, 10)
	})
	if avg < 0 {
		t.Errorf("AllocsPerRun returned negative: %f", avg)
	}
	t.Logf("Average allocations: %f", avg)
}

// Test Coverage functions
func TestCoverage(t *testing.T) {
	coverage := testing.Coverage()
	t.Logf("Coverage: %f", coverage)

	mode := testing.CoverMode()
	t.Logf("Cover mode: %s", mode)
}

// Test F (fuzzing support)
func FuzzExample(f *testing.F) {
	// Test F methods
	f.Log("Log message")
	f.Logf("Formatted: %d", 42)
	f.Helper()

	name := f.Name()
	if name == "" {
		f.Error("Name is empty")
	}

	if f.Failed() {
		f.Log("Fuzz failed")
	}

	if f.Skipped() {
		f.Log("Fuzz skipped")
	}

	dir := f.TempDir()
	if dir != "" {
		testChdir(f, dir, func() { f.Chdir(dir) })
	}

	f.Setenv("FUZZ_VAR", "value")
	f.Cleanup(func() {
		f.Log("Cleanup")
	})

	ctx := f.Context()
	if ctx == nil {
		f.Fatal("Context is nil")
	}

	// Test method existence
	var _ func(...any) = f.Error
	var _ func(string, ...any) = f.Errorf
	var _ func() = f.Fail
	var _ func() = f.FailNow
	var _ func(...any) = f.Fatal
	var _ func(string, ...any) = f.Fatalf
	var _ func(...any) = f.Skip
	var _ func(string, ...any) = f.Skipf
	var _ func() = f.SkipNow

	// Add seed corpus
	f.Add(5)
	f.Add(10)

	f.Fuzz(func(t *testing.T, n int) {
		if n < 0 {
			t.Skip("negative numbers")
		}
		// Test something
		_ = n * 2
	})
}

// Test TB interface
func helperUsingTB(tb testing.TB) {
	tb.Helper()
	tb.Log("Testing TB interface")
}

func TestTBInterface(t *testing.T) {
	helperUsingTB(t)
}

func BenchmarkTBInterface(b *testing.B) {
	helperUsingTB(b)
	for i := 0; i < b.N; i++ {
		_ = i
	}
}

// Test InternalTest
func TestInternalTest(t *testing.T) {
	it := testing.InternalTest{
		Name: "DummyTest",
		F: func(t *testing.T) {
			t.Log("Dummy test")
		},
	}

	if it.Name != "DummyTest" {
		t.Errorf("InternalTest.Name = %q", it.Name)
	}
}

// Test InternalBenchmark
func TestInternalBenchmark(t *testing.T) {
	ib := testing.InternalBenchmark{
		Name: "DummyBenchmark",
		F: func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = i
			}
		},
	}

	if ib.Name != "DummyBenchmark" {
		t.Errorf("InternalBenchmark.Name = %q", ib.Name)
	}
}

// Test InternalExample
func TestInternalExample(t *testing.T) {
	ie := testing.InternalExample{
		Name:   "ExampleTest",
		F:      func() { fmt.Println("example") },
		Output: "example\n",
	}

	if ie.Name != "ExampleTest" {
		t.Errorf("InternalExample.Name = %q", ie.Name)
	}
}

// Test InternalFuzzTarget
func TestInternalFuzzTarget(t *testing.T) {
	ift := testing.InternalFuzzTarget{
		Name: "FuzzTest",
		Fn: func(f *testing.F) {
			f.Add(1)
		},
	}

	if ift.Name != "FuzzTest" {
		t.Errorf("InternalFuzzTarget.Name = %q", ift.Name)
	}
}

// Test BenchmarkResult
func TestBenchmarkResult(t *testing.T) {
	result := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = i * 2
		}
	})

	if result.N == 0 {
		t.Error("Benchmark didn't run")
	}

	// Test String method
	str := result.String()
	if str == "" {
		t.Error("BenchmarkResult.String() is empty")
	}
	t.Logf("Benchmark result: %s", str)

	// Test NsPerOp
	nsPerOp := result.NsPerOp()
	t.Logf("ns/op: %d", nsPerOp)

	// Test AllocsPerOp
	allocsPerOp := result.AllocsPerOp()
	t.Logf("allocs/op: %d", allocsPerOp)

	// Test AllocedBytesPerOp
	bytesPerOp := result.AllocedBytesPerOp()
	t.Logf("bytes/op: %d", bytesPerOp)

	// Test MemString
	memStr := result.MemString()
	t.Logf("Memory: %s", memStr)
}

// Test Cover types
func TestCoverTypes(t *testing.T) {
	// Test Cover struct
	cover := testing.Cover{
		Mode:            "set",
		Counters:        make(map[string][]uint32),
		Blocks:          make(map[string][]testing.CoverBlock),
		CoveredPackages: "pkg1,pkg2",
	}

	if cover.Mode != "set" {
		t.Errorf("Cover.Mode = %q", cover.Mode)
	}

	// Test CoverBlock
	block := testing.CoverBlock{
		Line0: 10,
		Col0:  5,
		Line1: 15,
		Col1:  20,
		Stmts: 3,
	}

	if block.Line0 != 10 {
		t.Errorf("CoverBlock.Line0 = %d", block.Line0)
	}
}

// Test M (test main)
func TestMType(t *testing.T) {
	// We can't easily test M.Run() as it's meant for TestMain
	// But we can verify the type exists
	var _ *testing.M
	// Test method existence
	var m *testing.M
	if m != nil {
		var _ func() int = m.Run
	}
}

// Test other testing functions
func TestOtherFunctions(t *testing.T) {
	// Test Init
	testing.Init()

	// Test RunTests - we can reference it but not easily call it
	var _ func(func(string, string) (bool, error), []testing.InternalTest) bool = testing.RunTests

	// Test RunBenchmarks
	testing.RunBenchmarks(func(pat, str string) (bool, error) { return true, nil }, []testing.InternalBenchmark{})

	// Test RunExamples
	var _ func(func(string, string) (bool, error), []testing.InternalExample) bool = testing.RunExamples

	// Test Main - verify it exists but don't call it
	var _ func(func(string, string) (bool, error), []testing.InternalTest, []testing.InternalBenchmark, []testing.InternalExample) = testing.Main

	// Test RegisterCover
	cover := testing.Cover{
		Mode: "set",
	}
	testing.RegisterCover(cover)
}

// Test parallel execution
func TestParallel(t *testing.T) {
	t.Run("Parallel1", func(t *testing.T) {
		t.Parallel()
		t.Log("Running in parallel 1")
	})

	t.Run("Parallel2", func(t *testing.T) {
		t.Parallel()
		t.Log("Running in parallel 2")
	})
}

// Example test
func Example() {
	fmt.Println("Hello, World!")
	// Output: Hello, World!
}
