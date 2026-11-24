package directive

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

func commentGroup(lines ...string) *ast.CommentGroup {
	list := make([]*ast.Comment, len(lines))
	for i, line := range lines {
		list[i] = &ast.Comment{Slash: token.Pos(i + 1), Text: line}
	}
	return &ast.CommentGroup{List: list}
}

func TestParseAndGroup(t *testing.T) {
	doc := commentGroup(
		"// ordinary comment",
		"//go:noinline",
		"//llgo:cdecl",
		"// llgo:link Foo C.foo",
		"//llgointernal:tls",
		"//export Bar",
	)
	got := ParseGroup(doc)
	if len(got) != 5 {
		t.Fatalf("ParseGroup len = %d, want 5: %+v", len(got), got)
	}
	if got[0].Name != "go:noinline" || got[1].Name != "llgo:cdecl" ||
		got[2].Name != "llgo:link" || got[2].Args != "Foo C.foo" ||
		got[3].Name != "llgointernal:tls" || got[4].Name != "export" || got[4].Args != "Bar" {
		t.Fatalf("unexpected directives: %+v", got)
	}
	local, target := ParseLinknameArgs(got[2].Args)
	if local != "Foo" || target != "C.foo" {
		t.Fatalf("ParseLinknameArgs = (%q, %q)", local, target)
	}
}

func TestParseTypeCallConv(t *testing.T) {
	res, err := ParseTypeCallConv(commentGroup("//llgo:cdecl"))
	if err != nil || res.CallConv != CallConvC || res.Deprecated {
		t.Fatalf("cdecl = %+v, %v", res, err)
	}
	res, err = ParseTypeCallConv(commentGroup("//llgo:type stdcall"))
	if err != nil || res.CallConv != CallConvStdcall || !res.Deprecated || res.Replacement != "//llgo:stdcall" {
		t.Fatalf("legacy stdcall = %+v, %v", res, err)
	}
	if _, err := ParseTypeCallConv(commentGroup("//llgo:cdecl extra")); err == nil {
		t.Fatal("expected error for cdecl with args")
	}
	if _, err := ParseTypeCallConv(commentGroup("//llgo:cdecl", "//llgo:stdcall")); err == nil {
		t.Fatal("expected error for conflicting calling conventions")
	}
}

func TestParseLocality(t *testing.T) {
	loc, _, err := ParseLocality(commentGroup("//llgointernal:gls"))
	if err != nil || loc != LocalityGoroutine || loc.String() != "gls" {
		t.Fatalf("gls = %v, %v", loc, err)
	}
	if _, _, err := ParseLocality(commentGroup("//llgo:tls")); err == nil {
		t.Fatal("expected error for legacy //llgo:tls")
	}
	if _, _, err := ParseLocality(commentGroup("//llgointernal:tls", "//llgointernal:gls")); err == nil {
		t.Fatal("expected error for conflicting locality")
	}
}

func TestParseSemanticHelpers(t *testing.T) {
	doc := commentGroup(
		"//go:linkname foo C.foo",
		"//llgo:link bar llvm.ctpop",
		"//go:wasmimport env write",
		"//export Exported",
		"//llgo:env",
		"//llgo:skip A B",
		"//llgo:skipall",
	)
	links := ParseLinks(doc)
	if len(links) != 2 || links[0].Target != "C.foo" || links[1].Target != "llvm.ctpop" {
		t.Fatalf("ParseLinks = %+v", links)
	}
	wi, ok, err := ParseWasmImport(doc)
	if err != nil || !ok || wi.Module != "env" || wi.Name != "write" {
		t.Fatalf("ParseWasmImport = %+v, %v, %v", wi, ok, err)
	}
	if _, ok, err := ParseWasmImport(commentGroup("// ordinary comment")); ok || err != nil {
		t.Fatalf("absent ParseWasmImport = (%v, %v), want (false, nil)", ok, err)
	}
	// Last //go:wasmimport wins even when malformed, overriding earlier valid entries.
	if _, ok, err := ParseWasmImport(commentGroup("//go:wasmimport env write", "//go:wasmimport onlyone")); !ok || err == nil {
		t.Fatalf("malformed-last ParseWasmImport = (%v, %v), want (true, non-nil err)", ok, err)
	}
	if exp, _, ok := ParseExport(doc); !ok || exp != "Exported" {
		t.Fatalf("ParseExport = %q, %v", exp, ok)
	}
	if !HasClosureEnv(doc) {
		t.Fatal("HasClosureEnv = false")
	}
	skip := ParseSkip(doc)
	if !skip.All || !reflect.DeepEqual(skip.Names, []string{"A", "B"}) {
		t.Fatalf("ParseSkip = %+v", skip)
	}
}

func TestPackageKindAndFilesFromScope(t *testing.T) {
	cases := []struct {
		in    string
		kind  PackageKind
		param string
	}{
		{"link", PkgLinkIR, ""},
		{"decl", PkgDeclOnly, ""},
		{"noinit", PkgNoInit, ""},
		{"link: hello.a", PkgLinkExtern, " hello.a"},
		{"py.numpy", PkgPyModule, "numpy"},
		{"decl: test.ll", PkgDeclOnly, " test.ll"},
		{"", PkgLLGo, ""},
		{"llgo:ignored", PkgLLGo, ""},
		{"custom", PkgLLGo, ""},
	}
	for _, tc := range cases {
		kind, param := ParsePackageKind(tc.in)
		if kind != tc.kind || param != tc.param {
			t.Errorf("ParsePackageKind(%q) = (%v, %q), want (%v, %q)", tc.in, kind, param, tc.kind, tc.param)
		}
	}

	pkg := types.NewPackage("example.com/c", "c")
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "LLGoPackage", types.Typ[types.UntypedString], constant.MakeString("link: -lfoo")))
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "LLGoFiles", types.Typ[types.UntypedString], constant.MakeString("a.c; b.c")))

	kind, param := PackageKindFromScope(pkg.Scope())
	if kind != PkgLinkExtern || param != " -lfoo" {
		t.Fatalf("PackageKindFromScope = (%v, %q)", kind, param)
	}
	files, ok := LLGoFiles(pkg.Scope())
	if !ok || files != "a.c; b.c" {
		t.Fatalf("LLGoFiles = (%q, %v)", files, ok)
	}
}
