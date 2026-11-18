package cl

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llgo/internal/typepatch"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func init() {
	llssa.Initialize(llssa.InitAll | llssa.InitNative)
}

func TestStripLargeStaticByteArraySkipsSSAStores(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nvar Table = [%d]byte{%s}\n\nfunc Use() byte { return Table[0] + Table[%d] }\n", n, strings.Repeat("1, ", n), n-1)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: gpackages.NewImporter(fset)}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	payloads := StripLargeStaticByteArrays(pkg, []*ast.File{file}, info)
	got := payloads[StaticByteArrayKey(pkg.Path(), "Table")]
	if len(got) != n || got[0] != 1 || got[n-1] != 1 {
		t.Fatalf("payload = len %d data[0]=%d data[%d]=%d", len(got), got[0], n-1, got[n-1])
	}
	vs := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(vs.Values) != 0 {
		t.Fatalf("initializer was not removed: %#v", vs.Values)
	}
	at, ok := vs.Type.(*ast.ArrayType)
	if !ok || at.Len == nil {
		t.Fatalf("stripped type = %#v, want [N]byte", vs.Type)
	}

	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	ssaPkg := prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()
	initFn := ssaPkg.Func("init")
	stores := 0
	if initFn != nil {
		for _, b := range initFn.Blocks {
			for _, instr := range b.Instrs {
				if _, ok := instr.(*ssa.Store); ok {
					stores++
				}
				if call, ok := instr.(*ssa.Call); ok {
					if callee := call.Call.Value; callee != nil && strings.Contains(callee.String(), "memclr") {
						t.Fatalf("init still memclr's stripped array: %v", call)
					}
				}
			}
		}
	}
	if stores > 8 {
		t.Fatalf("init still has %d stores after stripping %d-byte array", stores, n)
	}

	llprog := ssatest.NewProgramEx(t, nil, conf.Importer)
	llprog.TypeSizes(types.SizesFor("gc", "arm64"))
	ret, _, err := NewPackageExWithEmbedMetaOptions(llprog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{StaticByteArrays: payloads})
	if err != nil {
		t.Fatal(err)
	}
	ir := ret.String()
	if !strings.Contains(ir, fmt.Sprintf("@p.Table = global [%d x i8]", n)) {
		t.Fatalf("missing static byte array global:\n%s", ir)
	}
	if strings.Contains(ir, "zeroinitializer") {
		t.Fatalf("static byte array still zero-initialized:\n%s", ir)
	}
}

func TestStripLargeStaticByteArrayIgnoresSmallTables(t *testing.T) {
	src := "package p\n\nvar SBox = [256]byte{1, 2, 3}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := StripLargeStaticByteArrays(types.NewPackage("p", "p"), []*ast.File{file}, nil); got != nil {
		t.Fatalf("small table was stripped: %v", got)
	}
	cl := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0].(*ast.CompositeLit)
	if len(cl.Elts) != 3 {
		t.Fatalf("small table elts = %d, want 3", len(cl.Elts))
	}
}

func TestStripLargeStaticByteArrayEllipsis(t *testing.T) {
	elts := strings.Repeat("0x01, ", minStripStaticByteArray)
	src := "package p\n\nvar Table = [...]byte{" + elts + "}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	payloads := StripLargeStaticByteArrays(pkg, []*ast.File{file}, nil)
	if len(payloads[StaticByteArrayKey(pkg.Path(), "Table")]) != minStripStaticByteArray {
		t.Fatalf("ellipsis payload len = %d, want %d", len(payloads["Table"]), minStripStaticByteArray)
	}
	for i, b := range payloads[StaticByteArrayKey(pkg.Path(), "Table")] {
		if b != 1 {
			t.Fatalf("payload[%d] = %d, want 1", i, b)
		}
	}
}

func TestStripLargeStaticByteArrayIdempotent(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nvar Fixed = [%d]byte{%s}\nvar Ellipsis = [...]byte{%s}\n", n, strings.Repeat("9, ", n), strings.Repeat("7, ", n))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg := types.NewPackage("p", "p")
	first := StripLargeStaticByteArrays(pkg, files, nil)
	fixed := first[StaticByteArrayKey(pkg.Path(), "Fixed")]
	ellipsis := first[StaticByteArrayKey(pkg.Path(), "Ellipsis")]
	if len(fixed) != n || fixed[0] != 9 || fixed[n-1] != 9 {
		t.Fatalf("first Fixed payload = len %d", len(fixed))
	}
	if len(ellipsis) != n || ellipsis[0] != 7 {
		t.Fatalf("first Ellipsis payload = len %d", len(ellipsis))
	}
	second := StripLargeStaticByteArrays(pkg, files, nil)
	if second != nil {
		t.Fatalf("second strip should skip empty literals, got %d keys", len(second))
	}
	merged := MergeStaticByteArrays(first, second)
	if merged[StaticByteArrayKey(pkg.Path(), "Fixed")][0] != 9 || merged[StaticByteArrayKey(pkg.Path(), "Ellipsis")][0] != 7 {
		t.Fatal("merging a nil second strip overwrote payloads")
	}
}

func TestStripLargeStaticByteArraySkipsSparse(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nvar Table = [%d]byte{0: 1, %d: 2}\n", n, n-1)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := StripLargeStaticByteArrays(types.NewPackage("p", "p"), []*ast.File{file}, nil); got != nil {
		t.Fatalf("sparse table was stripped: %v", got)
	}
	cl := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0].(*ast.CompositeLit)
	if len(cl.Elts) != 2 {
		t.Fatalf("sparse table elts = %d, want 2", len(cl.Elts))
	}
}

func TestStripLargeStaticByteArrayDistinctPackagePaths(t *testing.T) {
	const n = minStripStaticByteArray
	srcA := fmt.Sprintf("package main\n\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("1, ", n))
	srcB := fmt.Sprintf("package main\n\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("2, ", n))
	fset := token.NewFileSet()
	fileA, err := parser.ParseFile(fset, "a.go", srcA, 0)
	if err != nil {
		t.Fatal(err)
	}
	fileB, err := parser.ParseFile(fset, "b.go", srcB, 0)
	if err != nil {
		t.Fatal(err)
	}
	first := types.NewPackage("example.com/cmd/a", "main")
	second := types.NewPackage("example.com/cmd/b", "main")
	payloads := MergeStaticByteArrays(
		StripLargeStaticByteArrays(first, []*ast.File{fileA}, nil),
		StripLargeStaticByteArrays(second, []*ast.File{fileB}, nil),
	)
	if payloads[StaticByteArrayKey(llssa.PathOf(first), "Table")] != nil {
		t.Fatal("payloads collided under PathOf(\"main\")")
	}
	gotA := payloads[StaticByteArrayKey(first.Path(), "Table")]
	gotB := payloads[StaticByteArrayKey(second.Path(), "Table")]
	if len(gotA) != n || gotA[0] != 1 || len(gotB) != n || gotB[0] != 2 {
		t.Fatalf("distinct main paths missing: a=%v b=%v", gotA[:1], gotB[:1])
	}

	patched := types.NewPackage(abi.PatchPathPrefix+"runtime", "runtime")
	src2 := fmt.Sprintf("package runtime\n\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("2, ", n))
	file2, err := parser.ParseFile(fset, "runtime.go", src2, 0)
	if err != nil {
		t.Fatal(err)
	}
	payloads = StripLargeStaticByteArrays(patched, []*ast.File{file2}, nil)
	got := payloads[StaticByteArrayKey(patched.Path(), "Table")]
	if len(got) != n || got[0] != 2 {
		t.Fatalf("patched payload path=%q len %d data[0]=%d", patched.Path(), len(got), got[0])
	}
}

func TestMergeStaticByteArraysQualifiedKeys(t *testing.T) {
	dst := map[string][]byte{StaticByteArrayKey("a", "T"): {1}}
	src := map[string][]byte{StaticByteArrayKey("b", "T"): {2}}
	got := MergeStaticByteArrays(dst, src)
	if got[StaticByteArrayKey("a", "T")][0] != 1 || got[StaticByteArrayKey("b", "T")][0] != 2 {
		t.Fatalf("qualified keys collided: %v", got)
	}
	if StaticByteArrayKey("", "T") != "T" {
		t.Fatalf("empty pkg path key = %q", StaticByteArrayKey("", "T"))
	}
	mergedNil := MergeStaticByteArrays(nil, src)
	if mergedNil[StaticByteArrayKey("b", "T")][0] != 2 {
		t.Fatalf("merge into nil dst: %v", mergedNil)
	}
}

func TestStripLargeStaticByteArraySkipsNonValues(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nconst C = 1\nvar _ = [%d]byte{%s}\nvar Uninit [%d]byte\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("1, ", n), n, n, strings.Repeat("2, ", n))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	payloads := StripLargeStaticByteArrays(pkg, []*ast.File{nil, file}, nil)
	if payloads[StaticByteArrayKey(pkg.Path(), "Table")][0] != 2 {
		t.Fatalf("Table payload = %v", payloads[StaticByteArrayKey(pkg.Path(), "Table")])
	}
	if _, ok := payloads[StaticByteArrayKey(pkg.Path(), "_")]; ok {
		t.Fatal("blank identifier was stripped")
	}
	if _, ok := payloads[StaticByteArrayKey(pkg.Path(), "Uninit")]; ok {
		t.Fatal("uninitialized array was stripped")
	}
}

func TestStaticByteArrayLitRejects(t *testing.T) {
	if _, ok := staticByteArrayLit(nil); ok {
		t.Fatal("nil literal accepted")
	}
	n := minStripStaticByteArray
	mk := func(lenExpr ast.Expr, elt ast.Expr, values ...ast.Expr) *ast.CompositeLit {
		return &ast.CompositeLit{
			Type: &ast.ArrayType{Len: lenExpr, Elt: elt},
			Elts: values,
		}
	}
	intLit := func(s string) *ast.BasicLit { return &ast.BasicLit{Kind: token.INT, Value: s} }
	many := make([]ast.Expr, n)
	for i := range many {
		many[i] = intLit("1")
	}
	if _, ok := staticByteArrayLit(mk(intLit("4096"), &ast.Ident{Name: "int"}, many...)); ok {
		t.Fatal("[N]int accepted")
	}
	if _, ok := staticByteArrayLit(&ast.CompositeLit{Type: &ast.ArrayType{Elt: &ast.Ident{Name: "byte"}}, Elts: many}); ok {
		t.Fatal("slice literal accepted")
	}
	if _, ok := staticByteArrayLit(mk(intLit("100"), &ast.Ident{Name: "byte"}, many...)); ok {
		t.Fatal("declared length below threshold accepted")
	}
	badKey := append([]ast.Expr(nil), many...)
	badKey[0] = &ast.KeyValueExpr{Key: &ast.Ident{Name: "x"}, Value: intLit("1")}
	if _, ok := staticByteArrayLit(mk(intLit("4096"), &ast.Ident{Name: "byte"}, badKey...)); ok {
		t.Fatal("non-int keyed index accepted")
	}
	oob := append([]ast.Expr(nil), many...)
	oob[0] = &ast.KeyValueExpr{Key: intLit("8192"), Value: intLit("1")}
	if _, ok := staticByteArrayLit(mk(intLit("4096"), &ast.Ident{Name: "byte"}, oob...)); ok {
		t.Fatal("out-of-range keyed index accepted")
	}
	if _, ok := astIntLit(&ast.Ident{Name: "n"}); ok {
		t.Fatal("non-literal int accepted")
	}
	if _, ok := astIntLit(&ast.BasicLit{Kind: token.INT, Value: "-1"}); ok {
		t.Fatal("negative length accepted")
	}
	if _, ok := astByteLit(&ast.Ident{Name: "x"}); ok {
		t.Fatal("non-literal byte accepted")
	}
	if _, ok := astByteLit(&ast.BasicLit{Kind: token.INT, Value: "256"}); ok {
		t.Fatal("byte overflow accepted")
	}
	if _, ok := astByteLit(&ast.BasicLit{Kind: token.STRING, Value: `"a"`}); ok {
		t.Fatal("string literal accepted as byte")
	}
	if _, ok := astByteLit(&ast.BasicLit{Kind: token.CHAR, Value: `'\x`}); ok {
		t.Fatal("broken char literal accepted")
	}

	ellipsisBad := append([]ast.Expr(nil), many...)
	ellipsisBad[0] = &ast.KeyValueExpr{Key: &ast.Ident{Name: "x"}, Value: intLit("1")}
	if _, ok := staticByteArrayLit(mk(&ast.Ellipsis{}, &ast.Ident{Name: "byte"}, ellipsisBad...)); ok {
		t.Fatal("ellipsis with non-int key accepted")
	}

	file := &ast.File{
		Name: ast.NewIdent("p"),
		Decls: []ast.Decl{
			&ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
				&ast.TypeSpec{Name: ast.NewIdent("T"), Type: ast.NewIdent("int")},
			}},
		},
	}
	if got := StripLargeStaticByteArrays(types.NewPackage("p", "p"), []*ast.File{file}, nil); got != nil {
		t.Fatalf("non-value spec stripped: %v", got)
	}
}

func TestStripLargeStaticByteArrayKeyedAndChar(t *testing.T) {
	const n = minStripStaticByteArray
	var b strings.Builder
	b.WriteString("package p\n\nvar Table = [...]byte{")
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		if i%2 == 0 {
			fmt.Fprintf(&b, "%d: 0x01", i)
		} else {
			fmt.Fprintf(&b, "%d: 'A'", i)
		}
	}
	b.WriteString("}\n")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", b.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	payloads := StripLargeStaticByteArrays(pkg, []*ast.File{file}, nil)
	got := payloads[StaticByteArrayKey(pkg.Path(), "Table")]
	if len(got) != n || got[0] != 1 || got[1] != 'A' || got[n-1] != 'A' {
		t.Fatalf("keyed/char payload len %d [0]=%d [1]=%d [%d]=%d", len(got), got[0], got[1], n-1, got[n-1])
	}
}

func TestStripLargeStaticByteArrayDecreasingKeys(t *testing.T) {
	const n = minStripStaticByteArray
	var b strings.Builder
	b.WriteString("package p\n\nvar Table = [8192]byte{4095: 7, 0: 1")
	for i := 0; i < n-2; i++ {
		b.WriteString(", 2")
	}
	b.WriteString("}\n")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", b.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	payloads := StripLargeStaticByteArrays(pkg, []*ast.File{file}, nil)
	got := payloads[StaticByteArrayKey(pkg.Path(), "Table")]
	if len(got) != 8192 || got[0] != 1 || got[1] != 2 || got[4095] != 7 || got[4096] != 0 {
		t.Fatalf("decreasing keys: len=%d [0]=%d [1]=%d [4095]=%d [4096]=%d", len(got), got[0], got[1], got[4095], got[4096])
	}

	var ellipsis strings.Builder
	ellipsis.WriteString("package p\n\nvar Table = [...]byte{4095: 7, 0: 1")
	for i := 0; i < n-2; i++ {
		ellipsis.WriteString(", 2")
	}
	ellipsis.WriteString("}\n")
	file2, err := parser.ParseFile(fset, "e.go", ellipsis.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
	got = StripLargeStaticByteArrays(pkg, []*ast.File{file2}, nil)[StaticByteArrayKey(pkg.Path(), "Table")]
	if len(got) != n || got[0] != 1 || got[1] != 2 || got[n-1] != 7 {
		t.Fatalf("ellipsis decreasing keys: len=%d [0]=%d [1]=%d [%d]=%d", len(got), got[0], got[1], n-1, got[n-1])
	}
}

func TestStripLargeStaticByteArrayRequiresByteArrayType(t *testing.T) {
	const n = minStripStaticByteArray
	check := func(src string) *ast.File {
		t.Helper()
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "p.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object)}
		conf := types.Config{Importer: gpackages.NewImporter(fset)}
		pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		if got := StripLargeStaticByteArrays(pkg, []*ast.File{file}, info); got != nil {
			t.Fatalf("unexpected strip of %q: %v", src, got)
		}
		vs := file.Decls[len(file.Decls)-1].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
		if len(vs.Values) != 1 {
			t.Fatalf("initializer removed for %q", src)
		}
		return file
	}
	check(fmt.Sprintf("package p\n\nvar Table any = [%d]byte{%s}\n", n, strings.Repeat("1, ", n)))
	check(fmt.Sprintf("package p\n\ntype byte = uint16\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("1, ", n)))
}

func TestStripLargeStaticByteArraySkipsLocality(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\n//llgointernal:gls\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("1, ", n))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	if got := StripLargeStaticByteArrays(pkg, []*ast.File{file}, nil); got != nil {
		t.Fatalf("gls table was stripped: %v", got)
	}
	vs := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(vs.Values) != 1 {
		t.Fatal("gls initializer was removed")
	}
}

func TestStripLargeStaticByteArraySkipsLinknameAlias(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nimport _ \"unsafe\"\n\n//go:linkname alias example.com/target.Table\nvar alias = [%d]byte{%s}\n", n, strings.Repeat("1, ", n))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	if got := StripLargeStaticByteArrays(pkg, []*ast.File{file}, nil); got != nil {
		t.Fatalf("linkname alias was stripped: %v", got)
	}
	vs := file.Decls[len(file.Decls)-1].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(vs.Values) != 1 {
		t.Fatal("linkname alias initializer was removed")
	}
}

func TestSkipNamesRetainingInit(t *testing.T) {
	parse := func(src string) *ast.File {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), "alt.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	got := SkipNamesRetainingInit([]*ast.File{parse("package p\n//llgo:skip Table Other\ntype marker int\nvar Table [4096]byte\n")})
	if !got["Table"] || !got["Other"] || got["init"] {
		t.Fatalf("skip names = %v", got)
	}
	if got := SkipNamesRetainingInit([]*ast.File{parse("package p\n//llgo:skipall\ntype marker int\nvar Table [4096]byte\n")}); got != nil {
		t.Fatalf("skipall still retained names: %v", got)
	}
	if got := SkipNamesRetainingInit([]*ast.File{parse("package p\n//llgo:skip init Table\ntype marker int\n")}); got != nil {
		t.Fatalf("skip init still retained names: %v", got)
	}
	if got := SkipNamesRetainingInit([]*ast.File{parse("package p\nvar Table [4096]byte\n")}); got != nil {
		t.Fatalf("no skip directive returned %v", got)
	}
}

func TestStripLargeStaticByteArraySkipsPatchSkipSet(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nvar Table = [%d]byte{%s}\nvar Other = [%d]byte{%s}\n", n, strings.Repeat("1, ", n), n, strings.Repeat("2, ", n))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		InitOrder: nil,
	}
	conf := types.Config{Importer: gpackages.NewImporter(fset)}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.InitOrder) != 2 {
		t.Fatalf("InitOrder before strip = %d, want 2", len(info.InitOrder))
	}
	payloads := StripLargeStaticByteArraysExcept(pkg, []*ast.File{file}, info, map[string]bool{"Table": true})
	if payloads[StaticByteArrayKey(pkg.Path(), "Table")] != nil {
		t.Fatal("skipped Table was stripped")
	}
	gotOther := payloads[StaticByteArrayKey(pkg.Path(), "Other")]
	if len(gotOther) != n || gotOther[0] != 2 {
		t.Fatalf("Other payload = %v", gotOther)
	}
	table := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(table.Values) != 1 {
		t.Fatal("skipped Table initializer was removed")
	}
	other := file.Decls[1].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	if len(other.Values) != 0 {
		t.Fatal("Other initializer was kept")
	}
	if len(info.InitOrder) != 1 || info.InitOrder[0].Lhs[0].Name() != "Table" {
		t.Fatalf("InitOrder after strip = %v, want Table", info.InitOrder)
	}
}

func TestStripLargeStaticByteArraySkipAllStillStrips(t *testing.T) {
	const n = minStripStaticByteArray
	alt, err := parser.ParseFile(token.NewFileSet(), "alt.go", "package p\n//llgo:skipall\ntype marker int\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if SkipNamesRetainingInit([]*ast.File{alt}) != nil {
		t.Fatal("skipall should not retain original init stores")
	}
	src := fmt.Sprintf("package p\n\nvar Table = [%d]byte{%s}\n", n, strings.Repeat("1, ", n))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	payloads := StripLargeStaticByteArraysExcept(pkg, []*ast.File{file}, nil, SkipNamesRetainingInit([]*ast.File{alt}))
	if len(payloads[StaticByteArrayKey(pkg.Path(), "Table")]) != n {
		t.Fatalf("skipall unexpectedly kept Table: %v", payloads)
	}
}

func TestStripLargeStaticByteArrayKeepsStoresWhenPatchSkipsGlobal(t *testing.T) {
	const n = minStripStaticByteArray
	origSrc := fmt.Sprintf("package p\n\nvar Table = [%d]byte{%s}\n\nfunc Use() byte { return Table[0] + Table[%d] }\n", n, strings.Repeat("1, ", n), n-1)
	altSrc := fmt.Sprintf("package p\n\n//llgo:skip Table\ntype marker int\nvar Table [%d]byte\n", n)

	strippedStores := staticByteArrayPatchStores(t, origSrc, altSrc, false)
	if strippedStores >= n {
		t.Fatalf("stripped original still has %d init stores", strippedStores)
	}
	keptStores := staticByteArrayPatchStores(t, origSrc, altSrc, true)
	if keptStores < n {
		t.Fatalf("patch-skip original has %d init stores, want at least %d", keptStores, n)
	}
}

func staticByteArrayPatchStores(t *testing.T, origSrc, altSrc string, keepSkipInit bool) int {
	t.Helper()
	fset := token.NewFileSet()
	origFile, err := parser.ParseFile(fset, "p.go", origSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	altFile, err := parser.ParseFile(fset, "alt.go", altSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: gpackages.NewImporter(fset)}
	pkg, err := conf.Check("p", fset, []*ast.File{origFile}, info)
	if err != nil {
		t.Fatal(err)
	}
	var except map[string]bool
	if keepSkipInit {
		except = SkipNamesRetainingInit([]*ast.File{altFile})
		if !except["Table"] {
			t.Fatal("expected Table in skip set")
		}
	}
	StripLargeStaticByteArraysExcept(pkg, []*ast.File{origFile}, info, except)
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	ssaPkg := prog.CreatePackage(pkg, []*ast.File{origFile}, info, true)
	ssaPkg.Build()
	return countInitStores(ssaPkg)
}

func countInitStores(pkg *ssa.Package) int {
	initFn := pkg.Func("init")
	if initFn == nil {
		return 0
	}
	stores := 0
	for _, b := range initFn.Blocks {
		for _, instr := range b.Instrs {
			if _, ok := instr.(*ssa.Store); ok {
				stores++
			}
		}
	}
	return stores
}

func TestInitStaticByteArrayGlobalKeepsPatchSkipStores(t *testing.T) {
	const n = minStripStaticByteArray
	origSrc := fmt.Sprintf("package p\n\nvar Table = [%d]byte{%s}\n\nfunc Use() byte { return Table[0] + Table[%d] }\n", n, strings.Repeat("1, ", n), n-1)
	altSrc := fmt.Sprintf("package p\n\n//llgo:skip Table\ntype marker int\nvar Table [%d]byte\n", n)
	fset := token.NewFileSet()
	origFile, err := parser.ParseFile(fset, "p.go", origSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	altFile, err := parser.ParseFile(fset, "alt.go", altSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	origInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	altInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: gpackages.NewImporter(fset)}
	origPkg, err := conf.Check("p", fset, []*ast.File{origFile}, origInfo)
	if err != nil {
		t.Fatal(err)
	}
	altPkg, err := conf.Check(abi.PatchPathPrefix+"p", fset, []*ast.File{altFile}, altInfo)
	if err != nil {
		t.Fatal(err)
	}
	except := SkipNamesRetainingInit([]*ast.File{altFile})
	payloads := StripLargeStaticByteArraysExcept(origPkg, []*ast.File{origFile}, origInfo, except)
	if payloads[StaticByteArrayKey(origPkg.Path(), "Table")] != nil {
		t.Fatal("skipped original Table was stripped")
	}
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	origSSA := prog.CreatePackage(origPkg, []*ast.File{origFile}, origInfo, true)
	altSSA := prog.CreatePackage(altPkg, []*ast.File{altFile}, altInfo, true)
	origSSA.Build()
	altSSA.Build()
	if stores := countInitStores(origSSA); stores < n {
		t.Fatalf("original init stores = %d, want at least %d", stores, n)
	}

	llprog := ssatest.NewProgramEx(t, nil, conf.Importer)
	llprog.TypeSizes(types.SizesFor("gc", "arm64"))
	ret, _, err := NewPackageExWithEmbedMetaOptions(llprog, nil, Patches{
		"p": {Alt: altSSA, Types: typepatch.Clone(altPkg)},
	}, nil, origSSA, []*ast.File{origFile, altFile}, nil, false, Options{StaticByteArrays: payloads})
	if err != nil {
		t.Fatal(err)
	}
	ir := ret.String()
	if !strings.Contains(ir, fmt.Sprintf("@p.Table = global [%d x i8]", n)) {
		t.Fatalf("missing replacement table:\n%s", ir)
	}
	if strings.Count(ir, "store i8") < n {
		t.Fatalf("compiled init lost original stores:\n%s", ir)
	}
}

func TestInitStaticByteArrayGlobalUsesObjectPackage(t *testing.T) {
	const n = minStripStaticByteArray
	src := fmt.Sprintf("package p\n\nvar Table = [%d]byte{%s}\n\nfunc Use() byte { return Table[0] + Table[%d] }\n", n, strings.Repeat("1, ", n), n-1)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: gpackages.NewImporter(fset)}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	payloads := StripLargeStaticByteArrays(pkg, []*ast.File{file}, info)
	if len(payloads[StaticByteArrayKey(pkg.Path(), "Table")]) != n {
		t.Fatalf("payload missing: %v", payloads)
	}
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	ssaPkg := prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()
	ssaPkg.Pkg = types.NewPackage("github.com/xgo-dev/llgo/runtime/internal/lib/p", "p")

	llprog := ssatest.NewProgramEx(t, nil, conf.Importer)
	llprog.TypeSizes(types.SizesFor("gc", "arm64"))
	ret, _, err := NewPackageExWithEmbedMetaOptions(llprog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{StaticByteArrays: payloads})
	if err != nil {
		t.Fatal(err)
	}
	ir := ret.String()
	if !strings.Contains(ir, fmt.Sprintf("@p.Table = global [%d x i8]", n)) {
		t.Fatalf("missing static byte array global:\n%s", ir)
	}
	if strings.Contains(ir, "zeroinitializer") {
		t.Fatalf("patched package path missed payload restore:\n%s", ir)
	}
}

func TestInitStaticByteArrayGlobalRejects(t *testing.T) {
	var g llssa.Global
	p := &context{staticByteArrays: map[string][]byte{StaticByteArrayKey("p", "T"): {1, 2, 3}}}
	if p.initStaticByteArrayGlobal(false, &ssa.Global{}, g) {
		t.Fatal("define=false accepted")
	}
	empty := &context{}
	if empty.initStaticByteArrayGlobal(true, &ssa.Global{}, g) {
		t.Fatal("empty payloads accepted")
	}
	if p.initStaticByteArrayGlobal(true, &ssa.Global{}, g) {
		t.Fatal("nil pkg accepted")
	}
}

func compileWithPayload(t *testing.T, src string, payloads map[string][]byte) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object)}
	conf := types.Config{Importer: gpackages.NewImporter(fset)}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	ssaPkg := prog.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()
	llprog := ssatest.NewProgramEx(t, nil, conf.Importer)
	llprog.TypeSizes(types.SizesFor("gc", "arm64"))
	if _, _, err := NewPackageExWithEmbedMetaOptions(llprog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{StaticByteArrays: payloads}); err != nil {
		t.Fatal(err)
	}
}

func TestInitStaticByteArrayGlobalTypeMismatch(t *testing.T) {
	payload := map[string][]byte{StaticByteArrayKey("p", "Table"): {1, 2, 3, 4}}
	compileWithPayload(t, "package p\nvar Table int\n", payload)
	compileWithPayload(t, "package p\nvar Table [3]byte\n", payload)
	compileWithPayload(t, "package p\nvar Table [4]int\n", payload)
}
