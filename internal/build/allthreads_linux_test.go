package build

import (
	stdcontext "context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAllThreadsForeignLibrary(t *testing.T) {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("native Linux executable")
	}
	setPCLNIntegrationEnv(t)
	dir := t.TempDir()
	library := filepath.Join(dir, "libforeign.so")
	if out, err := exec.Command("cc", "-shared", "-fPIC", "-pthread", "-o", library,
		"testdata/allthreads/foreign.c").CombinedOutput(); err != nil {
		t.Fatalf("compile C pthread library: %v\n%s", err, out)
	}
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(`package main
import "syscall"
import _ "unsafe"
//go:linkname start C.foreign_thread_start
func start() int32
//go:linkname join C.foreign_thread_join
func join() int32
//go:linkname raw syscall.runtime_doAllThreadsSyscall
func raw(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)
func check() {
 for _, call := range []func() syscall.Errno{
  func() syscall.Errno { _,_,err:=syscall.AllThreadsSyscall(syscall.SYS_GETPID,0,0,0); return err },
  func() syscall.Errno { _,_,err:=syscall.AllThreadsSyscall6(syscall.SYS_SETRESUID,^uintptr(0),^uintptr(0),^uintptr(0),0,0,0); return err },
  func() syscall.Errno { _,_,err:=raw(syscall.SYS_SETRESUID,^uintptr(0),^uintptr(0),^uintptr(0),0,0,0); return syscall.Errno(err) },
 } {
  if call()!=syscall.ENOTSUP { panic("all-thread call enabled with a foreign C thread creator") }
 }
}
func main() {
 check() // A linked creator must be rejected before its first pthread exists.
 if start()!=0 { panic("pthread_create") }
 check()
 if join()!=0 { panic("pthread_join") }
 check() // Joining a foreign thread must not re-enable the operation.
 println("allthreads-foreign-ok")
}
`+fmt.Sprintf("\n//go:cgo_import_dynamic _ _ %q\n", library)), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "foreign")
	if _, err := Do([]string{source}, conf); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, conf.OutFile)
	cmd.Env = append(os.Environ(), "GC_MARKERS=1")
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "allthreads-foreign-ok") {
		t.Fatalf("foreign C library: %v\n%s", err, out)
	}
}

func TestAllThreadsNativeExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("native Linux executable")
	}
	setPCLNIntegrationEnv(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(`package main
import "syscall"
func main() {
 ready, done, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
 go func(){ close(ready); <-done; close(exited) }()
 <-ready
 for i:=0;i<25;i++ {
  r1,r2,err := syscall.AllThreadsSyscall(syscall.SYS_GETPID,0,0,0)
  if err!=syscall.ENOTSUP || r1!=^uintptr(0) || r2!=^uintptr(0) { panic("executable bypassed the libc/cgo boundary") }
 }
 close(done); <-exited
 println("allthreads-exe-ok")
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "owned")
	if _, err := Do([]string{source}, conf); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, conf.OutFile)
	cmd.Env = append(os.Environ(), "GC_MARKERS=1")
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "allthreads-exe-ok") {
		t.Fatalf("owned executable: %v\n%s", err, out)
	}
}

func TestAllThreadsNativeProtocol(t *testing.T) {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("native Linux 64-bit signal protocol")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, sanitize := range []bool{false, true} {
		name := "native"
		if sanitize {
			name = "asan-ubsan"
		}
		t.Run(name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "protocol")
			args := []string{"-Wall", "-Wextra", "-Werror", "-O2", "-pthread", "-o", binary,
				filepath.Join(root, "internal/build/testdata/allthreads/protocol.c"),
				filepath.Join(root, "runtime/internal/stacktrace/_wrap/traceback_unix.c")}
			if sanitize {
				args = append(args, "-fsanitize=address,undefined")
			}
			if out, err := exec.Command("cc", args...).CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, out)
			}
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), time.Minute)
			defer cancel()
			out, err := exec.CommandContext(ctx, binary).CombinedOutput()
			if err != nil || !strings.Contains(string(out), "allthreads-ok") {
				t.Fatalf("cohort, cancellation and creation/exit stress: %v\n%s", err, out)
			}
		})
	}
}

func TestAllThreadsControlledExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and executes controlled-thread native fixtures")
	}
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("native Linux 64-bit executable")
	}
	setPCLNIntegrationEnv(t)
	source, err := os.ReadFile("testdata/allthreads/controlled.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, tags := range []string{"llgo.allthreads", "llgo.allthreads,nogc"} {
		t.Run(strings.ReplaceAll(tags, ",", "+"), func(t *testing.T) {
			dir := t.TempDir()
			main := filepath.Join(dir, "main.go")
			if err := os.WriteFile(main, source, 0o600); err != nil {
				t.Fatal(err)
			}
			conf := NewDefaultConf(ModeBuild)
			conf.Tags, conf.OutFile = tags, filepath.Join(dir, "controlled")
			if _, err := Do([]string{main}, conf); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), time.Minute)
			defer cancel()
			command := exec.CommandContext(ctx, conf.OutFile)
			// Setup must select a serial collector before initialization, even
			// when the environment requests unregistered parallel GC markers.
			command.Env = append(os.Environ(), "GC_MARKERS=4")
			if out, err := command.CombinedOutput(); err != nil || !strings.Contains(string(out), "allthreads-controlled-ok") {
				t.Fatalf("%s: %v\n%s", tags, err, out)
			}
		})
	}
}

func TestAllThreadsControlledForeignInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and executes native constructor fixtures")
	}
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("native Linux 64-bit executable")
	}
	setPCLNIntegrationEnv(t)
	for name, native := range map[string]string{
		"collector": `#include <gc/gc.h>
__attribute__((constructor)) static void initialize(void) { GC_init(); fixture_initialized = 1; }
`,
		"thread": `#include <pthread.h>
#include <stdatomic.h>
#include <sched.h>
static atomic_int ready;
static void *foreign(void *unused) {
 (void)unused;
 atomic_store(&ready, 1);
 for (;;) sched_yield();
 return 0;
}
__attribute__((constructor)) static void initialize(void) {
 pthread_t thread;
 if (pthread_create(&thread, 0, foreign, 0)) __builtin_trap();
 while (!atomic_load(&ready)) sched_yield();
 fixture_initialized = 1;
}
`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "_wrap"), 0o700); err != nil {
				t.Fatal(err)
			}
			native = "static int fixture_initialized;\n" + native + "\nint llgo_allthreads_fixture_keep(void) { return fixture_initialized; }\n"
			if err := os.WriteFile(filepath.Join(dir, "_wrap", "early.c"), []byte(native), 0o600); err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(dir, "main.go")
			if err := os.WriteFile(source, []byte(`package main
import "syscall"
import _ "unsafe"
const LLGoFiles = "_wrap/early.c"
//go:linkname keep C.llgo_allthreads_fixture_keep
func keep() int32
//go:linkname raw syscall.runtime_doAllThreadsSyscall
func raw(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)
func main() {
 if keep()!=1 { panic("native constructor did not run") }
 if _,_,err:=syscall.AllThreadsSyscall(syscall.SYS_GETPID,0,0,0); err!=syscall.ENOTSUP { panic("early foreign initialization accepted") }
 if _,_,err:=syscall.AllThreadsSyscall6(syscall.SYS_SETRESUID,^uintptr(0),^uintptr(0),^uintptr(0),0,0,0); err!=syscall.ENOTSUP { panic("early foreign credentials accepted") }
 if _,_,err:=raw(syscall.SYS_SETRESUID,^uintptr(0),^uintptr(0),^uintptr(0),0,0,0); err!=uintptr(syscall.ENOTSUP) { panic("early foreign raw call accepted") }
 println("allthreads-early-rejected")
}
`), 0o600); err != nil {
				t.Fatal(err)
			}
			conf := NewDefaultConf(ModeBuild)
			conf.Tags, conf.OutFile = "llgo.allthreads", filepath.Join(dir, "early")
			if _, err := Do([]string{source}, conf); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), time.Minute)
			defer cancel()
			command := exec.CommandContext(ctx, conf.OutFile)
			command.Env = append(os.Environ(), "GC_MARKERS=1")
			if out, err := command.CombinedOutput(); err != nil || !strings.Contains(string(out), "allthreads-early-rejected") {
				t.Fatalf("early %s: %v\n%s", name, err, out)
			}
		})
	}
}
