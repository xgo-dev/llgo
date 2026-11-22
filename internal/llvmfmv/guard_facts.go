package llvmfmv

import "github.com/xgo-dev/llvm"

// A fact describes executions in which a boolean takes one particular value.
// An impossible alternative contributes nothing to a join. Unknown values can
// take either outcome, but prove no CPU predicates.
type guardFact struct {
	bits       uint64
	impossible bool
}

func either(a, b guardFact) guardFact {
	if a.impossible {
		return b
	}
	if b.impossible {
		return a
	}
	return guardFact{bits: a.bits & b.bits}
}

func together(a, b guardFact) guardFact {
	return guardFact{bits: a.bits | b.bits, impossible: a.impossible || b.impossible}
}

type guardOutcome struct {
	value llvm.Value
	taken bool
}

type guardFacts struct {
	queries map[llvm.Value]uint64
	helpers map[llvm.Value][2]guardFact
	cache   map[guardOutcome]guardFact
	// Facts established on block entry without inspecting PHIs. Keeping this
	// phase separate prevents a loop-carried boolean from proving itself.
	incoming map[llvm.BasicBlock]uint64
	simple   *guardFacts
}

func (g *guardFacts) outcome(v llvm.Value, taken bool) guardFact {
	key := guardOutcome{v, taken}
	if result, ok := g.cache[key]; ok {
		return result
	}
	// A backedge through a cyclic PHI supplies no additional facts.
	g.cache[key] = guardFact{}
	result := g.eval(v, taken)
	g.cache[key] = result
	return result
}

func (g *guardFacts) eval(v llvm.Value, taken bool) guardFact {
	if !v.IsAConstantInt().IsNil() {
		return guardFact{impossible: (v.ZExtValue() != 0) != taken}
	}
	if v.IsAInstruction().IsNil() {
		return guardFact{}
	}
	switch v.InstructionOpcode() {
	case llvm.Call:
		if facts, ok := g.helpers[v.CalledValue()]; ok {
			if taken {
				return facts[1]
			}
			return facts[0]
		}
		if taken {
			return guardFact{bits: g.queries[v.CalledValue()]}
		}
	case llvm.And, llvm.Or:
		a, b := g.outcome(v.Operand(0), taken), g.outcome(v.Operand(1), taken)
		if (v.InstructionOpcode() == llvm.And) == taken {
			return together(a, b)
		}
		return either(a, b)
	case llvm.Xor, llvm.ICmp:
		invert := true
		if v.InstructionOpcode() == llvm.ICmp {
			if v.Operand(0).Type().TypeKind() != llvm.IntegerTypeKind || v.Operand(0).Type().IntTypeWidth() != 1 {
				return guardFact{}
			}
			switch v.IntPredicate() {
			case llvm.IntEQ:
				invert = false
			case llvm.IntNE:
			default:
				return guardFact{}
			}
		}
		for i := 0; i < 2; i++ {
			if c := v.Operand(i).IsAConstantInt(); !c.IsNil() {
				// XOR/NE with true, or EQ with false, negates the operand.
				flip := (c.ZExtValue() != 0) == invert
				return g.outcome(v.Operand(1-i), taken != flip)
			}
		}
	case llvm.Select:
		return either(together(g.outcome(v.Operand(0), true), g.outcome(v.Operand(1), taken)),
			together(g.outcome(v.Operand(0), false), g.outcome(v.Operand(2), taken)))
	case llvm.PHI:
		if g.incoming == nil {
			return guardFact{}
		}
		result := guardFact{impossible: true}
		for i := 0; i < v.IncomingCount(); i++ {
			pred := v.IncomingBlock(i)
			mask, reachable := g.incoming[pred]
			if !reachable {
				continue
			}
			path := guardFact{bits: mask}
			term := pred.LastInstruction()
			if term.InstructionOpcode() == llvm.Br && term.SuccessorsCount() == 2 && term.Successor(0) != term.Successor(1) {
				// Only use PHI-independent evidence for the incoming edge.
				// The value itself can refer to another PHI.
				path = together(path, g.simple.outcome(term.Operand(0), term.Successor(0) == v.InstructionParent()))
			}
			result = either(result, together(path, g.outcome(v.IncomingValue(i), taken)))
		}
		return result
	}
	return guardFact{}
}

// Intersect edge facts at joins. Compute PHI-independent facts first; then a
// materialized short-circuit result can use the facts on its incoming paths.
func guardBlocks(fn llvm.Value, queries map[llvm.Value]uint64, helpers map[llvm.Value][2]guardFact) (map[llvm.BasicBlock]uint64, *guardFacts) {
	g := guardFacts{queries: queries, helpers: helpers, cache: make(map[guardOutcome]guardFact)}
	walk := func() map[llvm.BasicBlock]uint64 {
		entry := fn.EntryBasicBlock()
		known := map[llvm.BasicBlock]uint64{entry: 0}
		queue := []llvm.BasicBlock{entry}
		for len(queue) != 0 {
			block := queue[0]
			queue = queue[1:]
			term := block.LastInstruction()
			for i := 0; i < term.SuccessorsCount(); i++ {
				mask := known[block]
				if term.InstructionOpcode() == llvm.Br && term.SuccessorsCount() == 2 {
					fact := g.outcome(term.Operand(0), i == 0)
					if fact.impossible {
						continue
					}
					mask |= fact.bits
				}
				next := term.Successor(i)
				if prior, ok := known[next]; ok {
					mask &= prior
					if mask == prior {
						continue
					}
				}
				known[next] = mask
				queue = append(queue, next)
			}
		}
		return known
	}
	first := walk()
	// Only pay for the second walk if a branch expression actually used a PHI.
	simple := g
	g.simple = &simple
	g.incoming = first
	g.cache = make(map[guardOutcome]guardFact)
	for key := range simple.cache {
		if !key.value.IsAPHINode().IsNil() {
			return walk(), &g
		}
	}
	return first, &g
}
