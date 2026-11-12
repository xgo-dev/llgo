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

func TestNativeChannelSignalLifetime(t *testing.T) {
	// Darwin calls pthreadCondWait directly; Linux's GC wrapper waits from C
	// and would need a separate interception point.
	if runtime.GOOS != "darwin" {
		t.Skip("requires the Darwin pthread wait path")
	}
	if testing.Short() {
		t.Skip("builds and runs a native channel probe")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	// Redirect only the pthread boundary in this test build. The channel
	// implementation and its wait/finish/cleanup paths remain unmodified.
	file := filepath.Join(root, "runtime/internal/sync/sync_unix.go")
	source, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	patched := string(source)
	for _, op := range []string{"wait", "signal", "destroy"} {
		name := "C.pthread_cond_" + op
		if strings.Count(patched, name) != 1 {
			t.Fatalf("expected one binding for %s", name)
		}
		patched = strings.Replace(patched, name, "C.llgo_lifetime_cond_"+op, 1)
	}
	// Compile the shared Go/LLGo testcase with a companion that replaces only
	// its synchronization factory. Keep each scenario in a separate process so
	// the C probe can target exactly one wait. The probe checks notification
	// completion even when the waiter retains its signal until its G exits.
	dir := t.TempDir()
	for _, name := range []string{"channel_waiter_test.go", "channel_waiter_norace_test.go", "_wrap/channel_signal_lifetime.c"} {
		data, err := os.ReadFile(filepath.Join(root, "test/go", name))
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	companion := filepath.Join(dir, "probe_test.go")
	if err := os.WriteFile(companion, []byte(`package gotest

import _ "unsafe"

const LLGoFiles = "_wrap/channel_signal_lifetime.c"

//go:linkname lifetimeArm C.llgo_lifetime_arm
func lifetimeArm()

//go:linkname lifetimeAwaitWait C.llgo_lifetime_await_wait
func lifetimeAwaitWait()

//go:linkname lifetimeCheck C.llgo_lifetime_check
func lifetimeCheck()

func init() {
	newChannelWaiterSync = func() channelWaiterSync {
		return channelWaiterSync{lifetimeArm, lifetimeAwaitWait, lifetimeCheck}
	}
}
`), 0600); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeTest)
	conf.Overlay = map[string][]byte{file: []byte(patched)}
	conf.CompileOnly = true
	conf.OutFile = filepath.Join(dir, "channel-lifetime.test")
	if _, err := Do([]string{filepath.Join(dir, "channel_waiter_test.go"), filepath.Join(dir, "channel_waiter_norace_test.go"), companion}, conf); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"recv", "send", "close-recv", "close-send"} {
		t.Run(scenario, func(t *testing.T) {
			name := "TestChannelWaiterCompletion/" + scenario
			if scenario == "close-send" {
				name = "TestChannelWaiterCloseSend"
			}
			ctx, cancel := stdcontext.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, conf.OutFile,
				"-test.v", "-test.run=^"+strings.ReplaceAll(name, "/", "$/^")+"$").CombinedOutput()
			if err != nil {
				t.Fatalf("channel lifetime probe: %v (context: %v)\n%s", err, ctx.Err(), out)
			}
			if !strings.Contains(string(out), "--- PASS: "+name+" ") {
				t.Fatalf("probe did not complete its lifetime check: %s", out)
			}
		})
	}
}
