package build

import (
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

const (
	wasiArgcSymbol         = "__llgo_argc"
	wasiArgvSymbol         = "__llgo_argv"
	wasiRuntimeArgsSymbol  = "os.runtime_args"
	wasiArgsGlobal         = "os.Args"
	wasiArgsProvider       = "os_runtime_args"
	wasiArgsProviderSymbol = "runtime." + wasiArgsProvider
	wasiClitePath          = env.LLGoRuntimePkg + "/internal/clite"
	wasiRuntimePath        = env.LLGoRuntimePkg + "/internal/lib/runtime"
)

func canOmitWASIArgs(ctx *context, result *rta.Result, assembly []string, isSymbolRoot func(string) bool) bool {
	// The external symbol map may derive its path from argv[0] in native
	// code. Go reachability cannot see that dependency.
	if ctx.buildConf.PCLNMode == PCLNExternal || isSymbolRoot(wasiArgsGlobal) {
		return false
	}
	for _, symbol := range assembly {
		switch symbol {
		case wasiArgcSymbol, wasiArgvSymbol, wasiRuntimeArgsSymbol, wasiArgsProviderSymbol:
			return false
		}
	}
	// A second declaration, including a method, could reach the same raw
	// globals or the provider without naming os.Args in Go SSA.
	for source, target := range ctx.prog.Linknames() {
		switch strings.TrimPrefix(strings.TrimPrefix(target, "go:"), "C.") {
		case wasiRuntimeArgsSymbol:
			if source != wasiArgsProviderSymbol {
				return false
			}
		case wasiArgcSymbol:
			if source != wasiClitePath+".Argc" {
				return false
			}
		case wasiArgvSymbol:
			if source != wasiClitePath+".Argv" {
				return false
			}
		case wasiArgsGlobal, wasiArgsProviderSymbol, wasiClitePath + ".Argc", wasiClitePath + ".Argv":
			return false
		}
	}
	// The provider can be visible in both the runtime patch package and
	// the patched standard package.
	var providers []*ssa.Function
	if link, _ := ctx.prog.Linkname(wasiArgsProviderSymbol); link == wasiRuntimeArgsSymbol {
		for _, path := range []string{"runtime", wasiRuntimePath} {
			if pkg := ctx.progSSA.ImportedPackage(path); pkg != nil {
				providers = append(providers, pkg.Func(wasiArgsProvider))
			}
		}
	}

	var args *ssa.Global
	var runtimeArgs *ssa.Function
	if pkg := ctx.progSSA.ImportedPackage("os"); pkg != nil {
		args, _ = pkg.Members["Args"].(*ssa.Global)
		runtimeArgs = pkg.Func("runtime_args")
		if args == nil || runtimeArgs == nil || len(runtimeArgs.Blocks) != 0 {
			return false
		}
	}
	clite := ctx.progSSA.ImportedPackage(wasiClitePath)
	var argc, argv *ssa.Global
	if clite != nil {
		argc, _ = clite.Members["Argc"].(*ssa.Global)
		argv, _ = clite.Members["Argv"].(*ssa.Global)
	}
	for fn := range result.Reachable {
		// The runtime's linknamed provider is conservatively rooted by the
		// initializer analysis. Its reads are observable through callers only.
		isProvider := slices.Contains(providers, fn)
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if _, debug := instr.(*ssa.DebugRef); debug || wasiArgsInitializer(instr, args, runtimeArgs) {
					continue
				}
				for _, operand := range instr.Operands(nil) {
					if operand == nil || *operand == nil {
						continue
					}
					switch v := (*operand).(type) {
					case *ssa.Global:
						if v == args || !isProvider && (v == argc || v == argv) {
							return false
						}
					case *ssa.Function:
						if v == runtimeArgs || slices.Contains(providers, v) {
							return false
						}
					}
				}
			}
		}
	}
	return true
}

// The sole harmless observation is the standard-library assignment of
// runtime_args() to os.Args. The returned slice cannot escape elsewhere.
func wasiArgsInitializer(instr ssa.Instruction, args *ssa.Global, runtimeArgs *ssa.Function) bool {
	if args == nil || instr.Parent().Pkg != args.Pkg || !strings.HasPrefix(instr.Parent().Name(), "init#") {
		return false
	}
	var call *ssa.Call
	switch v := instr.(type) {
	case *ssa.Store:
		call, _ = v.Val.(*ssa.Call)
	case *ssa.Call:
		call = v
	}
	if call == nil || call.Common().StaticCallee() != runtimeArgs || len(call.Common().Args) != 0 {
		return false
	}
	stores := 0
	for _, ref := range *call.Referrers() {
		if _, debug := ref.(*ssa.DebugRef); debug {
			continue
		}
		store, ok := ref.(*ssa.Store)
		if !ok || store.Addr != args || store.Block() != call.Block() {
			return false
		}
		stores++
	}
	return stores == 1
}
