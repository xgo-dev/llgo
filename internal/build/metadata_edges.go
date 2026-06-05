package build

import (
	"strings"

	"github.com/goplus/llgo/internal/metadata"
	"github.com/xgo-dev/llvm"
)

func extractOrdinaryEdges(builder *metadata.Builder, mod llvm.Module) {
	if builder == nil {
		return
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		src := fn.Name()
		if src == "" || fn.IsDeclaration() {
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
		if src == "" {
			continue
		}
		init := global.Initializer()
		if init.IsNil() {
			continue
		}
		collector := ordinaryEdgeCollector{builder: builder, src: src}
		collector.scanGlobalInitializer(init)
	}
}

type ordinaryEdgeCollector struct {
	builder *metadata.Builder
	src     string
	seen    map[llvm.Value]struct{}
}

func (c *ordinaryEdgeCollector) scanGlobalInitializer(v llvm.Value) {
	if isUncommonTypeInitializer(v) {
		for i, n := 0, v.OperandsCount(); i < n; i++ {
			if i == 2 {
				continue
			}
			c.scan(v.Operand(i))
		}
		return
	}
	c.scan(v)
}

func (c *ordinaryEdgeCollector) scanOperands(v llvm.Value) {
	for i, n := 0, v.OperandsCount(); i < n; i++ {
		c.scan(v.Operand(i))
	}
}

func (c *ordinaryEdgeCollector) scan(v llvm.Value) {
	if v.IsNil() {
		return
	}
	if isMethodTable(v) {
		return
	}
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
	if dst == "" || dst == c.src {
		return
	}
	c.builder.AddEdge(c.builder.Symbol(c.src), c.builder.Symbol(dst))
}

func namedModuleSymbol(v llvm.Value) string {
	if !v.IsAFunction().IsNil() || !v.IsAGlobalVariable().IsNil() {
		return v.Name()
	}
	return ""
}

func isUncommonTypeInitializer(v llvm.Value) bool {
	if v.IsAConstantStruct().IsNil() || v.OperandsCount() != 3 {
		return false
	}
	return isMethodTable(v.Operand(2))
}

func isMethodTable(v llvm.Value) bool {
	if v.IsNil() || v.IsAConstantArray().IsNil() || v.OperandsCount() == 0 {
		return false
	}
	methodTy := v.Type().ElementType()
	if methodTy.TypeKind() != llvm.StructTypeKind {
		return false
	}
	if strings.HasSuffix(methodTy.StructName(), ".Method") {
		return true
	}
	for i, n := 0, v.OperandsCount(); i < n; i++ {
		method := v.Operand(i)
		if method.IsAConstantStruct().IsNil() || method.OperandsCount() != 4 {
			return false
		}
		if namedModuleSymbol(method.Operand(2)) == "" || namedModuleSymbol(method.Operand(3)) == "" {
			return false
		}
	}
	return true
}
