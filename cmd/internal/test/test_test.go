//go:build !llgo

package test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/build"
)

func TestBuildFlagsWiring(t *testing.T) {
	if goBuildFlags.Flag != &Cmd.Flag || Cmd.Flag.Lookup("ldflags") == nil || Cmd.Flag.Lookup("buildmode") == nil {
		t.Fatal("test build flags are not bound to the test command")
	}
}

func resetTestFlags() {
	flags.Verbose = false
	flags.TestRun = ""
	flags.TestBench = ""
	flags.TestTimeout = flags.DefaultTestTimeout
	flags.TestShort = false
	flags.TestCount = 1
	flags.TestCPU = ""
	flags.TestCover = false
	flags.TestCoverMode = ""
	flags.TestCoverProfile = ""
	flags.TestCoverPkg = ""
	flags.TestParallel = 0
	flags.TestFailfast = false
	flags.TestJSON = false
	flags.TestList = ""
	flags.TestSkip = ""
	flags.TestShuffle = ""
	flags.TestFullpath = false
	flags.TestBenchmem = false
	flags.TestBenchtime = ""
	flags.TestBlockProfileRate = 0
	flags.TestCPUProfile = ""
	flags.TestMemProfile = ""
	flags.TestMemProfileRate = 0
	flags.TestBlockProfile = ""
	flags.TestMutexProfile = ""
	flags.TestMutexProfileFrac = 0
	flags.TestTrace = ""
	flags.TestOutputDir = ""
	flags.TestPaniconexit0 = false
	flags.TestTestLogFile = ""
	flags.TestGoCoverDir = ""
	flags.TestFuzzWorker = false
	flags.TestFuzzCacheDir = ""
	flags.TestFuzz = ""
	flags.TestFuzzTime = ""
	flags.TestFuzzMinimizeTime = ""
}

func TestSplitArgsAt(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		separator string
		wantBef   []string
		wantAft   []string
	}{
		{
			name:      "no separator",
			args:      []string{"-v", ".", "-run", "TestFoo"},
			separator: "-args",
			wantBef:   []string{"-v", ".", "-run", "TestFoo"},
			wantAft:   nil,
		},
		{
			name:      "with separator",
			args:      []string{"-v", ".", "-args", "-custom", "value"},
			separator: "-args",
			wantBef:   []string{"-v", "."},
			wantAft:   []string{"-custom", "value"},
		},
		{
			name:      "separator at end",
			args:      []string{"-v", ".", "-args"},
			separator: "-args",
			wantBef:   []string{"-v", "."},
			wantAft:   []string{},
		},
		{
			name:      "separator at start",
			args:      []string{"-args", "-custom", "value"},
			separator: "-args",
			wantBef:   []string{},
			wantAft:   []string{"-custom", "value"},
		},
		{
			name:      "empty args",
			args:      []string{},
			separator: "-args",
			wantBef:   []string{},
			wantAft:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBef, gotAft := splitArgsAt(tt.args, tt.separator)
			if !reflect.DeepEqual(gotBef, tt.wantBef) {
				t.Errorf("splitArgsAt() before = %v, want %v", gotBef, tt.wantBef)
			}
			if !reflect.DeepEqual(gotAft, tt.wantAft) {
				t.Errorf("splitArgsAt() after = %v, want %v", gotAft, tt.wantAft)
			}
		})
	}
}

func TestBuildParallelChildArgs(t *testing.T) {
	got := buildParallelChildArgs(
		[]string{"-p", "3", "--p=4", "-run=TestOne", "--"},
		"example.com/p",
		[]string{"-custom", "value"},
	)
	want := []string{
		"test", "-run=TestOne", "-p=1", "example.com/p",
		"-args", "-custom", "value",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildParallelChildArgs() = %v, want %v", got, want)
	}
}

func TestCanRunPackagesInParallel(t *testing.T) {
	resetTestFlags()
	t.Setenv(parallelWorkerEnv, "")
	conf := build.NewDefaultConf(build.ModeTest)
	conf.BuildParallelism = 2
	if !canRunPackagesInParallel(conf, []string{"./..."}, 2) {
		t.Fatal("ordinary package pattern cannot run in parallel")
	}

	flags.TestCoverProfile = "cover.out"
	if canRunPackagesInParallel(conf, []string{"./..."}, 2) {
		t.Fatal("shared coverage profile can run in parallel")
	}
	flags.TestCoverProfile = ""

	if canRunPackagesInParallel(conf, []string{"one_test.go"}, 2) {
		t.Fatal("Go file arguments can run in parallel")
	}
	conf.CompileOnly = true
	if canRunPackagesInParallel(conf, []string{"./..."}, 2) {
		t.Fatal("-c can run in parallel")
	}
}

func TestListTestPackages(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeTest)
	conf.BuildParallelism = 2
	pkg := "github.com/xgo-dev/llgo/internal/goflags"
	got, err := listTestPackages(conf, []string{pkg, pkg})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{pkg}; !reflect.DeepEqual(got, want) {
		t.Fatalf("listTestPackages() = %v, want %v", got, want)
	}

	_, err = listTestPackages(conf, []string{"-definitely-not-a-package"})
	if err == nil {
		t.Fatal("leading-dash package pattern unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("leading-dash package pattern was parsed as a flag: %v", err)
	}
}

func TestReportTestPackageResult(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reportTestPackageResult(&stdout, &stderr, "example.com/pass", []byte("PASS"), nil, false)
	if got, want := stdout.String(), "PASS\nok  \texample.com/pass\n"; got != want {
		t.Fatalf("success stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("success stderr = %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	reportTestPackageResult(&stdout, &stderr, "example.com/fail", []byte("failure\n"), errors.New("failed"), false)
	if got, want := stdout.String(), "failure\n"; got != want {
		t.Fatalf("failure stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "FAIL\texample.com/fail\n"; got != want {
		t.Fatalf("failure stderr = %q, want %q", got, want)
	}

	stdout.Reset()
	stderr.Reset()
	reportTestPackageResult(&stdout, &stderr, "example.com/json", []byte("{\"Action\":\"pass\"}\n"), nil, true)
	if got, want := stdout.String(), "{\"Action\":\"pass\"}\n"; got != want {
		t.Fatalf("JSON stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("JSON stderr = %q", stderr.String())
	}
}

func TestRunTestPackagesLimitAndFailure(t *testing.T) {
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	result := make(chan testRunResult)
	var active atomic.Int32
	var maximum atomic.Int32
	go func() {
		result <- runTestPackages([]string{"a", "b", "c", "d"}, 2, false, func(pkg string) error {
			now := active.Add(1)
			for {
				old := maximum.Load()
				if now <= old || maximum.CompareAndSwap(old, now) {
					break
				}
			}
			started <- struct{}{}
			<-release
			active.Add(-1)
			if pkg == "d" {
				return errors.New("failed")
			}
			return nil
		})
	}()

	<-started
	<-started
	select {
	case <-started:
		t.Fatal("more than two packages started concurrently")
	default:
	}
	close(release)
	if got := <-result; !got.failed {
		t.Fatal("runTestPackages reported success after a package failed")
	}
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrency = %d, want 2", got)
	}
}

func TestRunTestPackagesFailFast(t *testing.T) {
	var runs atomic.Int32
	result := runTestPackages([]string{"a", "b", "c"}, 1, true, func(string) error {
		runs.Add(1)
		return errors.New("failed")
	})
	if !result.failed {
		t.Fatal("runTestPackages reported success")
	}
	if result.skipped != 2 {
		t.Fatalf("skipped %d packages, want 2", result.skipped)
	}
	if got := runs.Load(); got != 1 {
		t.Fatalf("ran %d packages after the first failure, want 1", got)
	}
}

func TestBuildTestArgs(t *testing.T) {
	tests := []struct {
		name        string
		setupFlags  func()
		customArgs  []string
		wantContain []string // flags that should be present
		wantAbsent  []string // flags that should NOT be forwarded
	}{
		{
			name: "verbose only",
			setupFlags: func() {
				flags.Verbose = true
			},
			customArgs:  nil,
			wantContain: []string{"-test.v"},
		},
		{
			name: "run flag",
			setupFlags: func() {
				flags.TestRun = "TestFoo"
			},
			customArgs:  nil,
			wantContain: []string{"-test.run=TestFoo"},
		},
		{
			name: "multiple flags",
			setupFlags: func() {
				flags.Verbose = true
				flags.TestRun = "TestFoo"
				flags.TestTimeout = "30s"
				flags.TestShort = true
			},
			customArgs:  nil,
			wantContain: []string{"-test.v", "-test.run=TestFoo", "-test.timeout=30s", "-test.short"},
		},
		{
			name: "with custom args",
			setupFlags: func() {
				flags.Verbose = true
			},
			customArgs:  []string{"-custom", "value"},
			wantContain: []string{"-test.v", "-custom", "value"},
		},
		{
			name: "benchmark flags",
			setupFlags: func() {
				flags.TestBench = "."
				flags.TestBenchtime = "5s"
				flags.TestBenchmem = true
			},
			customArgs:  nil,
			wantContain: []string{"-test.bench=.", "-test.benchtime=5s", "-test.benchmem"},
		},
		{
			name: "cpu list and parallel",
			setupFlags: func() {
				flags.TestCPU = "1,2,4"
				flags.TestParallel = 8
			},
			customArgs:  nil,
			wantContain: []string{"-test.cpu=1,2,4", "-test.parallel=8"},
		},
		{
			name: "profiling rate flags",
			setupFlags: func() {
				flags.TestBlockProfileRate = 10
				flags.TestMemProfileRate = 1
				flags.TestMutexProfileFrac = 5
			},
			customArgs:  nil,
			wantContain: []string{"-test.blockprofilerate=10", "-test.memprofilerate=1", "-test.mutexprofilefraction=5"},
		},
		{
			name: "panic on exit and test log file",
			setupFlags: func() {
				flags.TestPaniconexit0 = true
				flags.TestTestLogFile = "actions.log"
			},
			customArgs:  nil,
			wantContain: []string{"-test.paniconexit0", "-test.testlogfile=actions.log"},
		},
		{
			name: "fuzz worker and cache dir",
			setupFlags: func() {
				flags.TestFuzzWorker = true
				flags.TestFuzzCacheDir = "fuzzcache"
			},
			customArgs:  nil,
			wantContain: []string{"-test.fuzzworker", "-test.fuzzcachedir=fuzzcache"},
		},
		{
			name: "json and gocoverdir",
			setupFlags: func() {
				flags.TestJSON = true
				flags.TestGoCoverDir = "/tmp/cover"
			},
			customArgs:  nil,
			wantContain: []string{"-test.json", "-test.gocoverdir=/tmp/cover"},
		},
		{
			name: "coverage profile forwarded",
			setupFlags: func() {
				flags.TestCoverProfile = "coverage.out"
				flags.TestCover = true
				flags.TestCoverMode = "atomic"
			},
			customArgs:  nil,
			wantContain: []string{"-test.coverprofile=coverage.out"},
			wantAbsent:  []string{"-test.cover", "-test.covermode=atomic"},
		},
		{
			name: "count flag",
			setupFlags: func() {
				flags.TestCount = 3
			},
			customArgs:  nil,
			wantContain: []string{"-test.count=3"},
		},
		{
			name: "parallel flag",
			setupFlags: func() {
				flags.TestParallel = 4
			},
			customArgs:  nil,
			wantContain: []string{"-test.parallel=4"},
		},
		{
			name: "profiling flags",
			setupFlags: func() {
				flags.TestCPUProfile = "cpu.prof"
				flags.TestMemProfile = "mem.prof"
			},
			customArgs:  nil,
			wantContain: []string{"-test.cpuprofile=cpu.prof", "-test.memprofile=mem.prof"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTestFlags()

			// Setup test-specific flags
			tt.setupFlags()

			got := buildTestArgs(tt.customArgs)

			// Check that all expected flags are present
			for _, want := range tt.wantContain {
				found := false
				for _, arg := range got {
					if arg == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildTestArgs() missing expected flag %q, got %v", want, got)
				}
			}

			// Ensure excluded flags are not present
			for _, notWant := range tt.wantAbsent {
				for _, arg := range got {
					if arg == notWant {
						t.Errorf("buildTestArgs() forwarded unexpected flag %q, got %v", notWant, got)
						break
					}
				}
			}

			// Ensure we didn't drop required flags; allow default timeout to be present.
		})
	}
}
