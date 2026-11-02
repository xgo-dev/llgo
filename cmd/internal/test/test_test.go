package test

import (
	"bytes"
	"flag"
	"reflect"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/build"
)

func TestCoverageFlagsAfterPackages(t *testing.T) {
	for _, args := range [][]string{
		{"-covermode=count", ".", "-coverprofile=cover.out"},
		{".", "-covermode", "count", "-coverprofile", "cover.out"},
	} {
		resetTestFlags()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		flags.AddTestBinaryFlags(fs)
		ordered, err := interspersedTestFlags(fs, args)
		if err != nil {
			t.Fatal(err)
		}
		if err := fs.Parse(ordered); err != nil {
			t.Fatal(err)
		}
		if !flags.Cover || flags.CoverMode != "count" || flags.TestCoverProfile != "cover.out" {
			t.Fatalf("coverage flags not parsed: %v", args)
		}
		if !reflect.DeepEqual(fs.Args(), []string{"."}) {
			t.Fatalf("packages = %v", fs.Args())
		}
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.AddTestBinaryFlags(fs)
	if _, err := interspersedTestFlags(fs, []string{".", "-coverprofile"}); err == nil {
		t.Fatal("package argument was consumed as a missing flag value")
	}
	resetTestFlags()
}

func TestBuildFlagsWiring(t *testing.T) {
	if goBuildFlags.Flag != &Cmd.Flag || Cmd.Flag.Lookup("ldflags") == nil ||
		Cmd.Flag.Lookup("buildmode") == nil || Cmd.Flag.Lookup("debug-trace") == nil {
		t.Fatal("test build flags are not bound to the test command")
	}
}

func resetTestFlags() {
	flags.BuildTrace = ""
	flags.Verbose = false
	flags.TestRun = ""
	flags.TestBench = ""
	flags.TestTimeout = flags.DefaultTestTimeout
	flags.TestShort = false
	flags.TestCount = 1
	flags.Cover = false
	flags.CoverMode = ""
	flags.TestCoverProfile = ""
	flags.CoverPkg = ""
	flags.TestParallel = 0
	flags.TestFailfast = false
	flags.TestJSON = false
	flags.TestList = ""
	flags.TestSkip = ""
	flags.TestShuffle = ""
	flags.TestFullpath = false
	flags.TestBenchmem = false
	flags.TestBenchtime = ""
	flags.TestCPUProfile = ""
	flags.TestMemProfile = ""
	flags.TestMemProfileRate = 0
	flags.TestOutputDir = ""
	flags.TestPaniconexit0 = false
	flags.TestTestLogFile = ""
	flags.TestGoCoverDir = ""
	flags.TestArtifacts = false
}

func TestTestBinaryAliasesAfterPackages(t *testing.T) {
	resetTestFlags()
	defer resetTestFlags()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	flags.AddCommonFlags(fs)
	flags.AddTestBinaryFlags(fs)
	args, err := interspersedTestFlags(fs, []string{"-run=old", ".", "-test.run", "TestAlias", "-test.v", "-test.count=2", "-test.coverprofile=cover.out", "-test.artifacts"})
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if flags.TestRun != "TestAlias" || !flags.Verbose || flags.TestCount != 2 || !flags.Cover || !flags.TestArtifacts || flags.TestCoverProfile != "cover.out" || !reflect.DeepEqual(fs.Args(), []string{"."}) {
		t.Fatalf("aliases did not configure the test driver: %v", args)
	}
	got := buildTestArgs([]string{"-test.run=AfterArgs"})
	if got[len(got)-1] != "-test.run=AfterArgs" {
		t.Fatalf("custom test arguments lost precedence: %v", got)
	}
	for _, name := range []string{"test.json", "test.cover", "test.vet", "test.exec"} {
		if fs.Lookup(name) != nil {
			t.Errorf("driver-only flag has a test-binary alias: %s", name)
		}
	}
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

func TestTestRunsMustBeSequential(t *testing.T) {
	resetTestFlags()
	if testRunsMustBeSequential() {
		t.Fatal("ordinary tests must not be forced to run sequentially")
	}

	flags.TestCoverProfile = "cover.out"
	if testRunsMustBeSequential() {
		t.Fatal("per-package coverage fragments must allow parallel test execution")
	}
	flags.TestCoverProfile = ""

	flags.TestCPUProfile = "cpu.prof"
	if !testRunsMustBeSequential() {
		t.Fatal("profiling must force sequential test execution")
	}
}

func TestTestRunnerTimeout(t *testing.T) {
	maxDuration := time.Duration(1<<63 - 1)
	tests := []struct {
		value   string
		want    time.Duration
		wantErr bool
	}{
		{value: "10m", want: 10*time.Minute + testRunnerExitGrace},
		{value: "250ms", want: 250*time.Millisecond + testRunnerExitGrace},
		{value: "0", want: 0},
		{value: "-1s", want: 0},
		{value: maxDuration.String(), want: maxDuration},
		{value: "not-a-duration", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := testRunnerTimeout(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("testRunnerTimeout(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("testRunnerTimeout(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
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
			name: "panic on exit and test log file",
			setupFlags: func() {
				flags.TestPaniconexit0 = true
				flags.TestTestLogFile = "actions.log"
			},
			customArgs:  nil,
			wantContain: []string{"-test.paniconexit0", "-test.testlogfile=actions.log"},
		},
		{
			name: "json and gocoverdir",
			setupFlags: func() {
				flags.TestJSON = true
				flags.TestGoCoverDir = "/tmp/cover"
			},
			customArgs:  nil,
			wantContain: []string{"-test.v=test2json", "-test.gocoverdir=/tmp/cover"},
			wantAbsent:  []string{"-test.json", "-test.v"},
		},
		{
			name: "coverage is configured by the build driver",
			setupFlags: func() {
				flags.TestCoverProfile = "coverage.out"
				flags.Cover = true
				flags.CoverMode = "atomic"
			},
			customArgs: nil,
			wantAbsent: []string{"-test.cover", "-test.covermode=atomic", "-test.coverprofile=coverage.out"},
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

func TestUnsupportedRuntimeArgsRespectFlagValues(t *testing.T) {
	for _, args := range [][]string{{"-trace=out"}, {"-test.trace=out"}, {"-args", "-test.blockprofile=out"}, {"-run", "TestFoo", "-mutexprofile=out"}} {
		if err := rejectUnsupportedTestArgs(&Cmd.Flag, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	for _, args := range [][]string{{"-run", "-trace"}, {"-args", "-test.run", "-test.trace"}, {"-args", "-custom-trace=out"}, {"-args", "-trace=custom-value"}, {"-args", "--", "-test.trace=literal"}, {"-v", "-run=Trace"}} {
		if err := rejectUnsupportedTestArgs(&Cmd.Flag, args); err != nil {
			t.Fatalf("rejected a flag value: %v: %v", args, err)
		}
	}
}

func TestProfileTargetQualification(t *testing.T) {
	for _, tc := range []struct {
		goos, arch, target, option string
		wantErr                    bool
	}{
		{"linux", "amd64", "", "-test.cpuprofile=cpu", false},
		{"darwin", "arm64", "", "-test.cpuprofile=cpu", false},
		{"windows", "386", "", "-test.cpuprofile=cpu", false},
		{"linux", "386", "", "-test.cpuprofile=cpu", true},
		{"js", "wasm", "", "-test.cpuprofile=cpu", true},
		{"wasip1", "wasm", "wasi", "-test.memprofile=mem", true},
		{"linux", "arm64", "board", "-test.cpuprofile=cpu", true},
		{"linux", "386", "", "-test.memprofile=mem", false},
	} {
		conf := &build.Config{Goos: tc.goos, Goarch: tc.arch, Target: tc.target}
		err := validateProfileTarget(conf, []string{tc.option})
		if (err != nil) != tc.wantErr {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	for _, tag := range []string{"baremetal", "wasm", "llgo,baremetal", "llgo wasm"} {
		conf := &build.Config{Goos: "linux", Goarch: "amd64", Tags: tag}
		if err := validateProfileTarget(conf, []string{"-test.memprofile=mem"}); err == nil {
			t.Errorf("freestanding tag %q was accepted", tag)
		}
	}
	conf := &build.Config{Goos: "js", Goarch: "wasm"}
	for _, args := range [][]string{{"--", "-test.cpuprofile=literal"}, {"-test.run", "-test.cpuprofile=literal"}, {"-cpuprofile=custom"}, {"-memprofile=custom"}} {
		if err := validateProfileTarget(conf, args); err != nil {
			t.Errorf("rejected literal flag value %q: %v", args, err)
		}
	}
}
