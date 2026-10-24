package build

import (
	"debug/elf"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
)

func TestLinuxCExportArgs(t *testing.T) {
	exports := []cExport{{goName: "dep.Callback", cName: "Callback"}, {goName: "main.Export", cName: "Export"}}
	for _, mode := range []PCLNMode{PCLNEmbedded, PCLNExternal, PCLNNone} {
		ctx := &context{buildConf: &Config{Goos: "linux", BuildMode: BuildModeExe, PCLNMode: mode}}
		want := []string{"-Wl,--undefined=Callback", "-Wl,--export-dynamic-symbol=Callback", "-Wl,--undefined=Export", "-Wl,--export-dynamic-symbol=Export"}
		if got := linuxCExportArgs(ctx, exports); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s C exports = %v, want %v", mode, got, want)
		}
		if got := linuxCExportArgs(ctx, nil); got != nil {
			t.Fatalf("empty C exports = %v", got)
		}
	}
	for _, conf := range []Config{
		{Goos: "darwin", BuildMode: BuildModeExe},
		{Goos: "windows", BuildMode: BuildModeExe},
		{Goos: "linux", BuildMode: BuildModeCShared},
		{Goos: "linux", BuildMode: BuildModeCArchive},
		{Goos: "linux", BuildMode: BuildModeExe, Target: "wasi"},
	} {
		if got := linuxCExportArgs(&context{buildConf: &conf}, exports); got != nil {
			t.Fatalf("unexpected exports for %+v: %v", conf, got)
		}
	}
}

// DynamicOnly has no static caller, so publishing its C entry must also keep
// its body alive through LTO and section GC. DirectEntry exercises a regular C
// callback; publishing either qualified Go implementation would be incorrect.
const linuxExportsFixture = `package main
/*
#include <dlfcn.h>
#cgo LDFLAGS: -ldl
extern int DirectEntry(int);
static int direct_call(void) { return DirectEntry(41); }
static int dynamic_call(void) {
	void *handle = dlopen(0, RTLD_NOW);
	if (!handle) return -1;
	int (*fn)(int) = (int (*)(int))dlsym(handle, "DynamicOnly");
	int result = fn ? fn(40) : -1;
	dlclose(handle);
	return result;
}
*/
import "C"

//export DirectEntry
func DirectEntry(x C.int) C.int { return x + 1 }

//export DynamicOnly
func DynamicOnly(x C.int) C.int { return x + 2 }

func main() {
	if C.direct_call() != 42 || C.dynamic_call() != 42 { panic("C export lookup failed") }
	println("C exports OK")
}
`

func TestLinuxExecutableExportsIntegration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF executable exports")
	}
	setPCLNIntegrationEnv(t)
	t.Setenv("LDFLAGS", "")
	source := writePCLNIntegrationSource(t, linuxExportsFixture)
	for _, tt := range []struct {
		name string
		lto  lto.Mode
		pcln PCLNMode
	}{
		{"native", lto.Off, PCLNEmbedded},
		{"full", lto.Full, PCLNEmbedded},
		{"thin", lto.Thin, PCLNEmbedded},
		{"none", lto.Full, PCLNNone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "exports")
			conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: bin,
				LTO: tt.lto, PCLNMode: tt.pcln, PCLNModeSet: true}
			if _, err := Do([]string{source}, conf); err != nil {
				t.Fatal(err)
			}
			if got := runPCLNIntegrationBinary(t, bin); got != "C exports OK\n" {
				t.Fatal(got)
			}
			syms := linuxDynamicDefinitions(t, bin)
			for _, name := range []string{"DirectEntry", "DynamicOnly"} {
				if !syms[name] {
					t.Errorf("missing dynamic C entry %s", name)
				}
			}
			for name := range syms {
				if strings.HasPrefix(name, "main.") || strings.HasPrefix(name, "command-line-arguments.") {
					t.Errorf("unexpected Go dynamic export %s", name)
				}
			}
		})
	}
}

func TestLinuxExecutableExportPCLNFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF executable exports")
	}
	setPCLNIntegrationEnv(t)
	t.Setenv("LDFLAGS", "")
	source := writePCLNIntegrationFixture(t)
	for _, sites := range []string{"1", "0"} {
		t.Run("sites="+sites, func(t *testing.T) {
			t.Setenv(llgoFuncInfoSites, sites)
			bin := buildPCLNIntegrationBinary(t, source, PCLNEmbedded, LinkOptions{})
			if got := runPCLNIntegrationBinary(t, bin, "apis"); got != "caller=FULL frames=FULL func=FULL\n" {
				t.Fatal(got)
			}
			if got := linuxDynamicDefinitions(t, bin)["main.pclnTarget"]; got != (sites == "0") {
				t.Fatalf("pclnTarget exported = %v with sites=%s", got, sites)
			}
		})
	}
}

func linuxDynamicDefinitions(t *testing.T, path string) map[string]bool {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	syms, err := f.DynamicSymbols()
	if err != nil && err != elf.ErrNoSymbols {
		t.Fatal(err)
	}
	defined := make(map[string]bool)
	for _, sym := range syms {
		if sym.Section != elf.SHN_UNDEF {
			defined[sym.Name] = true
		}
	}
	return defined
}
