package build

import (
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/quoted"
	"github.com/xgo-dev/llgo/xtool/safesplit"
)

func (c *context) nativeFullLTOELFEnabled() bool {
	if c == nil || !dispatchInlineEnabled(c.buildConf) ||
		c.crossCompile.Toolchain.ObjectFormat != crosscompile.ObjectFormatELF ||
		c.crossCompile.Toolchain.Linker != crosscompile.LinkerFlavorELFLLD ||
		c.crossCompile.Linker != "" || c.buildConf.LinkOptions.ExternalLinker != "" {
		return false
	}
	for _, arg := range c.nativeBackendArgs() {
		if strings.HasPrefix(arg, "--ld-path=") ||
			(strings.HasPrefix(arg, "-fuse-ld=") && arg != "-fuse-ld=lld") {
			return false
		}
	}
	return true
}

// Match driver arguments rather than substrings of paths or ordinary C flags.
func (c *context) nativeBackendArgs() []string {
	flags, _ := quoted.Split(c.buildConf.LinkOptions.ExternalLinkerFlags)
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		flags = append(flags, safesplit.SplitPkgConfigFlags(os.Getenv(name))...)
		flags = append(flags, safesplit.SplitPkgConfigFlags(c.commands.lookup(name))...)
	}
	for _, args := range [][]string{c.crossCompile.CCArgs, c.crossCompile.CXXArgs,
		c.crossCompile.LinkerArgs, c.crossCompile.CCFLAGS, c.crossCompile.CFLAGS, c.crossCompile.LDFLAGS} {
		flags = append(flags, args...)
	}
	return flags
}

func (c *context) nativeBackendOptions() []string {
	var tokens, options []string
	for _, arg := range c.nativeBackendArgs() {
		if rest, ok := strings.CutPrefix(arg, "-Wl,"); ok {
			tokens = append(tokens, strings.Split(rest, ",")...)
		} else if arg != "-Xlinker" {
			tokens = append(tokens, arg)
		}
	}
	value := false
	for _, token := range tokens {
		var option string
		if value {
			option = token
			value = false
		} else {
			introducer, operand, assigned := strings.Cut(token, "=")
			switch introducer {
			case "-mllvm", "--mllvm":
				if !assigned {
					value = true
					continue
				}
			case "-plugin-opt", "--plugin-opt":
				if !assigned || !strings.HasPrefix(operand, "-") {
					continue
				}
			default:
				continue
			}
			option = operand
		}
		name, _, _ := strings.Cut(strings.TrimLeft(option, "-"), "=")
		options = append(options, name)
	}
	return options
}

// Set final-backend defaults only for optimized native x86-64 ELF executables.
// Aligning loops alone leaves dispatch performance sensitive to branch placement;
// combine cache-line loop alignment with the x86 branch-boundary padding pass.
func (c *context) nativeFullLTOLinkerArgs() []string {
	if !c.nativeFullLTOELFEnabled() {
		return nil
	}
	layout := true
	for _, option := range c.nativeBackendOptions() {
		if option == "align-loops" || strings.HasPrefix(option, "align-all-") ||
			strings.HasPrefix(option, "x86-align-branch") || strings.HasPrefix(option, "x86-branches-within") {
			layout = false
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

	return args
}
