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

package crosscompile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmscriptenRunnersForwardProgramArguments(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required to test the Emscripten runners: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "module.mjs")
	const source = `export default async function(config) {
	if (JSON.stringify(config.arguments) !== '["-test.run=TestOne","value"]') {
		throw new Error('arguments: ' + JSON.stringify(config.arguments));
	}
	const module = { ENV: {} };
	for (const hook of config.preRun) hook(module);
	if (module.ENV.LLGO_RUNNER_TEST !== 'present') {
		throw new Error('environment was not forwarded');
	}
}`
	if err := os.WriteFile(module, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, runner := range []string{"emscripten-runner.mjs", "emscripten-memory64-runner.mjs"} {
		t.Run(runner, func(t *testing.T) {
			cmd := exec.Command(node, filepath.Join(root, "targets", runner), module, "-test.run=TestOne", "value")
			cmd.Env = append(os.Environ(), "LLGO_RUNNER_TEST=present")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("runner failed: %v\n%s", err, output)
			}
		})
	}
}

func TestEmscriptenRunnerModelsBrowserForRawGoJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required to test the Emscripten runner: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "module.mjs")
	const source = `export default async function() {
	if (globalThis.window !== globalThis) throw new Error('browser window is missing');
	if (globalThis.process !== undefined) throw new Error('Node process leaked into browser-only output');
}`
	if err := os.WriteFile(module, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, filepath.Join(root, "targets", "emscripten-runner.mjs"), "--browser-only", module)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("browser-only runner failed: %v\n%s", err, output)
	}
}

func TestEmscriptenRunnersConsumeExitStatus(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required to test the Emscripten runners: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "module.mjs")
	const source = `function exitStatus(status) {
	const error = new Error('Program terminated with exit(' + status + ')');
	error.name = 'ExitStatus';
	error.status = status;
	return error;
}

export default async function(config) {
	const status = Number(config.arguments[0]);
	const mode = config.arguments[1];
	config.onExit(status);
	switch (mode) {
	case 'factory-rejection':
		throw exitStatus(status);
	case 'late-exception':
		setTimeout(() => { throw exitStatus(status); }, 0);
		return;
	case 'late-rejection':
		Promise.reject(exitStatus(status));
		return;
	case 'ordinary-error': {
		const error = new Error('ordinary failure');
		error.status = status;
		throw error;
	}
	default:
		throw new Error('unknown mode: ' + mode);
	}
}`
	if err := os.WriteFile(module, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, runner := range []string{"emscripten-runner.mjs", "emscripten-memory64-runner.mjs"} {
		t.Run(runner, func(t *testing.T) {
			command := func(status, mode string) *exec.Cmd {
				return exec.Command(node, filepath.Join(root, "targets", runner), module, status, mode)
			}
			for _, mode := range []string{"factory-rejection", "late-exception", "late-rejection"} {
				if output, err := command("0", mode).CombinedOutput(); err != nil {
					t.Fatalf("runner rejected normal %s ExitStatus: %v\n%s", mode, err, output)
				}
				output, err := command("7", mode).CombinedOutput()
				if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 7 {
					t.Fatalf("runner %s nonzero ExitStatus = %v, want exit 7\n%s", mode, err, output)
				}
			}
			output, err := command("0", "ordinary-error").CombinedOutput()
			if err == nil {
				t.Fatalf("runner consumed an ordinary error with a status field:\n%s", output)
			}
		})
	}
}

func TestEmscriptenRunnerReportsInstantiationFailure(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required to test the Emscripten runner: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	module := filepath.Join(tempDir, "invalid.mjs")
	const source = `export default async function(config) {
	config.instantiateWasm({}, () => {});
	await new Promise(() => {});
}`
	if err := os.WriteFile(module, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "invalid.wasm"), []byte("invalid wasm"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, filepath.Join(root, "targets", "emscripten-runner.mjs"), "--browser-only", module)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("runner accepted an invalid WebAssembly module:\n%s", output)
	}
	if !strings.Contains(string(output), "CompileError") {
		t.Fatalf("runner did not report the instantiation error: %v\n%s", err, output)
	}
}
