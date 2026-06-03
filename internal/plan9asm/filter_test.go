//go:build !llgo
// +build !llgo

package plan9asm

import (
	"reflect"
	"testing"

	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestFilterPlan9AsmFuncsKeepsDarwinRawSyscallAsm(t *testing.T) {
	funcs := []extplan9asm.Func{
		{Sym: "·RawSyscall"},
		{Sym: "·RawSyscall6"},
		{Sym: "·Syscall"},
	}
	got := FilterFuncs("syscall", "darwin", "arm64", funcs, resolveSymFuncForTarget("syscall", "darwin", "arm64"))
	want := funcs
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterFuncs = %#v, want %#v", got, want)
	}
}
