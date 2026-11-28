package directive

import (
	"go/ast"
	"go/token"
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
	if _, ok := Parse(nil); ok {
		t.Fatal("nil comment parsed as a directive")
	}
	if ParseGroup(nil) != nil {
		t.Fatal("nil comment group returned directives")
	}
	for _, tc := range []struct {
		text, name, args string
		ok               bool
	}{
		{text: "// ordinary"},
		{text: "//go:"},
		{text: "// llgo:type C", name: "llgo:type", args: "C", ok: true},
		{text: "// llgo:stdcall", name: "llgo:stdcall", ok: true},
		{text: "//llgo:link\tF C.f", name: "llgo:link", args: "F C.f", ok: true},
	} {
		parsed, ok := Parse(&ast.Comment{Text: tc.text})
		if ok != tc.ok || parsed.Name != tc.name || parsed.Args != tc.args {
			t.Fatalf("Parse(%q) = %+v, %v", tc.text, parsed, ok)
		}
	}
	for _, tc := range []struct{ args, local, target string }{
		{"", "", ""},
		{" Exported ", "Exported", ""},
		{" F C.f ", "F", "C.f"},
		{"F\tllvm.fshl.i8", "F", "llvm.fshl.i8"},
		{"(*T).Method\t C.method", "(*T).Method", "C.method"},
		{"F\t target  with\tspaces", "F", "target with spaces"},
	} {
		l, tgt := ParseLinknameArgs(tc.args)
		if l != tc.local || tgt != tc.target {
			t.Errorf("ParseLinknameArgs(%q) = (%q, %q), want (%q, %q)", tc.args, l, tgt, tc.local, tc.target)
		}
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
	if res, err := ParseTypeCallConv(commentGroup("//llgo:cdecl extra")); err == nil || !res.Pos.IsValid() {
		t.Fatalf("expected error with pos for cdecl with args, got (%+v, %v)", res, err)
	}
	if res, err := ParseTypeCallConv(commentGroup("//llgo:cdecl", "//llgo:stdcall")); err == nil || !res.Pos.IsValid() {
		t.Fatalf("expected error with pos for conflicting calling conventions, got (%+v, %v)", res, err)
	}
	if res, err := ParseTypeCallConv(commentGroup("//llgo:cdecl", "//llgo:cdecl")); err == nil || !res.Pos.IsValid() {
		t.Fatalf("expected error with pos for duplicate calling convention, got (%+v, %v)", res, err)
	}
}

func TestParseLocality(t *testing.T) {
	loc, _, err := ParseLocality(commentGroup("//llgointernal:gls"))
	if err != nil || loc != LocalityGoroutine || loc.String() != "gls" {
		t.Fatalf("gls = %v, %v", loc, err)
	}
	if _, pos, err := ParseLocality(commentGroup("//llgo:threadlocal")); err == nil || !pos.IsValid() {
		t.Fatalf("expected error with pos for legacy //llgo:threadlocal, got (%v, %v)", pos, err)
	}
	if _, pos, err := ParseLocality(commentGroup("//llgo:goroutinelocal")); err == nil || !pos.IsValid() {
		t.Fatalf("expected error with pos for legacy //llgo:goroutinelocal, got (%v, %v)", pos, err)
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

	flagsDoc := commentGroup(
		"//go:wasmexport hostMain",
		"//go:noinline",
		"//go:noescape",
		"//go:nointerface",
		"//go:uintptrescapes",
		"//go:nosplit",
	)
	if name, _, ok := ParseWasmExport(flagsDoc); !ok || name != "hostMain" {
		t.Fatalf("ParseWasmExport = (%q, %v)", name, ok)
	}
	if !HasNoInline(flagsDoc) || !HasNoEscape(flagsDoc) || !HasNoInterface(flagsDoc) || !HasUintptrEscapes(flagsDoc) || !HasNoSplit(flagsDoc) {
		t.Fatalf("directive flags mismatch: noinline=%v noescape=%v nointerface=%v uintptrescapes=%v nosplit=%v",
			HasNoInline(flagsDoc), HasNoEscape(flagsDoc), HasNoInterface(flagsDoc), HasUintptrEscapes(flagsDoc), HasNoSplit(flagsDoc))
	}
	mixedNoIfaceDoc := commentGroup(
		"//go:nointerface",
		"// Trailing prose comment after directive in the same comment group.",
	)
	if !HasNoInterface(mixedNoIfaceDoc) {
		t.Fatal("HasNoInterface returned false for //go:nointerface followed by prose comment in same group")
	}
	for _, tc := range []struct {
		target string
		ns     LinkNamespace
		sym    string
	}{
		{"C.printf", LinkC, "printf"},
		{"stdcall.ExitProcess", LinkStdcall, "ExitProcess"},
		{"py.print", LinkPy, "print"},
		{"llgo.alloca", LinkLLGo, "alloca"},
		{"llvm.ctpop.i32", LinkLLVM, "llvm.ctpop.i32"},
		{"runtime.mallocgc", LinkGo, "runtime.mallocgc"},
	} {
		ns, sym := ParseLinkTarget(tc.target)
		if ns != tc.ns || sym != tc.sym {
			t.Errorf("ParseLinkTarget(%q) = (%v, %q), want (%v, %q)", tc.target, ns, sym, tc.ns, tc.sym)
		}
	}
}

func TestParsePackageKind(t *testing.T) {
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
}
