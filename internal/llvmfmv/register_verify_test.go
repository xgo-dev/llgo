package llvmfmv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const registerEdgeFixture = `target triple = "x86_64-unknown-linux-gnu"
define void @kernel() noinline "target-features"="+avx,+avx2,+avx512f" "llgo.fmv.register-width"="512" {
 call void asm sideeffect "", ""()
 ret void
}
define void @start() noinline %s {
 call void @kernel()
 ret void
}`

func TestVerifyRegisterCalls(t *testing.T) {
	for _, tc := range []struct {
		attrs string
		valid bool
	}{
		{"", false}, {`"target-features"="+avx,+avx2"`, false}, {`"target-features"="+avx512f,-avx512f"`, false},
		{`"target-features"="+avx512f"`, true}, {`"llgo.fmv.register-baseline"="v4"`, true},
	} {
		m := parseModule(t, fmt.Sprintf(registerEdgeFixture, tc.attrs))
		if err := VerifyRegisterCalls(m); (err == nil) != tc.valid {
			t.Fatalf("%s: %v", tc.attrs, err)
		}
	}
}

func TestLinkTimeRegisterVerifier(t *testing.T) {
	plugin := os.Getenv("LLGO_LTO_PLUGIN")
	if plugin == "" {
		t.Skip("requires LLGO_LTO_PLUGIN")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	linker, err := exec.LookPath("ld.lld")
	if err != nil {
		t.Skip("ELF lld unavailable")
	}
	for _, mode := range []string{"thin", "full"} {
		for _, opt := range []string{"0", "2"} {
			for _, valid := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/O%s/valid=%t", mode, opt, valid), func(t *testing.T) {
					dir := t.TempDir()
					input := filepath.Join(dir, "edge.ll")
					bc := filepath.Join(dir, "edge.o")
					attrs := ""
					if valid {
						attrs = `"target-features"="+avx512f"`
					}
					if err := os.WriteFile(input, []byte(fmt.Sprintf(registerEdgeFixture, attrs)), 0600); err != nil {
						t.Fatal(err)
					}
					if output, err := exec.Command(clang, "--target=x86_64-unknown-linux-gnu", "-c", "-O0", "-flto="+mode, input, "-o", bc).CombinedOutput(); err != nil {
						t.Fatalf("%v: %s", err, output)
					}
					out, err := exec.Command(linker, "--lto-O"+opt, "--load-pass-plugin="+plugin, "-e", "start", bc, "-o", filepath.Join(dir, "result")).CombinedOutput()
					if valid && err != nil || !valid && (err == nil || !strings.Contains(string(out), "requires 512-bit features")) {
						t.Fatalf("link %v: %s", err, out)
					}
				})
			}
		}
	}
}
