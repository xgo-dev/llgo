package build

import (
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// Use LLVM 22's ordinary O3 cost budget in unprofiled switch cycles, where
// dividing probability across cases can otherwise suppress useful inlining.
// LLVM still checks cost and viability; no profile weights are manufactured.
func applyDispatchInlineBudget(conf *Config, mod llvm.Module) int {
	if !dispatchInlineEnabled(conf) {
		return 0
	}
	for _, row := range mod.NamedMetadataOperands("llvm.module.flags") {
		for _, field := range row.MDNodeOperands() {
			if field.IsAMDString() && strings.Contains(field.MDString(), "ProfileSummary") {
				return 0
			}
		}
	}
	ctx := mod.Context()
	prof := ctx.MDKindID("prof")
	hint := ctx.CreateStringAttribute("function-inline-threshold", "250")
	count := 0
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() || hasInlineBarrier(fn, false, "optnone", "optsize", "minsize", "cold") {
			continue
		}
		profiled := false
		for _, bb := range fn.BasicBlocks() {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				profiled = profiled || !inst.Metadata(prof).IsNil()
			}
		}
		if profiled {
			continue
		}
		for bb := range cyclicSwitchBlocks(fn.FirstBasicBlock()) {
			for call := bb.FirstInstruction(); !call.IsNil(); call = llvm.NextInstruction(call) {
				if call.InstructionOpcode() != llvm.Call && call.InstructionOpcode() != llvm.Invoke {
					continue
				}
				callee := call.CalledValue().IsAFunction()
				if callee.IsNil() || strings.HasPrefix(callee.Name(), "llvm.") ||
					hasInlineBarrier(callee, false, "cold", "noinline", "alwaysinline", "noreturn") ||
					hasInlineBarrier(call, true, "cold", "noinline", "alwaysinline", "noreturn") ||
					!call.GetCallSiteStringAttribute(-1, "function-inline-threshold").IsNil() {
					continue
				}
				call.AddCallSiteAttribute(-1, hint)
				count++
			}
		}
	}
	return count
}

func dispatchInlineEnabled(conf *Config) bool {
	return conf != nil && conf.Goos == "linux" && conf.Goarch == "amd64" &&
		conf.OptLevel == optlevel.O3 && conf.ltoMode() == lto.Full && conf.BuildMode == BuildModeExe
}

// Leave explicit backend policy and alternative linkers in control. The same
// predicate gates both the bitcode hints and the LLVM complete-cost option.
func (c *context) dispatchInlineBudgetEnabled() bool {
	if c == nil || !dispatchInlineEnabled(c.buildConf) ||
		c.crossCompile.Toolchain.ObjectFormat != crosscompile.ObjectFormatELF ||
		c.crossCompile.Toolchain.Linker != crosscompile.LinkerFlavorELFLLD ||
		c.crossCompile.Linker != "" || c.buildConf.LinkOptions.ExternalLinker != "" {
		return false
	}
	flags := []string{c.buildConf.LinkOptions.ExternalLinkerFlags}
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		flags = append(flags, os.Getenv(name), c.commands.lookup(name))
	}
	for _, args := range [][]string{c.crossCompile.CCArgs, c.crossCompile.CXXArgs,
		c.crossCompile.LinkerArgs, c.crossCompile.CCFLAGS, c.crossCompile.CFLAGS, c.crossCompile.LDFLAGS} {
		flags = append(flags, args...)
	}
	for _, flag := range flags {
		if strings.Contains(flag, "inline") || strings.Contains(flag, "callsite") {
			return false
		}
	}
	return true
}

func hasInlineBarrier(value llvm.Value, call bool, names ...string) bool {
	for _, name := range names {
		kind := llvm.AttributeKindID(name)
		attr := value.GetEnumAttributeAtIndex
		if call {
			attr = value.GetCallSiteEnumAttribute
		}
		if !attr(-1, kind).IsNil() {
			return true
		}
	}
	return false
}

// Tarjan's traversal includes irreducible cycles and excludes unreachable code.
func cyclicSwitchBlocks(entry llvm.BasicBlock) map[llvm.BasicBlock]bool {
	index := make(map[llvm.BasicBlock]int)
	low := make(map[llvm.BasicBlock]int)
	onStack := make(map[llvm.BasicBlock]bool)
	result := make(map[llvm.BasicBlock]bool)
	var stack []llvm.BasicBlock
	var visit func(llvm.BasicBlock)
	visit = func(bb llvm.BasicBlock) {
		index[bb] = len(index) + 1
		low[bb] = index[bb]
		stack = append(stack, bb)
		onStack[bb] = true
		term := bb.LastInstruction()
		self := false
		for i := 0; i < term.SuccessorsCount(); i++ {
			next := term.Successor(i)
			self = self || next == bb
			if index[next] == 0 {
				visit(next)
				low[bb] = min(low[bb], low[next])
			} else if onStack[next] {
				low[bb] = min(low[bb], index[next])
			}
		}
		if low[bb] != index[bb] {
			return
		}
		start := len(stack) - 1
		for stack[start] != bb {
			start--
		}
		members := stack[start:]
		dispatch := false
		for _, member := range members {
			dispatch = dispatch || member.LastInstruction().InstructionOpcode() == llvm.Switch
			onStack[member] = false
		}
		if dispatch && (self || len(members) > 1) {
			for _, member := range members {
				result[member] = true
			}
		}
		stack = stack[:start]
	}
	visit(entry)
	return result
}
