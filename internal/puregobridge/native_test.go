// Package puregobridge qualifies the explicit purego native ABI adapter.
// These tests are not evidence that unmodified Go runtime trampolines work.
package puregobridge

import (
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func command(t *testing.T, binary string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", binary, args, err, output)
	}
	return string(output)
}

func clang22(t *testing.T) string {
	t.Helper()
	config := os.Getenv("LLVM_CONFIG")
	if config == "" {
		config = "llvm-config"
	}
	version := strings.TrimSpace(command(t, config, "--version"))
	if !strings.HasPrefix(version, "22.") {
		t.Fatalf("LLVM 22 is required, got %q from %s", version, config)
	}
	binDir := strings.TrimSpace(command(t, config, "--bindir"))
	clang := filepath.Join(binDir, "clang")
	if !strings.Contains(command(t, clang, "--version"), "clang version 22.") {
		t.Fatal("LLVM_CONFIG and clang must both select LLVM 22")
	}
	return clang
}

func TestSysVAMD64Object(t *testing.T) {
	object := filepath.Join(t.TempDir(), "sysv.o")
	command(t, clang22(t), "--target=x86_64-unknown-linux-gnu", "-c", "native/sysv_amd64.S", "-o", object)
	file, err := elf.Open(object)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Machine != elf.EM_X86_64 {
		t.Fatalf("wrong ABI: %s", file.Machine)
	}
	if file.Section(".eh_frame") == nil {
		t.Fatal("native call bridge must retain unwind information")
	}
	stack := file.Section(".note.GNU-stack")
	if stack == nil || stack.Flags&elf.SHF_EXECINSTR != 0 {
		t.Fatal("native bridge must not require an executable stack")
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "llgo_purego_sysv_call" && symbol.Section != elf.SHN_UNDEF &&
			elf.ST_TYPE(symbol.Info) == elf.STT_FUNC {
			return
		}
	}
	t.Fatal("missing explicit C ABI entry")
}
