//go:build !llgo

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package debug

import (
	"bytes"
	"debug/dwarf"
	"debug/elf"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/targets"
)

func TestBackendRouting(t *testing.T) {
	tests := []struct {
		name   string
		conf   build.Config
		target *targets.Config
		want   backend
	}{
		{name: "native", conf: build.Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}, want: backendLLDB},
		{name: "embedded", conf: build.Config{Target: "board"}, target: &targets.Config{LLVMTarget: "thumbv7m-none-eabi"}, want: backendGDB},
		{name: "WASI", conf: build.Config{Target: "wasip1"}, target: &targets.Config{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasi"}, want: backendWasmtime},
		{name: "browser", conf: build.Config{Target: "wasm"}, target: &targets.Config{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasi"}, want: backendBrowser},
		{name: "Wasm name with WASI ABI", conf: build.Config{Target: "wasm-custom"}, target: &targets.Config{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1"}, want: backendWasmtime},
		{name: "raw Wasm target", conf: build.Config{Target: "wasm-unknown"}, target: &targets.Config{GOARCH: "wasm", LLVMTarget: "wasm32-unknown-unknown"}, want: backendWasmtime},
		{name: "inherited browser OS", conf: build.Config{Goos: "js", Goarch: "wasm"}, target: &targets.Config{LLVMTarget: "wasm32-unknown-unknown"}, want: backendBrowser},
		{name: "browser profile", target: &targets.Config{WasmProfile: "j64", LLVMTarget: "wasm64-unknown-unknown"}, want: backendBrowser},
		{name: "raw js source mode", conf: build.Config{Goos: "js", Goarch: "wasm"}, want: backendBrowser},
		{name: "memory64 inherited profile", conf: build.Config{Target: "custom"}, target: &targets.Config{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"}, want: backendBrowser},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kind := classifyTarget(&test.conf, test.target)
			got, err := selectBackend(backendAuto, kind)
			if err != nil || got != test.want {
				t.Fatalf("selectBackend(auto) = (%q, %v), want (%q, nil)", got, err, test.want)
			}
		})
	}
	if _, err := selectBackend(backendGDB, targetWASI); err == nil {
		t.Fatal("GDB unexpectedly accepted a WASI target")
	}
	if err := (options{backend: "unknown"}).validate(); err == nil {
		t.Fatal("unknown backend was accepted")
	}
}

func TestBrowserAndWASISessionBoundaries(t *testing.T) {
	conf := &build.Config{Goos: "js", Goarch: "wasm"}
	if err := validateSessionTarget(conf, nil, backendBrowser, options{}); err != nil {
		t.Fatalf("browser source mode must not require a native remote: %v", err)
	}
	if err := validateSessionTarget(conf, nil, backendLLDB, options{}); err == nil {
		t.Fatal("native cross-host launch accepted without remote")
	}
	for _, opts := range []options{{remote: ":3333"}, {server: "must-not-execute"}, {load: true}} {
		err := runSession(session{backend: backendBrowser, options: opts}, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-remote, -server and -load") {
			t.Fatalf("browser accepted native transport: %v", err)
		}
	}
	err := runSession(session{backend: backendWasmtime, target: &targets.Config{DebugServer: "must-not-execute"}}, nil, nil, nil)
	for _, requirement := range []string{"W32 pthread", "env.memory", "wasi.thread-spawn", "wasix_32v1.thread_exit", "Wasmer"} {
		if err == nil || !strings.Contains(err.Error(), requirement) {
			t.Fatalf("WASI diagnostic must identify %q: %v", requirement, err)
		}
	}
}

func TestSourceMapFlagsAndBrowserOutputs(t *testing.T) {
	var mappings sourceMapsFlag
	if err := mappings.Set("/build/src=/local/src"); err != nil {
		t.Fatal(err)
	}
	if err := mappings.Set("/build/src/vendor=/local/vendor"); err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 || mappings[0] != (browserdebug.PathMapping{From: filepath.Clean("/build/src"), To: filepath.Clean("/local/src")}) {
		t.Fatalf("repeated source mappings lost: %+v", mappings)
	}
	if err := mappings.Set("missing-destination"); err == nil || len(mappings) != 2 {
		t.Fatal("invalid source mapping changed accepted mappings")
	}
	for _, ext := range []string{".wasm", ".mjs", ".js", ".html"} {
		conf := &build.Config{Goos: "js", Goarch: "wasm", Target: "emscripten-memory64", OutFile: filepath.Join(t.TempDir(), "app"+ext)}
		cleanup, artifact, err := prepareArtifact(conf)
		if err != nil {
			t.Fatal(err)
		}
		cleanup()
		if !strings.HasSuffix(artifact, "app"+ext) || !strings.HasSuffix(browserModulePath(artifact), "app.wasm") || conf.Target != "emscripten-memory64" {
			t.Fatalf("browser output/profile changed: %+v, %q", conf, artifact)
		}
	}
}

func TestSessionPlanning(t *testing.T) {
	remote, err := makeServerPlan(nil, "program", options{remote: ":1234"})
	if err != nil || remote.address != "127.0.0.1:1234" || len(remote.command) != 0 {
		t.Fatalf("remote plan = (%+v, %v)", remote, err)
	}

	openocd, err := makeServerPlan(&targets.Config{
		Name:             "board",
		OpenOCDInterface: "cmsis-dap",
		OpenOCDTransport: "swd",
		OpenOCDTarget:    "stm32f4x",
	}, "program.elf", options{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(openocd.command, " ")
	for _, want := range []string{"openocd", "gdb_port", "interface/cmsis-dap.cfg", "transport select swd", "target/stm32f4x.cfg"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("OpenOCD command %q does not contain %q", joined, want)
		}
	}
	if !openocd.load {
		t.Fatal("OpenOCD plan does not request image loading")
	}
	if !openocd.stdio || !strings.Contains(joined, "gdb_port pipe") {
		t.Fatalf("OpenOCD must use pipes instead of reserving a TCP port: %+v", openocd)
	}
	// The loopback address is assigned by the owned relay at startup.
	openocd.address = "127.0.0.1:1234"

	gdbArgs, err := debuggerArguments(backendGDB, "program.elf", []string{"--batch"}, openocd)
	if err != nil {
		t.Fatal(err)
	}
	joined = strings.Join(gdbArgs, " ")
	for _, want := range []string{"target extended-remote", "monitor reset halt", " load ", "--batch"} {
		if !strings.Contains(" "+joined+" ", want) {
			t.Fatalf("GDB arguments %q do not contain %q", joined, want)
		}
	}
	if remote.load {
		t.Fatal("connecting to an existing server must not implicitly load an image")
	}
	loadArgs, err := debuggerArguments(backendLLDB, "dir with space/program.elf", []string{"--batch"}, openocd)
	wantLoadArgs := []string{
		"dir with space/program.elf", "-o", "gdb-remote " + openocd.address,
		"-o", "process plugin packet monitor reset halt",
		"-o", `target modules load --file "dir with space/program.elf" --slide 0 --load`,
		"-o", "process plugin packet monitor reset halt", "--batch",
	}
	if err != nil || !reflect.DeepEqual(loadArgs, wantLoadArgs) {
		t.Fatalf("LLDB image loading = %q, %v; want %q", loadArgs, err, wantLoadArgs)
	}
	command, err := parseServerCommand("server -kernel {} -port {debug-port}", filepath.Join("dir with space", "program.elf"), 4321)
	if err != nil || len(command) != 5 || command[2] != filepath.Join("dir with space", "program.elf") || command[4] != "4321" {
		t.Fatalf("parseServerCommand() = (%v, %v)", command, err)
	}

	lldbArgs, err := debuggerArguments(backendLLDB, `dir/program.elf`, []string{"--batch"}, remote)
	if err != nil {
		t.Fatal(err)
	}
	joined = strings.Join(lldbArgs, " ")
	for _, want := range []string{"gdb-remote 127.0.0.1:1234", "target modules load", "--slide 0", "--batch"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("LLDB arguments %q do not contain %q", joined, want)
		}
	}
}

func TestExplicitRemoteImageLoading(t *testing.T) {
	if _, err := makeServerPlan(nil, "app.elf", options{load: true}); err == nil {
		t.Fatal("-load without a remote server was accepted")
	}
	for _, tc := range []struct {
		name   string
		target *targets.Config
		opts   options
	}{
		{name: "existing remote", opts: options{remote: ":3333"}},
		{name: "target remote", target: &targets.Config{Name: "board"}, opts: options{remote: ":3333"}},
		{name: "custom server", target: &targets.Config{Name: "board"}, opts: options{server: "server --image {} --port {debug-port}"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, load := range []bool{false, true} {
				opts := tc.opts
				opts.load = load
				plan, err := makeServerPlan(tc.target, "app.elf", opts)
				if err != nil || plan.load != load {
					t.Fatalf("server image loading = %+v, %v; want %t", plan, err, load)
				}
			}
		})
	}
}

func TestGDBSessionStartsConfiguredServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	t.Setenv("LLGO_DEBUG_SERVER_HELPER", "1")
	capture := filepath.Join(t.TempDir(), "gdb-arguments")
	t.Setenv("LLGO_DEBUG_GDB_CAPTURE", capture)
	gdbPath := writeFakeGDB(t)
	template := fmt.Sprintf("%s -test.run=^TestDebugServerHelper$ -- {debug-port}", strconv.Quote(os.Args[0]))

	var stdout, stderr bytes.Buffer
	err := runSession(session{
		backend:  backendGDB,
		artifact: filepath.Join(t.TempDir(), "program.elf"),
		target: &targets.Config{
			Name:        "test-target",
			DebugServer: template,
			GDB:         []string{gdbPath},
		},
		options: options{backend: backendGDB},
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("runSession() error: %v; stderr=%s", err, stderr.String())
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	args := string(data)
	if !strings.Contains(args, "target remote 127.0.0.1:") || !strings.Contains(args, "program.elf") {
		t.Fatalf("GDB arguments do not contain the artifact and remote session: %q", args)
	}
}

func TestDebugServerFailureIncludesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	serverPath := filepath.Join(t.TempDir(), "failing-server")
	if err := os.WriteFile(serverPath, []byte("#!/bin/sh\necho 'server startup failed' >&2\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	port, err := freeTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	_, err = startServer(serverPlan{
		command: []string{serverPath},
		address: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
	})
	if err == nil || !strings.Contains(err.Error(), "server startup failed") {
		t.Fatalf("startServer() error = %v", err)
	}
}

func TestDebugServerHelper(t *testing.T) {
	if os.Getenv("LLGO_DEBUG_SERVER_HELPER") != "1" {
		return
	}
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		t.Fatal("missing helper port")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+os.Args[separator+1])
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	for {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		connection.Close()
	}
}

func TestArtifactAndArgumentHandling(t *testing.T) {
	command, debugger := splitDebuggerArgs([]string{"-target=board", ".", "--", "--batch", "-ex", "run"})
	if strings.Join(command, " ") != "-target=board ." || strings.Join(debugger, " ") != "--batch -ex run" {
		t.Fatalf("splitDebuggerArgs() = (%v, %v)", command, debugger)
	}

	conf := &build.Config{Target: "cortex-m-qemu", OutFile: filepath.Join(t.TempDir(), "firmware")}
	cleanup, artifact, err := prepareArtifact(conf)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if !strings.HasSuffix(artifact, "firmware.elf") || conf.OutFile != artifact || conf.AppExt != ".elf" {
		t.Fatalf("prepareArtifact() = %q, config=(%q, %q)", artifact, conf.OutFile, conf.AppExt)
	}
	for _, test := range []struct {
		conf build.Config
		want string
	}{
		{conf: build.Config{Goos: "windows"}, want: ".exe"},
		{conf: build.Config{Goos: "wasip1", Goarch: "wasm"}, want: ".wasm"},
		{conf: build.Config{Goos: "linux", Goarch: "amd64"}, want: ""},
	} {
		if got := debugArtifactExtension(&test.conf); got != test.want {
			t.Errorf("debugArtifactExtension(%+v) = %q, want %q", test.conf, got, test.want)
		}
	}
	if target, err := resolveTarget("cortex-m-qemu"); err != nil || target.DebugServer == "" {
		t.Fatalf("resolveTarget(cortex-m-qemu) = (%+v, %v)", target, err)
	}
}

func TestRunBuildsAndLaunchesNativeDebugger(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module debugcommandtest\n\ngo 1.20\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte("package main\nfunc main() { value := 42; println(value) }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(t.TempDir(), "lldb-arguments")
	t.Setenv("LLGO_DEBUG_LLDB_CAPTURE", capture)
	fakeLLDB := writeFakeLLDB(t)

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(moduleDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	flags.Target = ""
	flags.OutputFile = ""
	flags.OptLevel = optlevel.Unset
	flags.Tags = ""
	flags.Verbose = false
	flags.CompilerVerbose = false
	goBuildFlags.Args = nil
	var stdout, stderr bytes.Buffer
	if err := run(nil, []string{"--batch"}, options{backend: backendAuto, lldb: fakeLLDB}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run() error: %v; stderr=%s", err, stderr.String())
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 6 || lines[0] != "-O" || !strings.Contains(lines[1], "command script import") || lines[len(lines)-1] != "--batch" {
		t.Fatalf("LLDB arguments = %q", string(data))
	}
	artifact := lines[len(lines)-2]
	if !strings.Contains(artifact, "llgo-debug-") {
		t.Fatalf("LLDB artifact = %q, want temporary debug artifact", artifact)
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("temporary artifact still exists after debugger exit: %v", err)
	}

	if err := run([]string{".", "./other"}, nil, options{backend: backendAuto}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("multiple packages were accepted")
	}
	if err := run(nil, nil, options{backend: "invalid"}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("invalid backend was accepted")
	}
	goBuildFlags.Args = []string{"-ldflags=-s"}
	if err := run(nil, nil, options{backend: backendAuto}, strings.NewReader(""), &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "debug information is required") {
		t.Fatalf("run(-ldflags=-s) error = %v", err)
	}
	goBuildFlags.Args = nil
}

func TestRunBuildsEmbeddedDWARF(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv("LLGO_DEBUG_GDB_CAPTURE", filepath.Join(t.TempDir(), "gdb-arguments"))
	oldTarget, oldOutput, oldOpt, oldBuildFlags := flags.Target, flags.OutputFile, flags.OptLevel, goBuildFlags.Args
	t.Cleanup(func() {
		flags.Target, flags.OutputFile, flags.OptLevel, goBuildFlags.Args = oldTarget, oldOutput, oldOpt, oldBuildFlags
	})
	flags.Target = "cortex-m-qemu"
	flags.OutputFile = filepath.Join(t.TempDir(), "embedded-debug.elf")
	flags.OptLevel = optlevel.Unset
	goBuildFlags.Args = nil
	var stdout, stderr bytes.Buffer
	fixture := filepath.Join(repoRoot, "test", "debug", "embedded")
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(fixture); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	if err := run([]string{"."}, []string{"--batch"}, options{
		backend: backendGDB, gdb: writeFakeGDB(t), remote: "127.0.0.1:1234",
	}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run() error: %v; stderr=%s", err, stderr.String())
	}
	image, err := elf.Open(flags.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	info, err := image.DWARF()
	if err != nil {
		t.Fatalf("embedded debug image has no DWARF: %v", err)
	}
	reader := info.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			t.Fatal(err)
		}
		if entry == nil {
			break
		}
		if entry.Tag != dwarf.TagCompileUnit {
			continue
		}
		lines, err := info.LineReader(entry)
		if err != nil {
			t.Fatal(err)
		}
		if lines == nil {
			continue
		}
		for _, file := range lines.Files() {
			if file != nil && filepath.Base(file.Name) == "c.go" {
				return
			}
		}
	}
	t.Fatal("embedded debug image has no fixture source line table")
}

func writeFakeGDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gdb")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo 'GNU gdb (GDB) 15.1'
  exit 0
fi
if [ "$1" = "--batch" ] && [ "$2" = "--nx" ]; then
  exit 0
fi
printf '%s\n' "$@" > "$LLGO_DEBUG_GDB_CAPTURE"
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeLLDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lldb")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo 'lldb version 19.1.0'
  exit 0
fi
printf '%s\n' "$@" > "$LLGO_DEBUG_LLDB_CAPTURE"
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
