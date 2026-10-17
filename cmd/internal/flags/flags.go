package flags

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/compat"
	"github.com/xgo-dev/llgo/cmd/internal/compilerhash"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
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
var CompilerVerbose bool
var BuildMode string
var Tags string
var Target string
var Emulator bool
var Port string
var BaudRate int
var CheckLinkArgs bool
var CheckLLFiles bool
var GenLLFiles bool
var ForceEspClang bool
var SizeReport bool
var SizeFormat string
var SizeLevel string
var ForceRebuild bool
var PrintCommands bool
var BuildTrace string
var DeadcodeDrop bool
var PthreadStackSize byteSizeFlag
var OptLevel optlevel.Level

type byteSizeFlag struct {
	value int64
	set   bool
}

func (p *byteSizeFlag) String() string {
	if p == nil {
		return "0"
	}
	return strconv.FormatInt(p.value, 10)
}

func (p *byteSizeFlag) Set(v string) error {
	n, err := parseByteSize(v)
	if err != nil {
		return err
	}
	p.value = n
	p.set = true
	return nil
}

func parseByteSize(v string) (int64, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("size must be >= 0")
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid size %q", v)
	}
	n, err := strconv.ParseUint(s[:i], 10, 63)
	if err != nil {
		return 0, err
	}
	unit := strings.ToUpper(strings.TrimSpace(s[i:]))
	mul, ok := map[string]uint64{
		"":    1,
		"B":   1,
		"K":   1024,
		"KB":  1024,
		"KIB": 1024,
		"M":   1024 * 1024,
		"MB":  1024 * 1024,
		"MIB": 1024 * 1024,
		"G":   1024 * 1024 * 1024,
		"GB":  1024 * 1024 * 1024,
		"GIB": 1024 * 1024 * 1024,
	}[unit]
	if !ok {
		return 0, fmt.Errorf("invalid size unit %q", strings.TrimSpace(s[i:]))
	}
	if n > uint64(^uint64(0)>>1)/mul {
		return 0, fmt.Errorf("size %q overflows int64", v)
	}
	return int64(n * mul), nil
}

type ltoFlag struct {
	Specified bool
	Mode      lto.Mode
}

func (o *ltoFlag) String() string {
	return o.Mode.String()
}

func (o *ltoFlag) Set(v string) error {
	mode, err := lto.Parse(v)
	if err != nil {
		return err
	}
	o.Specified = true
	o.Mode = mode
	return nil
}

var LTO ltoFlag
var LTOPluginPath string

func AddLTOFlag(fs *flag.FlagSet) {
	LTO = ltoFlag{Mode: lto.Off}
	LTOPluginPath = ""
	fs.Var(&LTO, "lto", "Enable LTO optimization: thin or full (default: off)")
}

var GoGlobalDCE *bool

func AddGlobalDCEFlag(fs *flag.FlagSet) {
	GoGlobalDCE = nil
	if !buildenv.Dev {
		return
	}
	fs.BoolFunc("globaldce", "Enable Go global DCE with full LTO (default: true when -lto=full)", func(v string) error {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		GoGlobalDCE = &enabled
		return nil
	})
}

func ResolveLTOMode(defaultValue lto.Mode) lto.Mode {
	if LTO.Specified {
		return LTO.Mode
	}
	return defaultValue
}

const DefaultTestTimeout = "10m" // Matches Go's default test timeout

func AddCommonFlags(fs *flag.FlagSet) {
	AddDirectoryFlag(fs)
	fs.BoolVar(&Verbose, "v", false, "Verbose output")
}

func AddCompilerVerboseFlag(fs *flag.FlagSet) {
	fs.BoolVar(&CompilerVerbose, "compiler-verbose", false, "Print verbose compiler output")
	fs.BoolVar(&CompilerVerbose, "cv", false, "Print verbose compiler output (shorthand for -compiler-verbose)")
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
	fs.Func("O", "Optimization level (0,1,2,3,s,z; default s for host, z for targets)", func(val string) error {
		level, err := optlevel.Parse(val)
		if err != nil {
			return err
		}
		return setOptLevel(level, "-O="+val)
	})
}

func AddBuildFlags(fs *flag.FlagSet) {
	DeadcodeDrop = false
	PthreadStackSize = byteSizeFlag{}
	fs.BoolVar(&ForceRebuild, "a", false, "Force rebuilding of packages that are already up-to-date")
	fs.BoolVar(&PrintCommands, "x", false, "Print the commands")
	if buildenv.Dev {
		fs.BoolVar(&DeadcodeDrop, "deadcodedrop", false, "Enable Go dead code drop")
	}
	AddOptLevelFlags(fs)
	AddLTOFlag(fs)
	fs.StringVar(&LTOPluginPath, "lto-pass-plugin", "", "Load an LLVM LTO pass plugin during full LTO (ELF lld or Mach-O ld64.lld)")
	AddGlobalDCEFlag(fs)
	addPCLNFlag(fs)
	addDebugArtifactFlag(fs)
	fs.StringVar(&Tags, "tags", "", "Build tags")
	fs.Var(&PthreadStackSize, "goroutine-stack-size", "Stack size for native threads and WebAssembly goroutines, e.g. 2MB (0 uses the platform default)")
	fs.Var(&PthreadStackSize, "pthread-stack-size", "Alias for -goroutine-stack-size")
	if buildenv.Dev {
		fs.BoolVar(&CheckLinkArgs, "check-linkargs", false, "check link args valid")
		fs.BoolVar(&CheckLLFiles, "check-llfiles", false, "check .ll files valid")
		fs.BoolVar(&GenLLFiles, "gen-llfiles", false, "generate .ll files for pkg export")
		fs.BoolVar(&ForceEspClang, "force-espclang", false, "force to use esp-clang")
	}
}

// AddSizeFlags adds artifact size reports for llgo build.
func AddSizeFlags(fs *flag.FlagSet) {
	fs.BoolVar(&SizeReport, "size", false, "Print size report after build (default format=text, level=module)")
	fs.StringVar(&SizeFormat, "size-format", "", "Size report format (text,json). Default text.")
	fs.StringVar(&SizeLevel, "size-level", "", "Size report aggregation level (full,module,package). Default module.")
}

// AddBuildTraceFlag adds the build-scheduler trace flag. It is intentionally
// separate from AddBuildFlags for commands that execute one traced build
// invocation, currently "llgo build" and "llgo test".
func AddBuildTraceFlag(fs *flag.FlagSet) {
	BuildTrace = ""
	fs.StringVar(&BuildTrace, "debug-trace", "", "Write a Chrome/Perfetto build-scheduler trace to file")
}

func AddBuildModeFlags(fs *flag.FlagSet) {
	fs.StringVar(&BuildMode, "buildmode", "exe", "Build mode (exe, c-archive, c-shared)")
}

var Gen bool
var CompileOnly bool

// Coverage flags are shared by build and test. Profiles remain test-only;
// covered applications write Go's binary coverage data through GOCOVERDIR.
var (
	Cover     bool
	CoverMode string
	CoverPkg  string
)

func AddCoverageFlags(fs *flag.FlagSet) {
	// BoolVar resets Cover; Func does not reset its backing strings.
	CoverMode = ""
	CoverPkg = ""
	fs.BoolVar(&Cover, "cover", false, "Enable coverage analysis")
	fs.Func("covermode", "Coverage mode: set, count, atomic", func(value string) error {
		switch value {
		case "", "set", "count", "atomic":
			CoverMode = value
			Cover = true
			return nil
		default:
			return fmt.Errorf("valid modes are %q, %q, or %q", "set", "count", "atomic")
		}
	})
	fs.Func("coverpkg", "Apply coverage analysis to packages matching the patterns", func(value string) error {
		CoverPkg = value
		Cover = true
		return nil
	})
}

// Test binary flags
var (
	TestRun            string
	TestBench          string
	TestTimeout        string
	TestShort          bool
	TestCount          int
	TestCoverProfile   string
	TestParallel       int
	TestFailfast       bool
	TestJSON           bool
	TestList           string
	TestSkip           string
	TestShuffle        string
	TestFullpath       bool
	TestBenchmem       bool
	TestBenchtime      string
	TestCPUProfile     string
	TestMemProfile     string
	TestMemProfileRate int
	TestOutputDir      string
	TestPaniconexit0   bool
	TestTestLogFile    string
	TestGoCoverDir     string
	TestArtifacts      bool
)

func AddTestBinaryFlags(fs *flag.FlagSet) {
	AddCoverageFlags(fs)
	TestCoverProfile = ""
	fs.BoolVar(&TestArtifacts, "artifacts", false, "Save test artifacts in the output directory")
	fs.StringVar(&TestRun, "run", "", "Run only tests matching the regular expression")
	fs.StringVar(&TestBench, "bench", "", "Run benchmarks matching the regular expression")
	fs.StringVar(&TestTimeout, "timeout", DefaultTestTimeout, "Test timeout duration (e.g., 10m, 30s)")
	fs.BoolVar(&TestShort, "short", false, "Tell long-running tests to shorten their run time")
	fs.IntVar(&TestCount, "count", 1, "Run each test and benchmark n times")
	rejectValue(fs, "cpu", compat.Flag("test", "cpu").Note)
	fs.Func("coverprofile", "Write coverage profile to file", func(value string) error {
		TestCoverProfile = value
		Cover = true
		return nil
	})
	fs.IntVar(&TestParallel, "parallel", 0, "Maximum number of tests to run simultaneously")
	fs.BoolVar(&TestFailfast, "failfast", false, "Do not start new tests after the first test failure")
	fs.BoolVar(&TestJSON, "json", false, "Log verbose output in JSON format")
	fs.StringVar(&TestList, "list", "", "List tests, benchmarks, or examples matching the regular expression")
	fs.StringVar(&TestSkip, "skip", "", "Skip tests matching the regular expression")
	fs.StringVar(&TestShuffle, "shuffle", "", "Randomize the execution order of tests and benchmarks")
	fs.BoolVar(&TestFullpath, "fullpath", false, "Show full file names in error messages")
	fs.BoolVar(&TestBenchmem, "benchmem", false, "Print memory allocation statistics for benchmarks")
	fs.StringVar(&TestBenchtime, "benchtime", "", "Run benchmarks for duration d (e.g., 1s, 100x)")
	rejectValue(fs, "blockprofilerate", "LLGo does not implement goroutine blocking samples")
	fs.StringVar(&TestCPUProfile, "cpuprofile", "", "Write a CPU profile to the specified file")
	fs.StringVar(&TestMemProfile, "memprofile", "", "Write an allocation profile to the file")
	TestMemProfileRate = 0
	fs.Func("memprofilerate", "LLGo allocation profile rate: only 1 (full allocation counting) is supported", func(value string) error {
		if err := ValidateMemProfileRate(value); err != nil {
			return err
		}
		TestMemProfileRate = 1
		return nil
	})
	rejectValue(fs, "blockprofile", "LLGo does not implement goroutine blocking samples")
	rejectValue(fs, "mutexprofile", "LLGo does not implement mutex contention samples")
	rejectValue(fs, "mutexprofilefraction", "LLGo does not implement mutex contention samples")
	rejectValue(fs, "trace", "LLGo does not implement Go execution traces")
	fs.StringVar(&TestOutputDir, "outputdir", "", "Write output files to the specified directory")
	fs.BoolVar(&TestPaniconexit0, "paniconexit0", false, "Panic on call to os.Exit(0)")
	fs.StringVar(&TestTestLogFile, "testlogfile", "", "Write test action log to file")
	fs.StringVar(&TestGoCoverDir, "gocoverdir", "", "Directory where intermediate coverage files are written")
	rejectBool(fs, "fuzzworker", compat.Flag("test", "fuzz").Note)
	rejectValue(fs, "fuzzcachedir", compat.Flag("test", "fuzz").Note)
	for _, name := range []string{"fuzz", "fuzztime", "fuzzminimizetime"} {
		rejectValue(fs, name, compat.Flag("test", name).Note)
	}
	for _, name := range strings.Fields(compat.TestBinaryFlags) {
		if f := fs.Lookup(name); f != nil {
			fs.Var(f.Value, "test."+name, f.Usage+" (alias for -"+name+")")
		}
	}
}

func AddEmulatorFlags(fs *flag.FlagSet) {
	fs.BoolVar(&Emulator, "emulator", false, "Run in emulator mode")
}

func AddTestFlags(fs *flag.FlagSet) {
	fs.StringVar(&OutputFile, "o", "", "Compile test binary to the named file")
	fs.BoolVar(&CompileOnly, "c", false, "Compile test binary but do not run it")
}

func AddTargetFlag(fs *flag.FlagSet) {
	fs.StringVar(&Target, "target", "", "Target platform (e.g., rp2040, wasi)")
}

func AddEmbeddedFlags(fs *flag.FlagSet) {
	AddTargetFlag(fs)
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
	conf.PrintPackages = false
	switch conf.Mode {
	case build.ModeBuild, build.ModeRun, build.ModeInstall:
		// Keep Go build/run/install -v package output, reporting packages after
		// successful compilation. The legacy LLGo compiler output is available
		// through -compiler-verbose.
		conf.Verbose = CompilerVerbose
		conf.PrintPackages = Verbose
	case build.ModeTest:
		// For go test, -v controls the test binary only. buildTestArgs forwards
		// it as -test.v.
		conf.Verbose = CompilerVerbose
	}
	conf.PrintCommands = PrintCommands
	conf.DeadcodeDrop = DeadcodeDrop
	conf.OptLevel = OptLevel
	conf.Target = Target
	conf.Port = Port
	conf.BaudRate = BaudRate
	conf.ForceRebuild = ForceRebuild
	if PthreadStackSize.set {
		conf.PthreadStackSize = PthreadStackSize.value
		conf.PthreadStackSizeSet = true
	}
	if LTO.Specified {
		conf.LTO = LTO.Mode
	}
	if PCLN.Specified {
		conf.PCLNMode = PCLN.Mode
		conf.PCLNModeSet = true
	}
	if DebugArtifact.Specified {
		conf.DebugArtifactMode = DebugArtifact.Mode
		conf.DebugArtifactModeSet = true
	}
	if LTOPluginPath != "" {
		if conf.LTO != lto.Full {
			return fmt.Errorf("lto pass plugin can only be enabled with full LTO (-lto=full)")
		}
		conf.LTOPlugin = lto.PassPlugin{Path: LTOPluginPath}
	}
	if GoGlobalDCE != nil {
		if *GoGlobalDCE && conf.LTO != lto.Full {
			return fmt.Errorf("globaldce can only be enabled with full LTO (-lto=full)")
		}
		conf.DisableGoGlobalDCE = !*GoGlobalDCE
	}
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
