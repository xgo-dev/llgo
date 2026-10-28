package puregobridge

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func nativeOracle(t *testing.T, source string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sysv-oracle")
	command(t, clang22(t), "-std=c11", "-O2", "-fno-omit-frame-pointer", "-Wall", "-Wextra", "-Werror",
		"-Inative", "testdata/sysv.c", "testdata/guard_amd64.S", source, "-o", bin)
	return bin
}

func TestSysVAMD64NativeCall(t *testing.T) {
	command(t, nativeOracle(t, "native/sysv_amd64.S"))
}

// The upstream Go-runtime assembly is not a C ABI implementation: it clobbers
// R12 and saves its argument block at a positive BP offset into its caller's
// frame. The negative tests prove that these are detected by the C oracle,
// rather than relying on a strlen call happening to survive either defect.
func TestSysVAMD64OracleRejectsABIRegressions(t *testing.T) {
	original, err := os.ReadFile("native/sysv_amd64.S")
	if err != nil {
		t.Fatal(err)
	}
	for _, regression := range []struct{ name, from, to string }{
		{"callee-saved-r12", "callq *%r10", "movq $0, %r12\n\tcallq *%r10"},
		{"caller-frame-write", "movq %rdi, 72(%rsp)", "movq %rdi, 72(%rsp)\n\tmovq %rdi, 72(%rbp)"},
		{"variadic-float-count", "movl $8, %eax", "xorl %eax, %eax"},
	} {
		t.Run(regression.name, func(t *testing.T) {
			if strings.Count(string(original), regression.from) != 1 {
				t.Fatal("regression injection must change exactly one site")
			}
			mutant := filepath.Join(t.TempDir(), "mutant.S")
			if err := os.WriteFile(mutant, []byte(strings.Replace(string(original), regression.from, regression.to, 1)), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, nativeOracle(t, mutant)).CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || !strings.Contains(string(output), "purego SysV oracle:") {
				t.Fatalf("oracle did not reject %s with a controlled assertion: %v\n%s", regression.name, err, output)
			}
		})
	}
}
