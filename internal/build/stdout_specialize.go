package build

import (
	"go/types"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

const (
	pollWriteSymbol     = "internal/poll.(*FD).Write"
	stdoutPollInitGuard = "os.llgoSkipStdoutPollInit"
)

// The descriptor alias lives in the standard library archive. Only calls in
// the executable package change when a closed link proves it can use the alias.
type stdoutPlan struct {
	entry                                     *ssa.Package
	regular, typed, file, descriptor          *ssa.Function
	pollWrite, selected                       *ssa.Function
	discard                                   *ssa.Function
	calls                                     []stdoutCall
	builtinClosed, fileBytes, descriptorBytes *ssa.Function
	builtinCalls                              []builtinStdoutCall
}

type stdoutCall struct {
	call     ssa.CallInstruction
	original ssa.Value
}

type builtinStdoutCall struct {
	call     *ssa.Call
	original ssa.CallCommon
}

func newStdoutPlan(ctx *context) *stdoutPlan {
	if ctx.frontend != nil || len(ctx.initial) != 1 || ctx.initial[0].Name != "main" ||
		!(wasiGuardedStaticData(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) ||
			embeddedSizePruningConfig(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) ||
			ctx.buildConf.Goarch != "wasm" && hostedValueFormatSupport(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions)) {
		return nil
	}
	entry := ctx.progSSA.Package(ctx.initial[0].Types)
	fmtPkg := ctx.progSSA.ImportedPackage("fmt")
	osPkg := ctx.progSSA.ImportedPackage("os")
	if entry == nil || fmtPkg == nil || osPkg == nil {
		return nil
	}
	regular := fmtPkg.Func("llgoPrintfLiteral")
	file := osPkg.Func("llgoWriteClosedStdout")
	descriptor := osPkg.Func("llgoWriteStdoutDescriptor")
	if regular == nil || file == nil && descriptor == nil {
		return nil
	}
	plan := &stdoutPlan{entry: entry, regular: regular, typed: fmtPkg.Func("LlgoPrintfValue"), file: file, descriptor: descriptor, discard: osPkg.Func("llgoWriteStdoutDiscard")}
	plan.builtinClosed = fmtPkg.Func(fmtspecialize.BuiltinPrintfClosed)
	plan.fileBytes = osPkg.Func("llgoWriteClosedStdoutBytes")
	plan.descriptorBytes = osPkg.Func("llgoWriteStdoutDescriptorBytes")
	if poll := ctx.progSSA.ImportedPackage("internal/poll"); poll != nil {
		if fd := poll.Type("FD"); fd != nil {
			methods := ctx.progSSA.MethodSets.MethodSet(types.NewPointer(fd.Type()))
			if method := methods.Lookup(poll.Pkg, "Write"); method != nil {
				plan.pollWrite = ctx.progSSA.MethodValue(method)
			}
		}
	}
	return plan
}

func (p *stdoutPlan) record(instruction ssa.Instruction) {
	call, ok := instruction.(ssa.CallInstruction)
	if !ok {
		return
	}
	callee := call.Common().StaticCallee()
	if callee != nil && (callee == p.regular || callee == p.typed) {
		p.calls = append(p.calls, stdoutCall{call, call.Common().Value})
	}
}

func (p *stdoutPlan) recordBuiltin(plan *fmtspecialize.ValuePlan) {
	if p.builtinClosed == nil {
		return
	}
	for _, call := range plan.BuiltinPrintfCalls(p.entry) {
		p.builtinCalls = append(p.builtinCalls, builtinStdoutCall{call: call, original: *call.Common()})
	}
}

func (p *stdoutPlan) specialize(unused, candidates map[*ssa.Global]cl.OptionalInitializer, files []cl.OptionalInitializer,
	roots []*ssa.Function, keepFileWriter bool, analyze func(bool, ...*ssa.Function) (map[*ssa.Global]cl.OptionalInitializer, *callgraph.Graph)) map[*ssa.Global]cl.OptionalInitializer {
	for _, file := range files {
		if _, ok := candidates[file.Global]; !ok {
			return unused
		}
		if _, ok := unused[file.Global]; !ok && file.Global.Name() != "Stdout" {
			return unused
		}
	}
	// The regular formatter reads os.Stdout. Remove those reads tentatively
	// so the second pass can distinguish them from application observations.
	writer := p.file
	if p.descriptor != nil && p.pollWrite != nil && !keepFileWriter {
		writer = p.descriptor
	}
	if !p.canUseWriter(writer) {
		return unused
	}
	p.rewrite(writer)
	var alternate []*ssa.Function
	if writer == p.descriptor && p.file != nil && len(p.builtinCalls) == 0 {
		// Either choice only replaces direct calls. Include both writer bodies
		// in the proof, but check for poll.Write paths from real roots only.
		alternate = append(alternate, p.file)
	}
	updated, graph := analyze(writer == p.descriptor, alternate...)
	// A second writer is useful only when the link can drop the general
	// descriptor write path. File writes elsewhere should share that path.
	if writer == p.descriptor && hasCallPath(graph, roots, p.pollWrite, nil) {
		writer = p.file
		if !p.canUseWriter(writer) {
			p.rewrite(nil)
			return unused
		}
		p.rewrite(writer)
		if len(alternate) == 0 {
			updated, _ = analyze(false)
		}
	}
	for _, file := range files {
		if _, ok := updated[file.Global]; !ok {
			p.rewrite(nil)
			return unused
		}
	}
	p.selected = writer
	return updated
}

func (p *stdoutPlan) byteWriter(writer *ssa.Function) *ssa.Function {
	if writer == nil {
		return nil
	}
	if writer == p.descriptor {
		return p.descriptorBytes
	}
	if writer == p.file {
		return p.fileBytes
	}
	return nil
}

func (p *stdoutPlan) canUseWriter(writer *ssa.Function) bool {
	return writer != nil && (len(p.builtinCalls) == 0 || p.byteWriter(writer) != nil)
}

func (p *stdoutPlan) rewrite(writer *ssa.Function) {
	for _, call := range p.calls {
		call.call.Common().Value = call.original
		if writer != nil {
			call.call.Common().Value = writer
		}
	}
	byteWriter := p.byteWriter(writer)
	for _, site := range p.builtinCalls {
		common := site.call.Common()
		*common = site.original
		if byteWriter != nil {
			args := site.original.Args
			common.Value = p.builtinClosed
			common.Args = []ssa.Value{args[0], args[1], byteWriter, args[2]}
		}
	}
}

// RTA also scans every exported method of every boxed type, which can make
// file methods reachable even in a link with no calls to them. Look for an
// actual static or dynamic call path from a root when estimating link size.
func hasCallPath(graph *callgraph.Graph, roots []*ssa.Function, target *ssa.Function, omitted ssa.CallInstruction) bool {
	entry := graph.Nodes[target]
	if entry == nil {
		return false
	}
	seen := map[*callgraph.Node]bool{entry: true}
	queue := []*callgraph.Node{entry}
	for i := 0; i < len(queue); i++ {
		for _, edge := range queue[i].In {
			if omitted != nil && edge.Site == omitted {
				continue
			}
			if !seen[edge.Caller] {
				seen[edge.Caller] = true
				queue = append(queue, edge.Caller)
			}
		}
	}
	for _, fn := range roots {
		if seen[graph.Nodes[fn]] {
			return true
		}
	}
	return false
}
