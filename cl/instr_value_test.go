package cl

import (
	"fmt"
	"go/ast"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// Both target families use the same addressable entry and normal call lowering.
func TestInstructionFunctionValues(t *testing.T) {
	for _, tc := range []struct {
		name, target, signature, operation string
	}{
		{"funnel", "llvm.fshl", "(a,b,k uint8) uint8", "@llvm.fshl.i8("},
		{"population", "llvm.ctpop.i32", "(x uint32) uint32", "@llvm.ctpop.i32("},
		{"root", "llvm.sqrt", "(x float64) float64", "@llvm.sqrt.f64("},
		{"overflow", "llvm.umul.with.overflow", "(a,b uint64) (uint64,bool)", "@llvm.umul.with.overflow.i64("},
		{"legacy-overflow", "llgo.umulOverflow", "(a,b uint64) (uint64,bool)", "@llvm.umul.with.overflow.i64("},
		{"legacy-bits", "llgo.float64Bits", "(x float64) uint64", "bitcast double"},
		{"legacy-bool", "llgo.boolToUint8", "(x bool) uint8", "select i1"},
		{"legacy-void", "llgo.skip", "(x int)", "ret void"},
	} {
		for _, bodyless := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/bodyless=%v", tc.name, bodyless), func(t *testing.T) {
				directive := "//llgo:link operation " + tc.target
				body := ` { panic("binding body must be ignored") }`
				if bodyless {
					directive = "//go:linkname operation " + tc.target
					body = ""
				}
				src := "package entry\nimport _ \"unsafe\"\n" + directive + "\nfunc operation" + tc.signature + body + `
var Value = operation
var Again = operation
`
				ssapkg, _, files := buildGoSSAPkg(t, src)
				originalName := funcName(ssapkg.Pkg, ssapkg.Func("operation"), false)
				for _, target := range []llssa.Target{
					{GOOS: "linux", GOARCH: "amd64"},
					{GOOS: "linux", GOARCH: "arm64"},
					{GOOS: "windows", GOARCH: "386"},
					{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
					{GOOS: "linux", GOARCH: "riscv64"},
					{GOOS: "wasip1", GOARCH: "wasm"},
					{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"},
				} {
					t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
						prog := newLLSSAProgForTarget(t, &target)
						defer prog.Dispose()
						prog.EnableFuncInfoMetadata(true)
						pkg, err := NewPackage(prog, ssapkg, files)
						if err != nil {
							t.Fatal(err)
						}
						mod := pkg.Module()
						entry := mustNamedFunction(t, mod, originalName)
						if entry.Linkage() != llvm.LinkOnceAnyLinkage || !strings.Contains(entry.String(), tc.operation) {
							t.Fatalf("invalid original-symbol entry:\n%s", entry.String())
						}
						if target.GOOS == "windows" && !strings.Contains(entry.String(), "comdat") {
							t.Fatal("COFF entry lacks COMDAT")
						}
						init := mustNamedFunction(t, mod, "entry.init").String()
						descriptor := constantFunctionDescriptor(t, mod, entry)
						stores := 0
						for _, block := range mustNamedFunction(t, mod, "entry.init").BasicBlocks() {
							for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
								if inst.InstructionOpcode() == llvm.Store && uncastFunctionValue(inst.Operand(0)) == descriptor {
									stores++
								}
							}
						}
						if stores != 2 {
							t.Fatalf("function values did not share the original entry:\n%s", init)
						}
						mod.SetDataLayout(prog.DataLayout())
						mod.SetTarget(target.Spec().Triple)
						if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
							t.Fatal(err)
						}
						opts := llvm.NewPassBuilderOptions()
						defer opts.Dispose()
						opts.SetVerifyEach(true)
						if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
							t.Fatal(err)
						}
						asm, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
						if err != nil {
							t.Fatal(err)
						}
						asm.Dispose()
					})
				}
			})
		}
	}
}

func TestInstructionFunctionValuesAcrossPackages(t *testing.T) {
	for _, targetName := range []string{"llgo.umulOverflow", "llvm.umul.with.overflow"} {
		t.Run(targetName, func(t *testing.T) {
			// Alias predeclares the original symbol before taking its function
			// value, as happens when SIMD method expressions and aliases mix.
			dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
import _ "unsafe"
//go:linkname Multiply `+targetName+`
func Multiply(a,b uint64)(uint64,bool)
var Value = Multiply
`, "example.com/root", `package root
import "example.com/dep"
import _ "unsafe"
//go:linkname Alias example.com/dep.Multiply
func Alias(a,b uint64)(uint64,bool)
var Value = dep.Multiply
var Again = dep.Multiply
func Direct(a,b uint64)(uint64,bool){return dep.Multiply(a,b)}
`)
			files := []*ast.File{{Name: ast.NewIdent("dep"), Decls: []ast.Decl{dep.Func("Multiply").Syntax().(*ast.FuncDecl)}}}
			rootFiles := []*ast.File{{Name: ast.NewIdent("root"), Decls: []ast.Decl{root.Func("Alias").Syntax().(*ast.FuncDecl)}}}
			originalName := funcName(dep.Pkg, dep.Func("Multiply"), false)
			for _, target := range []llssa.Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "windows", GOARCH: "386"}} {
				t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
					prog := newLLSSAProgForTarget(t, &target)
					defer prog.Dispose()
					owner, err := NewPackage(prog, dep, files)
					if err != nil {
						t.Fatal(err)
					}
					consumer, err := NewPackage(prog, root, rootFiles)
					if err != nil {
						t.Fatal(err)
					}
					for _, pkg := range []llssa.Package{owner, consumer} {
						entry := mustNamedFunction(t, pkg.Module(), originalName)
						if entry.IsDeclaration() || entry.Linkage() != llvm.LinkOnceAnyLinkage {
							t.Fatalf("cross-package entry does not keep its original name/linkage:\n%s", entry.String())
						}
						if target.GOOS == "windows" && !strings.Contains(entry.String(), "comdat") {
							t.Fatal("imported COFF entry lacks COMDAT")
						}
						if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
							t.Fatal(err)
						}
					}
					direct := mustNamedFunction(t, consumer.Module(), "example.com/root.Direct").String()
					if !strings.Contains(direct, "@llvm.umul.with.overflow.i64(") || strings.Contains(direct, "call { i64, i1 } @\"example.com/dep.Multiply\"") {
						t.Fatalf("direct binding call did not lower to its operation:\n%s", direct)
					}
					// LLVM consumes the source module; verify canonical linkonce
					// entries from distinct packages coalesce without renaming.
					if err := llvm.LinkModules(owner.Module(), consumer.Module()); err != nil {
						t.Fatal(err)
					}
					if entry := mustNamedFunction(t, owner.Module(), originalName); entry.IsDeclaration() || entry.Linkage() != llvm.LinkOnceAnyLinkage {
						t.Fatal("linked entry did not retain the original name and linkage")
					}
					if err := llvm.VerifyModule(owner.Module(), llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

// Resolve the descriptor and inspect its code field rather than assuming that
// the addressable function entry is stored directly in a Go function value.
func constantFunctionDescriptor(t *testing.T, mod llvm.Module, entry llvm.Value) llvm.Value {
	t.Helper()
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		if !strings.HasPrefix(global.Name(), "_llgo_funcval$") || global.IsDeclaration() {
			continue
		}
		init := global.Initializer()
		if init.OperandsCount() < 2 {
			continue
		}
		code := init.Operand(0)
		// Go64 stores a physical Wasm32 pointer in a padded Go word.
		if code.Type().TypeKind() == llvm.StructTypeKind && code.OperandsCount() > 0 {
			code = code.Operand(0)
		}
		if uncastFunctionValue(code) == entry {
			return global
		}
	}
	t.Fatalf("missing canonical descriptor for %s:\n%s", entry.Name(), mod.String())
	return llvm.Value{}
}

func uncastFunctionValue(value llvm.Value) llvm.Value {
	for {
		var opcode llvm.Opcode
		switch {
		case !value.IsAConstantExpr().IsNil():
			opcode = value.Opcode()
		case !value.IsAInstruction().IsNil():
			opcode = value.InstructionOpcode()
		default:
			return value
		}
		switch opcode {
		case llvm.BitCast, llvm.PtrToInt, llvm.IntToPtr, llvm.ZExt:
			value = value.Operand(0)
		default:
			return value
		}
	}
}
