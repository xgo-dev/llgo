package llvmfmv

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Exercise positive and negative outcomes. Unknown arms and loop backedges
// cannot establish a feature unless every feasible way to that outcome does.
func TestMaterializedGuards(t *testing.T) {
	for _, tc := range []struct {
		name, cond string
		want       bool
	}{
		{"select", "%c = select i1 %flag, i1 %wide, i1 false", true},
		{"select alternative", "%c = select i1 %flag, i1 %wide, i1 true", false},
		{"select identical", "%c = select i1 %flag, i1 %wide, i1 %wide", true},
		{"equality", "%c = icmp eq i1 %wide, true", true},
		{"inequality", "%c = icmp ne i1 false, %wide", true},
		{"wrong polarity", "%c = icmp eq i1 %wide, false", false},
		{"nonboolean", "%c = icmp eq i32 %number, 1", false},
		{"de Morgan", "%not = xor i1 %wide,true\n %any = or i1 %not,%flag\n %c = xor i1 %any,true", true},
		{"mixed and", "%c = and i1 %wide, %flag", true},
		{"mixed or", "%c = or i1 %wide, %flag", false},
		{"short circuit", "br i1 %wide,label %rhs,label %join\nrhs: br label %join\njoin: %c = phi i1 [false,%entry],[%flag,%rhs]", true},
		{"bypass", "br i1 %wide,label %rhs,label %join\nrhs: br label %join\njoin: %c = phi i1 [true,%entry],[%flag,%rhs]", false},
		{"loop carried", "br label %loop\nloop: %c = phi i1 [%flag,%entry],[%wide,%loop]\n br i1 %flag,label %loop,label %done\ndone:", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
 declare i1 @wide() "llgo.cpu.query"="x86.avx512"
 define void @root(ptr %p,i1 %flag,i32 %number){
 entry: %wide = call i1 @wide()
 `+tc.cond+`
 br i1 %c,label %vec,label %exit
 vec: %v = load <16 x i32>,ptr %p
 %d = add <16 x i32> %v,%v
 store <16 x i32> %d,ptr %p
 br label %exit
 exit: ret void
 }`)
			if err := Run(m, "v1"); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			fn := m.NamedFunction("root.__llgo_fmv_guard_avx512")
			if tc.want == fn.IsNil() {
				t.Fatal(m.String())
			}
			if tc.want && !strings.Contains(stringAttribute(fn, "target-features"), "+avx512f") {
				t.Fatal(fn.String())
			}
		})
	}
}
