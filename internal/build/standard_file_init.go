package build

import (
	"go/constant"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/ssa"
)

// NewFile constructs a wrapper and registers cleanup. WASI also reads the
// descriptor flags, and LLGo's WASI poll stub declines the registration.
// An unread and unmodified global roots the wrapper for the lifetime of the
// executable, so its cleanup never runs.
func optionalStandardFileInitializers(init *ssa.Function) (groups []cl.OptionalInitializer) {
	if init == nil || init.Synthetic != "package initializer" {
		return nil
	}
	newFile := init.Pkg.Func("NewFile")
	for _, block := range init.Blocks {
		var call *ssa.Call
		var start int
		for i, inst := range block.Instrs {
			switch inst := inst.(type) {
			case *ssa.DebugRef:
			case *ssa.Call:
				call = nil
				if newFile != nil && inst.Common().StaticCallee() == newFile {
					call, start = inst, i
				}
			case *ssa.Store:
				global, ok := inst.Addr.(*ssa.Global)
				if call != nil && inst.Val == call && ok && global.Pkg == init.Pkg &&
					standardFileArguments(global.Name(), call.Common().Args) &&
					cl.InitializerHasLocalUses(block.Instrs[start:i+1]) {
					groups = append(groups, cl.OptionalInitializer{First: call, Store: inst, Global: global})
				}
				call = nil
			default:
				call = nil
			}
		}
	}
	return groups
}

func standardFileArguments(name string, args []ssa.Value) bool {
	if len(args) != 2 {
		return false
	}
	fd, hasFD := args[0].(*ssa.Const)
	path, hasPath := args[1].(*ssa.Const)
	if !hasFD || !hasPath || fd.Value == nil || path.Value == nil ||
		fd.Value.Kind() != constant.Int || path.Value.Kind() != constant.String {
		return false
	}
	number, ok := constant.Int64Val(fd.Value)
	if !ok {
		return false
	}
	filename := constant.StringVal(path.Value)
	switch name {
	case "Stdin":
		return number == 0 && filename == "/dev/stdin"
	case "Stdout":
		return number == 1 && filename == "/dev/stdout"
	case "Stderr":
		return number == 2 && filename == "/dev/stderr"
	}
	return false
}
