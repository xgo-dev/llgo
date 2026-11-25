//go:build darwin || linux || windows

package build

import (
	stdctx "context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
	args := []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "../../runtime/internal/clite/bdwgc/_test/gc_policy_test.c", "-o", output}
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

func TestBDWGCMarkersEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("pkg-config"); err != nil {
		if os.Getenv("CI") == "true" {
			t.Fatalf("native GC e2e test requires pkg-config: %v", err)
		}
		t.Skipf("native GC e2e test requires pkg-config: %v", err)
	}
	if out, err := exec.Command("pkg-config", "--atleast-version=8.2.0", "bdw-gc").CombinedOutput(); err != nil {
		if os.Getenv("CI") == "true" {
			t.Fatalf("native GC e2e test requires BDWGC >= 8.2.0: %v\n%s", err, out)
		}
		t.Skipf("native GC e2e test requires BDWGC >= 8.2.0: %v\n%s", err, out)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(`package main

import _ "unsafe"

//go:linkname gcGetParallel C.GC_get_parallel
func gcGetParallel() int32

//go:linkname initGC github.com/xgo-dev/llgo/runtime/internal/runtime.InitGC
func initGC()

//go:linkname parseGOMAXPROCS github.com/xgo-dev/llgo/runtime/internal/runtime.ParseGOMAXPROCS
func parseGOMAXPROCS(string) int

//go:linkname targetBDWGCMarkers github.com/xgo-dev/llgo/runtime/internal/runtime.targetBDWGCMarkers
func targetBDWGCMarkers(gomaxprocs string, gcMarkersSet bool, gcNprocs string, cpuCount int) int

func check(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}

func main() {
	// Calling InitGC after GC_init must be a safe no-op.
	initGC()

	// Exercise Go parser & policy edge cases inside the compiled runtime.
	check(parseGOMAXPROCS("") == 0, "empty")
	check(parseGOMAXPROCS("0") == 0, "zero")
	check(parseGOMAXPROCS("-1") == 0, "neg")
	check(parseGOMAXPROCS("+4") == 0, "plus")
	check(parseGOMAXPROCS(" 4") == 0, "lead-space")
	check(parseGOMAXPROCS("4 ") == 0, "trail-space")
	check(parseGOMAXPROCS("bad") == 0, "alpha")
	check(parseGOMAXPROCS("2147483648") == 0, "overflow")
	check(parseGOMAXPROCS("1") == 1, "one")
	check(parseGOMAXPROCS("4") == 4, "four")
	check(parseGOMAXPROCS("2147483647") == 2147483647, "maxint32")

	check(targetBDWGCMarkers("", false, "", 16) == 0, "unset")
	check(targetBDWGCMarkers("1", false, "", 16) == 1, "g1")
	check(targetBDWGCMarkers("4", true, "", 16) == 0, "gc-markers-set")
	check(targetBDWGCMarkers("4", false, "", 16) == 4, "gc-nprocs-empty")
	check(targetBDWGCMarkers("64", false, "", 16) == 16, "clamped")
	check(targetBDWGCMarkers("4", false, "8", 16) == 0, "gc-nprocs-set")
	check(targetBDWGCMarkers("0", false, "", 16) == 0, "g0")
	check(targetBDWGCMarkers("bad", false, "", 16) == 0, "gbad")

	println("parallel=", gcGetParallel())
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "markers")
	if runtime.GOOS == "windows" {
		outFile += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = outFile
	if _, err := Do([]string{source}, conf); err != nil {
		t.Fatalf("build native markers probe: %v", err)
	}
	runProbe := func(overrides map[string]string) int {
		t.Helper()
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, outFile)
		var env []string
		for _, kv := range os.Environ() {
			key, _, _ := strings.Cut(kv, "=")
			if strings.EqualFold(key, "GOMAXPROCS") ||
				strings.EqualFold(key, "GC_MARKERS") ||
				strings.EqualFold(key, "GC_NPROCS") {
				continue
			}
			env = append(env, kv)
		}
		for k, v := range overrides {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run markers probe %v: %v\n%s", overrides, err, out)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "parallel="); ok {
				val, err := strconv.Atoi(strings.TrimSpace(rest))
				if err != nil {
					t.Fatalf("parse markers probe line %q: %v", line, err)
				}
				return val
			}
		}
		t.Fatalf("missing parallel= line in markers probe output %q", string(out))
		return 0
	}
	def := runProbe(nil)
	if got := runProbe(map[string]string{"GOMAXPROCS": "1"}); got != 0 {
		t.Fatalf("GOMAXPROCS=1: GC_get_parallel() = %d, want 0", got)
	}
	if got := runProbe(map[string]string{"GOMAXPROCS": "2"}); got > 1 || (def >= 2 && got != 1) {
		t.Fatalf("GOMAXPROCS=2: GC_get_parallel() = %d (default=%d)", got, def)
	}
	if got := runProbe(map[string]string{"GOMAXPROCS": "99999"}); got > def {
		t.Fatalf("GOMAXPROCS=99999: GC_get_parallel() = %d, want <= default %d", got, def)
	}
	if got := runProbe(map[string]string{"GOMAXPROCS": "bad"}); got != def {
		t.Fatalf("GOMAXPROCS=bad: GC_get_parallel() = %d, want default %d", got, def)
	}
	wantMarkers := runProbe(map[string]string{"GC_MARKERS": "2"})
	if got := runProbe(map[string]string{"GOMAXPROCS": "1", "GC_MARKERS": "2"}); got != wantMarkers {
		t.Fatalf("GOMAXPROCS=1 GC_MARKERS=2: GC_get_parallel() = %d, want %d", got, wantMarkers)
	}
	wantEmptyMarkers := runProbe(map[string]string{"GC_MARKERS": ""})
	if got := runProbe(map[string]string{"GOMAXPROCS": "1", "GC_MARKERS": ""}); got != wantEmptyMarkers {
		t.Fatalf("GOMAXPROCS=1 GC_MARKERS=\"\": GC_get_parallel() = %d, want %d", got, wantEmptyMarkers)
	}
	wantNprocs := runProbe(map[string]string{"GC_NPROCS": "2"})
	if got := runProbe(map[string]string{"GOMAXPROCS": "1", "GC_NPROCS": "2"}); got != wantNprocs {
		t.Fatalf("GOMAXPROCS=1 GC_NPROCS=2: GC_get_parallel() = %d, want %d", got, wantNprocs)
	}
	if got := runProbe(map[string]string{"GOMAXPROCS": "1", "GC_NPROCS": ""}); got != 0 {
		t.Fatalf("GOMAXPROCS=1 GC_NPROCS=\"\": GC_get_parallel() = %d, want 0", got)
	}
}
