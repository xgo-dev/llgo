package build

import (
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func (c *context) nativeFullLTOELFEnabled() bool {
	if c == nil || !dispatchInlineEnabled(c.buildConf) ||
		c.crossCompile.Toolchain.ObjectFormat != crosscompile.ObjectFormatELF ||
		c.crossCompile.Toolchain.Linker != crosscompile.LinkerFlavorELFLLD ||
		c.crossCompile.Linker != "" || c.buildConf.LinkOptions.ExternalLinker != "" {
		return false
	}
	for _, flag := range c.nativeBackendFlags() {
		for _, arg := range strings.Fields(flag) {
			if strings.HasPrefix(arg, "--ld-path=") ||
				(strings.HasPrefix(arg, "-fuse-ld=") && arg != "-fuse-ld=lld") {
				return false
			}
		}
	}
	return true
}

func (c *context) nativeBackendFlags() []string {
	flags := []string{c.buildConf.LinkOptions.ExternalLinkerFlags}
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		flags = append(flags, os.Getenv(name), c.commands.lookup(name))
	}
	for _, args := range [][]string{c.crossCompile.CCArgs, c.crossCompile.CXXArgs,
		c.crossCompile.LinkerArgs, c.crossCompile.CCFLAGS, c.crossCompile.CFLAGS, c.crossCompile.LDFLAGS} {
		flags = append(flags, args...)
	}
	return flags
}

// Set final-backend defaults only for optimized native x86-64 ELF executables.
// Aligning loops alone leaves dispatch performance sensitive to branch placement;
// combine cache-line loop alignment with the x86 branch-boundary padding pass.
func (c *context) nativeFullLTOLinkerArgs() []string {
	if !c.nativeFullLTOELFEnabled() {
		return nil
	}
	layout := true
	for _, flag := range c.nativeBackendFlags() {
		for _, option := range []string{"align-loops", "align-all", "x86-align-branch", "x86-branches-within"} {
			if strings.Contains(flag, option) {
				layout = false
			}
		}
	}
	var args []string
	if layout {
		args = append(args, "-Wl,-mllvm,-align-loops=64",
			"-Wl,-mllvm,-x86-align-branch-boundary=32",
			// Exclude calls and returns: their adjacent PC labels participate
			// in runtime stack and source-position metadata.
			"-Wl,-mllvm,-x86-align-branch=fused+jcc+jmp+indirect")
	}
	if c.dispatchInlineBudgetEnabled() {
		// LLVM 22 reads function-inline-threshold only after completing its
		// cost walk; otherwise a cold-site early exit masks the hint.
		args = append(args, "-Wl,-mllvm,-inline-cost-full")
	}
	return args
}
