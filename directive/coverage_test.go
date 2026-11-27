package directive

import (
	"go/ast"
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
	if _, err := ParseTypeCallConv(commentGroup("//llgo:type C", "//llgo:cdecl")); err == nil {
		t.Fatal("expected error for duplicate calling convention")
	}
	if res, err := ParseTypeCallConv(commentGroup("//llgo:type Python")); err != nil || res.CallConv != CallConvNone {
		t.Fatalf("unrelated //llgo:type = (%+v, %v)", res, err)
	}
	if _, _, err := ParseLocality(commentGroup("//llgo:threadlocal")); err == nil {
		t.Fatal("expected error for legacy //llgo:threadlocal")
	}
	if _, _, err := ParseLocality(commentGroup("//llgo:goroutinelocal")); err == nil {
		t.Fatal("expected error for legacy //llgo:goroutinelocal")
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
}
