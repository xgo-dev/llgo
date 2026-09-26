//go:build !llgo

package test

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/mockable"
)

// llgo test
var Cmd = &base.Command{
	UsageLine: "llgo test [-target platform] [build flags] [test flags] package [test binary arguments...]",
	Short:     "Compile and run Go test",
}

var goBuildFlags *base.PassArgs

const testRunnerExitGrace = 30 * time.Second

func init() {
	Cmd.Run = runCmd
	goBuildFlags = flags.CaptureGoBuildFlags(Cmd)
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddCompilerVerboseFlag(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddBuildTraceFlag(&Cmd.Flag)
	flags.AddBuildModeFlags(&Cmd.Flag)
	flags.AddTestFlags(&Cmd.Flag)
	flags.AddTestBinaryFlags(&Cmd.Flag)
	flags.AddEmulatorFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {
	// Split args at -args to separate llgo flags from test binary args
	llgoArgs, testBinaryArgs := splitArgsAt(args, "-args")

	flagArgs, err := interspersedTestFlags(&cmd.Flag, llgoArgs)
	if err == nil {
		err = cmd.Flag.Parse(flagArgs)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}

	conf := build.NewDefaultConf(build.ModeTest)
	if err := flags.UpdateBuildConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
	conf.BuildTrace = flags.BuildTrace
	if err := flags.ApplyGoBuildFlags(conf, goBuildFlags.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}

	// Match `go test` behavior: set testing.Testing() to true by forcing the
	// stdlib testing package's testBinary marker to "1" in test binaries.
	// See `var testBinary` in $GOROOT/src/testing/testing.go.
	if conf.GlobalRewrites == nil {
		conf.GlobalRewrites = make(map[string]build.Rewrites)
	}
	vars := make(build.Rewrites)
	conf.GlobalRewrites["testing"] = vars
	vars["testBinary"] = "1"

	// Build test binary arguments from flags
	conf.RunArgs = buildTestArgs(testBinaryArgs)
	runnerTimeout, err := testRunnerTimeout(flags.TestTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	conf.RunnerTimeout = runnerTimeout
	conf.TestFailFast = flags.TestFailfast
	conf.TestJSON = flags.TestJSON
	conf.TestRunSequential = testRunsMustBeSequential()
	if flags.Cover || flags.CoverMode != "" || flags.CoverPkg != "" || flags.TestCoverProfile != "" {
		conf.Coverage = &build.CoverageConfig{
			Mode:      flags.CoverMode,
			Packages:  flags.CoverPkg,
			Profile:   flags.TestCoverProfile,
			OutputDir: flags.TestOutputDir,
		}
	}

	pkgArgs := cmd.Flag.Args()
	_, err = build.Do(pkgArgs, conf)
	if err != nil {
		if err != build.ErrTestFailed {
			fmt.Fprintln(os.Stderr, err)
		} else if !conf.TestJSON {
			fmt.Fprintln(os.Stdout, "FAIL")
		}
		mockable.Exit(1)
	}
}

// Go test accepts driver flags on either side of the package list. Move known
// flags ahead of the package arguments without reordering flags or their values.
// -args has already been split off, so custom binary flags remain untouched.
func interspersedTestFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var options, packages []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			packages = append(packages, args[i+1:]...)
			options = append(options, "--")
			break
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		f := fs.Lookup(name)
		if !strings.HasPrefix(arg, "-") || f == nil {
			packages = append(packages, arg)
			continue
		}
		options = append(options, arg)
		boolFlag, isBool := f.Value.(interface{ IsBoolFlag() bool })
		if !hasValue && !(isBool && boolFlag.IsBoolFlag()) {
			if i+1 == len(args) {
				return nil, fmt.Errorf("flag needs an argument: -%s", name)
			}
			i++
			options = append(options, args[i])
		}
	}
	return append(options, packages...), nil
}

// testRunnerTimeout gives the guest test watchdog time to print its panic and
// terminate the host normally. A non-positive -timeout retains testing's
// documented behavior and disables both watchdogs.
func testRunnerTimeout(value string) (time.Duration, error) {
	testTimeout, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q for -timeout: %w", value, err)
	}
	if testTimeout <= 0 {
		return 0, nil
	}
	const maxDuration = time.Duration(1<<63 - 1)
	if testTimeout > maxDuration-testRunnerExitGrace {
		return maxDuration, nil
	}
	return testTimeout + testRunnerExitGrace, nil
}

func testRunsMustBeSequential() bool {
	// These flags either name output paths shared by every test binary or, for
	// fuzzing, require one active test binary. Keep their execution sequential.
	return flags.TestCPUProfile != "" ||
		flags.TestMemProfile != "" ||
		flags.TestBlockProfile != "" ||
		flags.TestMutexProfile != "" ||
		flags.TestTrace != "" ||
		flags.TestTestLogFile != "" ||
		flags.TestFuzz != ""
}

// splitArgsAt splits args at the separator flag (e.g., "-args")
// Returns (before, after) where after includes everything after separator
func splitArgsAt(args []string, separator string) (before, after []string) {
	for i, arg := range args {
		if arg == separator {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

// buildTestArgs constructs arguments for the test binary.
// Go test binaries expect flags in -test.* form; we only emit
// non-default values to mirror go test's behavior. Custom args
// provided after "-args" are appended unchanged.
func buildTestArgs(customArgs []string) []string {
	args := make([]string, 0, 32)

	appendBool := func(cond bool, flagName string) {
		if cond {
			args = append(args, flagName)
		}
	}
	appendString := func(val, flagName string) {
		if val != "" {
			args = append(args, flagName+val)
		}
	}
	appendInt := func(val int, flagName string, defaultVal int) {
		if val != defaultVal {
			args = append(args, flagName+strconv.Itoa(val))
		}
	}

	appendBool(flags.Verbose, "-test.v")
	appendString(flags.TestRun, "-test.run=")
	appendString(flags.TestBench, "-test.bench=")
	appendString(flags.TestList, "-test.list=")
	appendString(flags.TestSkip, "-test.skip=")
	appendString(flags.TestCPU, "-test.cpu=")

	appendString(flags.TestTimeout, "-test.timeout=") // always has a default
	appendBool(flags.TestShort, "-test.short")
	appendInt(flags.TestCount, "-test.count=", 1)
	appendInt(flags.TestParallel, "-test.parallel=", 0)
	appendBool(flags.TestFailfast, "-test.failfast")
	appendString(flags.TestShuffle, "-test.shuffle=")

	appendBool(flags.TestJSON, "-test.json")
	appendBool(flags.TestFullpath, "-test.fullpath")

	appendBool(flags.TestBenchmem, "-test.benchmem")
	appendString(flags.TestBenchtime, "-test.benchtime=")
	appendInt(flags.TestBlockProfileRate, "-test.blockprofilerate=", 0)

	appendString(flags.TestCPUProfile, "-test.cpuprofile=")
	appendString(flags.TestMemProfile, "-test.memprofile=")
	appendInt(flags.TestMemProfileRate, "-test.memprofilerate=", 0)
	appendString(flags.TestBlockProfile, "-test.blockprofile=")
	appendString(flags.TestMutexProfile, "-test.mutexprofile=")
	appendInt(flags.TestMutexProfileFrac, "-test.mutexprofilefraction=", 0)
	appendString(flags.TestTrace, "-test.trace=")
	appendString(flags.TestOutputDir, "-test.outputdir=")
	appendBool(flags.TestPaniconexit0, "-test.paniconexit0")
	appendString(flags.TestTestLogFile, "-test.testlogfile=")
	appendString(flags.TestGoCoverDir, "-test.gocoverdir=")
	appendBool(flags.TestFuzzWorker, "-test.fuzzworker")
	appendString(flags.TestFuzzCacheDir, "-test.fuzzcachedir=")

	appendString(flags.TestFuzz, "-test.fuzz=")
	appendString(flags.TestFuzzTime, "-test.fuzztime=")
	appendString(flags.TestFuzzMinimizeTime, "-test.fuzzminimizetime=")

	return append(args, customArgs...)
}
