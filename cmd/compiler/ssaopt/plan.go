/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package ssaopt implements pure-Go, LLVM-free analysis and optimization passes
// over github.com/xgo-dev/llgo/ssa.
//
// In v1, passes leave the underlying llgo/ssa IR read-only and emit a verified
// FunctionPlan consumed by cmd/compiler/cl during LLVM lowering. At -O0,
// required semantic checks (such as receiver nil dereferences and bounds
// checks) are preserved without aggressive elimination to keep panic evaluation
// order and debuggability intact.
package ssaopt

import (
	"fmt"
	"go/token"
	"sync"

	"github.com/xgo-dev/llgo/callgraph/rta"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
)

// Config configures the ssaopt analysis and planning pipeline.
type Config struct {
	TypeCtx               *types.Context
	OptLevel              optlevel.Level
	DisableBoundsChecks   bool
	GCRoots               bool
	CooperativeSafepoints bool
	WholeProgram          bool
	Verify                bool

	Escape     *goescape.Result
	RTA        *rta.Result
	Patches    map[string]*ssa.Package
	PatchTypes map[string]*types.Package
	RuntimePkg *types.Package
}

// Optimizing reports whether optimization passes may elide redundant checks or
// perform non-trivial specializations beyond -O0 semantic lowering.
func (c Config) Optimizing() bool {
	return c.OptLevel.IsValid() && c.OptLevel != optlevel.O0
}

// NilProofKind classifies why a receiver or dereference nil check was elided.
type NilProofKind uint8

const (
	NilProofNone NilProofKind = iota
	NilProofAlloc
	NilProofGlobal
	NilProofWrapNilChk
	NilProofAlreadyProtectedLoad
	NilProofDomCheck
	NilProofDomCond
)

func (k NilProofKind) String() string {
	switch k {
	case NilProofAlloc:
		return "alloc"
	case NilProofGlobal:
		return "global"
	case NilProofWrapNilChk:
		return "wrapnilchk"
	case NilProofAlreadyProtectedLoad:
		return "protected-load"
	case NilProofDomCheck:
		return "dom-check"
	case NilProofDomCond:
		return "dom-cond"
	default:
		return "none"
	}
}

// NilCheckPlan records the single-owner lowering decision for a pointer-method
// receiver nil check (at a Call, Defer, Go, or MakeClosure instruction).
type NilCheckPlan struct {
	// NeedBaseCheck requests validating pointer dereference / FieldAddr bases
	// leading to the receiver value.
	NeedBaseCheck bool
	// NeedAddrCheck requests validating the final receiver address itself
	// (e.g. (*p).M or promoted value-field pointer method).
	NeedAddrCheck bool
	// Elided is true when a check would semantically apply but was proven
	// redundant by ProofKind (and optional dominating ProofAnchor).
	Elided      bool
	Pos         token.Pos
	ProofKind   NilProofKind
	ProofAnchor ssa.Instruction
}

// AllocPlan records the compiler's placement, GC scanning, and hoisting policy
// for an *ssa.Alloc instruction.
type AllocPlan struct {
	Heap            bool
	NoScan          bool
	UintptrEscapes  bool
	HoistToEntry    bool
	Dead            bool
	RecoverVolatile bool
}

// BoundsPlan records whether a slice/array/string indexing or slicing
// instruction requires runtime bounds checking.
type BoundsPlan struct {
	CheckBounds bool
	ProofAnchor ssa.Instruction
}

// IfaceBoxKind selects how a MakeInterface instruction constructs its data word.
type IfaceBoxKind uint8

const (
	IfaceBoxDefault IfaceBoxKind = iota
	IfaceBoxDirect
	IfaceBoxStatic
	IfaceBoxStack
	IfaceBoxHeap
)

// IfaceBoxPlan records boxing placement and scanning decisions for *ssa.MakeInterface.
type IfaceBoxPlan struct {
	Kind     IfaceBoxKind
	NoScan   bool
	DataOnly bool
}

// IfaceConvKind describes a planned specialization for ChangeInterface or TypeAssert.
type IfaceConvKind uint8

const (
	IfaceConvDefault IfaceConvKind = iota
	// IfaceConvKnownConcrete folds ChangeInterface(MakeInterface(x)) directly
	// to boxing x into the target interface with a static itab.
	IfaceConvKnownConcrete
	// IfaceConvStaticSuccess proves a TypeAssert always succeeds with KnownValue.
	IfaceConvStaticSuccess
	// IfaceConvStaticFail proves a TypeAssert always fails (comma-ok false or panic).
	IfaceConvStaticFail
)

// IfaceConvPlan records interface conversion or type-assertion folding decisions.
type IfaceConvPlan struct {
	Kind       IfaceConvKind
	KnownType  types.Type
	KnownValue ssa.Value
	ProofAnchor ssa.Instruction
}

// DevirtTarget records a statically resolved callee for an interface method call.
type DevirtTarget struct {
	Target        *ssa.Function
	Receiver      ssa.Value
	NeedNilCheck  bool
	WholeProgram  bool
	ProofAnchor   ssa.Instruction
}

// FunctionPlan is the read-only optimization and lowering decision table for a
// single *ssa.Function. Lowering consumes it without mutating the SSA IR.
type FunctionPlan struct {
	Fn *ssa.Function

	// NilChecks maps Call/Defer/Go/MakeClosure sites that carry pointer-method
	// receiver nil-check semantics to their lowering decision.
	NilChecks map[ssa.Instruction]NilCheckPlan
	// MethodNilDerefChecks marks *ssa.UnOp pointer loads passed as value
	// receivers to methods that must panic before load if nil.
	MethodNilDerefChecks map[*ssa.UnOp]struct{}
	// RecvNilDerefChecks marks *ssa.UnOp pointer loads inside pointer-method
	// receiver expressions whose nil dereference must be checked at the load.
	RecvNilDerefChecks map[*ssa.UnOp]token.Pos

	// Allocs maps each *ssa.Alloc in Fn to its compiler placement/scanning plan.
	Allocs map[*ssa.Alloc]AllocPlan

	// Bounds maps IndexAddr, Index, Slice, and SliceToArrayPointer sites to
	// bounds-check decisions (populated by M3 bounds pass).
	Bounds map[ssa.Instruction]BoundsPlan

	// IfaceBoxes maps MakeInterface sites to boxing decisions (M3).
	IfaceBoxes map[*ssa.MakeInterface]IfaceBoxPlan

	// IfaceConvs maps ChangeInterface and TypeAssert sites to folding decisions (M3).
	IfaceConvs map[ssa.Instruction]IfaceConvPlan

	// Devirt maps interface *ssa.Call sites to devirtualized direct targets (M3).
	Devirt map[*ssa.Call]DevirtTarget

	// SafepointBackedges records CFG backedge terminators requiring cooperative
	// safepoint polls (populated when Config.CooperativeSafepoints is true).
	SafepointBackedges map[ssa.Instruction]struct{}

	// InLoopBlock records whether block i participates in a CFG cycle (SCC).
	InLoopBlock []bool
}

// Pass populates one slice of FunctionPlan for fn without mutating fn.
type Pass interface {
	Name() string
	Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error
}

// Pipeline is an ordered sequence of pure-Go FunctionPlan passes.
type Pipeline struct {
	Passes []Pass
}

// DefaultPipeline returns the standard ssaopt pass pipeline.
func DefaultPipeline() Pipeline {
	return Pipeline{
		Passes: []Pass{
			NilCheckPass{},
			AllocPass{},
			BoundsPass{},
			IfacePass{},
			DevirtPass{},
			SafepointPass{},
		},
	}
}

// ProgramPlan caches verified FunctionPlans across a compiled *ssa.Program.
// It is safe for concurrent reads and on-demand planning across parallel
// package backend workers.
type ProgramPlan struct {
	cfg      Config
	pipeline Pipeline

	mu    sync.RWMutex
	plans map[*ssa.Function]*FunctionPlan
}

// NewProgramPlan creates a lazy ProgramPlan backed by cfg and pipeline.
func NewProgramPlan(cfg Config, pipeline Pipeline) *ProgramPlan {
	if len(pipeline.Passes) == 0 {
		pipeline = DefaultPipeline()
	}
	return &ProgramPlan{
		cfg:      cfg,
		pipeline: pipeline,
		plans:    make(map[*ssa.Function]*FunctionPlan),
	}
}

// Run plans all functions currently reachable in prog and returns a ProgramPlan.
func Run(prog *ssa.Program, cfg Config) (*ProgramPlan, error) {
	pp := NewProgramPlan(cfg, DefaultPipeline())
	if prog == nil {
		return pp, nil
	}
	for fn := range ssautil.AllFunctions(prog) {
		if fn == nil || len(fn.Blocks) == 0 {
			continue
		}
		if _, err := pp.Plan(fn); err != nil {
			return nil, err
		}
	}
	return pp, nil
}

// Function returns the cached FunctionPlan for fn, computing and caching it on
// demand if it was not planned up front (e.g., instantiated or synthetic methods).
func (pp *ProgramPlan) Function(fn *ssa.Function) *FunctionPlan {
	if pp == nil || fn == nil {
		return nil
	}
	plan, _ := pp.Plan(fn)
	return plan
}

// Plan returns the FunctionPlan for fn, running the pipeline if not yet cached.
func (pp *ProgramPlan) Plan(fn *ssa.Function) (*FunctionPlan, error) {
	if pp == nil || fn == nil {
		return nil, nil
	}
	pp.mu.RLock()
	if existing, ok := pp.plans[fn]; ok {
		pp.mu.RUnlock()
		return existing, nil
	}
	pp.mu.RUnlock()

	plan, err := pp.pipeline.RunFunction(fn, pp.cfg)
	if err != nil {
		return nil, err
	}
	pp.mu.Lock()
	if existing, ok := pp.plans[fn]; ok {
		pp.mu.Unlock()
		return existing, nil
	}
	pp.plans[fn] = plan
	pp.mu.Unlock()
	return plan, nil
}

// PlanFunction runs the default pipeline on a single function without a ProgramPlan.
func PlanFunction(fn *ssa.Function, cfg Config) (*FunctionPlan, error) {
	return DefaultPipeline().RunFunction(fn, cfg)
}

// RunFunction executes all passes in p on fn and optionally verifies the result.
func (p Pipeline) RunFunction(fn *ssa.Function, cfg Config) (*FunctionPlan, error) {
	plan := &FunctionPlan{
		Fn: fn,
	}
	if fn == nil || len(fn.Blocks) == 0 {
		return plan, nil
	}
	plan.InLoopBlock = computeLoopBlocks(fn.Blocks)
	for _, pass := range p.Passes {
		if err := pass.Run(fn, cfg, plan); err != nil {
			return nil, fmt.Errorf("ssaopt/%s: %w", pass.Name(), err)
		}
	}
	if cfg.Verify {
		if err := VerifyFunctionAndPlan(fn, plan); err != nil {
			return nil, err
		}
	}
	return plan, nil
}
