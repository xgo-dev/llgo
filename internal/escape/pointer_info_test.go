//go:build !llgo

package escape

import (
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestPotentialCopiesOfRoot(t *testing.T) {
	input := filepath.Join("testdata", "pointer-info", "in.txt")
	buffer, err := llvm.NewMemoryBufferFromFile(input)
	if err != nil {
		t.Fatal(err)
	}
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod, err := ctx.ParseIR(buffer)
	if err != nil {
		t.Fatalf("parse %s: %v", input, err)
	}
	defer mod.Dispose()

	analysis := newCopyAnalysis(mod)
	defer analysis.dispose()
	tests := []struct {
		function   string
		wantOK     bool
		wantCopies []string
	}{
		// Exact memory matching and conservative failures.
		{function: "exact_field", wantOK: true, wantCopies: []string{"q"}},
		{function: "disjoint_field", wantOK: true},
		{function: "overlapping_whole_read"},
		{function: "overwritten_before_read", wantOK: true},
		{function: "unknown_offset"},
		{function: "undef_destination", wantOK: true},
		{function: "null_destination", wantOK: true},
		{function: "null_pointer_valid"},
		{function: "unsupported_destination"},
		{function: "invalid_pointer_info"},
		// Address derivation through aggregate fields, casts, merges, and returned aliases.
		{function: "direct_slot", wantOK: true, wantCopies: []string{"q"}},
		{function: "array_element", wantOK: true, wantCopies: []string{"q"}},
		{function: "vector_element", wantOK: true, wantCopies: []string{"q"}},
		{function: "nested_field", wantOK: true, wantCopies: []string{"q"}},
		{function: "bitcast_destination", wantOK: true, wantCopies: []string{"q"}},
		{function: "addrspacecast_destination", wantOK: true, wantCopies: []string{"q"}},
		{function: "selected_destination", wantOK: true, wantCopies: []string{"q1", "q2"}},
		{function: "phi_destination", wantOK: true, wantCopies: []string{"q1", "q2"}},
		{function: "loop_invariant_phi", wantOK: true, wantCopies: []string{"q"}},
		{function: "loop_variant_phi"},
		{function: "frozen_reader"},
		{function: "returned_alias"},
		{function: "returned_internal_alias", wantOK: true, wantCopies: []string{"q"}},
		{function: "returned_address_taken_alias"},
		{function: "nested_object_copy", wantOK: true, wantCopies: []string{"q"}},
		// Supported stack, global, thread-local, and noalias allocation objects.
		{function: "internal_global_copy", wantOK: true, wantCopies: []string{"q"}},
		{function: "private_global_copy", wantOK: true, wantCopies: []string{"q"}},
		{function: "noalias_object_copy", wantOK: true, wantCopies: []string{"q"}},
		{function: "noalias_object_field_copy", wantOK: true, wantCopies: []string{"q"}},
		{function: "callsite_noalias_object_copy", wantOK: true, wantCopies: []string{"q"}},
		{function: "read_before_store", wantOK: true, wantCopies: []string{"before"}},
		{function: "read_before_store_norecurse", wantOK: true},
		{function: "global_read_before_store", wantOK: true, wantCopies: []string{"before"}},
		{function: "thread_local_read_before_store", wantOK: true},
		// Interprocedural and modeled memory-access behavior.
		{function: "defined_callee_reader", wantOK: true, wantCopies: []string{"callee_q"}},
		{function: "defined_callee_field_reader", wantOK: true, wantCopies: []string{"callee_q"}},
		{function: "defined_callee_finite_offset"},
		{function: "defined_callee_unknown_offset"},
		// TODO: LLGo currently emits calls, not invokes. Enable this when invoke enters generated IR.
		// {function: "invoke_callee_reader", wantOK: true, wantCopies: []string{"callee_q"}},
		{function: "lifetime_ignored", wantOK: true, wantCopies: []string{"q"}},
		{function: "memcpy_reader"},
		{function: "memmove_reader"},
		{function: "memset_overwrite", wantOK: true},
		{function: "callee_readnone", wantOK: true, wantCopies: []string{"q"}},
		{function: "callsite_readnone", wantOK: true, wantCopies: []string{"q"}},
		{function: "callee_readonly"},
		{function: "callee_readwrite"},
		{function: "callee_capture"},
		{function: "invalid_defined_callee"},
		{function: "variadic_callee"},
		{function: "atomicrmw_reader"},
		{function: "cmpxchg_reader"},
		{function: "vaarg_ignored"},
		{function: "finite_select_offset"},
		{function: "finite_phi_offset"},
		// Offset sets and interprocedural patterns adapted from LLVM 19 PointerInfo tests.
		{function: "chained_constant_offsets", wantOK: true, wantCopies: []string{"q"}},
		{function: "chained_finite_offsets"},
		{function: "nested_finite_offsets"},
		{function: "select_partial_overlap", wantOK: true, wantCopies: []string{"q"}},
		{function: "phi_partial_overlap", wantOK: true, wantCopies: []string{"q"}},
		{function: "same_callee_loads", wantOK: true, wantCopies: []string{"q1", "q2"}},
		{function: "different_callee_loads", wantOK: true, wantCopies: []string{"q0"}},
		{function: "constant_callsite_index"},
		{function: "arithmetic_index"},
		{function: "loaded_index"},
		{function: "constant_expr_destination"},
		{function: "byval_write_isolated", wantOK: true, wantCopies: []string{"q"}},
	}
	for _, test := range tests {
		t.Run(test.function, func(t *testing.T) {
			fn := mod.NamedFunction(test.function)
			if fn.IsNil() {
				t.Fatalf("function %s not found", test.function)
			}
			var root llvm.Value
			for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
				for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
					if instr.Name() == "root" {
						root = instr
					}
				}
			}
			if root.IsNil() {
				t.Fatalf("root in %s not found", test.function)
			}

			ok := true
			var copies []llvm.Value
			stores := 0
			for use := root.FirstUse(); !use.IsNil(); use = use.NextUse() {
				user := use.User()
				if user.InstructionOpcode() != llvm.Store || user.Operand(0) != root {
					continue
				}
				stores++
				storeCopies, complete := analysis.getPotentialCopiesOfStoredValue(user)
				if !complete {
					ok = false
					break
				}
				copies = append(copies, storeCopies...)
			}
			if stores == 0 {
				t.Fatalf("root %s has no stored-value use", root.Name())
			}
			if ok != test.wantOK {
				t.Fatalf("potentialCopies(%s) ok = %v, want %v", root.Name(), ok, test.wantOK)
			}
			got := make([]string, 0, len(copies))
			for _, copy := range copies {
				got = append(got, copy.Name())
			}
			sort.Strings(got)
			if !slices.Equal(got, test.wantCopies) {
				t.Fatalf("potentialCopies(%s) = %v, want %v", root.Name(), got, test.wantCopies)
			}
		})
	}
}
