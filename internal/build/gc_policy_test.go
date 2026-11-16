//go:build darwin || linux

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBDWGCAllocationPolicy(t *testing.T) {
	if out, err := exec.Command("pkg-config", "--atleast-version=8.2.0", "bdw-gc").CombinedOutput(); err != nil {
		t.Fatalf("native GC tests require BDWGC >= 8.2.0: %v\n%s", err, out)
	}
	flags, err := exec.Command("pkg-config", "--libs", "bdw-gc").Output()
	if err != nil {
		t.Fatal("native GC tests require bdw-gc pkg-config metadata:", err)
	}
	output := filepath.Join(t.TempDir(), "gc-policy")
	args := []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "../../runtime/internal/clite/bdwgc/_test/gc_policy_test.c", "-o", output}
	args = append(args, strings.Fields(string(flags))...)
	if out, err := exec.Command("clang", args...).CombinedOutput(); err != nil {
		t.Fatalf("compile BDWGC policy test: %v\n%s", err, out)
	}
	cmd := exec.Command(output)
	cmd.Env = append(os.Environ(), "GOGC=100")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("execute BDWGC policy test: %v\n%s", err, out)
	}
	cmd = exec.Command(output, "host-disabled")
	cmd.Env = append(os.Environ(), "GOGC=100")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("preserve host automatic-collection disablement: %v\n%s", err, out)
	}
	for _, test := range []struct{ input, want string }{
		{"off", "-1"}, {"-2", "-1"}, {"0", "0"}, {"25", "25"},
		{"", "100"}, {"bad", "100"}, {"2147483648", "100"},
		{"+25", "25"}, {" 25", "100"}, {"25 ", "100"},
	} {
		t.Run("GOGC="+test.input, func(t *testing.T) {
			cmd := exec.Command(output, test.want)
			cmd.Env = append(os.Environ(), "GOGC="+test.input)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("initialize policy from environment: %v\n%s", err, out)
			}
		})
	}
}
