package build

import (
	"slices"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

const (
	embeddedConsoleTag  = "llgo.baremetal.directconsole"
	embeddedConsoleFlag = "__llgo_baremetal_direct_console"
)

func embeddedConsoleProfile(conf *Config, target crosscompile.Export) bool {
	return (conf.Target == "esp32c3-flash" || conf.Target == "esp32c3-basic") &&
		conf.OptLevel == optlevel.Oz && conf.ltoMode() == lto.Full &&
		target.Libc == "newlib-esp32" &&
		!slices.Contains(target.BuildTags, "llgo.baremetal.cstdio") &&
		!slices.Contains(splitSourcePatchBuildTags(conf.Tags), "llgo.baremetal.cstdio")
}

// The built-in boot files only enter main and reference the target runtime.
// Extra native inputs can reference Go symbols that are invisible to SSA.
func embeddedBootFiles(target string) []string {
	switch target {
	case "cortex-m-mps2", "cortex-m-qemu":
		return []string{"targets/device/arm/cortexm.S", "targets/cortex-m-qemu.s", "targets/cortex-m-qemu.c"}
	case "esp32c3-flash":
		return []string{"targets/device/esp/esp32c3.S"}
	case "esp32c3-basic":
		return []string{}
	}
	return nil
}

func embeddedSizePruningConfig(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	boot := embeddedBootFiles(conf.Target)
	return boot != nil && slices.Equal(target.ExtraFiles, boot) &&
		!conf.disableSizeSpecialization && sizeCodecBuild(conf, frontend) &&
		conf.LinkOptions.ExternalLinker == "" && conf.LinkOptions.ExternalLinkerFlags == ""
}

func (ctx *context) sizePruningInputs(goroot string, forFormatter bool) *linkPruningInputs {
	if embeddedSizePruningConfig(ctx.buildConf, ctx.crossCompile, ctx.frontendOptions) {
		return &linkPruningInputs{goroot: goroot}
	}
	return ctx.wasiSizePruningInputs(goroot, forFormatter)
}
