package build

import (
	"fmt"
	"go/constant"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const espPrintfFloatRoot = "--undefined=_printf_float"

// The stock ESP32-C3 newlib is built with nano formatted I/O. Its printf
// implementation can omit the floating-point backend if every Go call site
// has a known integer-only format. Apply this to closed links whose
// Go startup and formatting paths have no floating-point use.
func configureEmbeddedPrintf(ctx *context, linkedOrder []*aPackage) {
	conf := ctx.buildConf
	if (conf.Target != "esp32c3-basic" && conf.Target != "esp32c3-flash") || conf.OptLevel != optlevel.Oz || conf.ltoMode() != lto.Full ||
		conf.BuildMode != BuildModeExe || ctx.mode == ModeGen || ctx.frontend != nil || len(ctx.initial) != 1 ||
		conf.ModuleHook != nil || conf.LTOPlugin.Enabled() || conf.Coverage != nil || len(conf.SourceOverlay) != 0 ||
		!slices.Equal(ctx.crossCompile.ExtraFiles, embeddedBootFiles(conf.Target)) || conf.LinkOptions.ExternalLinker != "" || conf.LinkOptions.ExternalLinkerFlags != "" ||
		!slices.Contains(ctx.crossCompile.LDFLAGS, espPrintfFloatRoot) {
		return
	}
	for _, name := range []string{"LDFLAGS", "CCFLAGS", "CFLAGS", "CGO_CFLAGS", "CGO_LDFLAGS"} {
		if os.Getenv(name) != "" || ctx.commands.lookup(name) != "" {
			return
		}
	}
	opaqueAssembly := &linkPruningInputs{}
	if opaqueGlobalInitPackage(ctx, make(map[string]bool), opaqueAssembly) != "" || opaqueAssembly.opaqueAsmPkg != "" {
		return
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
		return
	}
	ctx.crossCompile.LDFLAGS = slices.DeleteFunc(slices.Clone(ctx.crossCompile.LDFLAGS), func(flag string) bool {
		return flag == espPrintfFloatRoot
	})
	if conf.PrintCommands || conf.Verbose {
		fmt.Fprintln(os.Stderr, "llgo: omit unused ESP32-C3 floating-point printf")
	}
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
