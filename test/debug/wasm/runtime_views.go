package main

import "wasmdebug/model"

type debugNode struct {
	Name   string
	Values []int
}

func (n *debugNode) Label() string { return n.Name }

type debugLabel interface{ Label() string }
type debugLargeKey [17]int64
type debugLargeValue [18]int64

// Keep the same object graph live across growth, GC and both source stops.
var debugViews struct {
	EmptyArray      [3]struct{}
	Imported        any
	ImportedPointer any
	Text            string
	Nodes           []debugNode
	Box             any
	Pointer         any
	TypedNil        any
	Nil             any
	Label           debugLabel
	Function        func() int
	Closure         func() int
	Bound           func() string
	Nested          func() int
	Counter         func() int
	OtherCounter    func() int
	NilFunction     func() int
	Mapping         map[int]debugNode
	Indirect        map[debugLargeKey]debugLargeValue
	Set             map[int]struct{}
	Queue           chan debugNode
	Signals         chan struct{}
	Empty           []struct{}
	NilMap          map[int]int
	NilChannel      chan int
}

func debugAnswer() int { return 42 }

//go:noinline
func makeDebugCounter(value int) func() int {
	return func() int { value++; return value }
}

func prepareDebugViews() {
	imported := &shape.Record{Name: "imported", Value: 87}
	debugViews.Imported = *imported
	debugViews.ImportedPointer = imported
	debugViews.Text = "runtime 世界"
	node := &debugNode{Name: debugViews.Text, Values: []int{3, 5, 8}}
	debugViews.Nodes = []debugNode{*node, {Name: "nested", Values: []int{13, 21}}}
	debugViews.Box = *node
	debugViews.Pointer = node
	debugViews.TypedNil = (*debugNode)(nil)
	debugViews.Label = node
	debugViews.Function = debugAnswer
	debugViews.Closure = func() int { return node.Values[0] + len(node.Values) }
	debugViews.Bound = node.Label
	debugViews.Nested = func() func() int {
		value := 29
		return func() int { return value }
	}()
	debugViews.Counter = makeDebugCounter(11)
	debugViews.OtherCounter = makeDebugCounter(21)
	debugViews.Mapping = make(map[int]debugNode)
	// The 49th insertion starts a grow from eight buckets. Inspect both the
	// evacuated new buckets and the remaining old buckets at the breakpoint.
	for i := 0; i < 49; i++ {
		debugViews.Mapping[i] = debugNode{Name: "entry", Values: []int{i, i + 100}}
	}
	debugViews.Indirect = map[debugLargeKey]debugLargeValue{{7}: {11, 19}}
	debugViews.Set = map[int]struct{}{7: {}, 9: {}}
	debugViews.Queue = make(chan debugNode, 3)
	debugViews.Queue <- debugNode{Name: "discard"}
	debugViews.Queue <- debugViews.Nodes[0]
	<-debugViews.Queue
	debugViews.Queue <- debugViews.Nodes[1]
	debugViews.Queue <- debugNode{Name: "wrapped", Values: []int{34}}
	close(debugViews.Queue)
	debugViews.Signals = make(chan struct{}, 2)
	debugViews.Signals <- struct{}{}
	debugViews.Signals <- struct{}{}
	close(debugViews.Signals)
	debugViews.Empty = make([]struct{}, 3)
}

func checkDebugViews() {
	if debugViews.Imported.(shape.Record).Value != 87 || debugViews.ImportedPointer.(*shape.Record).Name != "imported" ||
		debugViews.Function() != 42 || debugViews.Closure() != 6 ||
		debugViews.Bound() != "runtime 世界" || debugViews.Nested() != 29 ||
		debugViews.Counter() != 12 || debugViews.OtherCounter() != 22 ||
		debugViews.Box.(debugNode).Values[2] != 8 ||
		debugViews.Pointer.(*debugNode).Name != debugViews.Label.Label() ||
		debugViews.TypedNil == nil || debugViews.Nil != nil ||
		debugViews.NilFunction != nil || debugViews.NilMap != nil || debugViews.NilChannel != nil ||
		len(debugViews.Mapping) != 49 || len(debugViews.Indirect) != 1 ||
		len(debugViews.Set) != 2 || len(debugViews.Empty) != 3 || len(debugViews.Signals) != 2 {
		panic("debug runtime views changed")
	}
	for i := 0; i < 49; i++ {
		if debugViews.Mapping[i].Values[0] != i {
			panic("debug map changed")
		}
	}
	for _, name := range []string{"runtime 世界", "nested", "wrapped"} {
		if (<-debugViews.Queue).Name != name {
			panic("debug channel changed")
		}
	}
	debugViews.Nodes[0].Values[0] = 55
	delete(debugViews.Mapping, 7)
}
