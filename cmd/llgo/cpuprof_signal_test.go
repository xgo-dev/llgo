//go:build !llgo

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeCPUProfileDefaultSiginfo(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix SIGPROF disposition")
	}
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/profiledefault\n\ngo 1.27.0\n")
	if err := os.Mkdir(filepath.Join(dir, "_wrap"), 0700); err != nil {
		t.Fatal(err)
	}
	nativeFile(t, filepath.Join(dir, "_wrap", "signal.c"), `#include <signal.h>
#include <string.h>
int profile_default_siginfo(void) {
    struct sigaction action;
    memset(&action, 0, sizeof(action));
    sigemptyset(&action.sa_mask);
    action.sa_handler = SIG_DFL;
    action.sa_flags = SA_SIGINFO;
    return sigaction(SIGPROF, &action, 0);
}
int profile_signal_restored(void) {
    struct sigaction action;
    if (sigaction(SIGPROF, 0, &action) != 0) return -1;
    return action.sa_handler == SIG_IGN && (action.sa_flags & SA_SIGINFO) == 0;
}
int profile_raise(void) { return raise(SIGPROF); }
`)
	nativeFile(t, filepath.Join(dir, "main.go"), `package main
import ("bytes"; "runtime/pprof"; _ "unsafe")
const LLGoFiles = "_wrap/signal.c"
//go:linkname defaultSiginfo C.profile_default_siginfo
func defaultSiginfo() int32
//go:linkname signalRestored C.profile_signal_restored
func signalRestored() int32
//go:linkname raiseProfile C.profile_raise
func raiseProfile() int32
func main() {
 for i:=0;i<3;i++ {
  if defaultSiginfo()!=0 { panic("sigaction failed") }
  var profile bytes.Buffer
  if err:=pprof.StartCPUProfile(&profile);err!=nil { panic(err) }
  pprof.StopCPUProfile()
  if signalRestored()!=1 { panic("StopCPUProfile did not restore SIG_IGN without SA_SIGINFO") }
  if raiseProfile()!=0 { panic("raise failed") }
 }
 println("profile signal ignored")
}
`)
	out, code := nativeCommand(t, dir, nil, "run", ".")
	if code != 0 || !strings.Contains(out, "profile signal ignored") {
		t.Fatalf("profile default disposition: exit=%d output=%s", code, out)
	}
}
