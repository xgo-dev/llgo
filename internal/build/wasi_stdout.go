package build

import (
	"go/types"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

const wasiPollWriteSymbol = "internal/poll.(*FD).Write"

// The descriptor alias lives in the standard library archive. Only calls in
// the executable package change when a closed link proves it can use the alias.
type wasiStdoutPlan struct {
	entry                     *ssa.Package
	regular, file, descriptor *ssa.Function
	pollWrite, selected       *ssa.Function
	calls                     []ssa.CallInstruction
}

func newWASIStdoutPlan(ctx *context) *wasiStdoutPlan {
	if ctx.frontend != nil || len(ctx.initial) != 1 || ctx.initial[0].Name != "main" ||
		!wasiGuardedStaticData(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) {
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
	if regular == nil || file == nil {
		return nil
	}
	plan := &wasiStdoutPlan{entry: entry, regular: regular, file: file, descriptor: osPkg.Func("llgoWriteStdoutDescriptor")}
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

func (p *wasiStdoutPlan) record(instruction ssa.Instruction) {
	if call, ok := instruction.(ssa.CallInstruction); ok && call.Common().StaticCallee() == p.regular {
		p.calls = append(p.calls, call)
	}
}

func (p *wasiStdoutPlan) specialize(unused, candidates map[*ssa.Global]cl.OptionalInitializer, files []cl.OptionalInitializer,
	roots []*ssa.Function, pollWriteRooted bool, analyze func(bool) (map[*ssa.Global]cl.OptionalInitializer, *callgraph.Graph)) map[*ssa.Global]cl.OptionalInitializer {
	if len(p.calls) == 0 {
		return unused
	}
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
	if p.descriptor != nil && p.pollWrite != nil && !pollWriteRooted {
		writer = p.descriptor
	}
	p.rewrite(writer)
	updated, graph := analyze(writer == p.descriptor)
	// A second writer is useful only when the link can drop the general
	// descriptor write path. File writes elsewhere should share that path.
	if writer == p.descriptor && hasCallPath(graph, roots, p.pollWrite) {
		writer = p.file
		p.rewrite(writer)
		updated, _ = analyze(false)
	}
	for _, file := range files {
		if _, ok := updated[file.Global]; !ok {
			p.rewrite(p.regular)
			return unused
		}
	}
	p.selected = writer
	return updated
}

func (p *wasiStdoutPlan) rewrite(writer *ssa.Function) {
	for _, call := range p.calls {
		call.Common().Value = writer
	}
}

// RTA also scans every exported method of every boxed type, which can make
// file methods reachable even in a link with no calls to them. Look for an
// actual static or dynamic call path from a root when estimating link size.
func hasCallPath(graph *callgraph.Graph, roots []*ssa.Function, target *ssa.Function) bool {
	entry := graph.Nodes[target]
	if entry == nil {
		return false
	}
	seen := map[*callgraph.Node]bool{entry: true}
	queue := []*callgraph.Node{entry}
	for i := 0; i < len(queue); i++ {
		for _, edge := range queue[i].In {
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
