//go:build darwin || linux || windows

package build

import (
	stdctx "context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBDWGCAllocationPolicy(t *testing.T) {
	unavailable := func(format string, args ...any) {
		t.Helper()
		if os.Getenv("CI") == "true" {
			t.Fatalf(format, args...)
		}
		t.Skipf(format, args...)
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		unavailable("native GC tests require Clang: %v", err)
	}
	if out, err := exec.Command("pkg-config", "--atleast-version=8.2.0", "bdw-gc").CombinedOutput(); err != nil {
		unavailable("native GC tests require BDWGC >= 8.2.0: %v\n%s", err, out)
	}
	flags, err := exec.Command("pkg-config", "--libs", "bdw-gc").Output()
	if err != nil {
		t.Fatal("native GC tests require bdw-gc pkg-config metadata:", err)
	}
	output := filepath.Join(t.TempDir(), "gc-policy.exe")
	args := []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "../../../runtime/internal/clite/bdwgc/_test/gc_policy_test.c", "-o", output}
	args = append(args, strings.Fields(string(flags))...)
	if out, err := exec.Command(clang, args...).CombinedOutput(); err != nil {
		t.Fatalf("compile BDWGC policy test: %v\n%s", err, out)
	}
	run := func(value string, args ...string) {
		t.Helper()
		// Collector aborts can display a modal Windows dialog while holding
		// the GC lock. Bound the child independently of its runtime timers.
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, output, args...)
		cmd.Env = append(os.Environ(), "GOGC="+value)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("BDWGC policy GOGC=%s %v: %v\n%s", value, args, err, out)
		}
	}
	run("100")
	run("100", "host-disabled")
	for _, test := range []struct{ input, want string }{
		{"off", "-1"}, {"-2", "-1"}, {"0", "0"}, {"25", "25"},
		{"", "100"}, {"bad", "100"}, {"2147483648", "100"},
		{"+25", "25"}, {" 25", "100"}, {"25 ", "100"},
	} {
		t.Run("GOGC="+test.input, func(t *testing.T) {
			run(test.input, test.want)
		})
	}
}
