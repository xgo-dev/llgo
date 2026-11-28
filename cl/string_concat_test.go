package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/internal/filecheck"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestStringConcatLowering(t *testing.T) {
	ir := compileInterfaceBoxIR(t, `package concatlower
func fused(a, b, c string) string { return a + b + c }
type named string
func namedFused(a, b, c named) named { return a + b + c }
type alias = named
func aliasFused(a, b, c alias) alias { return a + b + c }
func balanced(a, b, c, d string) string { return (a + b) + (c + d) }
func emptyNamed(s named) named { return "" + s + "" }
func emptyLocal(s string) int { return len("" + s + "...") }
func emptyLong(s string) int { return len("" + s + "0123456789abcdef0123456789abcdef!") }
func emptyMiddle(a,b,c string) string { return a + "" + b + c }
func local(a, b, c string) int { return len(a + b + c) }
func exactBuffer(s string) int { return len("0123456789abcdef" + s + "0123456789abcdef") }
func exceedsBuffer(s string) int { return len("0123456789abcdef" + s + "0123456789abcdef!") }
func sharedRoot(a, b, c string) (string, string) { x := a + b + c; return x, x }
func shared(a, b, c string) string { x := a + b; sink(x); return x + c }
func sharedLeaf(a, b, c, d string) string { x := a+b; sink(x); return x+c+d }
func sharedChains(a, b, c, d, e string) string { x := a+b+c; sink(x); return x+d+e }
func crossBlockChains(a, b, c, d, e string, branch bool) string { x := a+b+c; if branch { sink(d) }; return x+d+e }
func dominant(a, b, c string, branch bool) string { x := id(a); if branch { sink(b) }; return x+b+c }
func interleaved(a, b, c string) string { return a+b+id(c) }
func oversized(a string) string { return a+a+a+a+a+a+a+a+a+a+a+a+a+a+a+a+a }
func id(s string) string { return s }
func sink(string)
`, ssa.InstantiateGenerics, nil, false)
	for _, tc := range []struct {
		name           string
		concat, binary int
	}{
		{"fused", 1, 0}, {"namedFused", 1, 0}, {"aliasFused", 1, 0},
		{"balanced", 1, 0}, {"local", 1, 0}, {"sharedRoot", 1, 0},
		{"emptyNamed", 0, 1}, {"emptyLocal", 0, 1}, {"emptyLong", 0, 1}, {"emptyMiddle", 1, 0},
		{"exactBuffer", 1, 0}, {"exceedsBuffer", 1, 0},
		{"sharedLeaf", 1, 1}, {"sharedChains", 2, 0}, {"crossBlockChains", 2, 0},
		{"dominant", 1, 0}, {"shared", 0, 2}, {"interleaved", 0, 2}, {"oversized", 1, 1},
	} {
		body := llvmFunction(t, ir, "concatlower."+tc.name)
		concat := strings.Count(body, "runtime/internal/runtime.StringConcat")
		binary := strings.Count(body, "runtime/internal/runtime.StringCat")
		if concat != tc.concat || binary != tc.binary {
			t.Fatalf("%s: concat/binary calls = %d/%d, want %d/%d:\n%s", tc.name, concat, binary, tc.concat, tc.binary, body)
		}
	}
	for name, want := range map[string]bool{"fused": false, "local": true, "exactBuffer": true, "exceedsBuffer": false, "emptyLocal": true, "emptyLong": false} {
		if got := strings.Contains(llvmFunction(t, ir, "concatlower."+name), "alloca [32 x i8]"); got != want {
			t.Errorf("%s: result stack buffer = %t, want %t", name, got, want)
		}
	}
}

func TestStringConcatLoweringWithRoots(t *testing.T) {
	ir := compileInterfaceBoxIR(t, `package concatroots
func fused(a, b, c string) string { return a + b + c }
`, ssa.InstantiateGenerics, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"}, true)
	// W32 has four-byte pointers but 16-byte strings. Bind all three data
	// addresses to the indirect tail of the compact root map (high bit plus 3)
	// and track each padding-normalized header
	// from its argument through the ordered table stores to the call and release.
	checks := `CHECK: [[STR:%".*String"]] = type { { ptr, i32 }, i64 }
CHECK: @"concatroots.fused$gcmap" = {{.*}}constant { i32, i32 } { i32 6, i32 -2147483645 }
CHECK-LABEL: @concatroots.fused(
CHECK-SAME: [[STR]] [[A:%[0-9]+]], [[STR]] [[B:%[0-9]+]], [[STR]] [[C:%[0-9]+]])
CHECK: [[TABLE:%.*]] = alloca [3 x [[STR]]]
CHECK: [[ROOT0:%.*]] = getelementptr inbounds [6 x ptr], ptr [[ROOTS:%.*]], i32 0, i32 3
CHECK: [[ROOT1:%.*]] = getelementptr inbounds [6 x ptr], ptr [[ROOTS]], i32 0, i32 4
CHECK: [[ROOT2:%.*]] = getelementptr inbounds [6 x ptr], ptr [[ROOTS]], i32 0, i32 5
CHECK: call void @llvm.memset.p0.i64(ptr [[TABLE]], i8 0, i64 48, i1 false)
CHECK: store ptr [[TABLE]], ptr [[ROOT0]], align 4
CHECK: [[DATA1:%.*]] = getelementptr inbounds i8, ptr [[TABLE]], i64 16
CHECK-NEXT: store ptr [[DATA1]], ptr [[ROOT1]], align 4
CHECK: [[DATA2:%.*]] = getelementptr inbounds i8, ptr [[TABLE]], i64 32
CHECK-NEXT: store ptr [[DATA2]], ptr [[ROOT2]], align 4
CHECK-DAG: [[APTR:%[0-9]+]] = extractvalue [[STR]] [[A]], 0
CHECK-DAG: [[APAD:%[0-9]+]] = insertvalue { ptr, i32 } [[APTR]], i32 0, 1
CHECK-DAG: [[ANORM:%[0-9]+]] = insertvalue [[STR]] [[A]], { ptr, i32 } [[APAD]], 0
CHECK-DAG: [[BPTR:%[0-9]+]] = extractvalue [[STR]] [[B]], 0
CHECK-DAG: [[BPAD:%[0-9]+]] = insertvalue { ptr, i32 } [[BPTR]], i32 0, 1
CHECK-DAG: [[BNORM:%[0-9]+]] = insertvalue [[STR]] [[B]], { ptr, i32 } [[BPAD]], 0
CHECK-DAG: [[CPTR:%[0-9]+]] = extractvalue [[STR]] [[C]], 0
CHECK-DAG: [[CPAD:%[0-9]+]] = insertvalue { ptr, i32 } [[CPTR]], i32 0, 1
CHECK-DAG: [[CNORM:%[0-9]+]] = insertvalue [[STR]] [[C]], { ptr, i32 } [[CPAD]], 0
CHECK: [[SLOT0:%.*]] = getelementptr [[STR]], ptr [[TABLE]], i32 0
CHECK-NEXT: store [[STR]] [[ANORM]], ptr [[SLOT0]]
CHECK-NEXT: [[SLOT1:%.*]] = getelementptr [[STR]], ptr [[TABLE]], i32 1
CHECK-NEXT: store [[STR]] [[BNORM]], ptr [[SLOT1]]
CHECK-NEXT: [[SLOT2:%.*]] = getelementptr [[STR]], ptr [[TABLE]], i32 2
CHECK-NEXT: store [[STR]] [[CNORM]], ptr [[SLOT2]]
CHECK-NEXT: {{%.*}} = call [[STR]] @"github.com/xgo-dev/llgo/runtime/internal/runtime.StringConcat"(ptr [[TABLE]], i64 3, ptr null)
CHECK-NEXT: call void @llvm.memset.p0.i64(ptr [[TABLE]], i8 0, i64 48, i1 false)
`
	path := filepath.Join(t.TempDir(), "concat.check")
	if err := os.WriteFile(path, []byte(checks), 0600); err != nil {
		t.Fatal(err)
	}
	if err := filecheck.Match(path, ir); err != nil {
		t.Fatal(err)
	}
}

func TestStringConcatTargetABI(t *testing.T) {
	const stringType = `%"github.com/xgo-dev/llgo/runtime/internal/runtime.String"`
	for _, tc := range []struct {
		target      llssa.Target
		result      string
		pointerSize int
	}{
		{llssa.Target{GOOS: "windows", GOARCH: "amd64"}, "void", 8},
		{llssa.Target{GOOS: "linux", GOARCH: "amd64"}, "{ ptr, i64 }", 8},
		{llssa.Target{GOOS: "linux", GOARCH: "arm64"}, stringType, 8},
		{llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"}, "void", 4},
		{llssa.Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", LLVMTarget: "wasm64-unknown-emscripten"}, "void", 8},
	} {
		target := tc.target
		t.Run(strings.TrimRight(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, "/"), func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, `package concatabi
func fused(a, b, c string) string { return a+b+c }
func local(a, b, c string) int { return len(a+b+c) }
`)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			if prog.PointerSize() != tc.pointerSize || prog.GoWordSize() != 8 || prog.SizeOf(prog.String()) != 16 {
				t.Fatalf("pointer/Go word/string sizes = %d/%d/%d, want %d/8/16", prog.PointerSize(), prog.GoWordSize(), prog.SizeOf(prog.String()), tc.pointerSize)
			}
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(target.Spec().Triple)
			config := llabi.AggregateLoweringConfig{GoWordSize: prog.GoWordSize(), Wasm: target.GOARCH == "wasm"}
			llabi.LowerLargeAggregates(prog.TargetData(), mod, config)
			cabi.NewTransformer(prog, target.Spec().Triple, "", false).TransformModule(pkg.Path(), mod)
			llabi.LowerAggregateCopies(prog.TargetData(), mod, config)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			// Bind the real table, optional byte buffer, and ABI result slot so a
			// declaration-only check cannot hide swapped or missing arguments.
			for _, name := range []string{"fused", "local"} {
				checks, outArg, buffer := "", "", "null"
				if tc.result == "void" {
					checks += "CHECK: [[OUT:%.*]] = alloca " + stringType + ", align 8\n"
					outArg = "ptr sret(" + stringType + ") [[OUT]], "
				}
				if name == "local" {
					checks += "CHECK: [[BUFFER:%.*]] = alloca [32 x i8]\n"
					buffer = "[[BUFFER]]"
				}
				checks += "CHECK: [[TABLE:%.*]] = alloca [3 x " + stringType + "], align 8\n"
				checks += "CHECK: call " + tc.result + " @\"github.com/xgo-dev/llgo/runtime/internal/runtime.StringConcat\"(" + outArg + "ptr [[TABLE]], i64 3, ptr " + buffer + ")\n"
				if tc.result == "void" {
					checks += "CHECK-NEXT: {{%.*}} = load " + stringType + ", ptr [[OUT]], align 8\n"
				}
				path := filepath.Join(t.TempDir(), "concat.check")
				if err := os.WriteFile(path, []byte(checks), 0600); err != nil {
					t.Fatal(err)
				}
				if err := filecheck.Match(path, mustNamedFunction(t, mod, "concatabi."+name).String()); err != nil {
					t.Fatalf("%s: %v", name, err)
				}
			}
			assembly, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			assembly.Dispose()
		})
	}
}

func TestStringConcatPlans(t *testing.T) {
	pkg, _, _ := buildGoSSAPkg(t, `package concat
func sink(string)
func id(s string) string { return s }
func fused(a, b, c string) string { return a + b + c }
func shared(a, b, c string) string { x := a + b; sink(x); return x + c }
func interleaved(a, b, c string) string { x := a + b; y := id(c); return x + y }
func oversized(a string) string {
	return a+a+a+a+a+a+a+a+a+a+a+a+a+a+a+a+a
}
func limit(a string) string { return `+strings.Repeat("a+", maxStringConcatParts-1)+`a }
func crossBlock(a, b, c string, branch bool) string { x := a+b; if branch { return x+c }; return x }
func repeated(a, b string) string { x := a+b; return x+x }
func reverse(a, b, c, d string) string { right := c+d; left := a+b; return left+right }
func many(a, b string) string { return `+strings.Repeat("a+", maxStringConcatParts-1)+`a + (`+strings.Repeat("b+", maxStringConcatParts-1)+`b) }
func halves(a,b string) string { return (`+strings.Repeat("a+", 8)+`a) + (`+strings.Repeat("b+", 8)+`b) }
func interleavedMutation(buf []byte, b, c string) string { x := string(buf)+b; buf[0]='x'; return x+c }
func dominant(a, b, c string, branch bool) string { x := id(a); if branch { sink(b) }; return x+b+c }
func sharedLeaf(a, b, c, d string) string { x := a+b; sink(x); return x+c+d }
func crossBlockLeaf(a, b, c, d string, branch bool) string { x := a+b; if branch { sink(c) }; return x+c+d }
`)

	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	module := prog.NewPackage("concat", "concat")

	for _, tc := range []struct {
		name   string
		groups []int // Number of operands per fused call, in execution order.
	}{
		{"fused", []int{3}}, {"limit", []int{16}}, {"oversized", []int{16}},
		{"many", []int{16, 16}}, {"halves", []int{9, 10}}, {"reverse", []int{4}},
		{"shared", nil}, {"interleaved", nil}, {"crossBlock", nil},
		{"repeated", nil}, {"interleavedMutation", nil},
		{"dominant", []int{3}}, {"sharedLeaf", []int{3}}, {"crossBlockLeaf", []int{3}},
	} {
		fn := pkg.Func(tc.name)
		ctx := &context{prog: prog, fn: module.NewFunc(tc.name, fn.Signature, llssa.GoDecl)}
		ctx.prepareStringConcatPlans(fn)
		var groups []int
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				add, _ := instr.(*ssa.BinOp)
				if parts := ctx.concatPlans[add]; len(parts) > 0 {
					groups = append(groups, len(parts))
				}
			}
		}
		if !slices.Equal(groups, tc.groups) {
			t.Errorf("%s: group sizes %v, want %v", tc.name, groups, tc.groups)
		}
		if tc.name == "reverse" {
			for i, part := range ctx.concatPlans[lastStringAdd(fn)] {
				if part != fn.Params[i] {
					t.Errorf("reverse: operand %d = %v, want %v", i, part, fn.Params[i])
				}
			}
		}
	}

	fn := pkg.Func("fused")
	root := lastStringAdd(fn)
	child := root.X.(*ssa.BinOp)
	for _, skipped := range []*ssa.BinOp{child, root} {
		if parts := collectStringConcatParts([]*ssa.BinOp{child, root}, map[ssa.Instruction]none{skipped: {}}); parts != nil {
			t.Errorf("fused an addition handled by static initialization: %v", skipped)
		}
	}
}

func TestStringConcatDebugRefs(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "concat.go", `package concatdebug
func join(a, b, c string) string { prefix := a+b; return prefix+c }
func tail(a, b, c, d string) string { tailPrefix := a+b; return tailPrefix+c+d }
func groups(a, b, c, d, e string) string { chunk := a+b+c; return chunk+d+e }
func root(a, b, c string) string { result := a+b+c; return result }
func expression(a, b, c string) string { return a+b+c }
func reassigned(seed, b, c string) string { seed = seed+b; return seed+c }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("concatdebug", "concatdebug"), files,
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		level   optlevel.Level
		symbols bool
	}{
		{"O0", optlevel.O0, true},
		{"Os", optlevel.Os, true},
		{"O2", optlevel.O2, true},
		{"O0/no-symbols", optlevel.O0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64", OptLevel: tc.level})
			defer prog.Dispose()
			compiled, _, err := newPackageEx(prog, nil, nil, nil, pkg, files, nil, false,
				Options{Debug: tc.symbols, DebugSymbols: tc.symbols})
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			ir := compiled.String()
			preserve := tc.level == optlevel.O0 && tc.symbols
			if preserve {
				for _, variable := range []string{"prefix", "tailPrefix", "chunk"} {
					assertDebugRecords(t, ir, `name: "`+variable+`"`, false, true)
				}
				assertDebugHomeStores(t, ir, llvmFunction(t, ir, "concatdebug.reassigned"), `name: "seed", arg: 1`, 2)
			}
			if tc.symbols {
				assertDebugRecords(t, ir, `name: "result"`, false, true)
			}
			for _, fn := range []struct {
				name           string
				concat, binary int // Calls when intermediate debug values are preserved.
			}{
				{"join", 0, 2}, {"tail", 1, 1}, {"groups", 2, 0},
				{"root", 1, 0}, {"expression", 1, 0}, {"reassigned", 0, 2},
			} {
				body := llvmFunction(t, ir, "concatdebug."+fn.name)
				concat := strings.Count(body, "runtime/internal/runtime.StringConcat")
				binary := strings.Count(body, "runtime/internal/runtime.StringCat")
				wantConcat, wantBinary := 1, 0
				if preserve {
					wantConcat, wantBinary = fn.concat, fn.binary
				}
				if concat != wantConcat || binary != wantBinary {
					t.Errorf("%s: concat/binary calls = %d/%d, want %d/%d:\n%s",
						fn.name, concat, binary, wantConcat, wantBinary, body)
				}
			}
		})
	}
}

func TestStringConcatPlanBudget(t *testing.T) {
	// A rejected four-part chain must not fall back to fusing its cheaper
	// three-part prefix, even when that prefix would fit the remaining budget.
	const parts = 4
	pkg, _, _ := buildGoSSAPkg(t, `package concatbudget
func join(a, b, c, d string) string { return a+b+c+d }
func twoGroups(s string) string { return `+strings.Repeat("s+", maxStringConcatParts+1)+`s }
func empty(s string) string { return ""+s+"!" }
`)
	for _, roots := range []bool{false, true} {
		prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
		defer prog.Dispose()
		prog.EnableGCRoots(roots)
		module := prog.NewPackage("concatbudget", "concatbudget")
		fn := pkg.Func("join")
		lowered := module.NewFunc("join", fn.Signature, llssa.GoDecl)
		ctx := &context{prog: prog, fn: lowered}
		table := prog.Type(types.NewArray(types.Typ[types.String], parts), llssa.GoDecl)
		cost := prog.SizeOf(table) + prog.StackRootBytes(table)
		lowered.SetEscapeStackBytes(llabi.MaxEscapeStackBytes - cost + 1)
		ctx.prepareStringConcatPlans(fn)
		if ctx.concatPlans != nil {
			t.Fatalf("roots=%v: over-budget plan was accepted", roots)
		}
		lowered.SetEscapeStackBytes(llabi.MaxEscapeStackBytes - cost)
		ctx.prepareStringConcatPlans(fn)
		if len(ctx.concatPlans[lastStringAdd(fn)]) != parts {
			t.Fatalf("roots=%v: exact-budget plan was rejected", roots)
		}

		// Only the second (three-part) group fits. Its first operand must
		// remain the ordinary binary result of the rejected sixteen-part group.
		fn = pkg.Func("twoGroups")
		ctx.fn = module.NewFunc("twoGroups", fn.Signature, llssa.GoDecl)
		ctx.fn.SetEscapeStackBytes(llabi.MaxEscapeStackBytes - cost)
		ctx.prepareStringConcatPlans(fn)
		tail := ctx.concatPlans[lastStringAdd(fn)]
		if len(tail) != 3 || len(ctx.concatPlans) != 2 {
			t.Fatalf("roots=%v: failed to isolate the over-budget group", roots)
		}
		prefix := tail[0].(*ssa.BinOp)
		if _, planned := ctx.concatPlans[prefix]; planned {
			t.Fatalf("roots=%v: over-budget prefix was not materialized", roots)
		}

		fn = pkg.Func("empty")
		ctx.fn = module.NewFunc("empty", fn.Signature, llssa.GoDecl)
		ctx.fn.SetEscapeStackBytes(llabi.MaxEscapeStackBytes)
		ctx.prepareStringConcatPlans(fn)
		if len(ctx.concatPlans[lastStringAdd(fn)]) != 2 {
			t.Fatalf("roots=%v: binary concat charged for an operand table", roots)
		}
	}
}

func lastStringAdd(fn *ssa.Function) *ssa.BinOp {
	for i := len(fn.Blocks) - 1; i >= 0; i-- {
		instrs := fn.Blocks[i].Instrs
		for j := len(instrs) - 1; j >= 0; j-- {
			if add, ok := instrs[j].(*ssa.BinOp); ok && isStringAdd(add) {
				return add
			}
		}
	}
	panic("missing string add")
}
