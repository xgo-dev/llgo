package cl

import (
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestReflectTypeMethodRetainsCallableDemand(t *testing.T) {
	const source = `package constiface
import (
    "reflect"
    "unsafe"
)
var saved reflect.Method
var savedPtr *reflect.Method
func consume(reflect.Method) {}
func consumePtr(*reflect.Method) {}
func name(t reflect.Type, i int) any { return t.Method(i).Name }
func fields(t reflect.Type, i int) any { m := t.Method(i); return []any{m.Name, m.PkgPath, m.Type, m.Index} }
func named(t reflect.Type, i int) any { m, ok := t.MethodByName("M"); if ok { return m.Type }; return nil }
func dynamic(t reflect.Type, i int) any { m, ok := t.MethodByName(saved.Name); if ok { return m.Name }; return nil }
func exists(t reflect.Type, i int) any { _, ok := t.MethodByName("M"); return ok }
func discarded(t reflect.Type, i int) any { t.Method(i); return nil }
func copied(t reflect.Type, i int) any { m := t.Method(i); n := m; return n.Type }
func merged(t reflect.Type, i int) any { m := t.Method(i); if i == 0 { m = reflect.Method{} }; return m.Name }
func funcValid(t reflect.Type, i int) any { return t.Method(i).Func.IsValid() }
func funcInterface(t reflect.Type, i int) any { return t.Method(i).Func.Interface() }
func funcCall(t reflect.Type, i int) any { return t.Method(i).Func.Call(nil) }
func aggregate(t reflect.Type, i int) any { return t.Method(i) }
func argument(t reflect.Type, i int) any { consume(t.Method(i)); return nil }
func global(t reflect.Type, i int) any { saved = t.Method(i); return nil }
func ptrEscape(t reflect.Type, i int) any { m := t.Method(i); consumePtr(&m); return nil }
func ptrGlobal(t reflect.Type, i int) any { m := t.Method(i); savedPtr = &m; return nil }
func mixed(t reflect.Type, i int) any { m := t.Method(i); if i == 0 { return m.Func }; return m.Name }
func metadataAddr(t reflect.Type, i int) any { m := t.Method(i); p := &m.Name; return *p }
func metadataAddrEscape(t reflect.Type, i int) any { m := t.Method(i); return &m.Name }
func funcAddr(t reflect.Type, i int) any { m := t.Method(i); p := &m.Func; return p.IsValid() }
func unsafeWhole(t reflect.Type, i int) any { m := t.Method(i); p := (*reflect.Method)(unsafe.Pointer(&m)); return p.Func.IsValid() }
func unsafeFuncOffset(t reflect.Type, i int) any { m := t.Method(i); p := (*reflect.Value)(unsafe.Add(unsafe.Pointer(&m), unsafe.Offsetof(m.Func))); return p.IsValid() }
func unsafeFieldOffset(t reflect.Type, i int) any { m := t.Method(i); p := (*reflect.Value)(unsafe.Add(unsafe.Pointer(&m.Name), unsafe.Offsetof(m.Func)-unsafe.Offsetof(m.Name))); return p.IsValid() }
func unsafeUintptr(t reflect.Type, i int) any { m := t.Method(i); p := (*reflect.Value)(unsafe.Pointer(uintptr(unsafe.Pointer(&m)) + unsafe.Offsetof(m.Func))); return p.IsValid() }
func unsafeLayoutOnly(t reflect.Type, i int) any { m := t.Method(i); return unsafe.Sizeof(m) + unsafe.Offsetof(m.Func) }
`
	cases := []string{
		"name",
		"fields",
		"named",
		"dynamic",
		"exists",
		"discarded",
		"copied",
		"merged",
		"funcValid",
		"funcInterface",
		"funcCall",
		"aggregate",
		"argument",
		"global",
		"ptrEscape",
		"ptrGlobal",
		"mixed",
		"metadataAddr",
		"metadataAddrEscape",
		"funcAddr",
		"unsafeWhole",
		"unsafeFuncOffset",
		"unsafeFieldOffset",
		"unsafeUintptr",
		"unsafeLayoutOnly",
	}
	for _, debug := range []bool{false, true} {
		ssapkg := buildConstIfaceSSA(t, source, debug)
		for _, name := range cases {
			t.Run(name+map[bool]string{false: "", true: "/debug"}[debug], func(t *testing.T) {
				fn := ssapkg.Func(name)
				var call *ssa.Call
				for _, block := range fn.Blocks {
					for _, instr := range block.Instrs {
						c, ok := instr.(*ssa.Call)
						if ok && c.Call.Method != nil && isReflectType(c.Call.Value.Type()) &&
							(c.Call.Method.Name() == "Method" || c.Call.Method.Name() == "MethodByName") {
							call = c
						}
					}
				}
				if call == nil {
					t.Fatal("missing reflect method lookup")
				}
				for _, drop := range []bool{false, true} {
					prog := llssa.NewProgram(nil)
					prog.EnableDeadcodeDrop(drop)
					pkg := prog.NewPackageEx("pkg", "pkg", true)
					llfn := pkg.NewFunc("pkg.caller", types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.GoDecl)
					ctx := &context{pkg: pkg, fn: llfn, goFn: fn}
					check := ctx.reflectTypeMethodCheck(&call.Call, call.Call.Method)
					if check.Kind == 0 || pkg.NeedAbiInit != check.Kind {
						t.Fatal("metadata-only lookup lost ABI initialization/check flags")
					}
					if err := pkg.FinishMetaCollection(); err != nil {
						t.Fatal(err)
					}
					meta := pkg.Meta.String()
					pkg.Meta.Close()
					prog.Dispose()
					if strings.Contains(meta, "[ReflectMethodMetadata]") {
						t.Fatalf("retired metadata-only demand was emitted:\n%s", meta)
					}
					if drop {
						want := "[Reflect]"
						if check.Kind == llssa.ReflectTypeMethodByName {
							want = "[UseNamedMethod]"
						}
						if !strings.Contains(meta, want) {
							t.Fatalf("lookup lost callable demand %s:\n%s", want, meta)
						}
					}
				}
			})
		}
	}
}
