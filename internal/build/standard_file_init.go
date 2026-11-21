package build

import (
	"go/constant"
	"go/token"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

// On WASI and bare metal, an unobserved standard File wrapper has no
// observable initialization. These targets do not register it with a poller.
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
				if call != nil && inst.Val == call && ok && global.Pkg == init.Pkg {
					first := standardFileStart(global.Name(), block.Instrs, start)
					if first >= 0 && cl.InitializerHasLocalUses(block.Instrs[first:i+1]) {
						groups = append(groups, cl.OptionalInitializer{First: block.Instrs[first], Store: inst, Global: global})
					}
				}
				call = nil
			default:
				call = nil
			}
		}
	}
	return groups
}

func standardFileStart(name string, instructions []ssa.Instruction, index int) int {
	var number int64
	var filename string
	switch name {
	case "Stdin":
		number, filename = 0, "/dev/stdin"
	case "Stdout":
		number, filename = 1, "/dev/stdout"
	case "Stderr":
		number, filename = 2, "/dev/stderr"
	default:
		return -1
	}
	args := instructions[index].(*ssa.Call).Common().Args
	if len(args) != 2 {
		return -1
	}
	path, ok := args[1].(*ssa.Const)
	if !ok || path.Value == nil || path.Value.Kind() != constant.String || constant.StringVal(path.Value) != filename {
		return -1
	}
	if isDescriptorNumber(args[0], number) {
		return index
	}

	// Unix initializes the wrappers from writable syscall globals.
	convert, ok := args[0].(*ssa.Convert)
	if !ok || index < 2 || instructions[index-1] != convert {
		return -1
	}
	load, ok := convert.X.(*ssa.UnOp)
	if !ok || load.Op != token.MUL || instructions[index-2] != load {
		return -1
	}
	fd, ok := load.X.(*ssa.Global)
	if !ok || fd.Pkg.Pkg.Path() != "syscall" || fd.Name() != name {
		return -1
	}
	return index - 2
}

func isDescriptorNumber(value ssa.Value, want int64) bool {
	fd, ok := value.(*ssa.Const)
	if !ok || fd.Value == nil || fd.Value.Kind() != constant.Int {
		return false
	}
	number, ok := constant.Int64Val(fd.Value)
	return ok && number == want
}

// fixedStdoutDescriptor proves that the variable captured by os.init
// receives only the normal descriptor and its address cannot escape.
func fixedStdoutDescriptor(group cl.OptionalInitializer, reachable *rta.Result, isSymbolRoot func(string) bool) bool {
	load, usesVariable := group.First.(*ssa.UnOp)
	if !usesVariable {
		return true
	}
	fd := load.X.(*ssa.Global)
	if isSymbolRoot("syscall.Stdout") {
		return false
	}
	init := fd.Pkg.Func("init")
	found := false
	var operands [8]*ssa.Value
	for fn := range reachable.Reachable {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				switch instr := instr.(type) {
				case *ssa.DebugRef:
					continue
				case *ssa.UnOp:
					if instr.Op == token.MUL {
						continue
					}
				case *ssa.Store:
					if instr.Addr == fd && fn == init && isDescriptorNumber(instr.Val, 1) && !found {
						found = true
						continue
					}
				}
				for _, operand := range instr.Operands(operands[:0]) {
					if operand != nil && *operand == fd {
						return false
					}
				}
			}
		}
	}
	return found
}
