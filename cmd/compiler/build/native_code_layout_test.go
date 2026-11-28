package build

import (
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
)

func TestNativeFullLTOCodeLayout(t *testing.T) {
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		t.Setenv(name, "")
	}
	for _, driver := range []string{"CC", "CXX"} {
		t.Run(driver, func(t *testing.T) {
			for _, override := range []string{"", "-Wl,-mllvm,-align-loops=16", "-Wl,-mllvm,-x86-align-branch-boundary=64"} {
				t.Run(override, func(t *testing.T) {
					c := &context{buildConf: dispatchInlineConfig(), crossCompile: crosscompile.Export{
						CC: "clang", LDFLAGS: []string{override},
						Toolchain: crosscompile.NativeToolchain{ObjectFormat: crosscompile.ObjectFormatELF, Linker: crosscompile.LinkerFlavorELFLLD},
					}}
					if driver == "CXX" {
						c.crossCompile.CXX = "clang++"
					}
					args := c.linker().LinkArguments()
					for _, flag := range []string{"-Wl,-mllvm,-align-loops=64", "-Wl,-mllvm,-x86-align-branch-boundary=32", "-Wl,-mllvm,-x86-align-branch=fused+jcc+jmp+indirect"} {
						if got, want := slices.Contains(args, flag), override == ""; got != want {
							t.Errorf("%s present = %v, want %v: %v", flag, got, want, args)
						}
					}
					if override != "" && !slices.Contains(args, override) {
						t.Fatalf("lost explicit layout flag: %v", args)
					}
					if got := c.compiler().CompileArguments(); slices.Contains(got, "-Wl,-mllvm,-align-loops=64") {
						t.Fatalf("link-time default leaked to compilation: %v", got)
					}
				})
			}
		})
	}
}

func TestNativeFullLTOCodeLayoutPCLN(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux amd64 full-LTO backend policy")
	}
	setPCLNIntegrationEnv(t)
	source := writePCLNIntegrationFixture(t)
	for _, mode := range []PCLNMode{PCLNEmbedded, PCLNExternal} {
		t.Run(string(mode), func(t *testing.T) {
			conf := dispatchInlineConfig()
			conf.Mode = ModeBuild
			conf.PCLNMode, conf.PCLNModeSet = mode, true
			conf.OutFile = filepath.Join(t.TempDir(), "layout-pcln")
			if _, err := Do([]string{source}, conf); err != nil {
				t.Fatal(err)
			}
			for arg, want := range map[string]string{
				"apis":    "caller=FULL frames=FULL func=FULL\n",
				"closure": "target=FULL funcval=FULL same-entry=true\n",
			} {
				if got := runPCLNIntegrationBinary(t, conf.OutFile, arg); got != want {
					t.Fatalf("%s: got %q, want %q", arg, got, want)
				}
			}
		})
	}
}

func TestNativeFullLTOCodeLayoutScope(t *testing.T) {
	for _, change := range []func(*context){
		func(c *context) { c.buildConf.Goos = "windows" },
		func(c *context) { c.buildConf.Goarch = "arm64" },
		func(c *context) { c.buildConf.OptLevel = optlevel.Os },
		func(c *context) { c.buildConf.LTO = lto.Thin },
		func(c *context) { c.buildConf.BuildMode = BuildModeCArchive },
		func(c *context) { c.buildConf.LinkOptions.ExternalLinker = "clang" },
		func(c *context) { c.crossCompile.Linker = "ld.lld" },
		func(c *context) { c.crossCompile.CXXArgs = []string{"-fuse-ld=bfd"} },
		func(c *context) { c.buildConf.LinkOptions.ExternalLinkerFlags = "--ld-path=custom-ld" },
		func(c *context) { c.crossCompile.Toolchain.Linker = "" },
	} {
		c := &context{buildConf: dispatchInlineConfig(), crossCompile: crosscompile.Export{
			Toolchain: crosscompile.NativeToolchain{ObjectFormat: crosscompile.ObjectFormatELF, Linker: crosscompile.LinkerFlavorELFLLD},
		}}
		change(c)
		if got := c.nativeFullLTOLinkerArgs(); len(got) != 0 {
			t.Fatalf("unexpected defaults outside target policy: %v", got)
		}
	}
}
