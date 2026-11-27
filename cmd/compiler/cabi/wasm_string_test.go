package cabi_test

import (
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/cmd/compiler/abi"
	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/cabi"
	"github.com/xgo-dev/llgo/cmd/compiler/heap2stack"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestWasmSparseHeaderStorageCodegen(t *testing.T) {
	runWasmHeaderConfigs(t, checkWasmSparseHeaderStorageCodegen)
}

func checkWasmSparseHeaderStorageCodegen(t *testing.T, tags string, level optlevel.Level) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Goos, conf.Goarch, conf.OptLevel = "wasip1", "wasm", level
	conf.Tags = tags
	// ModeGen skips optimization. Inspect the first lowering sequence
	// here so a repeated pass cannot hide a redundant header snapshot.
	conf.ModuleHook = func(pkg build.Package) {
		if pkg.Name != "main" {
			return
		}
		prog, mod := pkg.LPkg.Prog, pkg.LPkg.Module()
		cfg := llabi.AggregateLoweringConfig{
			GoWordSize: prog.GoWordSize(), GCRoots: prog.GCRootsEnabled(), Wasm: true,
		}
		llabi.LowerLargeAggregates(prog.TargetData(), mod, cfg)
		if level != optlevel.O0 {
			if err := heap2stack.Run(mod, prog.TargetMachine()); err != nil {
				t.Fatal(err)
			}
		}
		cabi.NewTransformer(prog, prog.Target().Spec().Triple, "", true).TransformModule(pkg.PkgPath, mod)
		llabi.LowerAggregateCopies(prog.TargetData(), mod, cfg)
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		checkWasmHeaderABICopies(t, mod, cfg.GCRoots)
	}
	pkgs, err := build.Do([]string{"./_testdata/wasmstring/strings.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	// Check snapshot semantics before LLVM may split the copies around
	// header fields. Final code generation must remain bounded below.
	receiveFn := mod.NamedFunction("ReceiveSparseArrayStorage")
	allocations, copies, volatileReads := 0, 0, 0
	for bb := receiveFn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if inst.IsACallInst().IsNil() {
				continue
			}
			name := inst.CalledValue().Name()
			switch {
			case name == llvmgen.PkgRuntime+".AllocU", name == llvmgen.PkgRuntime+".AllocUAligned":
				allocations++
			case strings.HasPrefix(name, "llvm.memcpy."), strings.HasPrefix(name, "llvm.memmove."):
				if !inst.Operand(2).IsAConstantInt().IsNil() && inst.Operand(2).ZExtValue() == 2*(250*8+16) {
					copies++
					if inst.Operand(3).ZExtValue() != 0 {
						volatileReads++
					}
				}
			}
		}
	}
	if allocations != 0 || copies != 2 || volatileReads != 1 {
		t.Errorf("array receive has %d allocations, %d copies and %d volatile reads, want 0, 2 and 1:\n%s",
			allocations, copies, volatileReads, receiveFn.String())
	}
	optimizeWasmModule(t, pkg, level)
	for _, tc := range []struct {
		name string
		size uint64
	}{{"CopySparseStorage", 4096*8 + 16}, {"CopySparseArrayStorage", 2 * (250*8 + 16)}} {
		copyFn := mod.NamedFunction(tc.name)
		if copyFn.IsNil() {
			t.Fatal("missing", tc.name)
		}
		allocations, copies := 0, 0
		var copyBlock llvm.BasicBlock
		var clear llvm.Value
		var guards [2]llvm.Value // dst, src
		var nonNilBlocks [2]llvm.BasicBlock
		for bb := copyFn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if !inst.IsACallInst().IsNil() {
					name := inst.CalledValue().Name()
					switch {
					case name == llvmgen.PkgRuntime+".AllocU", name == llvmgen.PkgRuntime+".AllocUAligned":
						allocations++
					case strings.HasPrefix(name, "__llgo_clear_header_range"):
						clear = inst
						if inst.Operand(0) != copyFn.Param(0) || copies != 1 || copyBlock != bb {
							t.Errorf("padding clear must follow the copy and only write dst: %s", inst.String())
						}
					case strings.HasPrefix(name, "llvm.memcpy."), strings.HasPrefix(name, "llvm.memmove."):
						copies++
						copyBlock = bb
						if inst.Operand(0) != copyFn.Param(0) || inst.Operand(1) != copyFn.Param(1) ||
							inst.Operand(2).IsAConstantInt().IsNil() || inst.Operand(2).ZExtValue() != tc.size {
							t.Errorf("copy must transfer the whole value directly from src to dst: %s", inst.String())
						}
					}
				}
				if inst.IsAICmpInst().IsNil() || !inst.Operand(1).IsNull() {
					continue
				}
				for i := range guards {
					branch := bb.LastInstruction()
					if inst.Operand(0) != copyFn.Param(i) || branch.IsABranchInst().IsNil() ||
						branch.SuccessorsCount() != 2 || branch.Operand(0) != inst {
						continue
					}
					switch inst.IntPredicate() {
					case llvm.IntEQ:
						guards[i], nonNilBlocks[i] = inst, branch.Successor(1)
					case llvm.IntNE:
						guards[i], nonNilBlocks[i] = inst, branch.Successor(0)
					}
				}
			}
		}
		if allocations != 0 || copies != 1 {
			t.Errorf("ordinary copy has %d allocations and %d copies, want 0 and 1:\n%s", allocations, copies, copyFn.String())
		}
		if level == optlevel.O0 && clear.IsNil() {
			t.Errorf("ordinary copy lost its padding clear:\n%s", copyFn.String())
		}
		if guards[0].IsNil() || guards[1].IsNil() {
			t.Errorf("ordinary copy lost a source or destination nil check:\n%s", copyFn.String())
		} else if nonNilBlocks[1] != guards[0].InstructionParent() || nonNilBlocks[0] != copyBlock {
			t.Errorf("copy must follow both nil checks, with src checked before dst:\n%s", copyFn.String())
		}
	}
	text := wasmAssembly(t, pkg)
	for _, name := range []string{"CopySparseStorage", "ReceiveSparseStorage", "CopySparseArrayStorage", "ReceiveSparseArrayStorage"} {
		body := wasmAssemblyBody(t, text, name)
		// An insertvalue around the aggregate load prevents copy lowering.
		// Channel reads are volatile, so even O2 cannot undo that expansion.
		if lines := strings.Count(body, "\n"); lines > 512 {
			t.Errorf("%s expanded a sparse header copy to %d assembly lines", name, lines)
		}
	}
}

// Call arguments and returns become stores only during ABI lowering. Check
// before LLVM optimization, which can hide a redundant aggregate snapshot.
func checkWasmHeaderABICopies(t *testing.T, mod llvm.Module, gc bool) {
	t.Helper()
	for _, tc := range []struct {
		name, callee string
		size         uint64
		delayed      bool
	}{
		{"PassHeaderArray", "takeHeaderArray", 256 * 16, false},
		{"PassLargeHeaderArray", "takeLargeHeaderArray", 4097 * 16, false},
		{"ReturnHeaderArray", "", 4097 * 16, false},
		{"PassHeaderArrayAfterMutation", "takeHeaderArray", 256 * 16, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := mod.NamedFunction(tc.name)
			if fn.IsNil() {
				t.Fatal("missing", tc.name)
			}
			src, dst := fn.Param(0), llvm.Value{}
			if tc.callee == "" {
				dst, src = fn.Param(0), fn.Param(1)
			}
			allocations, copies, sourceReads := 0, 0, 0
			var read, clear, mutate llvm.Value
			order := make(map[llvm.Value]int)
			writes := make(map[llvm.Value]llvm.Value)
			for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					order[inst] = len(order)
					if inst.IsACallInst().IsNil() {
						continue
					}
					name := inst.CalledValue().Name()
					switch {
					case name == llvmgen.PkgRuntime+".AllocU", name == llvmgen.PkgRuntime+".AllocUAligned":
						allocations++
					case name == tc.callee:
						dst = inst.Operand(0)
					case name == "mutateHeaderArray":
						mutate = inst
					case strings.HasPrefix(name, "__llgo_clear_header_range"):
						clear = inst
					case strings.HasPrefix(name, "llvm.memcpy."), strings.HasPrefix(name, "llvm.memmove."):
						if !inst.Operand(2).IsAConstantInt().IsNil() && inst.Operand(2).ZExtValue() == tc.size {
							copies++
							writes[inst.Operand(0)] = inst
							if inst.Operand(1) == src {
								sourceReads++
								read = inst
							}
						}
					}
				}
			}
			if sourceReads != 1 || dst.IsNil() || clear.IsNil() {
				t.Fatalf("missing one source read, ABI storage or header clear:\n%s", fn.String())
			}
			if tc.delayed {
				if mutate.IsNil() || order[read] >= order[mutate] || order[clear] >= order[mutate] ||
					read.Operand(0) == src || clear.Operand(0) != read.Operand(0) {
					t.Fatalf("intervening mutation must retain the original normalized snapshot:\n%s", fn.String())
				}
				if write := writes[dst]; dst != read.Operand(0) &&
					(write.IsNil() || write.Operand(1) != read.Operand(0)) {
					t.Fatalf("call must receive the captured value, not the mutated source:\n%s", fn.String())
				}
				return
			}
			if gc && tc.callee != "" {
				// Publishing each argument's string pointers needs one live
				// snapshot until the call; normalization must reuse that storage.
				write := writes[dst]
				if allocations != 1 || copies != 2 || read.Operand(0) == src || clear.Operand(0) != read.Operand(0) ||
					write.IsNil() || write.Operand(1) != read.Operand(0) || order[clear] <= order[read] {
					t.Fatalf("GC call must reuse one normalized snapshot; got %d allocations and %d copies:\n%s",
						allocations, copies, fn.String())
				}
				return
			}
			if allocations != 0 || copies != 1 || read.Operand(0) != dst || clear.Operand(0) != dst || order[clear] <= order[read] {
				t.Fatalf("ABI storage must be filled and normalized directly; got %d allocations and %d copies:\n%s",
					allocations, copies, fn.String())
			}
		})
	}
}

func TestWasmStringBoundaryPadding(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Goos, conf.Goarch, conf.Tags = "wasip1", "wasm", "nogc"
	pkgs, err := build.Do([]string{"./_testdata/wasmstring/strings.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	ctx := mod.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()

	// Simulate C after ABI lowering, so an outgoing Go conversion cannot
	// accidentally hide a missing incoming conversion. C may leave these
	// padding bytes dirty; the low pointer words and lengths stay intact.
	checks := []string{"LocalStringStorage", "ImportedGlobal"}
	returnCall := func(fn, arg llvm.Value) {
		call := llvm.CreateCall(b, fn.GlobalValueType(), fn, []llvm.Value{arg})
		for _, attr := range fn.GetAttributesAtIndex(1) {
			call.AddCallSiteAttribute(1, attr)
		}
		b.CreateRet(call)
	}
	constantWords := func(words []uint64) llvm.Value {
		values := make([]llvm.Value, len(words))
		for i, word := range words {
			values[i] = llvm.ConstInt(ctx.Int64Type(), word, false)
		}
		return llvm.ConstArray(ctx.Int64Type(), values)
	}
	global := mod.NamedGlobal("foreignGlobal")
	if global.IsNil() {
		t.Fatal("missing C GoString global")
	}
	global.SetGlobalConstant(true)
	layout := global.GlobalValueType()
	slot := layout.StructElementTypes()[0]
	parts := slot.StructElementTypes()
	global.SetInitializer(llvm.ConstNamedStruct(layout, []llvm.Value{
		llvm.ConstNamedStruct(slot, []llvm.Value{
			llvm.ConstIntToPtr(llvm.ConstInt(ctx.Int32Type(), 0x6000, false), parts[0]),
			llvm.ConstInt(parts[1], 0xa5a5a5a5, false),
		}),
		llvm.ConstInt(layout.StructElementTypes()[1], 19, false),
	}))
	for _, tc := range []struct {
		name             string
		words, alternate []uint64
	}{
		{"String", []uint64{0xa5a5a5a500001000, 7}, []uint64{0x3c3c3c3c00004000, 17}},
		{"Envelope",
			[]uint64{11, 0xa5a5a5a500001000, 7, 0x7e7e7e7e00005000, 3, 8, 0xdeadbeef00003000, 23},
			[]uint64{37, 0xa5a5a5a500001000, 7, 0x7e7e7e7e00005000, 3, 8, 0xdeadbeef00003000, 23}},
	} {
		value := constantWords(tc.words)
		foreign := mod.NamedFunction("foreign" + tc.name)
		alternate := mod.NamedFunction("foreign" + tc.name + "Alternate")
		exported := mod.NamedFunction("Exported" + tc.name)
		for _, entry := range []struct {
			fn   llvm.Value
			attr string
		}{{foreign, "sret"}, {alternate, "sret"}, {exported, "byval"}} {
			if entry.fn.IsNil() || len(entry.fn.Params()) != 1 {
				t.Fatalf("missing indirect %s boundary for %s", entry.attr, tc.name)
			}
			attr := entry.fn.GetEnumAttributeAtIndex(1, llvm.AttributeKindID(entry.attr))
			if attr.IsNil() || pkg.Prog.TargetData().TypeAllocSize(attr.GetTypeValue()) != uint64(8*len(tc.words)) {
				t.Fatalf("unexpected %s storage for %s:\n%s", entry.attr, tc.name, entry.fn.String())
			}
		}
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(foreign, "native"))
		b.CreateStore(value, foreign.Param(0))
		b.CreateRetVoid()
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(alternate, "native"))
		b.CreateStore(constantWords(tc.alternate), alternate.Param(0))
		b.CreateRetVoid()

		probe := llvm.AddFunction(mod, "Probe"+tc.name, llvm.FunctionType(ctx.Int1Type(), nil, false))
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(probe, "native"))
		storage := llvm.CreateAlloca(b, value.Type())
		b.CreateStore(value, storage)
		returnCall(exported, storage)
		checks = append(checks, probe.Name(), "Imported"+tc.name)

		funcval := mod.NamedFunction("Imported" + tc.name + "Funcval")
		if funcval.IsNil() {
			t.Fatalf("missing native funcval observer for %s", tc.name)
		}
		for i, suffix := range []string{"Primary", "Alternate"} {
			probe := llvm.AddFunction(mod, "Probe"+tc.name+"Funcval"+suffix, llvm.FunctionType(ctx.Int1Type(), nil, false))
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(probe, "native"))
			returnCall(funcval, llvm.ConstInt(ctx.Int1Type(), uint64(i), false))
			checks = append(checks, probe.Name())
		}
	}

	// Stub the runtime context calls so LLVM can evaluate the header values
	// without linking a Wasm runtime.
	for _, name := range []string{"EnterLocalContext", "LeaveLocalContext"} {
		fn := mod.NamedFunction(llvmgen.PkgRuntime + "." + name)
		if !fn.IsNil() && fn.IsDeclaration() {
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "stub"))
			if fn.GlobalValueType().ReturnType().TypeKind() == llvm.VoidTypeKind {
				b.CreateRetVoid()
			} else {
				b.CreateRet(llvm.ConstNull(fn.GlobalValueType().ReturnType()))
			}
		}
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !fn.IsDeclaration() {
			fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("alwaysinline"), 0))
		}
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	pbo := llvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	pbo.SetVerifyEach(true)
	// Inline the observers and fold unused header projections before escape
	// analysis, so it can promote the address-taken aggregates in the fixture.
	if err := mod.RunPasses("always-inline,function(instcombine)", pkg.Prog.TargetMachine(), pbo); err != nil {
		t.Fatal(err)
	}
	if err := heap2stack.Run(mod, pkg.Prog.TargetMachine()); err != nil {
		t.Fatal(err)
	}
	if err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), pbo); err != nil {
		t.Fatal(err)
	}
	// Check behavior rather than a particular insertvalue/mask sequence.
	// Both native directions must preserve every data field while clearing
	// only string and slice padding. Typed loads leave the source bytes intact.
	checks = append(checks, "CheckSparseStorage", "CheckSparseArrayStorage")
	for _, name := range checks {
		fn := mod.NamedFunction(name)
		if fn.IsNil() {
			t.Errorf("%s disappeared after optimization", name)
			continue
		}
		returns := 0
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			if ret := bb.LastInstruction(); !ret.IsNil() && !ret.IsAReturnInst().IsNil() {
				returns++
				if value := ret.Operand(0); value.IsAConstantInt().IsNil() || value.ZExtValue() != 1 {
					t.Errorf("%s did not preserve the expected string value:\n%s", name, fn.String())
				}
			}
		}
		if returns == 0 {
			t.Errorf("%s has no return after optimization:\n%s", name, fn.String())
		}
	}
}

func runWasmHeaderConfigs(t *testing.T, run func(*testing.T, string, optlevel.Level)) {
	t.Helper()
	for _, gc := range []struct{ name, tags string }{{"gc", ""}, {"nogc", "nogc"}} {
		for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
			t.Run(gc.name+"/"+level.String(), func(t *testing.T) {
				run(t, gc.tags, level)
			})
		}
	}
}

func optimizeWasmModule(t *testing.T, pkg llvmgen.Package, level optlevel.Level) {
	t.Helper()
	opts := llvm.NewPassBuilderOptions()
	defer opts.Dispose()
	opts.SetVerifyEach(true)
	if err := pkg.Module().RunPasses("default<"+level.String()+">", pkg.Prog.TargetMachine(), opts); err != nil {
		t.Fatal(err)
	}
}

func wasmAssembly(t *testing.T, pkg llvmgen.Package) string {
	t.Helper()
	buf, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.AssemblyFile)
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Dispose()
	return string(buf.Bytes())
}

func wasmAssemblyBody(t *testing.T, text, name string) string {
	t.Helper()
	_, rest, found := strings.Cut(text, "\n"+name+":")
	if !found {
		t.Fatalf("missing %s assembly", name)
	}
	body, _, found := strings.Cut(rest, "end_function")
	if !found {
		t.Fatalf("missing end of %s", name)
	}
	return body
}
