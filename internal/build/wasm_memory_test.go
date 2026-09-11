//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestWASIHeapReachesLinker(t *testing.T) {
	t.Setenv("CCFLAGS", "")
	t.Setenv("LDFLAGS", "")
	t.Setenv("LLGO_TEST_LINKER_HELPER", "write")
	for _, test := range []struct {
		name     string
		goos     string
		config   []string
		args     []string
		wantHeap bool
	}{
		{name: "single worker default", goos: "wasip1", wantHeap: true},
		{name: "explicit initial memory", goos: "wasip1", args: []string{"-Wl,--initial-memory=33554432"}},
		{name: "explicit initial heap", goos: "wasip1", args: []string{"-Wl,--initial-heap=1048576"}},
		{name: "driver response", goos: "wasip1", args: []string{"@user flags.rsp"}},
		{name: "linker response", goos: "wasip1", args: []string{"-Wl,@user flags.rsp"}},
		{name: "configured response", goos: "wasip1", config: []string{"-Xlinker", "@user flags.rsp"}},
		{name: "shared memory contract", goos: "wasip1", config: []string{"-Wl,--initial-memory=67108864", "-Wl,--import-memory"}},
		{name: "emscripten unchanged", goos: "js"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			argsFile := filepath.Join(dir, "link-args.txt")
			t.Setenv("LINK_ARGS_FILE", argsFile)
			ctx := &context{
				buildConf: &Config{
					Goos: test.goos, Goarch: "wasm", BuildMode: BuildModeExe,
					LinkOptions: LinkOptions{DWARF: DWARFOmit},
				},
				crossCompile: crosscompile.Export{Linker: os.Args[0], LDFLAGS: test.config},
			}
			if err := linkObjFiles(ctx, filepath.Join(dir, "app.wasm"), nil, test.args, false); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Split(strings.TrimSpace(string(data)), "\n")
			if got := slices.Contains(args, defaultWASIHeapFlag); got != test.wantHeap {
				t.Fatalf("default heap in linker arguments = %v, want %v: %q", got, test.wantHeap, args)
			}
			for _, flag := range slices.Concat(test.config, test.args) {
				if !slices.Contains(args, flag) {
					t.Errorf("explicit linker option %q was lost: %q", flag, args)
				}
			}
		})
	}
}

func TestDefaultWASIHeapArgs(t *testing.T) {
	for _, test := range []struct {
		name     string
		goos     string
		goarch   string
		args     []string
		config   []string
		prefix   []string
		ccflags  string
		ldflags  string
		wantHeap bool
	}{
		{name: "default", goos: "wasip1", goarch: "wasm", wantHeap: true},
		{name: "named WASI flags", goos: "wasip1", goarch: "wasm", config: []string{"-target", "wasm32-unknown-wasip1", "-Wl,--stack-first"}, wantHeap: true},
		{name: "native", goos: "linux", goarch: "amd64"},
		{name: "non-wasm architecture", goos: "wasip1", goarch: "amd64"},
		{name: "emscripten", goos: "js", goarch: "wasm"},
		{name: "package memory", goos: "wasip1", goarch: "wasm", args: []string{"-Wl,--initial-memory=33554432"}},
		{name: "package heap", goos: "wasip1", goarch: "wasm", args: []string{"-Wl,--initial-heap=1048576"}},
		{name: "separate driver option", goos: "wasip1", goarch: "wasm", args: []string{"-Wl,--initial-memory,33554432"}},
		{name: "xlinker separate", goos: "wasip1", goarch: "wasm", args: []string{"-Xlinker", "--initial-heap", "-Xlinker", "0"}},
		{name: "xlinker equals", goos: "wasip1", goarch: "wasm", args: []string{"-Xlinker", "--initial-memory=33554432"}},
		{name: "direct linker option", goos: "wasip1", goarch: "wasm", args: []string{"--initial-memory", "33554432"}},
		{name: "config or extldflags", goos: "wasip1", goarch: "wasm", config: []string{"-Wl,--initial-memory=33554432"}},
		{name: "linker prefix", goos: "wasip1", goarch: "wasm", prefix: []string{"-Wl,--initial-heap=1048576"}},
		{name: "CCFLAGS", goos: "wasip1", goarch: "wasm", ccflags: "-Wl,--initial-memory=33554432"},
		{name: "LDFLAGS", goos: "wasip1", goarch: "wasm", ldflags: "-Xlinker --initial-heap=1048576"},
		{name: "maximum unchanged", goos: "wasip1", goarch: "wasm", args: []string{"-Wl,--max-memory=268435456"}, wantHeap: true},
		{name: "unrelated names", goos: "wasip1", goarch: "wasm", args: []string{"data-initial-memory.o", "objects@user.o", "objects,@user.o", "-Wl,-Map,initial-heap.map"}, wantHeap: true},
		{name: "driver response", goos: "wasip1", goarch: "wasm", args: []string{"@user flags.rsp"}},
		{name: "linker response", goos: "wasip1", goarch: "wasm", args: []string{"-Wl,@user flags.rsp"}},
		{name: "combined linker response", goos: "wasip1", goarch: "wasm", args: []string{"-Wl,--export=main,@objects.rsp"}},
		{name: "xlinker response", goos: "wasip1", goarch: "wasm", args: []string{"-Xlinker", "@user flags.rsp"}},
		{name: "config response", goos: "wasip1", goarch: "wasm", config: []string{"@user flags.rsp"}},
		{name: "linker prefix response", goos: "wasip1", goarch: "wasm", prefix: []string{"@user flags.rsp"}},
		{name: "CCFLAGS response", goos: "wasip1", goarch: "wasm", ccflags: "-O2 -Wl,@user flags.rsp"},
		{name: "LDFLAGS response", goos: "wasip1", goarch: "wasm", ldflags: "-O2 -Wl,@user flags.rsp"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CCFLAGS", test.ccflags)
			t.Setenv("LDFLAGS", test.ldflags)
			ctx := &context{
				buildConf: &Config{Goos: test.goos, Goarch: test.goarch},
				crossCompile: crosscompile.Export{
					Linker: "clang", LinkerArgs: test.prefix, LDFLAGS: test.config,
				},
			}
			linker := ctx.linker()
			before := linker.LinkArguments(test.args...)
			got := defaultWASIHeapArgs(ctx, linker, test.args)
			var want []string
			if test.wantHeap {
				want = []string{defaultWASIHeapFlag}
			}
			if !slices.Equal(got, want) {
				t.Fatalf("flags = %q, want %q", got, want)
			}
			if after := linker.LinkArguments(test.args...); !slices.Equal(before, after) {
				t.Fatalf("explicit arguments changed: before %q, after %q", before, after)
			}
		})
	}
	if got := defaultWASIHeapArgs(nil, nil, nil); len(got) != 0 {
		t.Fatalf("nil context: %q", got)
	}
	if got := defaultWASIHeapArgs(&context{}, nil, nil); len(got) != 0 {
		t.Fatalf("missing config: %q", got)
	}
}
