package build

import (
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

func TestStdoutPlanRestoresMixedCalleesAfterFileDowngrade(t *testing.T) {
	p := &stdoutPlan{regular: new(ssa.Function), typed: new(ssa.Function), file: new(ssa.Function), descriptor: new(ssa.Function), pollWrite: new(ssa.Function)}
	literal, typed := new(ssa.Call), new(ssa.Call)
	literal.Call.Value, typed.Call.Value = p.regular, p.typed
	p.record(literal)
	p.record(typed)
	root := new(ssa.Function)
	graph := callgraph.New(root)
	callgraph.AddEdge(graph.Root, nil, graph.CreateNode(p.pollWrite))
	file := cl.OptionalInitializer{Global: new(ssa.Global)}
	candidates := map[*ssa.Global]cl.OptionalInitializer{file.Global: file}
	passes := 0
	p.specialize(candidates, candidates, []cl.OptionalInitializer{file}, []*ssa.Function{root}, false, func(wantGraph bool, alternate ...*ssa.Function) (map[*ssa.Global]cl.OptionalInitializer, *callgraph.Graph) {
		passes++
		if wantGraph {
			if len(alternate) != 1 || alternate[0] != p.file || typed.Call.Value != p.descriptor {
				t.Fatal("descriptor rewrite missing")
			}
			return nil, graph
		}
		if typed.Call.Value != p.file {
			t.Fatal("file writer downgrade missing")
		}
		return nil, nil
	})
	if passes != 1 || literal.Call.Value != p.regular || typed.Call.Value != p.typed || p.selected != nil {
		t.Fatal("failed proof changed original callees")
	}
}
