package heap2stack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func tailModule(marker, path string) string {
	use := "%p"
	derive := ""
	switch path {
	case "cross-block-gep":
		derive = "br label %use\nuse:\n%q = getelementptr i8, ptr %p, i64 8\nstore i64 42, ptr %q, align 8\n"
		use = "%q"
	case "phi":
		derive = "br i1 %choice, label %left, label %right\nleft:\nbr label %use\nright:\nbr label %use\nuse:\n%q = phi ptr [%p, %left], [%unused, %right]\n"
		use = "%q"
	case "select":
		derive = "%q = select i1 %choice, ptr %p, ptr %unused\n"
		use = "%q"
	case "cyclic-phi":
		derive = "br label %loop\nloop:\n%q = phi ptr [%p, %entry], [%r, %loop]\n%r = getelementptr i8, ptr %q, i64 0\nbr i1 %choice, label %use, label %loop\nuse:\n"
		use = "%q"
	case "returned":
		derive = "%q = call ptr @identity(ptr %p)\n"
		use = "%q"
	case "spill":
		derive = "%slot = alloca ptr\nstore ptr %p, ptr %slot\n%q = load ptr, ptr %slot\n"
		use = "%q"
	}
	return fmt.Sprintf(`
declare nonnull ptr @malloc(i64) allocsize(0)
declare i64 @take(ptr captures(none), i1) nofree
declare ptr @identity(ptr returned captures(none)) nofree memory(none)
define i64 @caller(ptr %%unused, i1 %%choice) {
entry:
  %%p = call ptr @malloc(i64 16) allockind("alloc,uninitialized")
  store i64 42, ptr %%p, align 8
  %s
  %%v = %s call i64 @take(ptr %s, i1 %%choice)
  ret i64 %%v
}
`, derive, marker, use)
}

func tailTarget(t *testing.T, mod llvm.Module) llvm.TargetMachine {
	t.Helper()
	llvm.InitializeNativeTarget()
	llvm.InitializeNativeAsmPrinter()
	triple := llvm.DefaultTargetTriple()
	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		t.Fatal(err)
	}
	tm := target.CreateTargetMachine(triple, "", "", llvm.CodeGenLevelAggressive, llvm.RelocPIC, llvm.CodeModelDefault)
	t.Cleanup(tm.Dispose)
	data := tm.CreateTargetData()
	defer data.Dispose()
	mod.SetTarget(triple)
	mod.SetDataLayout(data.String())
	return tm
}

func TestTailAllocationPlacement(t *testing.T) {
	for _, marker := range []string{"tail", "musttail"} {
		for _, path := range []string{"direct", "cross-block-gep", "phi", "select", "cyclic-phi", "returned", "spill"} {
			t.Run(marker+"/"+path, func(t *testing.T) {
				mod := parseModule(t, tailModule(marker, path))
				tm := tailTarget(t, mod)
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if err := Run(mod, tm); err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				body := mod.NamedFunction("caller").String()
				if !strings.Contains(body, "@malloc(") {
					t.Errorf("tail-call argument lost heap lifetime:\n%s", body)
				}
				if !strings.Contains(body, marker+" call") {
					t.Errorf("tail marker changed:\n%s", body)
				}
			})
		}
	}
	// Exclusions are per allocation: ordinary calls and independent objects
	// in a function containing a tail call must still be eligible.
	t.Run("independent", func(t *testing.T) {
		mod := parseModule(t, `
@sink = external global i64
declare nonnull ptr @malloc(i64) allocsize(0)
declare i64 @take(ptr captures(none), i1) nofree
define i64 @caller(ptr %unused, i1 %choice) {
  %p = call ptr @malloc(i64 16) allockind("alloc,uninitialized")
  %q = call ptr @malloc(i64 16) allockind("alloc,uninitialized")
  store atomic i64 42, ptr %q seq_cst, align 8
  %value = load atomic i64, ptr %q seq_cst, align 8
  store i64 %value, ptr @sink
  store i64 42, ptr %p, align 8
  %result = musttail call i64 @take(ptr %p, i1 %choice)
  ret i64 %result
}
`)
		tm := tailTarget(t, mod)
		if err := Run(mod, tm); err != nil {
			t.Fatal(err)
		}
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		body := mod.NamedFunction("caller").String()
		if strings.Count(body, "@malloc(") != 1 || !strings.Contains(body, "alloca ") {
			t.Fatalf("must retain p on heap and promote independent q:\n%s", body)
		}
	})
	for _, marker := range []string{"", "notail"} {
		t.Run("ordinary/"+marker, func(t *testing.T) {
			mod := parseModule(t, tailModule(marker, "direct"))
			tm := tailTarget(t, mod)
			if err := Run(mod, tm); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := mod.NamedFunction("caller").String()
			if strings.Contains(body, "@malloc(") || !strings.Contains(body, "alloca ") {
				t.Fatalf("ordinary call unnecessarily excluded:\n%s", body)
			}
		})
	}
}

// VerifyModule accepts the broken stack-lifetime combination. Execute both
// the original and transformed module against a separately compiled callee
// that overwrites its stack, so a pointer into the released caller frame is
// observably wrong. Object emission uses the binding's matching LLVM version;
// clang only compiles the C harness and links the native object.
func TestTailAllocationExecution(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("64-bit native execution regression")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is unavailable for the native C harness")
	}
	for _, marker := range []string{"tail", "musttail"} {
		for _, transformed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/transformed=%v", marker, transformed), func(t *testing.T) {
				mod := parseModule(t, tailModule(marker, "direct"))
				tm := tailTarget(t, mod)
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if transformed {
					if err := Run(mod, tm); err != nil {
						t.Fatal(err)
					}
				}
				opts := llvm.NewPassBuilderOptions()
				defer opts.Dispose()
				if err := mod.RunPasses("default<O2>", tm, opts); err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				obj, err := tm.EmitToMemoryBuffer(mod, llvm.ObjectFile)
				if err != nil {
					t.Fatal(err)
				}
				defer obj.Dispose()
				dir := t.TempDir()
				object := filepath.Join(dir, "caller.o")
				if err := os.WriteFile(object, obj.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
				harness := filepath.Join(dir, "main.c")
				if err := os.WriteFile(harness, []byte(`
#include <stdio.h>
#include <stdbool.h>
extern unsigned long long caller(void *, bool);
unsigned long long take(const volatile unsigned long long *p, bool choice) {
  volatile unsigned long long scratch[128];
  for (unsigned i = 0; i < 128; ++i) scratch[i] = 99;
  return *p;
}
int main(void) { printf("%llu\n", caller(0, true)); return 0; }
`), 0600); err != nil {
					t.Fatal(err)
				}
				binary := filepath.Join(dir, "tail.exe")
				if out, err := exec.CommandContext(t.Context(), clang, "-O2", harness, object, "-o", binary).CombinedOutput(); err != nil {
					t.Fatalf("link: %v\n%s", err, out)
				}
				out, err := exec.CommandContext(t.Context(), binary).CombinedOutput()
				if err != nil || strings.TrimSpace(string(out)) != "42" {
					t.Fatalf("tail-call lifetime: want 42, got %q (%v)", out, err)
				}
			})
		}
	}
}
