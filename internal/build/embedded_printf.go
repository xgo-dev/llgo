package build

import (
	"bytes"
	"fmt"
	"go/constant"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const (
	espPrintfFloatRoot   = "--undefined=_printf_float"
	espPrintfFloatTag    = "llgo.esp32c3.printf_float"
	espPrintfFloatMarker = "__llgo_esp32c3_requires_printf_float"
)

// Every program built with this profile shares the same runtime archive.
// The only program-dependent choice is whether to force libc's weak float
// formatter into the final link.
func embeddedPrintfFloatProfile(conf *Config, target crosscompile.Export) bool {
	return (conf.Target == "esp32c3-basic" || conf.Target == "esp32c3-flash") &&
		conf.OptLevel == optlevel.Oz && conf.ltoMode() == lto.Full &&
		target.Libc == "newlib-esp32" && slices.Contains(target.LDFLAGS, espPrintfFloatRoot)
}

// Retain the forced root for application C calls that may need floating-point
// formatting. When Go needs a runtime, probe whether LTO can discard its
// floating-point marker before omitting the root.
func configureEmbeddedPrintf(ctx *context, linkedOrder []*aPackage, needRuntime bool, runtimeLinkArgs []string) (probe bool) {
	conf := ctx.buildConf
	if !embeddedPrintfFloatProfile(conf, ctx.crossCompile) || conf.BuildMode != BuildModeExe ||
		ctx.mode == ModeGen || ctx.frontend != nil || len(ctx.initial) != 1 ||
		conf.ModuleHook != nil || conf.LTOPlugin.Enabled() || conf.Coverage != nil || len(conf.SourceOverlay) != 0 ||
		!slices.Equal(ctx.crossCompile.ExtraFiles, embeddedBootFiles(conf.Target)) ||
		conf.LinkOptions.ExternalLinker != "" || conf.LinkOptions.ExternalLinkerFlags != "" {
		return false
	}
	if needRuntime {
		// The stock collector only redirects allocation. Other runtime link
		// arguments may link libraries whose format strings we cannot inspect.
		for _, arg := range runtimeLinkArgs {
			switch arg {
			case "--wrap=malloc", "--wrap=realloc", "--wrap=calloc":
			default:
				return false
			}
		}
		if !ctx.embeddedRuntimeFree {
			symbol, ok := ctx.prog.Linkname(llssa.PkgRuntime + ".requirePrintfFloat")
			if !ok || symbol != "C."+espPrintfFloatMarker {
				return false
			}
		}
	}
	for _, name := range []string{"LDFLAGS", "CCFLAGS", "CFLAGS", "CGO_CFLAGS", "CGO_LDFLAGS"} {
		if os.Getenv(name) != "" || ctx.commands.lookup(name) != "" {
			return false
		}
	}
	opaqueAssembly := &linkPruningInputs{}
	if opaqueGlobalInitPackage(ctx, make(map[string]bool), opaqueAssembly) != "" ||
		opaqueAssembly.opaqueAsmPkg != "" || slices.ContainsFunc(opaqueAssembly.roots, isPrintfSymbol) {
		return false
	}
	extern := func(fn *ssa.Function) string {
		if fn.Pkg == nil {
			return ""
		}
		pkg := fn.Pkg.Pkg
		symbol := llssa.FuncName(pkg, fn.Name(), fn.Signature.Recv(), true)
		name, _ := ctx.prog.Linkname(symbol)
		return name
	}
	linked := make(map[*ssa.Package]bool)
	for _, archive := range linkedOrder {
		if isRuntimePkg(archive.PkgPath) {
			continue
		}
		linked[archive.SSA] = true
		// The alternate package is emitted into this ordinary archive, even
		// though the alternate source belongs to the runtime module.
		if archive.AltPkg != nil {
			linked[ctx.progSSA.Package(archive.AltPkg.Types)] = true
		}
	}
	if needsEmbeddedPrintfFloat(ssautil.AllFunctions(ctx.progSSA), linked, extern) {
		return false
	}
	if needRuntime && !ctx.embeddedRuntimeFree {
		return true
	}
	omitEmbeddedPrintfFloatRoot(&ctx.crossCompile)
	if conf.PrintCommands || conf.Verbose {
		fmt.Fprintln(os.Stderr, "llgo: omit unused ESP32-C3 floating-point printf")
	}
	return false
}

func omitEmbeddedPrintfFloatRoot(target *crosscompile.Export) {
	target.LDFLAGS = slices.DeleteFunc(slices.Clone(target.LDFLAGS), func(flag string) bool {
		return flag == espPrintfFloatRoot
	})
}

func defineEmbeddedPrintfFloatMarker(pkg llssa.Package) {
	fn := pkg.NewFunc(espPrintfFloatMarker, llssa.NoArgsNoRet, llssa.CDecl)
	pkg.Module().NamedFunction(espPrintfFloatMarker).SetVisibility(llvm.HiddenVisibility)
	fn.MakeBody(1).Return()
}

func buildEmbeddedPrintfFloatMarker(ctx *context, exportFile string) (string, error) {
	pkg := ctx.prog.NewPackage("", "esp32c3.printf.float")
	defer pkg.Module().Dispose()
	defineEmbeddedPrintfFloatMarker(pkg)
	return exportObject(ctx, "esp32c3_printf_float", exportFile+"-printf-float", pkg)
}

func linkEmbeddedPrintfFloat(ctx *context, plan *mainLinkPlan, output string, verbose bool) error {
	linked, err := tryEmbeddedLink(ctx, plan, output, plan.linkInputs, true, verbose, espPrintfFloatMarker)
	if err != nil {
		return err
	}
	if linked {
		if ctx.shouldPrintCommands(verbose) {
			fmt.Fprintln(os.Stderr, "llgo: omit unused ESP32-C3 floating-point printf")
		}
		return nil
	}
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintln(os.Stderr, "llgo: retain ESP32-C3 floating-point printf for Go builtin printing")
	}
	inputs := slices.Concat([]string{plan.embeddedPrintfFloatMarker}, plan.linkInputs)
	return linkObjFiles(ctx, output, inputs, plan.linkArgs, verbose)
}

// A missing marker rejects the speculative link. Leave the caller's link
// flags and output alone until the reduced link has succeeded.
func tryEmbeddedLink(ctx *context, plan *mainLinkPlan, output string, inputs []string,
	omitFloat, verbose bool, fallbackMarkers ...string) (bool, error) {
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	temp, err := os.MkdirTemp(dir, ".llgo-embedded-link-*")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(temp)
	pending := filepath.Join(temp, filepath.Base(output))
	probe := newLinkExecutionContext(ctx, plan)
	if omitFloat {
		omitEmbeddedPrintfFloatRoot(&probe.crossCompile)
	}
	var diagnostics bytes.Buffer
	probe.linkDiagnostics = &diagnostics
	if err := linkObjFiles(probe, pending, inputs, plan.linkArgs, verbose); err != nil {
		for _, marker := range fallbackMarkers {
			if bytes.Contains(diagnostics.Bytes(), []byte("undefined symbol: "+marker)) {
				return false, nil
			}
		}
		_, _ = diagnostics.WriteTo(os.Stderr)
		return false, err
	}
	_, _ = diagnostics.WriteTo(os.Stderr)
	if err := os.Rename(pending, output); err != nil {
		return false, err
	}
	ctx.crossCompile.LDFLAGS = probe.crossCompile.LDFLAGS
	return true, nil
}

func needsEmbeddedPrintfFloat(functions map[*ssa.Function]bool, linked map[*ssa.Package]bool, extern func(*ssa.Function) string) bool {
	var scratch [8]*ssa.Value
	for fn := range functions {
		if fn.Pkg != nil && !linked[fn.Pkg] {
			continue
		}
		// A C-linked method can derive the format from its receiver.
		if fn.Signature.Recv() != nil && isPrintfSymbol(extern(fn)) {
			return true
		}
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				var direct *ssa.Function
				var directValue *ssa.Value
				if ins, ok := inst.(ssa.CallInstruction); ok {
					call := ins.Common()
					direct = call.StaticCallee()
					directValue = &call.Value
					if direct != nil {
						if symbol := extern(direct); isPrintfSymbol(symbol) {
							index := printfFormatIndex(symbol)
							if index < 0 || index >= len(call.Args) || !knownIntegerPrintfFormat(call.Args[index], extern) {
								return true
							}
						}
					}
				}
				for _, operand := range inst.Operands(scratch[:0]) {
					if operand == nil {
						continue
					}
					target, _ := (*operand).(*ssa.Function)
					if target == nil {
						continue
					}
					if operand != directValue && isPrintfSymbol(extern(target)) {
						return true
					}
				}
			}
		}
	}
	return false
}

func isPrintfSymbol(name string) bool {
	name = strings.TrimPrefix(name, "C.")
	switch name {
	case "err", "errx", "verr", "verrx", "warn", "warnx", "vwarn", "vwarnx", "error", "error_at_line", "syslog", "vsyslog":
		return true
	}
	return strings.Contains(name, "printf")
}

func printfFormatIndex(name string) int {
	switch strings.TrimPrefix(name, "C.") {
	case "printf", "vprintf":
		return 0
	case "fprintf", "sprintf", "vfprintf", "vsprintf", "asprintf", "vasprintf":
		return 1
	case "snprintf", "vsnprintf", "asnprintf", "vasnprintf":
		return 2
	}
	return -1
}

func knownIntegerPrintfFormat(value ssa.Value, extern func(*ssa.Function) string) bool {
	call, ok := value.(*ssa.Call)
	if !ok {
		return false
	}
	target := call.Common().StaticCallee()
	if target == nil || extern(target) != "llgo.cstr" || len(call.Common().Args) != 1 {
		return false
	}
	format, ok := call.Common().Args[0].(*ssa.Const)
	return ok && format.Value != nil && format.Value.Kind() == constant.String &&
		integerPrintfFormat(constant.StringVal(format.Value))
}

func integerPrintfFormat(format string) bool {
	format, _, _ = strings.Cut(format, "\x00")
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i < len(format) && format[i] == '%' {
			continue
		}
		for i < len(format) && strings.ContainsRune("#0- +'123456789.*$hljztL", rune(format[i])) {
			i++
		}
		if i == len(format) || !strings.ContainsRune("diuoxXcspn", rune(format[i])) {
			return false
		}
	}
	return true
}
