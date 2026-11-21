package cabi_test

import (
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/heap2stack"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWasmSparseHeaderStorageCodegen(t *testing.T) {
	for _, gc := range []struct{ name, tags string }{{"gc", ""}, {"nogc", "nogc"}} {
		for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
			t.Run(gc.name+"/"+level.String(), func(t *testing.T) {
				conf := build.NewDefaultConf(build.ModeGen)
				conf.Goos, conf.Goarch, conf.OptLevel = "wasip1", "wasm", level
				conf.Tags = gc.tags
				if level != optlevel.O0 {
					// ModeGen skips optimization. Run the allocation pass at its real
					// position, before C ABI and copy lowering, rather than letting a
					// later heap2stack pass hide a redundant header snapshot.
					conf.ModuleHook = func(pkg build.Package) {
						if pkg.Name != "main" {
							return
						}
						prog, mod := pkg.LPkg.Prog, pkg.LPkg.Module()
						llabi.LowerLargeAggregates(prog.TargetData(), mod, llabi.AggregateLoweringConfig{
							GoWordSize: prog.GoWordSize(), GCRoots: prog.GCRootsEnabled(), Wasm: true,
						})
						if err := heap2stack.Run(mod, prog.TargetMachine()); err != nil {
							t.Fatal(err)
						}
					}
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
						case name == ssa.PkgRuntime+".AllocU", name == ssa.PkgRuntime+".AllocUAligned":
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
				options := llvm.NewPassBuilderOptions()
				defer options.Dispose()
				options.SetVerifyEach(true)
				if err := mod.RunPasses("default<"+level.String()+">", pkg.Prog.TargetMachine(), options); err != nil {
					t.Fatal(err)
				}
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
								case name == ssa.PkgRuntime+".AllocU", name == ssa.PkgRuntime+".AllocUAligned":
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
				asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				defer asm.Dispose()
				text := string(asm.Bytes())
				for _, name := range []string{"CopySparseStorage", "ReceiveSparseStorage", "CopySparseArrayStorage", "ReceiveSparseArrayStorage"} {
					start := strings.Index(text, name+":")
					if start < 0 {
						t.Fatalf("missing %s", name)
					}
					body := text[start:]
					end := strings.Index(body, "end_function")
					if end < 0 {
						t.Fatalf("missing end of %s", name)
					}
					// An insertvalue around the aggregate load prevents copy lowering.
					// Channel reads are volatile, so even O2 cannot undo that expansion.
					if lines := strings.Count(body[:end], "\n"); lines > 512 {
						t.Errorf("%s expanded a sparse header copy to %d assembly lines", name, lines)
					}
				}
			})
		}
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
	// Exported entries install a runtime-local context. Its behavior is
	// independent of string representation; provide inert bodies so LLVM can
	// evaluate the value checks without linking or executing a Wasm runtime.
	for _, name := range []string{"EnterLocalContext", "LeaveLocalContext"} {
		fn := mod.NamedFunction(ssa.PkgRuntime + "." + name)
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
