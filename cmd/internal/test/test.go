package test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
var testExec *[]string

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
	testExec = flags.AddExecFlag(&Cmd.Flag)
	flags.AddEmulatorFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag)
	flags.FinalizeNativeFlags("test", &Cmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {
	var err error
	args, err = flags.ChangeDirectory(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	if err := rejectUnsupportedTestArgs(&cmd.Flag, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	// Split args at -args to separate llgo flags from test binary args
	llgoArgs, testBinaryArgs := splitArgsAt(args, "-args")

	flagArgs, err := interspersedTestFlags(&cmd.Flag, llgoArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	if err := flags.ParseAfterDirectory(&cmd.Flag, flagArgs); err != nil {
		if err == flag.ErrHelp {
			return
		}
		// flag.Parse has already printed the diagnostic and usage.
		mockable.Exit(1)
		return
	}

	conf := build.NewDefaultConf(build.ModeTest)
	conf.Exec = *testExec
	if err := flags.UpdateBuildConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
	if conf.BuildMode != build.BuildModeExe && !conf.CompileOnly {
		fmt.Fprintln(os.Stderr, "test execution requires -buildmode=exe; use -c to produce a library artifact")
		mockable.Exit(1)
		return
	}
	conf.BuildTrace = flags.BuildTrace
	if err := flags.ApplyGoBuildFlags(conf, goBuildFlags.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
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
	conf.RetainTestBinary = flags.TestCPUProfile != "" || flags.TestMemProfile != ""
	if flags.TestOutputDir != "" || flags.TestArtifacts || conf.RetainTestBinary {
		flags.TestOutputDir, err = filepath.Abs(flags.TestOutputDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			mockable.Exit(1)
			return
		}
	}
	conf.RunArgs = buildTestArgs(testBinaryArgs)
	if err := validateProfileTarget(conf, conf.RunArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
		return
	}
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

// Profiles are qualified on native targets. Named target runtimes may lack
// sampling or filesystem support and must not return placeholder profiles.
func validateProfileTarget(conf *build.Config, args []string) error {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		name, value, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") {
			break
		}
		if !strings.HasPrefix(name, "test.") {
			// Bare names after -args belong to user-defined test flags.
			continue
		}
		name = strings.TrimPrefix(name, "test.")
		if f := Cmd.Flag.Lookup(name); f != nil && !hasValue {
			isBool, ok := f.Value.(interface{ IsBoolFlag() bool })
			if !ok || !isBool.IsBoolFlag() {
				i++
				if i < len(args) {
					value = args[i]
				}
			}
		}
		if value == "" || name != "cpuprofile" && name != "memprofile" {
			continue
		}
		nativeOS := conf.Goos == "linux" || conf.Goos == "darwin" || conf.Goos == "windows"
		cpuArch := conf.Goarch == "amd64" || conf.Goarch == "arm64" || conf.Goos == "windows" && conf.Goarch == "386"
		freestanding := false
		for _, tag := range strings.Fields(strings.ReplaceAll(conf.Tags, ",", " ")) {
			freestanding = freestanding || tag == "baremetal" || tag == "wasm"
		}
		if conf.Target != "" || !nativeOS || conf.Goarch == "wasm" || freestanding || name == "cpuprofile" && !cpuArch {
			return fmt.Errorf("-%s is unsupported for target %s (%s/%s): LLGo profiling requires a qualified native runtime", name, conf.Target, conf.Goos, conf.Goarch)
		}
	}
	return nil
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
	// These flags name output paths shared by test binaries. Keep their
	// execution sequential.
	return flags.TestCPUProfile != "" ||
		flags.TestMemProfile != "" ||
		flags.TestTestLogFile != ""
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

	if flags.TestJSON {
		args = append(args, "-test.v=test2json")
	} else {
		appendBool(flags.Verbose, "-test.v")
	}
	appendString(flags.TestRun, "-test.run=")
	appendString(flags.TestBench, "-test.bench=")
	appendString(flags.TestList, "-test.list=")
	appendString(flags.TestSkip, "-test.skip=")

	appendString(flags.TestTimeout, "-test.timeout=") // always has a default
	appendBool(flags.TestShort, "-test.short")
	appendInt(flags.TestCount, "-test.count=", 1)
	appendInt(flags.TestParallel, "-test.parallel=", 0)
	appendBool(flags.TestFailfast, "-test.failfast")
	appendString(flags.TestShuffle, "-test.shuffle=")

	appendBool(flags.TestFullpath, "-test.fullpath")
	appendBool(flags.TestArtifacts, "-test.artifacts")

	appendBool(flags.TestBenchmem, "-test.benchmem")
	appendString(flags.TestBenchtime, "-test.benchtime=")

	appendString(flags.TestCPUProfile, "-test.cpuprofile=")
	appendString(flags.TestMemProfile, "-test.memprofile=")
	appendInt(flags.TestMemProfileRate, "-test.memprofilerate=", 0)
	appendString(flags.TestOutputDir, "-test.outputdir=")
	appendBool(flags.TestPaniconexit0, "-test.paniconexit0")
	appendString(flags.TestTestLogFile, "-test.testlogfile=")
	appendString(flags.TestGoCoverDir, "-test.gocoverdir=")

	return append(args, customArgs...)
}

// The -args boundary must not bypass rejection of placeholder runtime data.
func rejectUnsupportedTestArgs(fs *flag.FlagSet, args []string) error {
	custom := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "-args" {
			custom = true
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if custom && !strings.HasPrefix(name, "test.") {
			continue
		}
		name = strings.TrimPrefix(name, "test.")
		switch name {
		case "trace", "blockprofile", "blockprofilerate", "mutexprofile", "mutexprofilefraction", "cpu", "fuzz", "fuzztime", "fuzzminimizetime", "fuzzworker", "fuzzcachedir":
			return fmt.Errorf("-%s is recognized but unsupported: %s", name, strings.TrimPrefix(fs.Lookup(name).Usage, "Unsupported: "))
		case "memprofilerate":
			_, value, _ := strings.Cut(arg, "=")
			if !hasValue && i+1 < len(args) {
				value = args[i+1]
			}
			if err := flags.ValidateMemProfileRate(value); err != nil {
				return err
			}
		}
		if f := fs.Lookup(name); f != nil && !hasValue {
			isBool, ok := f.Value.(interface{ IsBoolFlag() bool })
			if !ok || !isBool.IsBoolFlag() {
				i++
			}
		}
	}
	return nil
}
