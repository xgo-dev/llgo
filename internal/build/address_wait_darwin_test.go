package build

import (
	stdcontext "context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDarwinAddressWaitFallback(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "no-address-wait.dylib")
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "clang", "-dynamiclib", "-o", lib, "testdata/address_wait_darwin/no_address_wait.c").CombinedOutput(); err != nil {
		t.Fatalf("build symbol lookup interposer: %v\n%s", err, out)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "fallback")
	if _, err := Do([]string{"testdata/address_wait_darwin/main.go"}, conf); err != nil {
		t.Fatal(err)
	}
	runCtx, cancelRun := stdcontext.WithTimeout(stdcontext.Background(), 15*time.Second)
	defer cancelRun()
	cmd := exec.CommandContext(runCtx, conf.OutFile)
	cmd.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+lib)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Darwin semaphore and timer fallback: %v\n%s", err, out)
	}
}
