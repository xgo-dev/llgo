package llvmfmv

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestRegisterImportAvailability(t *testing.T) {
	const ir = `target triple = "x86_64-unknown-linux-gnu"
declare void @pkg.Add(ptr sret(<8 x float>), ptr) "llgo.fmv.wide-origin"="pkg.Add" "llgo.fmv.register-owner"="pkg"
define void @root(ptr %out, ptr %in) "target-features"="+avx,+avx2" {
 call void @pkg.Add(ptr sret(<8 x float>) %out, ptr %in)
 ret void
}`
	const name = "pkg.Add.__llgo_fmv_reg256_go"
	for _, tc := range []struct {
		p    RegisterProvider
		want bool
	}{
		{RegisterProvider{}, false},
		{RegisterProvider{Symbols: []string{"other"}}, false},
		{RegisterProvider{Fresh: true}, true},
		{RegisterProvider{Symbols: []string{name}}, true},
	} {
		m := parseModule(t, ir)
		v := llvm.VectorType(m.Context().FloatType(), 8)
		s := WideCalls{"pkg.Add": {llvm.FunctionType(v, []llvm.Type{v}, false), 256}}
		if err := s.Promote(m, "v1", map[string]RegisterProvider{"pkg": tc.p}); err != nil {
			t.Fatal(err)
		}
		if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(m.NamedFunction("root").String(), name) != tc.want {
			t.Fatal(m.String())
		}
	}
}
