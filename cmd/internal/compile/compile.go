//go:build !llgo

package compile

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/goplus/llgo/cmd/internal/base"
	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/mockable"
	"github.com/goplus/llgo/internal/optlevel"
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
	output      string
	pkgPath     string
	importCfg   string
	localImport string
	includes    stringListFlag

	noBounds  countFlag
	noColumns countFlag
	noOpt     countFlag
	noInline  countFlag
	allErrors countFlag
	showOpt   countFlag
	writeBar  countFlag

	complete    bool
	dynlink     bool
	live        bool
	race        bool
	smallFrames bool
	standard    bool
	runtimePkg  bool
	version     bool
}

func newFlagSet(opts *options) *flag.FlagSet {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Var(&opts.noBounds, "B", "disable bounds checking")
	fs.Var(&opts.noColumns, "C", "disable printing of columns in error messages")
	fs.StringVar(&opts.localImport, "D", "", "set relative path for local imports")
	fs.Var(&opts.includes, "I", "add directory to import search path")
	fs.Var(&opts.noOpt, "N", "disable optimizations")
	fs.BoolVar(&opts.runtimePkg, "+", false, "compiling runtime")
	fs.BoolVar(&opts.version, "V", false, "print version and exit")
	fs.IntVar(&opts.concurrency, "c", 0, "concurrency during compilation (1 means no concurrency)")
	fs.BoolVar(&opts.complete, "complete", false, "compiling complete package (no C or assembly)")
	fs.Var(&opts.debug, "d", "enable debugging settings")
	fs.BoolVar(&opts.dynlink, "dynlink", false, "support references to Go symbols defined in shared libraries")
	fs.Var(&opts.allErrors, "e", "no limit on number of errors reported")
	fs.StringVar(&opts.importCfg, "importcfg", "", "read import configuration from file")
	fs.StringVar(&opts.lang, "lang", "", "Go language version source code expects")
	fs.Var(&opts.noInline, "l", "disable inlining")
	fs.Var(&opts.showOpt, "m", "print optimization decisions")
	fs.BoolVar(&opts.live, "live", false, "debug liveness analysis")
	fs.StringVar(&opts.output, "o", "", "write output to file")
	fs.StringVar(&opts.pkgPath, "p", "", "set expected package import path")
	fs.BoolVar(&opts.race, "race", false, "enable race detector")
	fs.BoolVar(&opts.smallFrames, "smallframes", false, "reduce the size limit for stack allocated objects")
	fs.BoolVar(&opts.standard, "std", false, "compiling standard library")
	fs.Var(&opts.writeBar, "wb", "enable write barrier")
	return fs
}

func init() {
	Cmd.Run = runCmd
}

func runCmd(_ *base.Command, args []string) {
	opts := new(options)
	fs := newFlagSet(opts)
	if err := fs.Parse(args); err != nil {
		mockable.Exit(2)
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
	appendFlag(opts.noBounds.value != 0, "-B")
	appendFlag(opts.dynlink, "-dynlink")
	appendFlag(opts.showOpt.value != 0, "-m")
	appendFlag(opts.live, "-live")
	appendFlag(opts.race, "-race")
	appendFlag(opts.smallFrames, "-smallframes")
	appendFlag(opts.standard, "-std")
	appendFlag(opts.runtimePkg, "-+")
	appendFlag(opts.writeBar.set, "-wb")
	for _, setting := range opts.debug {
		for _, item := range strings.Split(setting, ",") {
			if item == "panic" || item == "ssa/check/on" {
				continue
			}
			out = append(out, "-d="+item)
		}
	}
	return out
}
