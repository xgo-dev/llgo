//go:build !llgo

package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

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

const parallelWorkerEnv = "LLGO_TEST_PARALLEL_WORKER"

func init() {
	Cmd.Run = runCmd
	goBuildFlags = flags.CaptureGoBuildFlags(Cmd)
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddCompilerVerboseFlag(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddBuildModeFlags(&Cmd.Flag)
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
	if err := flags.UpdateBuildConfig(conf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
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

	pkgArgs := cmd.Flag.Args()
	parallelism := effectiveParallelism(conf.BuildParallelism)
	if canRunPackagesInParallel(conf, pkgArgs, parallelism) {
		pkgs, err := listTestPackages(conf, pkgArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			mockable.Exit(1)
		}
		if len(pkgs) > 1 {
			executable, err := os.Executable()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				mockable.Exit(1)
			}
			// flag.FlagSet stops at the first non-flag, so pkgArgs is a
			// contiguous suffix of llgoArgs.
			flagArgs := llgoArgs[:len(llgoArgs)-len(pkgArgs)]
			var outputMu sync.Mutex
			result := runTestPackages(pkgs, parallelism, flags.TestFailfast, func(pkg string) error {
				childArgs := buildParallelChildArgs(flagArgs, pkg, testBinaryArgs)
				child := exec.Command(executable, childArgs...)
				child.Env = append(os.Environ(), parallelWorkerEnv+"=1")
				var output bytes.Buffer
				child.Stdout = &output
				child.Stderr = &output
				err := child.Run()

				// Flush one completed package at a time so parallel child
				// output cannot interleave.
				outputMu.Lock()
				if err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						fmt.Fprintf(os.Stderr, "failed to run test package %s: %v\n", pkg, err)
					}
				}
				reportTestPackageResult(os.Stdout, os.Stderr, pkg, output.Bytes(), err, flags.TestJSON)
				outputMu.Unlock()
				return err
			})
			if result.skipped != 0 {
				fmt.Fprintf(os.Stderr, "FAIL\t%d package(s) skipped by -failfast\n", result.skipped)
			}
			if result.failed {
				mockable.Exit(1)
			}
			return
		}
	}

	_, err := build.Do(pkgArgs, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}

func canRunPackagesInParallel(conf *build.Config, pkgArgs []string, parallelism int) bool {
	if os.Getenv(parallelWorkerEnv) != "" || conf.Target != "" || conf.CompileOnly || conf.OutFile != "" {
		return false
	}
	if parallelism == 1 {
		return false
	}
	for _, arg := range pkgArgs {
		if strings.HasSuffix(arg, ".go") {
			return false
		}
	}
	// These flags name process-wide output files. Until LLGo merges or
	// disambiguates them like cmd/go, keep the existing sequential behavior.
	return flags.TestCoverProfile == "" &&
		flags.TestCPUProfile == "" &&
		flags.TestMemProfile == "" &&
		flags.TestBlockProfile == "" &&
		flags.TestMutexProfile == "" &&
		flags.TestTrace == "" &&
		flags.TestTestLogFile == "" &&
		flags.TestFuzz == ""
}

func effectiveParallelism(parallelism int) int {
	if parallelism == 0 {
		parallelism = runtime.GOMAXPROCS(0)
	}
	if parallelism < 1 {
		return 1
	}
	return parallelism
}

func listTestPackages(conf *build.Config, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	tags := build.DefaultBuildTags(conf.Goarch, conf.Target)
	if conf.Tags != "" {
		tags += "," + conf.Tags
	}
	args := make([]string, 0, 4+len(conf.GoBuildFlags)+len(patterns))
	args = append(args, "list", "-tags="+tags)
	args = append(args, conf.GoBuildFlags...)
	args = append(args, "--")
	args = append(args, patterns...)

	list := exec.Command("go", args...)
	list.Env = append(os.Environ(), "GOOS="+conf.Goos, "GOARCH="+conf.Goarch)
	var stderr bytes.Buffer
	list.Stderr = &stderr
	output, err := list.Output()
	if err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return nil, fmt.Errorf("%s", message)
		}
		return nil, err
	}

	lines := strings.Fields(string(output))
	pkgs := make([]string, 0, len(lines))
	seen := make(map[string]bool, len(lines))
	for _, pkg := range lines {
		if !seen[pkg] {
			seen[pkg] = true
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs, nil
}

func buildParallelChildArgs(flagArgs []string, pkg string, testBinaryArgs []string) []string {
	args := make([]string, 0, 3+len(flagArgs)+len(testBinaryArgs))
	args = append(args, "test")
	for i := 0; i < len(flagArgs); i++ {
		arg := flagArgs[i]
		switch {
		case arg == "-p" || arg == "--p":
			i++ // The successfully parsed flag always has a following value.
		case strings.HasPrefix(arg, "-p=") || strings.HasPrefix(arg, "--p="):
		case arg == "--":
			// The package pattern has already been resolved by go list, so
			// the parent's flag terminator is no longer needed.
		default:
			args = append(args, arg)
		}
	}
	// The parent owns package-level fan-out. Keep each worker's go/packages
	// loading serial to avoid multiplying -p across child processes.
	args = append(args, "-p=1")
	args = append(args, pkg)
	if len(testBinaryArgs) != 0 {
		args = append(args, "-args")
		args = append(args, testBinaryArgs...)
	}
	return args
}

func reportTestPackageResult(stdout, stderr io.Writer, pkg string, output []byte, err error, json bool) {
	if len(output) != 0 {
		_, _ = stdout.Write(output)
		if output[len(output)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "FAIL\t%s\n", pkg)
	} else if !json {
		fmt.Fprintf(stdout, "ok  \t%s\n", pkg)
	}
}

type testRunResult struct {
	failed  bool
	skipped int
}

func runTestPackages(pkgs []string, parallelism int, failFast bool, run func(string) error) testRunResult {
	if parallelism < 1 {
		parallelism = 1
	}
	if parallelism > len(pkgs) {
		parallelism = len(pkgs)
	}
	results := make(chan error, parallelism)
	start := func(pkg string) {
		go func() {
			results <- run(pkg)
		}()
	}

	next := 0
	running := 0
	for next < len(pkgs) && running < parallelism {
		start(pkgs[next])
		next++
		running++
	}
	var result testRunResult
	for running != 0 {
		if err := <-results; err != nil {
			result.failed = true
		}
		running--
		if next < len(pkgs) && !(failFast && result.failed) {
			start(pkgs[next])
			next++
			running++
		}
	}
	result.skipped = len(pkgs) - next
	return result
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
