//go:build !llgo

package build

import (
	stdcontext "context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTracebackConfiguration(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "traceback-config")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = bin
	if _, err := Do([]string{"./testdata/tracebackconfig"}, conf); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, env, set, stack  string
		current, other, system bool
	}{
		{name: "default", current: true},
		{name: "none", env: "none"},
		{name: "single", env: "single", current: true},
		{name: "all", env: "all", current: true, other: true},
		{name: "system", env: "system", current: true, other: true, system: true},
		{name: "crash", env: "crash", current: true, other: true, system: true},
		{name: "numeric_zero", env: "0"},
		{name: "numeric_one", env: "1", current: true, other: true},
		{name: "numeric_two", env: "2", current: true, other: true, system: true},
		{name: "invalid", env: "invalid"},
		{name: "set_all", env: "none", set: "all", current: true, other: true},
		{name: "set_system", env: "none", set: "system", current: true, other: true, system: true},
		{name: "set_higher_than_env", env: "single", set: "system", current: true, other: true, system: true},
		{name: "env_higher_than_set", env: "system", set: "single", current: true, other: true, system: true},
		{name: "environment_floor", env: "all", set: "none", current: true, other: true},
		{name: "stack_ignores_none", env: "none", stack: "single", current: true},
		{name: "stack_ignores_system", env: "system", stack: "single", current: true},
		{name: "stack_all", env: "none", stack: "all", current: true, other: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin)
			cmd.Env = append(os.Environ(), "GOTRACEBACK="+tc.env, "LLGO_STACK_ONLY="+tc.stack)
			if tc.set != "" {
				cmd.Env = append(cmd.Env, "LLGO_SET_TRACEBACK="+tc.set)
			}
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("traceback hung: %v\n%s", ctx.Err(), output)
			}
			if (err == nil) != (tc.stack != "") {
				t.Fatalf("unexpected process result: %v\n%s", err, output)
			}
			if tc.env == "crash" && runtime.GOOS != "windows" && cmd.ProcessState.ExitCode() == 2 {
				t.Errorf("crash mode used normal panic exit status 2:\n%s", output)
			}
			for _, check := range []struct {
				text string
				want bool
			}{
				{"goroutine ", tc.current},
				{"main.main(", tc.current},
				{"main.parkedWorker(", tc.other},
				{"runtime.goexit(", tc.system},
			} {
				if got := strings.Contains(string(output), check.text); got != check.want {
					t.Errorf("contains %q = %v, want %v:\n%s", check.text, got, check.want, output)
				}
			}
		})
	}
	if runtime.GOOS == "windows" {
		for _, tc := range []struct {
			name, env, set, again string
			disabled              bool
		}{
			{name: "wer_default", disabled: true},
			{name: "wer_environment", env: "wer"},
			{name: "wer_set", set: "wer"},
			{name: "wer_sticky", set: "wer", again: "all"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				cmd := exec.Command(bin)
				cmd.Env = append(os.Environ(), "GOTRACEBACK="+tc.env, "LLGO_WER_PROBE=1", "LLGO_SET_TRACEBACK="+tc.set)
				if tc.again != "" {
					cmd.Env = append(cmd.Env, "LLGO_SET_TRACEBACK_AGAIN="+tc.again)
				}
				out, err := cmd.CombinedOutput()
				want := "disabled=false no-ui=true"
				if tc.disabled {
					want = "disabled=true no-ui=true"
				}
				if err != nil || strings.TrimSpace(string(out)) != want {
					t.Fatalf("WER state: %v %s, want %s", err, out, want)
				}
			})
		}
	}

}
