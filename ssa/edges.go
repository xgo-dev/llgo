package ssa

import (
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llvm"
)

const abiTypeMethodTableOperand = 2

func extractOrdinaryEdges(builder *meta.Builder, mod llvm.Module, abiTypeWithUncommon map[llvm.Value]struct{}) {
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		src := fn.Name()
		if fn.IsDeclaration() {
			continue
		}
		collector := ordinaryEdgeCollector{builder: builder, src: src}
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				collector.scanOperands(instr)
			}
		}
	}
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		src := global.Name()
		if global.IsDeclaration() {
			continue
		}
		init := global.Initializer()
		collector := ordinaryEdgeCollector{builder: builder, src: src}
		if _, ok := abiTypeWithUncommon[global]; ok {
			for i, n := 0, init.OperandsCount(); i < n; i++ {
				if i != abiTypeMethodTableOperand {
					collector.scan(init.Operand(i))
				}
			}
		} else {
			collector.scan(init)
		}
	}
}

type ordinaryEdgeCollector struct {
	builder  *meta.Builder
	src      string
	seen     map[llvm.Value]struct{}
	addedDst map[string]struct{} // dedup (src, dst) pairs
}

func (c *ordinaryEdgeCollector) scanOperands(v llvm.Value) {
	for i, n := 0, v.OperandsCount(); i < n; i++ {
		c.scan(v.Operand(i))
	}
}

func (c *ordinaryEdgeCollector) scan(v llvm.Value) {
	if name := namedModuleSymbol(v); name != "" {
		c.add(name)
		return
	}
	if v.IsAConstant().IsNil() {
		return
	}
	if c.seen == nil {
		c.seen = make(map[llvm.Value]struct{})
	}
	if _, ok := c.seen[v]; ok {
		return
	}
	c.seen[v] = struct{}{}
	for i, n := 0, v.OperandsCount(); i < n; i++ {
		c.scan(v.Operand(i))
	}
}

func (c *ordinaryEdgeCollector) add(dst string) {
	if dst == c.src {
		return
	}
	if c.addedDst == nil {
		c.addedDst = make(map[string]struct{})
	}
	if _, ok := c.addedDst[dst]; ok {
		return
	}
	c.addedDst[dst] = struct{}{}
	c.builder.AddOrdinaryEdge(c.builder.Sym(c.src), c.builder.Sym(dst))
}

func namedModuleSymbol(v llvm.Value) string {
	if !v.IsAFunction().IsNil() || !v.IsAGlobalVariable().IsNil() {
		return v.Name()
	}
	return ""
}
