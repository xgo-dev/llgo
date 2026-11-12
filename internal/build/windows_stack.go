package build

import (
	"strconv"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

const windowsAMD64StackReserve = 8 << 20

func applyWindowsAMD64StackReserve(conf *Config, export *crosscompile.Export) {
	if conf.Goos != "windows" || conf.Goarch != "amd64" || conf.BuildMode != BuildModeExe || export.Toolchain.ObjectFormat != crosscompile.ObjectFormatCOFF {
		return
	}
	// The main thread cannot grow its Go stack. Match common Unix main-stack
	// headroom without committing all reserved pages. Keep 32-bit targets on
	// their existing reserve, where virtual address space is limited.
	reserve := strconv.Itoa(windowsAMD64StackReserve)
	var flag string
	switch export.Toolchain.ABI {
	case crosscompile.PlatformABIMsvc:
		flag = "-Wl,/stack:" + reserve
	case crosscompile.PlatformABIGNU:
		flag = "-Wl,--stack," + reserve
	default:
		return
	}
	// Explicit user linker flags remain later and can override this default.
	export.LDFLAGS = append([]string{flag}, export.LDFLAGS...)
}
