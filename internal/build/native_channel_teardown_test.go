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

func TestNativeChannelWaiterTeardown(t *testing.T) {
	// Linux's collector waits through a C wrapper rather than this binding.
	if runtime.GOOS != "darwin" {
		t.Skip("requires the Darwin pthread wait path")
	}
	if testing.Short() {
		t.Skip("builds and runs native lifecycle probes")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	t.Cleanup(func() { cacheRootFunc = oldCacheRoot })
	file := filepath.Join(root, "runtime/internal/sync/sync_unix.go")
	source, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	patched := string(source)
	for _, op := range []string{"cond_wait", "cond_destroy", "mutex_destroy"} {
		name := "C.pthread_" + op
		if strings.Count(patched, name) != 1 {
			t.Fatalf("expected one binding for %s", name)
		}
		patched = strings.Replace(patched, name, "C.llgo_teardown_"+op, 1)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.Overlay = map[string][]byte{file: []byte(patched)}
	conf.OutFile = filepath.Join(t.TempDir(), "channel-teardown")
	fixture := filepath.Join(root, "internal/build/testdata/native-channel-teardown")
	if _, err := Do([]string{fixture}, conf); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"main-goexit", "foreign-thread"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := stdcontext.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, conf.OutFile, scenario).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "ok" {
				t.Fatalf("channel waiter teardown: %v (context: %v)\n%s", err, ctx.Err(), out)
			}
		})
	}
}
