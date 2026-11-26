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

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestNativeChannelSignalLifetime(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires the Darwin channel wait paths")
	}
	if testing.Short() {
		t.Skip("builds and runs a native channel probe")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	// Instrument the native boundaries in one test build. Each process
	// chooses the backend before runtime initialization.
	overlay := make(map[string][]byte)
	patch := func(relative string, bindings [][2]string) {
		file := filepath.Join(root, relative)
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		patched := string(source)
		for _, pair := range bindings {
			binding := "C." + pair[0]
			if strings.Count(patched, binding) != 1 {
				t.Fatalf("expected one binding for %s in %s", binding, file)
			}
			patched = strings.Replace(patched, binding, "C."+pair[1], 1)
		}
		overlay[file] = []byte(patched)
	}
	patch("runtime/internal/sync/sync_unix_native.go", [][2]string{
		{"pthread_cond_wait", "llgo_lifetime_cond_wait"},
		{"pthread_cond_signal", "llgo_lifetime_cond_signal"},
		{"pthread_cond_destroy", "llgo_lifetime_cond_destroy"},
		{"pthread_mutex_unlock", "llgo_lifetime_mutex_unlock"},
		{"pthread_mutex_destroy", "llgo_lifetime_mutex_destroy"},
	})
	patch("runtime/internal/runtime/chan_sync_darwin.go", [][2]string{
		{"llgo_bind_channel_address_wait", "llgo_lifetime_bind_address_wait"},
		{"llgo_channel_address_wait", "llgo_lifetime_address_wait"},
		{"llgo_channel_address_wake", "llgo_lifetime_address_wake"},
	})
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
	conf.Overlay = overlay
	conf.OptLevel, conf.LTO = optlevel.O3, lto.Full
	conf.CompileOnly = true
	conf.OutFile = filepath.Join(dir, "channel-lifetime.test")
	if _, err := Do([]string{filepath.Join(dir, "channel_waiter_test.go"), filepath.Join(dir, "channel_waiter_norace_test.go"), companion}, conf); err != nil {
		t.Fatal(err)
	}
	for _, backend := range []string{"condition", "address"} {
		t.Run(backend, func(t *testing.T) {
			for _, scenario := range []string{"recv", "send", "close-recv", "close-send"} {
				t.Run(scenario, func(t *testing.T) {
					name := "TestChannelWaiterCompletion/" + scenario
					if scenario == "close-send" {
						name = "TestChannelWaiterCloseSend"
					}
					ctx, cancel := stdcontext.WithTimeout(t.Context(), 30*time.Second)
					defer cancel()
					cmd := exec.CommandContext(ctx, conf.OutFile,
						"-test.v", "-test.run=^"+strings.ReplaceAll(name, "/", "$/^")+"$")
					cmd.Env = append(os.Environ(), "LLGO_CHANNEL_LIFETIME_BACKEND="+backend)
					out, err := cmd.CombinedOutput()
					if status, ok := err.(*exec.ExitError); ok && status.ExitCode() == 77 && backend == "address" {
						t.Skip("requires Darwin address-wait APIs")
					}
					if err != nil {
						t.Fatalf("channel lifetime probe: %v (context: %v)\n%s", err, ctx.Err(), out)
					}
					if !strings.Contains(string(out), "--- PASS: "+name+" ") {
						t.Fatalf("probe did not complete its lifetime check: %s", out)
					}
				})
			}
		})
	}
}
