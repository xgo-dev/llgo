//go:build !llgo
// +build !llgo

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package cl

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"
	"unsafe"

	llssa "github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestConstBool(t *testing.T) {
	if v, ok := constBool(nil); v || ok {
		t.Fatal("constBool?")
	}
}

func TestIsLargeNonPointerValue(t *testing.T) {
	prog := llssa.NewProgram(nil)
	ctx := &context{prog: prog}

	largeArray := prog.Type(types.NewArray(types.Typ[types.Byte], int64(maxDirectDerefSize)+1), llssa.InGo)
	if !ctx.isLargeNonPointerValue(largeArray) {
		t.Fatal("large array should require explicit nil-deref guard")
	}

	smallArray := prog.Type(types.NewArray(types.Typ[types.Byte], 16), llssa.InGo)
	if ctx.isLargeNonPointerValue(smallArray) {
		t.Fatal("small array should not require explicit nil-deref guard")
	}

	largePointer := prog.Type(types.NewPointer(types.NewArray(types.Typ[types.Byte], int64(maxDirectDerefSize)+1)), llssa.InGo)
	if ctx.isLargeNonPointerValue(largePointer) {
		t.Fatal("pointer values should not be classified as large direct values")
	}
}

func TestCompileLargeNilDerefInterfaceGuards(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

type large [1 << 21]byte
type largeStruct struct {
	data [1 << 21]byte
}
type holder struct {
	pad [1 << 21]byte
	value largeStruct
}

var sink any

func arrayIface(p *large) {
	sink = *p
}

func standalone(p *large) {
	_ = *p
}

func fieldIface(p *holder) {
	sink = p.value
}
`)
	ir := m.String()
	for _, want := range []string{"AssertNilDeref", "Typedmemmove"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("compiled IR missing %s for large nil-deref guard path:\n%s", want, ir)
		}
	}
}

func TestCompileInterfaceCompareDerefNilGuard(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

func compareInterfacePtr(p *interface{}, q interface{}) bool {
	return *p == q
}
`)
	if ir := m.String(); !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("compiled IR missing AssertNilDeref for interface compare deref:\n%s", ir)
	}
}

func TestCompileNestedLargeNilDerefBaseGuard(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

type large [1 << 21]byte

func nested(pp **large) {
	_ = **pp
}
`)
	ir := m.String()
	if strings.Count(ir, "AssertNilDerefPtr") == 0 {
		t.Fatalf("compiled IR missing nested AssertNilDerefPtr guard:\n%s", ir)
	}
	if !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("compiled IR missing outer AssertNilDeref guard:\n%s", ir)
	}
}

func TestCompileWrapNilCheckGuard(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
	package foo

type value struct {
	n int
}

func (v value) method() int {
	return v.n
}

func methodExpr(p *value) int {
	return (*value).method(p)
}

func methodValue(p *value) func() int {
	return p.method
}
`)
	if !strings.Contains(m.String(), "AssertNilDeref") {
		t.Fatalf("compiled IR missing AssertNilDeref for ssa:wrapnilchk:\n%s", m.String())
	}
}

func TestCompilePromotedValueMethodNilDerefGuard(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
	package foo

	type embedded struct {
		n int
	}

	func (v embedded) value() int {
		return v.n
	}

	type outer struct {
		*embedded
	}

	func call(o outer) int {
		return o.value()
	}
	`)
	if !strings.Contains(m.String(), "AssertNilDeref") {
		t.Fatalf("compiled IR missing AssertNilDeref for promoted value method receiver:\n%s", m.String())
	}
}

func TestCompileValueReceiverNilDerefKeepsDominance(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
	package foo

	type stamp struct {
		n int
	}

	func (s stamp) value() int {
		return 1
	}

	type conn struct {
		idle stamp
	}

	func pick(conns []*conn, cond bool) int {
		if len(conns) == 0 {
			return 0
		}
		pc := conns[0]
		if cond {
			_ = pc.idle.value()
		}
		return pc.idle.n
	}
	`)
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("compiled IR failed verifier: %v\n%s", err, m.String())
	}
}

func TestMethodNilDerefHelperBranches(t *testing.T) {
	ctx := &context{}
	alloc := &ssa.Alloc{}
	field := &ssa.FieldAddr{X: alloc}

	ctx.emitNilDerefBaseCheck(nil, field)
	ctx.assertNilDerefBase(nil, field)
	ctx.assertNilDerefBase(nil, &ssa.UnOp{Op: token.ADD})

	if arg, ok := valueReceiverNilDerefArg(nil, nil); ok || arg != nil {
		t.Fatal("nil function should not request value receiver nil check")
	}

	recv := types.NewVar(token.NoPos, nil, "recv", types.Typ[types.Int])
	valueMethod := &ssa.Function{
		Signature: types.NewSignatureType(recv, nil, nil, nil, nil, false),
	}
	unop := &ssa.UnOp{Op: token.MUL, X: alloc}
	if arg, ok := valueReceiverNilDerefArg(valueMethod, []ssa.Value{unop}); ok || arg != nil {
		t.Fatal("known non-nil receiver address should not request nil check")
	}

	bound := &ssa.Function{
		Synthetic: "bound method wrapper for value",
		FreeVars:  []*ssa.FreeVar{nil},
	}
	if arg, ok := boundValueReceiverNilDerefArg(bound, nil); ok || arg != nil {
		t.Fatal("missing bound method binding should not request nil check")
	}

	if arg, ok := boundValueReceiverNilDerefArg(
		&ssa.Function{Synthetic: "plain closure", FreeVars: []*ssa.FreeVar{nil}},
		[]ssa.Value{unop},
	); ok || arg != nil {
		t.Fatal("non-bound wrapper should not request nil check")
	}

	ssapkg := buildSSAPackage(t, `
package foo

type pointer struct{}

func (*pointer) pointerMethod() int { return 1 }

type value struct{}

func (value) valueMethod() int { return 2 }

func bindPointer(p *pointer) func() int {
	return p.pointerMethod
}

func bindLocalValue() func() int {
	var v value
	return v.valueMethod
}
`)
	if arg, ok := boundValueReceiverNilDerefArgFromFunc(t, ssapkg.Func("bindPointer")); ok || arg != nil {
		t.Fatal("pointer receiver bound method should not request nil check")
	}
	if arg, ok := boundValueReceiverNilDerefArgFromFunc(t, ssapkg.Func("bindLocalValue")); ok || arg != nil {
		t.Fatal("known non-nil local value binding should not request nil check")
	}
}

func boundValueReceiverNilDerefArgFromFunc(t *testing.T, fn *ssa.Function) (*ssa.UnOp, bool) {
	t.Helper()
	if fn == nil {
		t.Fatal("missing function")
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			closure, ok := instr.(*ssa.MakeClosure)
			if !ok {
				continue
			}
			bound, ok := closure.Fn.(*ssa.Function)
			if !ok {
				t.Fatalf("closure function has type %T, want *ssa.Function", closure.Fn)
			}
			return boundValueReceiverNilDerefArg(bound, closure.Bindings)
		}
	}
	t.Fatalf("bound method closure not found in %s", fn.Name())
	return nil, false
}

func TestCollectMethodNilDerefChecksSkipsDynamicDeferGo(t *testing.T) {
	fn := &ssa.Function{
		Blocks: []*ssa.BasicBlock{{
			Instrs: []ssa.Instruction{
				&ssa.Defer{},
				&ssa.Go{},
			},
		}},
	}
	if got := collectMethodNilDerefChecks(fn); len(got) != 0 {
		t.Fatalf("collectMethodNilDerefChecks() = %v, want no static checks", got)
	}
}

func TestToBackground(t *testing.T) {
	if v := toBackground(""); v != llssa.InGo {
		t.Fatal("toBackground:", v)
	}
}

func TestCollectSkipNames(t *testing.T) {
	ctx := &context{skips: make(map[string]none)}
	ctx.collectSkipNames("//llgo:skipall")
	ctx.collectSkipNames("//llgo:skip")
	ctx.collectSkipNames("//llgo:skip abs")
}

func TestCollectSkipNamesByDoc(t *testing.T) {
	ftest := func(comments string, wantSkips []string, wantAll bool) {
		t.Helper()
		ctx := &context{skips: make(map[string]none)}
		doc := parseComments(t, comments)
		ctx.collectSkipNamesByDoc(doc)

		// Check skipall
		if wantAll != ctx.skipall {
			t.Errorf("skipall = %v, want %v", ctx.skipall, wantAll)
		}

		// Check collected symbols
		var gotSkips []string
		for sym := range ctx.skips {
			gotSkips = append(gotSkips, sym)
		}
		if len(gotSkips) != len(wantSkips) {
			t.Errorf("got %d skips %v, want %d skips %v", len(gotSkips), gotSkips, len(wantSkips), wantSkips)
			return
		}
		// Check each expected symbol exists
		for _, want := range wantSkips {
			if _, ok := ctx.skips[want]; !ok {
				t.Errorf("missing expected symbol %q", want)
			}
		}
	}

	// Multiple llgo:skip mixed - stops at first non-directive
	ftest(`
				//llgo:skip sym1 sym2
				//llgo:skip sym3
				//llgo:skipall
				// normal comment
				// llgo:skip sym4
				//llgo:skip sym5
			`,
		[]string{"sym4", "sym5"},
		false,
	)

	// llgo:skip and go: mixed - processes until non-directive
	ftest(`
				//llgo:skip sym1
				//llgo:skipall
				//go:generate
				// normal comment
				//go:build linux
				//llgo:skip sym2
			`,
		[]string{"sym2"},
		false,
	)

	// Only directives - processes all
	ftest(`
				// llgo:skip sym1
				//go:generate
				// llgo:skip sym2 sym3
				// llgo:skipall
			`,
		[]string{"sym1", "sym2", "sym3"},
		true,
	)

	// Starts with non-directive - stops immediately
	ftest(`
				//llgo:skip sym1
				// normal comment
				//llgo:skip sym2
				//llgo:skipall
			`,
		[]string{"sym2"},
		true,
	)

	// Only normal comments
	ftest(`
				// normal comment 1
				// normal comment 2
			`,
		[]string{},
		false,
	)
}

func parseComments(t *testing.T, text string) *ast.CommentGroup {
	t.Helper()
	var comments []*ast.Comment
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		comments = append(comments, &ast.Comment{Text: line})
	}
	return &ast.CommentGroup{List: comments}
}

func TestReplaceGoName(t *testing.T) {
	if ret := replaceGoName("foo", 0); ret != "foo" {
		t.Fatal("replaceGoName:", ret)
	}
}

func TestIsAllocVargs(t *testing.T) {
	if isAllocVargs(nil, ssaAlloc(&ssa.Return{})) {
		t.Fatal("isVargs?")
	}
	if isAllocVargs(nil, ssaAlloc(ssaSlice(&ssa.Go{}))) {
		t.Fatal("isVargs?")
	}
	if isAllocVargs(nil, ssaAlloc(ssaSlice(&ssa.Return{}))) {
		t.Fatal("isVargs?")
	}
}

func ssaSlice(refs ...ssa.Instruction) *ssa.Slice {
	a := &ssa.Slice{}
	setRefs(unsafe.Pointer(a), refs...)
	return a
}

func ssaAlloc(refs ...ssa.Instruction) *ssa.Alloc {
	a := &ssa.Alloc{}
	setRefs(unsafe.Pointer(a), refs...)
	return a
}

func setRefs(v unsafe.Pointer, refs ...ssa.Instruction) {
	off := unsafe.Offsetof(ssa.Alloc{}.Comment) - unsafe.Sizeof([]int(nil))
	ptr := uintptr(v) + off
	*(*[]ssa.Instruction)(unsafe.Pointer(ptr)) = refs
}

func TestRecvTypeName(t *testing.T) {
	if ret := recvTypeName(&ast.IndexExpr{
		X:     &ast.Ident{Name: "Pointer"},
		Index: &ast.Ident{Name: "T"},
	}); ret != "Pointer" {
		t.Fatal("recvTypeName IndexExpr:", ret)
	}
	if ret := recvTypeName(&ast.IndexListExpr{
		X:       &ast.Ident{Name: "Pointer"},
		Indices: []ast.Expr{&ast.Ident{Name: "T"}},
	}); ret != "Pointer" {
		t.Fatal("recvTypeName IndexListExpr:", ret)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("recvTypeName: no error?")
		}
	}()
	recvTypeName(&ast.BadExpr{})
}

func TestRecvType(t *testing.T) {
	pkg := types.NewPackage("", "main")
	obj := types.NewTypeName(token.NoPos, pkg, "T", nil)
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	if ret := recvNamed(named); ret != named {
		t.Fatal("error")
	}
	if ret := recvNamed(types.NewPointer(named)); ret != named {
		t.Fatal("error")
	}
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("must panic")
		}
	}()
	recvNamed(types.NewPointer(types.Typ[types.Int]))
}

/*
func TestErrCompileValue(t *testing.T) {
	defer func() {
		if r := recover(); r != "can't use llgo instruction as a value" {
			t.Fatal("TestErrCompileValue:", r)
		}
	}()
	pkg := types.NewPackage("foo", "foo")
	ctx := &context{
		goTyps: pkg,
		link: map[string]string{
			"foo.": "llgo.unreachable",
		},
	}
	ctx.compileValue(nil, &ssa.Function{
		Pkg:       &ssa.Package{Pkg: pkg},
		Signature: types.NewSignatureType(nil, nil, nil, nil, nil, false),
	})
}
*/

func TestErrCompileInstrOrValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("compileInstrOrValue: no error?")
		}
	}()
	ctx := &context{
		bvals: make(map[ssa.Value]llssa.Expr),
	}
	ctx.compileInstrOrValue(nil, &ssa.Call{}, true)
}

func TestErrBuiltin(t *testing.T) {
	test := func(builtin string, fn func(ctx *context)) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal(builtin, ": no error?")
			}
		}()
		var ctx context
		fn(&ctx)
	}
	test("advance", func(ctx *context) { ctx.advance(nil, nil) })
	test("alloca", func(ctx *context) { ctx.alloca(nil, nil) })
	test("allocaCStr", func(ctx *context) { ctx.allocaCStr(nil, nil) })
	test("allocaCStrs", func(ctx *context) { ctx.allocaCStrs(nil, nil) })
	test("allocaCStrs(Nonconst)", func(ctx *context) { ctx.allocaCStrs(nil, []ssa.Value{nil, &ssa.Parameter{}}) })
	test("string", func(ctx *context) { ctx.string(nil, nil) })
	test("stringData", func(ctx *context) { ctx.stringData(nil, nil) })
	test("funcAddr", func(ctx *context) { ctx.funcAddr(nil, nil) })
	test("sigsetjmp", func(ctx *context) { ctx.sigsetjmp(nil, nil) })
	test("siglongjmp", func(ctx *context) { ctx.siglongjmp(nil, nil) })
	test("cstr(NoArgs)", func(ctx *context) { cstr(nil, nil) })
	test("cstr(Nonconst)", func(ctx *context) { cstr(nil, []ssa.Value{&ssa.Parameter{}}) })
	test("pystr(NoArgs)", func(ctx *context) { pystr(nil, nil) })
	test("pystr(Nonconst)", func(ctx *context) { pystr(nil, []ssa.Value{&ssa.Parameter{}}) })
	test("atomic", func(ctx *context) { ctx.atomic(nil, 0, nil) })
	test("atomicLoad", func(ctx *context) { ctx.atomicLoad(nil, nil) })
	test("atomicStore", func(ctx *context) { ctx.atomicStore(nil, nil) })
	test("atomicCmpXchg", func(ctx *context) { ctx.atomicCmpXchg(nil, nil) })
}

func TestErrAsm(t *testing.T) {
	test := func(testName string, fn func(ctx *context)) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal(testName, ": no error?")
			}
		}()
		var ctx context
		fn(&ctx)
	}

	test("asm(NoArgs)", func(ctx *context) { ctx.asm(nil, []ssa.Value{}) })
	test("asm(Nonconst)", func(ctx *context) { ctx.asm(nil, []ssa.Value{&ssa.Parameter{}}) })
	test("asmFull(Nonconst)", func(ctx *context) { ctx.asm(nil, []ssa.Value{&ssa.Parameter{}, &ssa.Parameter{}}) })
	test("asmFull(NonConstKey)", func(ctx *context) {
		makeMap := &ssa.MakeMap{}
		nonConstKey := &ssa.Parameter{}
		mapUpdate := &ssa.MapUpdate{Key: nonConstKey}
		referrers := []ssa.Instruction{mapUpdate}
		setRefs(unsafe.Pointer(makeMap), referrers...)
		strConst := &ssa.Const{
			Value: constant.MakeString("nop"),
		}
		ctx.asm(nil, []ssa.Value{strConst, makeMap})
	})
	test("asmFull(RegisterNotFound)", func(ctx *context) {
		makeMap := &ssa.MakeMap{}
		referrers := []ssa.Instruction{}
		setRefs(unsafe.Pointer(makeMap), referrers...)
		strConst := &ssa.Const{
			Value: constant.MakeString("test {missing}"),
		}
		ctx.asm(nil, []ssa.Value{strConst, makeMap})
	})
	test("asmFull(UnknownReferrer)", func(ctx *context) {
		makeMap := &ssa.MakeMap{}
		unknownRef := &ssa.Return{}
		referrers := []ssa.Instruction{unknownRef}
		setRefs(unsafe.Pointer(makeMap), referrers...)
		strConst := &ssa.Const{
			Value: constant.MakeString("test"),
		}
		ctx.asm(nil, []ssa.Value{strConst, makeMap})
	})
}

func TestPkgNoInit(t *testing.T) {
	pkg := types.NewPackage("foo", "foo")
	ctx := &context{
		goTyps: pkg,
		loaded: make(map[*types.Package]*pkgInfo),
	}
	if ctx.pkgNoInit(pkg) {
		t.Fatal("pkgNoInit?")
	}
}

func TestPkgKind(t *testing.T) {
	if v, _ := pkgKind("link: hello.a"); v != PkgLinkExtern {
		t.Fatal("pkgKind:", v)
	}
	if v, _ := pkgKind("noinit"); v != PkgNoInit {
		t.Fatal("pkgKind:", v)
	}
	if v, _ := pkgKind("link"); v != PkgLinkIR {
		t.Fatal("pkgKind:", v)
	}
	if v, _ := pkgKind(""); v != PkgLLGo {
		t.Fatal("pkgKind:", v)
	}
	if v, _ := pkgKind("decl"); v != PkgDeclOnly {
		t.Fatal("pkgKind:", v)
	}
	if v, _ := pkgKind("decl: test.ll"); v != PkgDeclOnly {
		t.Fatal("pkgKind:", v)
	}
}

func TestPkgKindOf(t *testing.T) {
	if v, _ := PkgKindOf(types.Unsafe); v != PkgDeclOnly {
		t.Fatal("PkgKindOf unsafe:", v)
	}
	pkg := types.NewPackage("foo", "foo")
	pkg.Scope().Insert(
		types.NewConst(
			0, pkg, "LLGoPackage", types.Typ[types.String],
			constant.MakeString("noinit")),
	)
	if v, _ := PkgKindOf(pkg); v != PkgNoInit {
		t.Fatal("PkgKindOf foo:", v)
	}
}

func TestIsAny(t *testing.T) {
	if isAny(types.Typ[types.UntypedInt]) {
		t.Fatal("isAny?")
	}
}

func TestIntVal(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("intVal: no error?")
		}
	}()
	intVal(&ssa.Parameter{})
}

func TestErrImport(t *testing.T) {
	var ctx context
	pkg := types.NewPackage("foo", "foo")
	ctx.importPkg(pkg, nil)

	alt := types.NewPackage("bar", "bar")
	alt.Scope().Insert(
		types.NewConst(0, alt, "LLGoPackage", types.Typ[types.String], constant.MakeString("noinit")),
	)
	ctx.patches = Patches{"foo": Patch{Alt: &ssa.Package{Pkg: alt}, Types: alt}}
	ctx.importPkg(pkg, &pkgInfo{})
}

func TestErrInitLinkname(t *testing.T) {
	var ctx context
	ctx.initLinkname("//llgo:link abc", true, func(name string, isExport bool) (string, bool, bool) {
		return "", false, false
	})
	ctx.initLinkname("//go:linkname Printf printf", true, func(name string, isExport bool) (string, bool, bool) {
		return "", false, false
	})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("initLinkname: no error?")
		}
	}()
	ctx.initLinkname("//go:linkname Printf printf", true, func(name string, isExport bool) (string, bool, bool) {
		return "foo.Printf", false, name == "Printf"
	})
}

func TestErrVarOf(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("varOf: no error?")
		}
	}()
	prog := llssa.NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	pkgTypes := types.NewPackage("foo", "foo")
	ctx := &context{
		pkg:    pkg,
		goTyps: pkgTypes,
	}
	ssaPkg := &ssa.Package{Pkg: pkgTypes}
	g := &ssa.Global{Pkg: ssaPkg}
	ctx.varOf(nil, g)
}

func TestContextResolveLinkname(t *testing.T) {
	tests := []struct {
		name   string
		link   map[string]string
		input  string
		want   string
		panics bool
	}{
		{
			name: "Normal",
			link: map[string]string{
				"foo": "C.bar",
			},
			input: "foo",
			want:  "bar",
		},
		{
			name: "MultipleLinks",
			link: map[string]string{
				"foo1": "C.bar1",
				"foo2": "C.bar2",
			},
			input: "foo2",
			want:  "bar2",
		},
		{
			name:  "NoLink",
			link:  map[string]string{},
			input: "foo",
			want:  "foo",
		},
		{
			name: "InvalidLink",
			link: map[string]string{
				"foo": "invalid.bar",
			},
			input:  "foo",
			panics: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				defer func() {
					if r := recover(); r == nil {
						t.Error("want panic")
					}
				}()
			}
			ctx := &context{prog: llssa.NewProgram(nil)}
			for k, v := range tt.link {
				ctx.prog.SetLinkname(k, v)
			}
			got := ctx.resolveLinkname(tt.input)
			if !tt.panics {
				if got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestInstantiate(t *testing.T) {
	obj := types.NewTypeName(0, nil, "T", nil)
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	if typ := obj.Type(); typ != instantiate(typ, named) {
		t.Fatal("error")
	}
	tparam := types.NewTypeParam(types.NewTypeName(0, nil, "P", nil), types.NewInterface(nil, nil))
	named.SetTypeParams([]*types.TypeParam{tparam})
	inamed, err := types.Instantiate(nil, named, []types.Type{types.Typ[types.Int]}, true)
	if err != nil {
		t.Fatal(err)
	}
	if typ := instantiate(obj.Type(), inamed.(*types.Named)); typ == obj.Type() || typ.(*types.Named).TypeArgs() == nil {
		t.Fatal("error")
	}
}

func TestHandleExportDiffName(t *testing.T) {
	tests := []struct {
		name               string
		enableExportRename bool
		line               string
		fullName           string
		inPkgName          string
		wantHasLinkname    bool
		wantLinkname       string
		wantExport         string
	}{
		{
			name:               "ExportDiffNames_DifferentName",
			enableExportRename: true,
			line:               "//export IRQ_Handler",
			fullName:           "pkg.HandleInterrupt",
			inPkgName:          "HandleInterrupt",
			wantHasLinkname:    true,
			wantLinkname:       "IRQ_Handler",
			wantExport:         "IRQ_Handler",
		},
		{
			name:               "ExportDiffNames_SameName",
			enableExportRename: true,
			line:               "//export SameName",
			fullName:           "pkg.SameName",
			inPkgName:          "SameName",
			wantHasLinkname:    true,
			wantLinkname:       "SameName",
			wantExport:         "SameName",
		},
		{
			name:               "ExportDiffNames_WithSpaces",
			enableExportRename: true,
			line:               "//export   Timer_Callback  ",
			fullName:           "pkg.OnTimerTick",
			inPkgName:          "OnTimerTick",
			wantHasLinkname:    true,
			wantLinkname:       "Timer_Callback",
			wantExport:         "Timer_Callback",
		},
		{
			name:               "ExportDiffNames_Disabled_MatchingName",
			enableExportRename: false,
			line:               "//export Func",
			fullName:           "pkg.Func",
			inPkgName:          "Func",
			wantHasLinkname:    true,
			wantLinkname:       "Func",
			wantExport:         "Func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global state
			oldEnableExportRename := enableExportRename
			defer func() {
				EnableExportRename(oldEnableExportRename)
			}()
			EnableExportRename(tt.enableExportRename)

			// Setup context
			prog := llssa.NewProgram(nil)
			pkg := prog.NewPackage("test", "test")
			ctx := &context{
				prog: prog,
				pkg:  pkg,
			}

			// Call initLinkname with closure that mimics initLinknameByDoc behavior
			ret := ctx.initLinkname(tt.line, true, func(name string, isExport bool) (string, bool, bool) {
				return tt.fullName, false, name == tt.inPkgName || (isExport && enableExportRename)
			})

			// Verify result
			hasLinkname := (ret == hasLinkname)
			if hasLinkname != tt.wantHasLinkname {
				t.Errorf("hasLinkname = %v, want %v", hasLinkname, tt.wantHasLinkname)
			}

			if tt.wantHasLinkname {
				// Check linkname was set
				if link, ok := prog.Linkname(tt.fullName); !ok || link != tt.wantLinkname {
					t.Errorf("linkname = %q (ok=%v), want %q", link, ok, tt.wantLinkname)
				}

				// Check export was set
				exports := pkg.ExportFuncs()
				if export, ok := exports[tt.fullName]; !ok || export != tt.wantExport {
					t.Errorf("export = %q (ok=%v), want %q", export, ok, tt.wantExport)
				}
			}
		})
	}
}

func TestInitLinknameByDocExportDiffNames(t *testing.T) {
	tests := []struct {
		name               string
		enableExportRename bool
		doc                *ast.CommentGroup
		fullName           string
		inPkgName          string
		wantExported       bool // Whether the symbol should be exported with different name
		wantLinkname       string
		wantExport         string
	}{
		{
			name:               "WithExportDiffNames_DifferentNameExported",
			enableExportRename: true,
			doc: &ast.CommentGroup{
				List: []*ast.Comment{
					{Text: "//export IRQ_Handler"},
				},
			},
			fullName:     "pkg.HandleInterrupt",
			inPkgName:    "HandleInterrupt",
			wantExported: true,
			wantLinkname: "IRQ_Handler",
			wantExport:   "IRQ_Handler",
		},
		{
			name:               "WithoutExportDiffNames_NotExported",
			enableExportRename: false,
			doc: &ast.CommentGroup{
				List: []*ast.Comment{
					{Text: "//export DifferentName"},
				},
			},
			fullName:     "pkg.HandleInterrupt",
			inPkgName:    "HandleInterrupt",
			wantExported: false,
			// Without enableExportRename, it goes through normal flow which expects same name
			// The symbol "DifferentName" won't be found, so no export happens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Without enableExportRename, export with different name will panic
			if !tt.wantExported && !tt.enableExportRename {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic for export with different name when enableExportRename=false")
					}
				}()
			}

			// Save and restore global state
			oldEnableExportRename := enableExportRename
			defer func() {
				EnableExportRename(oldEnableExportRename)
			}()
			EnableExportRename(tt.enableExportRename)

			// Setup context
			prog := llssa.NewProgram(nil)
			pkg := prog.NewPackage("test", "test")
			ctx := &context{
				prog: prog,
				pkg:  pkg,
			}

			// Call initLinknameByDoc
			ctx.processLinknameByDoc(tt.doc, tt.fullName, tt.inPkgName, false, true)

			// Verify export behavior
			exports := pkg.ExportFuncs()
			if tt.wantExported {
				// Should have exported the symbol with different name
				if export, ok := exports[tt.fullName]; !ok || export != tt.wantExport {
					t.Errorf("export = %q (ok=%v), want %q", export, ok, tt.wantExport)
				}
				// Check linkname was also set
				if link, ok := prog.Linkname(tt.fullName); !ok || link != tt.wantLinkname {
					t.Errorf("linkname = %q (ok=%v), want %q", link, ok, tt.wantLinkname)
				}
			}
		})
	}
}

func TestInitLinkExportDiffNames(t *testing.T) {
	tests := []struct {
		name               string
		enableExportRename bool
		line               string
		wantPanic          bool
	}{
		{
			name:               "ExportDiffNames_Enabled_NoError",
			enableExportRename: true,
			line:               "//export IRQ_Handler",
			wantPanic:          false,
		},
		{
			name:               "ExportDiffNames_Disabled_Panic",
			enableExportRename: false,
			line:               "//export IRQ_Handler",
			wantPanic:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but didn't panic")
					}
				}()
			}

			oldEnableExportRename := enableExportRename
			defer func() {
				EnableExportRename(oldEnableExportRename)
			}()
			EnableExportRename(tt.enableExportRename)

			prog := llssa.NewProgram(nil)
			pkg := prog.NewPackage("test", "test")
			ctx := &context{
				prog: prog,
				pkg:  pkg,
			}

			ctx.initLinkname(tt.line, true, func(inPkgName string, isExport bool) (fullName string, isVar, ok bool) {
				// Simulate initLinknames scenario: symbol not found (like in decl packages)
				return "", false, false
			})
		})
	}
}
