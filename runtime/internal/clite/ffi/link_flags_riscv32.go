//go:build tinygo.riscv32

package ffi

// The riscv32 libffi archive is searched via the baremetal -L path added by
// internal/crosscompile for riscv32 named targets.
const (
	LLGoPackage = "link: -lffi" + riscv32CacheSyncLinkFlags
	LLGoFiles   = "_wrap/libffi_riscv32.c" + riscv32CacheSyncFiles
)
