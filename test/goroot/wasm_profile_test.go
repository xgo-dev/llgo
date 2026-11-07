package goroot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type gorootWasmProfile struct {
	name        string
	target      string
	goos        string
	llgoSuffix  string
	runner      string
	browserOnly bool
}

func selectGOROOTWasmProfile(name string) (gorootWasmProfile, bool, error) {
	switch name {
	case "":
		return gorootWasmProfile{}, false, nil
	case "J32-Emscripten":
		return gorootWasmProfile{name: name, target: "emscripten", goos: "js", llgoSuffix: ".mjs", runner: "emscripten-runner.mjs"}, true, nil
	case "J64-Emscripten":
		return gorootWasmProfile{name: name, target: "emscripten-memory64", goos: "js", llgoSuffix: ".mjs", runner: "emscripten-memory64-runner.mjs"}, true, nil
	case "W32-WASI":
		return gorootWasmProfile{name: name, target: "wasi", goos: "wasip1", llgoSuffix: ".wasm", runner: "wasmer"}, true, nil
	case "J32-GoJS":
		return gorootWasmProfile{name: name, goos: "js", llgoSuffix: ".mjs", runner: "emscripten-runner.mjs", browserOnly: true}, true, nil
	default:
		return gorootWasmProfile{}, false, fmt.Errorf("unknown -wasm-profile=%q", name)
	}
}

func activeGOROOTWasmProfile() (gorootWasmProfile, bool) {
	p, ok, err := selectGOROOTWasmProfile(*flagWasmProfile)
	if err != nil {
		panic(err) // TestGoRootRunCases validates the flag before executing cases.
	}
	return p, ok
}

func gorootTargetEnv(env []string) []string {
	p, ok := activeGOROOTWasmProfile()
	if !ok {
		return env
	}
	out := append([]string{}, env...)
	out = upsertEnv(out, "GOOS="+p.goos)
	out = upsertEnv(out, "GOARCH=wasm")
	out = upsertEnv(out, "CGO_ENABLED=0")
	if p.goos == "wasip1" {
		out = upsertEnv(out, "GOWASIRUNTIME=wasmtime")
	}
	return out
}

func gorootRuntimeEnv(env []string) []string {
	out := gorootTargetEnv(env)
	if p, ok := activeGOROOTWasmProfile(); ok {
		if p.goos == "js" {
			out = gorootJSRuntimeEnv(out)
		}
		// Keep the official Go baseline deterministic. The LLGo pthread backend
		// uses Wasmer and can still create Ms.
		out = upsertEnv(out, "GOMAXPROCS=1")
	}
	return out
}

// Go's JS runner copies the host environment into a fixed-size Wasm startup
// area. CI metadata can exhaust that area before main runs. Give both Go and
// LLGo the same runtime environment, retaining test controls and the variables
// needed to launch Node. Compilation still receives the full environment.
func gorootJSRuntimeEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		key, _, _ := strings.Cut(item, "=")
		switch strings.ToUpper(key) {
		case "PATH", "HOME", "USERPROFILE", "SYSTEMROOT", "WINDIR",
			"TMPDIR", "TMP", "TEMP", "TZ",
			"NODE_OPTIONS", "NODE_PATH", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH",
			"GOROOT", "GOPATH", "GOOS", "GOARCH", "CGO_ENABLED", "GO111MODULE",
			"GOEXPERIMENT", "GODEBUG", "GOGC", "GOMEMLIMIT", "GOTRACEBACK", "GOSSAINTERP":
			out = append(out, item)
		}
	}
	return out
}

func gorootArtifactPath(rootDir, label string, llgo bool) string {
	p, ok := activeGOROOTWasmProfile()
	if ok {
		if llgo {
			return filepath.Join(rootDir, label+p.llgoSuffix)
		}
		return filepath.Join(rootDir, label+".wasm")
	}
	out := filepath.Join(rootDir, label+".out")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	return out
}

func gorootBuildArgs(llgo bool, flags []string, output, target string) []string {
	args := []string{"build"}
	if llgo {
		if p, ok := activeGOROOTWasmProfile(); ok && p.target != "" {
			args = append(args, "-target", p.target)
		}
	}
	for i := 0; i < len(flags); i++ {
		// These gc backend controls stay on the official Go baseline;
		// LLVM runs the same source assertions with its own backend.
		// Keep other payloads intact so unsupported options still fail.
		if payload, ok := strings.CutPrefix(flags[i], "-gcflags="); llgo && ok && gorootGCBackendOnlyFlags(payload) {
			continue
		}
		if (flags[i] == "-gcflags" || flags[i] == "-tags" || flags[i] == "-ldflags") && i+1 < len(flags) {
			if !llgo || flags[i] != "-gcflags" || !gorootGCBackendOnlyFlags(flags[i+1]) {
				args = append(args, flags[i], flags[i+1])
			}
			i++
			continue
		}
		args = append(args, flags[i])
	}
	return append(args, "-o", output, target)
}

func gorootGCBackendOnlyFlags(payload string) bool {
	switch payload {
	case "-l=4", "-d=ssa/check/on", "-d=softfloat", "-d=checkptr", "all=-d=checkptr":
		return true
	}
	return false
}

func envEntry(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func runGOROOTArtifact(dir, artifact string, llgo bool, env []string, timeout time.Duration, programArgs ...string) ([]byte, []byte, int, time.Duration, error) {
	app, args, runEnv, err := gorootArtifactCommand(dir, artifact, llgo, env, programArgs...)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	return runProgram(dir, app, runEnv, timeout, args...)
}

func gorootArtifactCommand(dir, artifact string, llgo bool, env []string, programArgs ...string) (string, []string, []string, error) {
	p, ok := activeGOROOTWasmProfile()
	if !ok {
		return artifact, programArgs, env, nil
	}
	if p.runner == "wasmer" {
		cwd, err := filepath.Abs(dir)
		if err != nil {
			return "", nil, nil, fmt.Errorf("resolve Wasmer working directory: %w", err)
		}
		args := gorootWASIArgs(cwd, os.TempDir(), artifact, runtime.GOOS == "windows")
		// Source-tree tests such as winbatch.go read runtime.GOROOT(). WASI
		// guests need that directory preopened in addition to their workdir.
		if goroot := envEntry(env, "GOROOT"); goroot != "" {
			volume, guest := goroot, goroot
			if runtime.GOOS == "windows" {
				volume, guest = goroot+":/goroot", "/goroot"
			}
			args = append([]string{"run", "--volume=" + volume, "--env=GOROOT=" + guest}, args[1:]...)
		}
		// Wasmer does not inherit the host environment into the WASI guest.
		// Keep PATH explicit, as in the public runner, so both baselines see it.
		if path := envEntry(env, "PATH"); path != "" {
			args = append([]string{"run", "--env=PATH=" + path}, args[1:]...)
		}
		// Suppress engine tracing at its source while preserving the module
		// cache and guest stdout/stderr, including log-shaped guest output.
		runEnv := upsertEnv(gorootRuntimeEnv(env), "RUST_LOG=off")
		return "wasmer", append(args, programArgs...), runEnv, nil
	}
	if !llgo {
		goroot := envEntry(env, "GOROOT")
		if goroot == "" {
			return "", nil, nil, errors.New("target Go execution requires GOROOT")
		}
		runner := filepath.Join(goroot, "lib", "wasm", "go_"+p.goos+"_wasm_exec")
		return runner, append([]string{artifact}, programArgs...), gorootRuntimeEnv(env), nil
	}
	root := envEntry(env, "LLGO_ROOT")
	if root == "" {
		return "", nil, nil, errors.New("target LLGo execution requires LLGO_ROOT")
	}
	runner := filepath.Join(root, "targets", p.runner)
	// Match Go's go_js_wasm_exec wrapper: the default 984 KiB V8 stack
	// cannot run deep-recursion cases such as stackobj2.go.
	args := []string{"--stack-size=8192", runner}
	if p.browserOnly {
		args = append(args, "--browser-only")
	}
	args = append(args, artifact)
	args = append(args, programArgs...)
	return "node", args, gorootRuntimeEnv(env), nil
}

// uintptrescapes deliberately forces stack growth with 4096 recursive frames.
// LLGo's wasm goroutine stacks are fixed, so reserve a test-specific budget.
// Other cases retain the default to avoid multiplying every goroutine's memory.
func gorootWasmCaseBuildFlags(casePath string, flags []string) []string {
	if _, ok := activeGOROOTWasmProfile(); ok && casePath == "uintptrescapes.go" {
		return append(append([]string(nil), flags...), "-goroutine-stack-size=8MB")
	}
	return flags
}

// gorootWASIArgs applies the same host directory contract as the
// public WASI runner. The host is explicit so both contracts can be tested.
func gorootWASIArgs(cwd, tempDir, artifact string, windows bool) []string {
	workVolume, tempVolume, guestCwd := cwd, "/tmp", cwd
	if windows {
		// Map drive-letter paths to POSIX guest names, including Go's default
		// temporary directory. Wasmer selects its available backend itself.
		workVolume, tempVolume, guestCwd = cwd+":/work", tempDir+":/tmp", "/work"
	}
	return []string{"run", "--enable-exceptions", "--enable-simd", "--stack-size=1048576",
		"--volume=" + workVolume, "--volume=" + tempVolume, "--env=PWD=" + guestCwd, artifact, "--"}
}
