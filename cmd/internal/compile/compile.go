package compile

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/mockable"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

// Cmd implements the compiler action used by GOROOT/test. Its option names
// intentionally follow "go tool compile" rather than the llgo build command.
var Cmd = &base.Command{
	UsageLine: "llgo tool compile [options] file.go...",
	Short:     "Compile Go source files without linking",
}

type countFlag struct {
	value int
	set   bool
}

func (f *countFlag) String() string { return strconv.Itoa(f.value) }

func (f *countFlag) Set(value string) error {
	f.set = true
	if value == "true" {
		f.value++
		return nil
	}
	if value == "false" {
		f.value = 0
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	f.value = n
	return nil
}

func (*countFlag) IsBoolFlag() bool { return true }

type stringListFlag []string

func (f *stringListFlag) String() string { return strings.Join(*f, ",") }

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

type options struct {
	concurrency int
	debug       stringListFlag
	lang        string

	noBounds  countFlag
	noColumns countFlag
	noOpt     countFlag
	noInline  countFlag
	allErrors countFlag

	complete bool
	standard bool
	version  bool
}

func newFlagSet(opts *options) *flag.FlagSet {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Var(&opts.noBounds, "B", "disable bounds checking")
	fs.Var(&opts.noColumns, "C", "disable printing of columns in error messages")
	fs.Func("D", "Unsupported: gc local import paths", func(string) error { return fmt.Errorf("-D is unsupported by LLGo") })
	fs.Func("I", "Unsupported: gc import search paths", func(string) error { return fmt.Errorf("-I is unsupported by LLGo") })
	fs.Var(&opts.noOpt, "N", "disable optimizations")
	fs.BoolVar(&opts.version, "V", false, "print version and exit")
	fs.IntVar(&opts.concurrency, "c", 0, "concurrency during compilation (1 means no concurrency)")
	fs.BoolVar(&opts.complete, "complete", false, "compiling complete package (no C or assembly)")
	fs.Var(&opts.debug, "d", "enable debugging settings")
	fs.Var(&opts.allErrors, "e", "no limit on number of errors reported")
	fs.Func("importcfg", "Unsupported: gc object import configuration", func(string) error { return fmt.Errorf("-importcfg is unsupported by LLGo") })
	fs.StringVar(&opts.lang, "lang", "", "Go language version source code expects")
	fs.Var(&opts.noInline, "l", "disable inlining")
	fs.Func("o", "Unsupported: gc object output", func(string) error { return fmt.Errorf("-o is unsupported by LLGo") })
	fs.Func("p", "Unsupported: gc package import identity", func(string) error { return fmt.Errorf("-p is unsupported by LLGo") })
	fs.BoolVar(&opts.standard, "std", false, "compiling standard library")
	for _, name := range []string{"+", "dynlink", "h", "live", "m", "race", "smallframes", "wb"} {
		fs.BoolFunc(name, "Unsupported: gc backend or diagnostics", func(string) error {
			return fmt.Errorf("-%s is recognized but unsupported by LLGo", name)
		})
	}
	return fs
}

func init() {
	Cmd.Run = runCmd
}

func runCmd(_ *base.Command, args []string) {
	opts := new(options)
	fs := newFlagSet(opts)
	if err := fs.Parse(args); err != nil {
		if err != flag.ErrHelp {
			mockable.Exit(2)
		}
		return
	}
	if opts.version {
		fmt.Printf("compile version llgo %s\n", env.Version())
		return
	}
	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "compile: no Go source files")
		mockable.Exit(2)
		return
	}
	if unsupported := opts.unsupported(); len(unsupported) != 0 {
		fmt.Fprintf(os.Stderr, "compile: unsupported llgo compiler option(s): %s\n", strings.Join(unsupported, ", "))
		mockable.Exit(2)
		return
	}
	if opts.concurrency < 0 {
		fmt.Fprintln(os.Stderr, "compile: -c must be non-negative")
		mockable.Exit(2)
		return
	}
	if opts.lang != "" && !strings.HasPrefix(opts.lang, "go1.") {
		fmt.Fprintf(os.Stderr, "compile: invalid value %q for -lang\n", opts.lang)
		mockable.Exit(2)
		return
	}

	if opts.concurrency > 0 {
		previous := runtime.GOMAXPROCS(opts.concurrency)
		defer runtime.GOMAXPROCS(previous)
	}
	conf := build.NewDefaultConf(build.ModeGen)
	conf.GoVersion = opts.lang
	conf.NoErrorColumn = opts.noColumns.value != 0
	conf.AllowNoBody = !opts.complete
	conf.DisableBoundsChecks = opts.noBounds.value != 0
	var loaderCompilerFlags []string
	if opts.allErrors.value != 0 {
		loaderCompilerFlags = append(loaderCompilerFlags, "-e")
	}
	if opts.lang != "" {
		loaderCompilerFlags = append(loaderCompilerFlags, "-lang="+opts.lang)
	}
	if opts.complete {
		loaderCompilerFlags = append(loaderCompilerFlags, "-complete")
	}
	if opts.standard {
		loaderCompilerFlags = append(loaderCompilerFlags, "-std")
	}
	if len(loaderCompilerFlags) != 0 {
		conf.GoBuildFlags = append(conf.GoBuildFlags, "-gcflags=command-line-arguments="+strings.Join(loaderCompilerFlags, " "))
	}
	if opts.noOpt.value != 0 || opts.noInline.value != 0 {
		conf.OptLevel = optlevel.O0
	}
	pkgs, err := build.Do(files, conf)
	if len(pkgs) != 0 && pkgs[0].LPkg != nil {
		pkgs[0].LPkg.Prog.Dispose()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}

func (opts *options) unsupported() []string {
	var out []string
	appendFlag := func(condition bool, name string) {
		if condition {
			out = append(out, name)
		}
	}
	appendFlag(opts.noInline.value < 0 || opts.noInline.value > 1, "-l (only 0 and 1 are supported)")
	for _, setting := range opts.debug {
		for _, item := range strings.Split(setting, ",") {
			if compatibleDebugSetting(item) {
				continue
			}
			out = append(out, "-d="+item)
		}
	}
	return out
}

func compatibleDebugSetting(setting string) bool {
	if setting == "panic" || setting == "ssa/check/on" || setting == "ssa/check/seed" {
		return true
	}
	// LLGo's x/tools SSA sanity checking is always enabled. gc uses this seed
	// for its own randomized checking order, so it needs no additional LLGo
	// action.
	return strings.HasPrefix(setting, "ssa/check/seed=")
}
