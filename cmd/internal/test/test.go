//go:build !llgo

package test

import (
	"fmt"
	"os"
	"strconv"

	"github.com/goplus/llgo/cmd/internal/base"
	"github.com/goplus/llgo/cmd/internal/flags"
	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/mockable"
)

// llgo test
var Cmd = &base.Command{
	UsageLine: "llgo test [-target platform] [build flags] [test flags] package [test binary arguments...]",
	Short:     "Compile and run Go test",
}

func init() {
	Cmd.Run = runCmd
	base.PassBuildFlags(Cmd)
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddTestFlags(&Cmd.Flag)
	flags.AddTestBinaryFlags(&Cmd.Flag)
	flags.AddEmulatorFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag)
}

func runCmd(cmd *base.Command, args []string) {
	// Split args at -args to separate llgo flags from test binary args
	llgoArgs, testBinaryArgs := splitArgsAt(args, "-args")

	if err := cmd.Flag.Parse(llgoArgs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}

	conf := build.NewDefaultConf(build.ModeTest)
	if err := flags.UpdateConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
	if err := flags.UpdatePassBuildConfig(conf, cmd.PassArgs.Args); err != nil {
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

	args = cmd.Flag.Args()
	_, err := build.Do(args, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
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
	appendString(flags.TestCoverProfile, "-test.coverprofile=")

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
