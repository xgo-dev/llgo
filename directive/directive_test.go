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
	wi, ok := ParseWasmImport(doc)
	if !ok || wi.Module != "env" || wi.Name != "write" {
		t.Fatalf("ParseWasmImport = %+v, %v", wi, ok)
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
	pkg := types.NewPackage("example.com/c", "c")
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "LLGoPackage", types.Typ[types.UntypedString], constant.MakeString("link: -lfoo")))
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "LLGoFiles", types.Typ[types.UntypedString], constant.MakeString("a.c; b.c")))

	kind, param := PackageKindFromScope(pkg.Scope())
	if kind != PkgLinkExtern || param != "-lfoo" {
		t.Fatalf("PackageKindFromScope = (%v, %q)", kind, param)
	}
	files, ok := LLGoFiles(pkg.Scope())
	if !ok || files != "a.c; b.c" {
		t.Fatalf("LLGoFiles = (%q, %v)", files, ok)
	}
}
