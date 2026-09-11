package goroot

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type gorootWasmProfile struct {
	name       string
	target     string
	goos       string
	llgoSuffix string
	runner     string
}

func selectGOROOTWasmProfile(name string) (gorootWasmProfile, bool, error) {
	switch name {
	case "":
		return gorootWasmProfile{}, false, nil
	case "EC32":
		return gorootWasmProfile{name: name, target: "emscripten", goos: "js", llgoSuffix: ".mjs", runner: "emscripten-runner.mjs"}, true, nil
	case "EC64":
		return gorootWasmProfile{name: name, target: "emscripten-memory64", goos: "js", llgoSuffix: ".mjs", runner: "emscripten-memory64-runner.mjs"}, true, nil
	case "WC32":
		return gorootWasmProfile{name: name, target: "wasi", goos: "wasip1", llgoSuffix: ".wasm", runner: "wasmtime"}, true, nil
	case "GJS":
		return gorootWasmProfile{name: name, goos: "js", llgoSuffix: ".mjs", runner: "emscripten-runner.mjs"}, true, nil
	case "GWASI":
		return gorootWasmProfile{name: name, goos: "wasip1", llgoSuffix: ".wasm", runner: "wasmtime"}, true, nil
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
	if _, ok := activeGOROOTWasmProfile(); ok {
		// Official Go's current js/wasm and wasip1/wasm ports do not create
		// operating-system threads. These profiles intentionally exercise the
		// same single-worker contract in LLGo while the native driver and the
		// compiler may continue to use the CI job's wider GOMAXPROCS setting.
		out = upsertEnv(out, "GOMAXPROCS=1")
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
	args = append(args, flags...)
	return append(args, "-o", output, target)
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
	if p.runner == "wasmtime" {
		args := []string{"run", "-W", "exceptions=y", "--dir=."}
		args = append(args, artifact)
		args = append(args, programArgs...)
		return "wasmtime", args, gorootRuntimeEnv(env), nil
	}
	runner := filepath.Join(root, "targets", p.runner)
	args := append([]string{runner, artifact}, programArgs...)
	return "node", args, gorootRuntimeEnv(env), nil
}
