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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/browser"
	"github.com/xgo-dev/llgo/cmd/internal/gdb"
	"github.com/xgo-dev/llgo/cmd/internal/lldb"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/shellparse"
	"github.com/xgo-dev/llgo/internal/targets"
)

type backend string

const (
	backendAuto     backend = "auto"
	backendLLDB     backend = "lldb"
	backendGDB      backend = "gdb"
	backendWasmtime backend = "wasmtime"
	backendBrowser  backend = "browser"
)

type targetKind uint8

const (
	targetNative targetKind = iota
	targetEmbedded
	targetWASI
	targetBrowser
)

type options struct {
	backend backend
	lldb    string
	gdb     string
	remote  string
	server  string
	load    bool
	browser browser.Options
}

func (o options) validate() error {
	switch o.backend {
	case backendAuto, backendLLDB, backendGDB, backendWasmtime, backendBrowser:
		return nil
	default:
		return fmt.Errorf("llgo debug: unknown backend %q; use auto, lldb, gdb, wasmtime, or browser", o.backend)
	}
}

func classifyTarget(conf *build.Config, target *targets.Config) targetKind {
	goos, goarch, llvmTarget := conf.Goos, conf.Goarch, ""
	if target != nil {
		if target.GOOS != "" {
			goos = target.GOOS
		}
		if target.GOARCH != "" {
			goarch = target.GOARCH
		}
		llvmTarget = target.LLVMTarget
	}
	if goarch == "wasm" || strings.HasPrefix(llvmTarget, "wasm") {
		if goos == "js" || (target != nil && (target.WasmProfile == "j32" || target.WasmProfile == "j64")) {
			return targetBrowser
		}
		return targetWASI
	}
	if target != nil {
		return targetEmbedded
	}
	return targetNative
}

func selectBackend(requested backend, kind targetKind) (backend, error) {
	if requested != backendAuto {
		switch kind {
		case targetWASI:
			if requested != backendWasmtime {
				return "", wasiDebuggerUnavailable()
			}
		case targetBrowser:
			if requested != backendBrowser {
				return "", fmt.Errorf("llgo debug: backend %s cannot debug a browser target; use browser", requested)
			}
		default:
			if requested != backendLLDB && requested != backendGDB {
				return "", fmt.Errorf("llgo debug: backend %s cannot debug this target", requested)
			}
		}
		return requested, nil
	}
	switch kind {
	case targetEmbedded:
		return backendGDB, nil
	case targetWASI:
		return backendWasmtime, nil
	case targetBrowser:
		return backendBrowser, nil
	default:
		return backendLLDB, nil
	}
}

type session struct {
	backend      backend
	artifact     string
	debuggerArgs []string
	target       *targets.Config
	options      options
}

func runSession(s session, stdin io.Reader, stdout, stderr io.Writer) error {
	if err := validateBackendOptions(s.backend, s.options); err != nil {
		return err
	}
	if s.backend == backendBrowser {
		opts := s.options.browser
		opts.ChromeArgs = append(append([]string(nil), opts.ChromeArgs...), s.debuggerArgs...)
		return browser.Run(browserModulePath(s.artifact), opts, stdin, stdout, stderr)
	}
	plan, err := makeServerPlan(s.target, s.artifact, s.options)
	if err != nil {
		return err
	}
	var server *debugServer
	if plan != nil {
		server, err = startServer(*plan)
		if err != nil {
			return err
		}
		defer server.stop()
		if server != nil {
			plan.address = server.address
		}
	}
	args, err := debuggerArguments(s.backend, s.artifact, s.debuggerArgs, plan)
	if err != nil {
		return err
	}

	var debugErr error
	switch s.backend {
	case backendLLDB:
		if err := lldb.Run(s.options.lldb, args, stdin, stdout, stderr); err != nil {
			debugErr = fmt.Errorf("llgo debug: %w", err)
		}
	case backendGDB:
		var candidates []string
		if s.target != nil {
			candidates = s.target.GDB
		}
		if err := gdb.Run(s.options.gdb, candidates, args, stdin, stdout, stderr); err != nil {
			debugErr = err
		}
	default:
		debugErr = fmt.Errorf("llgo debug: backend %s is not implemented", s.backend)
	}
	if server != nil {
		select {
		case err := <-server.done:
			server.finished = true
			if err != nil && debugErr == nil {
				debugErr = fmt.Errorf("llgo debug: debug server exited: %w", err)
			}
		default:
		}
	}
	if debugErr != nil && server != nil {
		if output := server.logSuffix(); output != "" {
			return fmt.Errorf("%w\ndebug server output:%s", debugErr, output)
		}
	}
	return debugErr
}

func wasiDebuggerUnavailable() error {
	return errors.New("llgo debug: WASI source-debug sessions are unavailable for W32 pthread modules: the current Wasmtime backend does not implement shared env.memory, wasi.thread-spawn and wasix_32v1.thread_exit; use llgo run -target=wasi with Wasmer for execution")
}

func validateBackendOptions(selected backend, opts options) error {
	if selected == backendWasmtime {
		return wasiDebuggerUnavailable()
	}
	if selected == backendBrowser && (opts.remote != "" || opts.server != "" || opts.load) {
		return errors.New("llgo debug: browser sessions use their own loopback HTTP server; -remote, -server and -load apply to native/embedded debugger transports")
	}
	return nil
}

type serverPlan struct {
	command []string
	address string
	load    bool
	stdio   bool
	openOCD bool
}

func makeServerPlan(target *targets.Config, artifact string, opts options) (*serverPlan, error) {
	if target == nil {
		if opts.server != "" {
			return nil, errors.New("llgo debug: -server requires -target")
		}
		if opts.remote == "" {
			if opts.load {
				return nil, errors.New("llgo debug: -load requires a remote debug server")
			}
			return nil, nil
		}
		return &serverPlan{address: normalizeRemoteAddress(opts.remote), load: opts.load}, nil
	}
	if opts.remote != "" {
		if opts.server != "" {
			return nil, errors.New("llgo debug: -remote and -server are mutually exclusive")
		}
		return &serverPlan{address: normalizeRemoteAddress(opts.remote), load: opts.load}, nil
	}

	serverTemplate := opts.server
	if serverTemplate == "" {
		serverTemplate = target.DebugServer
	}
	if serverTemplate != "" {
		stdio := strings.Contains(serverTemplate, "{debug-stdio}")
		port := 0
		if !stdio {
			var err error
			port, err = freeTCPPort()
			if err != nil {
				return nil, fmt.Errorf("llgo debug: allocate debug-server port: %w", err)
			}
		}
		command, err := parseServerCommand(serverTemplate, artifact, port)
		if err != nil {
			return nil, err
		}
		return &serverPlan{command: command, address: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), load: opts.load, stdio: stdio}, nil
	}
	if target.OpenOCDInterface == "" && target.OpenOCDTarget == "" {
		return nil, fmt.Errorf("llgo debug: target %s has no debug server; use -remote or -server", target.Name)
	}
	command := []string{"openocd", "-c", "gdb_port pipe", "-c", "tcl_port disabled", "-c", "telnet_port disabled"}
	if target.OpenOCDInterface != "" {
		command = append(command, "-f", "interface/"+target.OpenOCDInterface+".cfg")
	}
	if target.OpenOCDTransport != "" {
		command = append(command, "-c", "transport select "+target.OpenOCDTransport)
	}
	if target.OpenOCDTarget != "" {
		command = append(command, "-f", "target/"+target.OpenOCDTarget+".cfg")
	}
	return &serverPlan{
		command: command,
		load:    true,
		stdio:   true,
		openOCD: true,
	}, nil
}

func normalizeRemoteAddress(address string) string {
	address = strings.TrimPrefix(address, "tcp://")
	if strings.HasPrefix(address, ":") {
		return "127.0.0.1" + address
	}
	return address
}

func parseServerCommand(template, artifact string, port int) ([]string, error) {
	replacer := strings.NewReplacer(
		"{debug-stdio}", "stdio",
		"{debug-port}", strconv.Itoa(port),
		"{root}", quoteServerArgument(env.LLGoROOT()),
		"{tmpDir}", quoteServerArgument(os.TempDir()),
		"{elf}", quoteServerArgument(artifact),
		"{}", quoteServerArgument(artifact),
	)
	command, err := shellparse.Parse(replacer.Replace(template))
	if err != nil {
		return nil, fmt.Errorf("llgo debug: parse debug-server command: %w", err)
	}
	if len(command) == 0 {
		return nil, errors.New("llgo debug: debug-server command is empty")
	}
	return command, nil
}

func quoteServerArgument(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

func freeTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func debuggerArguments(selected backend, artifact string, extra []string, server *serverPlan) ([]string, error) {
	if server == nil {
		return append([]string{artifact}, extra...), nil
	}
	if server.address == "" {
		return nil, errors.New("llgo debug: remote debug-server address is empty")
	}
	switch selected {
	case backendGDB:
		args := []string{"--quiet", artifact}
		remoteCommand := "target remote " + server.address
		if server.load {
			remoteCommand = "target extended-remote " + server.address
		}
		args = append(args, "-ex", remoteCommand)
		if server.load {
			args = append(args, "-ex", "monitor reset halt", "-ex", "load", "-ex", "monitor reset halt")
		}
		return append(args, extra...), nil
	case backendLLDB:
		args := []string{
			artifact,
			"-o", "gdb-remote " + server.address,
		}
		load := "target modules load --file " + quoteLLDBArgument(artifact) + " --slide 0"
		if server.load {
			args = append(args, "-o", "process plugin packet monitor reset halt")
			load += " --load"
		}
		args = append(args, "-o", load)
		if server.load {
			args = append(args, "-o", "process plugin packet monitor reset halt")
		}
		return append(args, extra...), nil
	default:
		return nil, fmt.Errorf("llgo debug: backend %s does not use GDB Remote", selected)
	}
}

func quoteLLDBArgument(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(filepath.ToSlash(value)) + `"`
}
