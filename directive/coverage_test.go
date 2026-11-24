package directive

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"testing"
)

func TestDirectiveSemanticsEdgeCases(t *testing.T) {
	if CallConvC.String() != "C" || CallConvStdcall.String() != "stdcall" || CallConvNone.String() != "" {
		t.Fatalf("unexpected CallConv.String() values: %q %q %q", CallConvC.String(), CallConvStdcall.String(), CallConvNone.String())
	}
	if LocalityThread.String() != "tls" || LocalityGoroutine.String() != "gls" || LocalityNone.String() != "" {
		t.Fatalf("unexpected Locality.String() values: %q %q %q", LocalityThread.String(), LocalityGoroutine.String(), LocalityNone.String())
	}

	if _, ok := Parse(&ast.Comment{Text: "// not a directive"}); ok {
		t.Fatal("plain comment parsed as directive")
	}
	if _, ok := Parse(nil); ok {
		t.Fatal("nil comment parsed as directive")
	}
	if ParseGroup(nil) != nil {
		t.Fatal("ParseGroup(nil) != nil")
	}
	if _, err := ParseTypeCallConv(commentGroup("//llgo:cdecl extra")); err == nil {
		t.Fatal("expected error for //llgo:cdecl with args")
	}
	if _, err := ParseTypeCallConv(commentGroup("//llgo:stdcall extra")); err == nil {
		t.Fatal("expected error for //llgo:stdcall with args")
	}
	if _, err := ParseTypeCallConv(commentGroup("//llgo:cdecl", "//llgo:stdcall")); err == nil {
		t.Fatal("expected error for conflicting calling conventions")
	}
	if _, _, err := ParseLocality(commentGroup("//llgo:tls")); err == nil {
		t.Fatal("expected error for legacy //llgo:tls")
	}
	if _, _, err := ParseLocality(commentGroup("//llgo:gls")); err == nil {
		t.Fatal("expected error for legacy //llgo:gls")
	}
	if _, _, err := ParseLocality(commentGroup("//llgointernal:tls extra")); err == nil {
		t.Fatal("expected error for //llgointernal:tls with args")
	}
	if _, _, err := ParseLocality(commentGroup("//llgointernal:tls", "//llgointernal:gls")); err == nil {
		t.Fatal("expected error for conflicting locality directives")
	}
	if name, _, ok := ParseExport(commentGroup("//export MyFn")); !ok || name != "MyFn" {
		t.Fatalf("ParseExport = (%q, %v)", name, ok)
	}
	if _, _, ok := ParseExport(commentGroup("//export")); ok {
		t.Fatal("empty //export unexpectedly parsed")
	}
	if HasClosureEnv(nil) {
		t.Fatal("HasClosureEnv(nil) = true")
	}

	pkg := types.NewPackage("example.com/clib", "clib")
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "LLGoPackage", types.Typ[types.UntypedString], constant.MakeString("link: -lm")))
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "LLGoFiles", types.Typ[types.UntypedString], constant.MakeString("a.c;b.c")))
	kind, param := PackageKindFromScope(pkg.Scope())
	if kind != PkgLinkExtern || param != " -lm" {
		t.Fatalf("PackageKindFromScope = (%v, %q), want (PkgLinkExtern, \" -lm\")", kind, param)
	}
	if got, ok := LLGoFiles(pkg.Scope()); !ok || got != "a.c;b.c" {
		t.Fatalf("LLGoFiles() = (%q, %v), want (a.c;b.c, true)", got, ok)
	}
	if kind, param := PackageKindFromScope(nil); kind != PkgNormal || param != "" {
		t.Fatalf("PackageKindFromScope(nil) = (%v, %q)", kind, param)
	}
	if got, ok := LLGoFiles(nil); ok || got != "" {
		t.Fatalf("LLGoFiles(nil) = (%q, %v)", got, ok)
	}
}
