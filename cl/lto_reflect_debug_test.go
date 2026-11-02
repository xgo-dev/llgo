package cl_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
	llvmenv "github.com/xgo-dev/llgo/xtool/env/llvm"
)

func TestLTOPluginReflectFoldedReturnLength(t *testing.T) {
	conf := testltoLTOPluginConf(t, build.ModeGen)
	// This is the x86-64 pre-LTO shape with optimized DWARF: IPSCCP folds
	// the length of a noinline helper's return, but keeps its pointer extract.
	const input = `
target datalayout = "e-p:64:64-i64:64-n8:16:32:64-S128"
%value = type { ptr, ptr, i64 }
@name.a = private constant [5 x i8] c"KeepA"
@name.b = private constant [5 x i8] c"KeepB"

define internal { ptr, i64 } @methodName(i1 %choice) noinline {
  %p = select i1 %choice, ptr @name.a, ptr @name.b
  %s0 = insertvalue { ptr, i64 } poison, ptr %p, 0
  %s = insertvalue { ptr, i64 } %s0, i64 5, 1
  ret { ptr, i64 } %s
}
declare void @methodByName(ptr sret(%value), ptr, i64)
declare { ptr, i1 } @llvm.type.checked.load(ptr, i32, metadata)
declare void @llvm.assume(i1)
declare void @consume(ptr)

define void @entry(i1 %choice, i64 %unknown) {
  %ret = alloca %value
  %name = call { ptr, i64 } @methodName(i1 %choice)
  %p = extractvalue { ptr, i64 } %name, 0
  %n = extractvalue { ptr, i64 } %name, 1
  call void @methodByName(ptr sret(%value) %ret, ptr "llgo.reflect.methodbyname.name"="1" %p, i64 NAME_LENGTH) #0
  %slot = getelementptr %value, ptr %ret, i32 0, i32 1
  %method = load ptr, ptr %slot
  %checked = call { ptr, i1 } @llvm.type.checked.load(ptr %method, i32 0, metadata !"go.method.value.reflect")
  %ok = extractvalue { ptr, i1 } %checked, 1
  call void @llvm.assume(i1 %ok)
  call void @consume(ptr %method)
  ret void
}
attributes #0 = { "llgo.reflect.methodbyname"="value" }
`
	for _, tt := range []struct {
		name   string
		length string
		known  bool
		mixed  bool
	}{
		{"paired_extracts", "%n", true, false},
		{"folded_length", "5", true, false},
		{"mismatched_length", "4", false, false},
		{"unknown_length", "%unknown", false, false},
		{"different_candidate_lengths", "5", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := strings.Replace(input, "NAME_LENGTH", tt.length, 1)
			if tt.mixed {
				source = strings.Replace(source, `[5 x i8] c"KeepB"`, `[6 x i8] c"KeepBB"`, 1)
				source = strings.Replace(source,
					"%s = insertvalue { ptr, i64 } %s0, i64 5, 1",
					"%len = select i1 %choice, i64 5, i64 6\n  %s = insertvalue { ptr, i64 } %s0, i64 %len, 1", 1)
			}
			opt := filepath.Join(llvmenv.New("").BinDir(), "opt")
			cmd := exec.Command(opt, "-load-pass-plugin="+conf.LTOPlugin.Path,
				"-passes=llgo-lto-pre-globaldce", "-verify-each", "-S", "-o", "-")
			cmd.Stdin = strings.NewReader(source)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run folded string-length refinement: %v\n%s", err, out)
			}
			ir := string(out)
			generic := strings.Contains(ir, `metadata !"go.method.value.reflect"`)
			if generic == tt.known {
				t.Fatalf("generic method check present = %v, want %v\n%s", generic, !tt.known, ir)
			}
			for _, name := range []string{"KeepA", "KeepB"} {
				if got := strings.Contains(ir, `metadata !"go.method.value.reflect.`+name+`"`); got != tt.known {
					t.Fatalf("refined check for %s = %v, want %v\n%s", name, got, tt.known, ir)
				}
			}
		})
	}
}
