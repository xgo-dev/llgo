package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"regexp"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func runtimeContractSignature(params []types.Type, result types.Type) *types.Signature {
	vars := make([]*types.Var, len(params))
	for i, typ := range params {
		vars[i] = types.NewParam(token.NoPos, nil, "", typ)
	}
	var results *types.Tuple
	if result != nil {
		results = types.NewTuple(types.NewParam(token.NoPos, nil, "", result))
	}
	return types.NewSignatureType(nil, nil, nil, types.NewTuple(vars...), results, false)
}

func TestRuntimeContracts(t *testing.T) {
	tests := []struct {
		name   string
		want   []string
		absent []string
	}{
		{"AssertNilDerefPtr", []string{"declare nonnull ptr", "ptr returned"}, []string{"ptr nonnull", "memory(", "noreturn", "willreturn", "nounwind"}},
		{"CStrCopy", []string{"ptr returned writeonly captures(ret: address, provenance)", "memory(read, argmem: readwrite)"}, []string{"noalias", "nonnull"}},
		{"memequal", []string{"ptr readonly captures(none)", "memory(argmem: read)"}, []string{"nonnull", "noalias"}},
		{"StringEqual", []string{"memory(read)"}, []string{"memory(argmem:"}},
		{"StringLess", []string{"memory(read)"}, nil},
		{"Typedmemmove", []string{"ptr readonly captures(none), ptr writeonly captures(none), ptr readonly captures(none)", "memory(argmem: readwrite)"}, []string{"nonnull", "noalias"}},
		{"Typedmemclr", []string{"ptr readonly captures(none), ptr writeonly captures(none)", "memory(argmem: readwrite)"}, []string{"nonnull"}},
		{"MapLen", []string{"range(i%d 0, %s)", "ptr readonly captures(none)", "memory(argmem: read)"}, []string{"nonnull"}},
		{"ChanCap", []string{"range(i%d 0, %s)", "memory(argmem: read)"}, []string{"nonnull"}},
		{"Memhash", []string{"ptr readonly captures(none)", "memory(read)"}, []string{"memory(argmem:"}},
		{"Memhash32", []string{"memory(read)"}, nil},
		{"Memhash64", []string{"memory(read)"}, nil},
	}
	for _, target := range []Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm64"}} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					// Exercise definitions and declarations using the runtime source signature.
					for _, definition := range []bool{false, true} {
						pkg := prog.NewPackage(fmt.Sprint(test.name, definition), "test/contracts")
						var sig *types.Signature
						if test.name == "memequal" {
							// This private linkname entry is absent from export data.
							sig = runtimeContractSignature([]types.Type{types.Typ[types.UnsafePointer], types.Typ[types.UnsafePointer], types.Typ[types.Uintptr]}, types.Typ[types.Bool])
						} else {
							sig = prog.runtime().Scope().Lookup(test.name).Type().(*types.Signature)
						}
						fn := pkg.NewFunc(PkgRuntime+"."+test.name, sig, InGo)
						if definition {
							b := prog.ctx.NewBuilder()
							b.SetInsertPointAtEnd(prog.ctx.AddBasicBlock(fn.impl, "entry"))
							b.CreateUnreachable()
							b.Dispose()
						}
						ir := pkg.String()
						ir = regexp.MustCompile(` %[0-9]+`).ReplaceAllString(ir, "")
						for _, want := range test.want {
							if strings.Contains(want, "%d") {
								bits := prog.Int().ll.IntTypeWidth()
								upper := "-9223372036854775808"
								if bits == 32 {
									upper = "-2147483648"
								}
								want = fmt.Sprintf(want, bits, upper)
							}
							if definition {
								want = strings.ReplaceAll(want, "declare ", "define ")
							}
							if !strings.Contains(ir, want) {
								t.Errorf("missing %q:\n%s", want, ir)
							}
						}
						for _, absent := range test.absent {
							if strings.Contains(ir, absent) {
								t.Errorf("unexpected %q:\n%s", absent, ir)
							}
						}
						if test.name != "AssertNilDerefPtr" {
							for _, want := range []string{"nofree", "nosync", "nounwind", "willreturn"} {
								if !strings.Contains(ir, want) {
									t.Errorf("missing %s:\n%s", want, ir)
								}
							}
						}
						if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
							t.Fatal(err)
						}
					}
				})
			}
		})
	}
}

func TestRuntimePanicContractsAndExclusions(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	for _, name := range []string{"Panic", "PanicErrorString", "PanicIndex", "PanicIndexU", "PanicSliceConvert", "PanicTypeAssert", "PanicTypeAssertionError", "PanicExtendIndex", "PanicExtendIndexU", "Rethrow", "throw", "AssertIndex", "PanicWrapNilPointer", "ChanLen", "Recover", "Implements", "AllocU", "AllocZ", "AllocRoot", "NewItab"} {
		pkg := prog.NewPackage(name, "test/panic")
		fn := pkg.NewFunc(PkgRuntime+"."+name, NoArgsNoRet, InGo)
		panicEntry := strings.HasPrefix(name, "Panic") && name != "PanicWrapNilPointer"
		for _, attr := range []string{"cold", "noreturn"} {
			if got := !fn.impl.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(attr)).IsNil(); got != panicEntry {
				t.Errorf("%s %s = %v, want %v", name, attr, got, panicEntry)
			}
		}
		for _, attr := range []string{"memory", "nounwind", "nofree", "nosync", "willreturn"} {
			if !fn.impl.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(attr)).IsNil() {
				t.Errorf("%s unexpectedly has %s", name, attr)
			}
		}
	}
	// A matching short name in another package must not inherit runtime contracts.
	pkg := prog.NewPackage("ordinary", "example.com/ordinary")
	fn := pkg.NewFunc("example.com/ordinary.Panic", NoArgsNoRet, InGo)
	if !fn.impl.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("noreturn")).IsNil() {
		t.Fatal("ordinary Panic got noreturn")
	}
}

func TestRuntimeResolvedLinknameContract(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("linkname", "test/linkname")
	pkg.SetResolveLinkname(func(name string) string {
		if name == PkgRuntime+".Typedmemmove" {
			return "reflect.typedmemmove"
		}
		return name
	})
	fn := pkg.RuntimeFunc("Typedmemmove").impl
	if fn.Name() != "reflect.typedmemmove" {
		t.Fatal(fn.Name())
	}
	for i, access := range []string{"readonly", "writeonly", "readonly"} {
		if fn.GetEnumAttributeAtIndex(i+1, llvm.AttributeKindID(access)).IsNil() {
			t.Fatalf("linkname lost parameter %d %s", i, access)
		}
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeContractOptimizations(t *testing.T) {
	for _, target := range []Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm64"}} {
		for _, pipeline := range []string{"default<O2>", "lto<O2>"} {
			t.Run(target.GOARCH+"/"+pipeline, func(t *testing.T) {
				prog := NewProgram(&target)
				defer prog.Dispose()
				setTestRuntime(t, prog)
				pkg := prog.NewPackage("opt", "test/opt")
				ptr, integer := types.Typ[types.UnsafePointer], types.Typ[types.Int]
				checked := pkg.NewFunc(PkgRuntime+".AssertNilDerefPtr", runtimeContractSignature([]types.Type{ptr}, ptr), InGo).impl
				length := pkg.NewFunc(PkgRuntime+".MapLen", runtimeContractSignature([]types.Type{ptr}, integer), InGo).impl
				equal := pkg.NewFunc(PkgRuntime+".memequal", runtimeContractSignature([]types.Type{ptr, ptr, integer}, types.Typ[types.Bool]), InGo).impl
				hash := pkg.NewFunc(PkgRuntime+".Memhash64", runtimeContractSignature([]types.Type{ptr, integer}, integer), InGo).impl
				copystr := pkg.RuntimeFunc("CStrCopy").impl
				global := llvm.AddGlobal(pkg.Module(), prog.Int().ll, "hashkey")
				b := prog.ctx.NewBuilder()
				defer b.Dispose()
				for _, name := range []string{"checked", "negative", "copy", "repeat", "write", "globalwrite"} {
					fn := llvm.AddFunction(pkg.Module(), name, llvm.FunctionType(prog.ctx.Int1Type(), []llvm.Type{prog.VoidPtr().ll, prog.VoidPtr().ll}, false))
					b.SetInsertPointAtEnd(prog.ctx.AddBasicBlock(fn, "entry"))
					p, q := fn.Param(0), fn.Param(1)
					call := func(f llvm.Value, args ...llvm.Value) llvm.Value {
						return b.CreateCall(f.GlobalValueType(), f, args, "")
					}
					switch name {
					case "checked":
						r := call(checked, p)
						b.CreateRet(b.CreateICmp(llvm.IntEQ, r, llvm.ConstNull(r.Type()), ""))
					case "negative":
						r := call(length, p)
						b.CreateRet(b.CreateICmp(llvm.IntSLT, r, llvm.ConstNull(r.Type()), ""))
					case "copy":
						r := call(copystr, p, llvm.ConstNull(copystr.GlobalValueType().ParamTypes()[1]))
						b.CreateRet(b.CreateICmp(llvm.IntEQ, r, p, ""))
					case "repeat", "write":
						args := []llvm.Value{p, q, llvm.ConstInt(prog.Int().ll, 1, false)}
						x := call(equal, args...)
						if name == "write" {
							b.CreateStore(llvm.ConstInt(prog.ctx.Int8Type(), 42, false), p)
						}
						y := call(equal, args...)
						b.CreateRet(b.CreateXor(x, y, ""))
					case "globalwrite":
						seed := llvm.ConstInt(prog.Int().ll, 1, false)
						x := call(hash, p, seed)
						b.CreateStore(seed, global)
						y := call(hash, p, seed)
						b.CreateRet(b.CreateICmp(llvm.IntEQ, x, y, ""))
					}
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				pbo := llvm.NewPassBuilderOptions()
				defer pbo.Dispose()
				if err := pkg.Module().RunPasses(pipeline, prog.TargetMachine(), pbo); err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"checked", "negative", "repeat"} {
					ir := pkg.Module().NamedFunction(name).String()
					if !strings.Contains(ir, "ret i1 false") {
						t.Errorf("%s failed to fold:\n%s", name, ir)
					}
					if name == "checked" && !strings.Contains(ir, "call ptr @") {
						t.Errorf("nil check call was removed:\n%s", ir)
					}
				}
				for _, name := range []string{"write", "globalwrite"} {
					ir := pkg.Module().NamedFunction(name).String()
					if strings.Count(ir, " call ") != 2 {
						t.Errorf("%s lost memory invalidation:\n%s", name, ir)
					}
				}
				if ir := pkg.Module().NamedFunction("copy").String(); !strings.Contains(ir, "ret i1 true") || !strings.Contains(ir, "call ptr @") {
					t.Errorf("copy must retain the call and fold its returned identity:\n%s", ir)
				}
				asm, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				asm.Dispose()
			})
		}
	}
}
