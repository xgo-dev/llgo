package flags

import (
	"flag"
	"fmt"

	"github.com/goplus/llgo/cmd/internal/compilerhash"
	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/buildenv"
	"github.com/goplus/llgo/internal/optlevel"
)

var OutputFile string
var OutBin bool
var OutHex bool
var OutImg bool
var OutUf2 bool
var OutZip bool

func AddOutputFlags(fs *flag.FlagSet) {
	fs.StringVar(&OutputFile, "o", "", "Output file")
	fs.BoolVar(&OutBin, "obin", false, "Generate binary output (.bin)")
	fs.BoolVar(&OutHex, "ohex", false, "Generate Intel hex output (.hex)")
	fs.BoolVar(&OutImg, "oimg", false, "Generate image output (.img)")
	fs.BoolVar(&OutUf2, "ouf2", false, "Generate UF2 output (.uf2)")
	fs.BoolVar(&OutZip, "ozip", false, "Generate ZIP/DFU output (.zip)")
}

var Verbose bool
var BuildEnv string
var BuildMode string
var Tags string
var Target string
var Emulator bool
var Port string
var BaudRate int
var AbiMode int
var CheckLinkArgs bool
var CheckLLFiles bool
var GenLLFiles bool
var ForceEspClang bool
var SizeReport bool
var SizeFormat string
var SizeLevel string
var ForceRebuild bool
var PrintCommands bool
var OptLevel optlevel.Level

const DefaultTestTimeout = "10m" // Matches Go's default test timeout

func AddCommonFlags(fs *flag.FlagSet) {
	fs.BoolVar(&Verbose, "v", false, "Verbose output")
}

func AddOptLevelFlags(fs *flag.FlagSet) {
	OptLevel = optlevel.Unset
	var optSource string
	setOptLevel := func(level optlevel.Level, source string) error {
		if optSource != "" {
			return fmt.Errorf("optimization flags are mutually exclusive: %s and %s", optSource, source)
		}
		OptLevel = level
		optSource = source
		return nil
	}
	optLevelBoolFunc := func(level optlevel.Level, source string) func(string) error {
		return func(string) error {
			return setOptLevel(level, source)
		}
	}

	fs.BoolFunc("O0", "Disable optimizations", optLevelBoolFunc(optlevel.O0, "-O0"))
	fs.BoolFunc("O1", "Optimize lightly", optLevelBoolFunc(optlevel.O1, "-O1"))
	fs.BoolFunc("O2", "Optimize for performance", optLevelBoolFunc(optlevel.O2, "-O2"))
	fs.BoolFunc("O3", "Optimize aggressively", optLevelBoolFunc(optlevel.O3, "-O3"))
	fs.BoolFunc("Os", "Optimize for size", optLevelBoolFunc(optlevel.Os, "-Os"))
	fs.BoolFunc("Oz", "Optimize aggressively for size", optLevelBoolFunc(optlevel.Oz, "-Oz"))
	fs.Func("O", "Optimization level (0,1,2,3,s,z)", func(val string) error {
		level, err := optlevel.Parse(val)
		if err != nil {
			return err
		}
		return setOptLevel(level, "-O="+val)
	})
}

func AddBuildFlags(fs *flag.FlagSet) {
	fs.BoolVar(&ForceRebuild, "a", false, "Force rebuilding of packages that are already up-to-date")
	fs.BoolVar(&PrintCommands, "x", false, "Print the commands")
	AddOptLevelFlags(fs)
	fs.StringVar(&Tags, "tags", "", "Build tags")
	fs.StringVar(&BuildEnv, "buildenv", "", "Build environment")
	if buildenv.Dev {
		fs.IntVar(&AbiMode, "abi", 2, "ABI mode (default 2). 0 = none, 1 = cfunc, 2 = allfunc.")
		fs.BoolVar(&CheckLinkArgs, "check-linkargs", false, "check link args valid")
		fs.BoolVar(&CheckLLFiles, "check-llfiles", false, "check .ll files valid")
		fs.BoolVar(&GenLLFiles, "gen-llfiles", false, "generate .ll files for pkg export")
		fs.BoolVar(&ForceEspClang, "force-espclang", false, "force to use esp-clang")
	}

	fs.BoolVar(&SizeReport, "size", false, "Print size report after build (default format=text, level=module)")
	fs.StringVar(&SizeFormat, "size-format", "", "Size report format (text,json). Default text.")
	fs.StringVar(&SizeLevel, "size-level", "", "Size report aggregation level (full,module,package). Default module.")
}

func AddBuildModeFlags(fs *flag.FlagSet) {
	fs.StringVar(&BuildMode, "buildmode", "exe", "Build mode (exe, c-archive, c-shared)")
}

var Gen bool
var CompileOnly bool

// Test binary flags
var (
	TestRun              string
	TestBench            string
	TestTimeout          string
	TestShort            bool
	TestCount            int
	TestCPU              string
	TestCover            bool
	TestCoverMode        string
	TestCoverProfile     string
	TestCoverPkg         string
	TestParallel         int
	TestFailfast         bool
	TestJSON             bool
	TestList             string
	TestSkip             string
	TestShuffle          string
	TestFullpath         bool
	TestBenchmem         bool
	TestBenchtime        string
	TestBlockProfileRate int
	TestCPUProfile       string
	TestMemProfile       string
	TestMemProfileRate   int
	TestBlockProfile     string
	TestMutexProfile     string
	TestMutexProfileFrac int
	TestTrace            string
	TestOutputDir        string
	TestPaniconexit0     bool
	TestTestLogFile      string
	TestGoCoverDir       string
	TestFuzzWorker       bool
	TestFuzzCacheDir     string
	TestFuzz             string
	TestFuzzTime         string
	TestFuzzMinimizeTime string
)

func AddTestBinaryFlags(fs *flag.FlagSet) {
	fs.StringVar(&TestRun, "run", "", "Run only tests matching the regular expression")
	fs.StringVar(&TestBench, "bench", "", "Run benchmarks matching the regular expression")
	fs.StringVar(&TestTimeout, "timeout", DefaultTestTimeout, "Test timeout duration (e.g., 10m, 30s)")
	fs.BoolVar(&TestShort, "short", false, "Tell long-running tests to shorten their run time")
	fs.IntVar(&TestCount, "count", 1, "Run each test and benchmark n times")
	fs.StringVar(&TestCPU, "cpu", "", "Comma-separated list of GOMAXPROCS values for which the tests or benchmarks should be executed")
	fs.BoolVar(&TestCover, "cover", false, "Enable coverage analysis")
	fs.StringVar(&TestCoverMode, "covermode", "", "Coverage mode: set, count, atomic")
	fs.StringVar(&TestCoverProfile, "coverprofile", "", "Write coverage profile to file")
	fs.StringVar(&TestCoverPkg, "coverpkg", "", "Apply coverage analysis to packages matching the patterns")
	fs.IntVar(&TestParallel, "parallel", 0, "Maximum number of tests to run simultaneously")
	fs.BoolVar(&TestFailfast, "failfast", false, "Do not start new tests after the first test failure")
	fs.BoolVar(&TestJSON, "json", false, "Log verbose output in JSON format")
	fs.StringVar(&TestList, "list", "", "List tests, benchmarks, or examples matching the regular expression")
	fs.StringVar(&TestSkip, "skip", "", "Skip tests matching the regular expression")
	fs.StringVar(&TestShuffle, "shuffle", "", "Randomize the execution order of tests and benchmarks")
	fs.BoolVar(&TestFullpath, "fullpath", false, "Show full file names in error messages")
	fs.BoolVar(&TestBenchmem, "benchmem", false, "Print memory allocation statistics for benchmarks")
	fs.StringVar(&TestBenchtime, "benchtime", "", "Run benchmarks for duration d (e.g., 1s, 100x)")
	fs.IntVar(&TestBlockProfileRate, "blockprofilerate", 0, "Control the detail provided in goroutine blocking profiles by calling runtime.SetBlockProfileRate")
	fs.StringVar(&TestCPUProfile, "cpuprofile", "", "Write a CPU profile to the specified file")
	fs.StringVar(&TestMemProfile, "memprofile", "", "Write an allocation profile to the file")
	fs.IntVar(&TestMemProfileRate, "memprofilerate", 0, "Enable more precise (and expensive) memory allocation profiles by setting runtime.MemProfileRate")
	fs.StringVar(&TestBlockProfile, "blockprofile", "", "Write a goroutine blocking profile to the specified file")
	fs.StringVar(&TestMutexProfile, "mutexprofile", "", "Write a mutex contention profile to the specified file")
	fs.IntVar(&TestMutexProfileFrac, "mutexprofilefraction", 0, "Sample 1 in n stack traces of goroutines holding a contended mutex")
	fs.StringVar(&TestTrace, "trace", "", "Write an execution trace to the specified file")
	fs.StringVar(&TestOutputDir, "outputdir", "", "Write output files to the specified directory")
	fs.BoolVar(&TestPaniconexit0, "paniconexit0", false, "Panic on call to os.Exit(0)")
	fs.StringVar(&TestTestLogFile, "testlogfile", "", "Write test action log to file")
	fs.StringVar(&TestGoCoverDir, "gocoverdir", "", "Directory where intermediate coverage files are written")
	fs.BoolVar(&TestFuzzWorker, "fuzzworker", false, "Coordinate with the parent process to fuzz random values (for use only by cmd/go)")
	fs.StringVar(&TestFuzzCacheDir, "fuzzcachedir", "", "Directory where interesting fuzzing inputs are stored (for use only by cmd/go)")
	fs.StringVar(&TestFuzz, "fuzz", "", "Run the fuzz test matching the regular expression")
	fs.StringVar(&TestFuzzTime, "fuzztime", "", "Run fuzzing for the specified duration (e.g., 10s, 1m)")
	fs.StringVar(&TestFuzzMinimizeTime, "fuzzminimizetime", "", "Time to spend minimizing a value after finding a crash (default: 60s)")
}

func AddEmulatorFlags(fs *flag.FlagSet) {
	fs.BoolVar(&Emulator, "emulator", false, "Run in emulator mode")
}

func AddTestFlags(fs *flag.FlagSet) {
	fs.StringVar(&OutputFile, "o", "", "Compile test binary to the named file")
	fs.BoolVar(&CompileOnly, "c", false, "Compile test binary but do not run it")
}

func AddEmbeddedFlags(fs *flag.FlagSet) {
	fs.StringVar(&Target, "target", "", "Target platform (e.g., rp2040, wasi)")
	fs.StringVar(&Port, "port", "", "Target port for flashing")
	fs.IntVar(&BaudRate, "baudrate", 115200, "Baudrate for serial communication")
}

func AddCmpTestFlags(fs *flag.FlagSet) {
	fs.BoolVar(&Gen, "gen", false, "Generate llgo.expect file")
}

func UpdateConfig(conf *build.Config) error {
	conf.CompilerHash = compilerhash.Value()
	conf.Tags = Tags
	conf.Verbose = Verbose
	conf.PrintCommands = PrintCommands
	conf.OptLevel = OptLevel
	conf.Target = Target
	conf.Port = Port
	conf.BaudRate = BaudRate
	conf.ForceRebuild = ForceRebuild
	if SizeReport || SizeFormat != "" || SizeLevel != "" {
		conf.SizeReport = true
		if SizeFormat != "" {
			conf.SizeFormat = SizeFormat
		}
		if SizeLevel != "" {
			conf.SizeLevel = SizeLevel
		}
	}

	switch conf.Mode {
	case build.ModeBuild:
		conf.OutFile = OutputFile
		conf.OutFmts = build.OutFmts{
			Bin: OutBin,
			Hex: OutHex,
			Img: OutImg,
			Uf2: OutUf2,
			Zip: OutZip,
		}
	case build.ModeRun:
		conf.Emulator = Emulator
	case build.ModeTest:
		conf.OutFile = OutputFile
		conf.CompileOnly = CompileOnly
		conf.Emulator = Emulator
	case build.ModeInstall:

	case build.ModeCmpTest:
		conf.Emulator = Emulator
		conf.GenExpect = Gen
	}
	if buildenv.Dev {
		conf.AbiMode = build.AbiMode(AbiMode)
		conf.CheckLinkArgs = CheckLinkArgs
		conf.CheckLLFiles = CheckLLFiles
		conf.GenLL = GenLLFiles
		conf.ForceEspClang = ForceEspClang
	}
	return nil
}

func UpdateBuildConfig(conf *build.Config) error {
	// First apply common config
	if err := UpdateConfig(conf); err != nil {
		return err
	}
	if err := build.ValidateBuildMode(BuildMode); err != nil {
		return err
	}
	conf.BuildMode = build.BuildMode(BuildMode)

	return nil
}
