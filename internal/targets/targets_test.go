//go:build !llgo

package targets

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestConfigBasics(t *testing.T) {
	config := &Config{
		Name:       "test",
		LLVMTarget: "arm-none-eabi",
		GOOS:       "linux",
		GOARCH:     "arm",
	}

	if config.IsEmpty() {
		t.Error("Config should not be empty when fields are set")
	}

	empty := &Config{}
	if !empty.IsEmpty() {
		t.Error("Empty config should report as empty")
	}
}

func TestRawConfigInheritance(t *testing.T) {
	raw := &RawConfig{
		Inherits: []string{"parent1", "parent2"},
		Config: Config{
			Name: "child",
		},
	}

	if !raw.HasInheritance() {
		t.Error("RawConfig should report having inheritance")
	}

	inherits := raw.GetInherits()
	if len(inherits) != 2 || inherits[0] != "parent1" || inherits[1] != "parent2" {
		t.Errorf("Expected inheritance list [parent1, parent2], got %v", inherits)
	}

	noInherit := &RawConfig{}
	if noInherit.HasInheritance() {
		t.Error("RawConfig with no inherits should not report having inheritance")
	}
}

func TestLoaderLoadRaw(t *testing.T) {
	// Create a temporary directory for test configs
	tempDir := t.TempDir()

	// Create a test config file
	testConfig := `{
		"llvm-target": "thumbv6m-unknown-unknown-eabi",
		"cpu": "cortex-m0plus",
		"goos": "linux",
		"goarch": "arm",
		"build-tags": ["test", "embedded"],
		"cflags": ["-Os", "-g"],
		"ldflags": ["--gc-sections"],
		"binary-format": "uf2"
	}`

	configPath := filepath.Join(tempDir, "test-target.json")
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewLoader(tempDir)
	config, err := loader.LoadRaw("test-target")
	if err != nil {
		t.Fatalf("Failed to load raw config: %v", err)
	}

	if config.Name != "test-target" {
		t.Errorf("Expected name 'test-target', got '%s'", config.Name)
	}
	if config.LLVMTarget != "thumbv6m-unknown-unknown-eabi" {
		t.Errorf("Expected llvm-target 'thumbv6m-unknown-unknown-eabi', got '%s'", config.LLVMTarget)
	}
	if config.CPU != "cortex-m0plus" {
		t.Errorf("Expected cpu 'cortex-m0plus', got '%s'", config.CPU)
	}
	if len(config.BuildTags) != 2 || config.BuildTags[0] != "test" || config.BuildTags[1] != "embedded" {
		t.Errorf("Expected build-tags [test, embedded], got %v", config.BuildTags)
	}
	if config.BinaryFormat != "uf2" {
		t.Errorf("Expected binary-format 'uf2', got '%s'", config.BinaryFormat)
	}
}

func TestLoaderInheritance(t *testing.T) {
	tempDir := t.TempDir()

	// Create parent config
	parentConfig := `{
		"llvm-target": "thumbv6m-unknown-unknown-eabi",
		"cpu": "cortex-m0plus",
		"goos": "linux",
		"goarch": "arm",
		"cflags": ["-Os"],
		"ldflags": ["--gc-sections"],
		"binary-format": "elf"
	}`

	// Create child config that inherits from parent
	childConfig := `{
		"inherits": ["parent"],
		"cpu": "cortex-m4",
		"build-tags": ["child"],
		"cflags": ["-O2"],
		"ldflags": ["-g"],
		"binary-format": "uf2"
	}`

	parentPath := filepath.Join(tempDir, "parent.json")
	childPath := filepath.Join(tempDir, "child.json")

	if err := os.WriteFile(parentPath, []byte(parentConfig), 0644); err != nil {
		t.Fatalf("Failed to write parent config: %v", err)
	}
	if err := os.WriteFile(childPath, []byte(childConfig), 0644); err != nil {
		t.Fatalf("Failed to write child config: %v", err)
	}

	loader := NewLoader(tempDir)
	config, err := loader.Load("child")
	if err != nil {
		t.Fatalf("Failed to load child config: %v", err)
	}

	// Check inherited values
	if config.LLVMTarget != "thumbv6m-unknown-unknown-eabi" {
		t.Errorf("Expected inherited llvm-target 'thumbv6m-unknown-unknown-eabi', got '%s'", config.LLVMTarget)
	}
	if config.GOOS != "linux" {
		t.Errorf("Expected inherited goos 'linux', got '%s'", config.GOOS)
	}
	if config.GOARCH != "arm" {
		t.Errorf("Expected inherited goarch 'arm', got '%s'", config.GOARCH)
	}

	// Check overridden values
	if config.CPU != "cortex-m4" {
		t.Errorf("Expected overridden cpu 'cortex-m4', got '%s'", config.CPU)
	}

	// Check binary-format override
	if config.BinaryFormat != "uf2" {
		t.Errorf("Expected overridden binary-format 'uf2', got '%s'", config.BinaryFormat)
	}

	// Check merged arrays
	expectedCFlags := []string{"-Os", "-O2"}
	if len(config.CFlags) != 2 || config.CFlags[0] != "-Os" || config.CFlags[1] != "-O2" {
		t.Errorf("Expected merged cflags %v, got %v", expectedCFlags, config.CFlags)
	}

	expectedLDFlags := []string{"--gc-sections", "-g"}
	if len(config.LDFlags) != 2 || config.LDFlags[0] != "--gc-sections" || config.LDFlags[1] != "-g" {
		t.Errorf("Expected merged ldflags %v, got %v", expectedLDFlags, config.LDFlags)
	}

	// Check child-specific values
	if len(config.BuildTags) != 1 || config.BuildTags[0] != "child" {
		t.Errorf("Expected build-tags [child], got %v", config.BuildTags)
	}
}

func TestLoaderListTargets(t *testing.T) {
	tempDir := t.TempDir()

	// Create some test config files
	configs := []string{"target1.json", "target2.json", "not-a-target.txt"}
	for _, config := range configs {
		configPath := filepath.Join(tempDir, config)
		if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to write config %s: %v", config, err)
		}
	}

	loader := NewLoader(tempDir)
	targets, err := loader.ListTargets()
	if err != nil {
		t.Fatalf("Failed to list targets: %v", err)
	}

	expectedTargets := []string{"target1", "target2"}
	if len(targets) != len(expectedTargets) {
		t.Errorf("Expected %d targets, got %d", len(expectedTargets), len(targets))
	}

	for _, expected := range expectedTargets {
		found := false
		for _, target := range targets {
			if target == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected target %s not found in list %v", expected, targets)
		}
	}
}

func TestResolver(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test config
	testConfig := `{
		"llvm-target": "wasm32-unknown-wasi",
		"cpu": "generic",
		"goos": "wasip1",
		"goarch": "wasm"
	}`

	configPath := filepath.Join(tempDir, "wasi.json")
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	resolver := NewResolver(tempDir)

	// Test resolve
	config, err := resolver.Resolve("wasi")
	if err != nil {
		t.Fatalf("Failed to resolve target: %v", err)
	}

	if config.Name != "wasi" {
		t.Errorf("Expected name 'wasi', got '%s'", config.Name)
	}

	// Test has target
	if !resolver.HasTarget("wasi") {
		t.Error("Resolver should report having 'wasi' target")
	}
	if resolver.HasTarget("nonexistent") {
		t.Error("Resolver should not report having 'nonexistent' target")
	}

	// Test list available targets
	targets, err := resolver.ListAvailableTargets()
	if err != nil {
		t.Fatalf("Failed to list available targets: %v", err)
	}

	if len(targets) != 1 || targets[0] != "wasi" {
		t.Errorf("Expected targets [wasi], got %v", targets)
	}
}

func TestResolverWithRealTargets(t *testing.T) {
	// Test with actual targets directory if it exists
	resolver := NewDefaultResolver()
	targetsDir := resolver.GetTargetsDirectory()

	// Check if targets directory exists
	if _, err := os.Stat(targetsDir); os.IsNotExist(err) {
		t.Skipf("Targets directory %s does not exist, skipping real targets test", targetsDir)
	}

	// Test listing real targets
	targets, err := resolver.ListAvailableTargets()
	if err != nil {
		t.Fatalf("Failed to list real targets: %v", err)
	}

	t.Logf("Found %d targets in %s", len(targets), targetsDir)

	// Test resolving some known targets
	knownTargets := []string{"wasi", "cortex-m", "rp2040"}
	for _, targetName := range knownTargets {
		if resolver.HasTarget(targetName) {
			config, err := resolver.Resolve(targetName)
			if err != nil {
				t.Errorf("Failed to resolve known target %s: %v", targetName, err)
				continue
			}
			t.Logf("Resolved target %s: LLVM=%s, CPU=%s, GOOS=%s, GOARCH=%s",
				targetName, config.LLVMTarget, config.CPU, config.GOOS, config.GOARCH)
		}
	}
}

func TestWebAssemblyProfileTargets(t *testing.T) {
	resolver := NewDefaultResolver()
	tests := []struct {
		name, llvmTarget, goos, goarch, profile, provider string
		wantTags                                          []string
	}{
		{"emscripten", "wasm32-unknown-emscripten", "js", "wasm", "j32", "emscripten", []string{"llgo.wasm.emscripten"}},
		{"emscripten-memory64", "wasm64-unknown-emscripten", "js", "wasm", "j64", "emscripten", []string{"llgo.wasm.emscripten", "llgo.wasm.emscripten.memory64"}},
		{"wasm", "wasm32-unknown-emscripten", "js", "wasm", "j32", "emscripten", []string{"llgo.wasm.emscripten"}},
		{"wasi", "wasm32-unknown-wasip1", "wasip1", "wasm", "w32", "wasi", []string{"llgo.wasm.wasi"}},
		{"wasip1", "wasm32-unknown-wasip1", "wasip1", "wasm", "w32", "wasi", []string{"llgo.wasm.wasi"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := resolver.Resolve(test.name)
			if err != nil {
				t.Fatal(err)
			}
			if config.LLVMTarget != test.llvmTarget || config.GOOS != test.goos || config.GOARCH != test.goarch ||
				config.WasmProfile != test.profile || config.WasmProvider != test.provider {
				t.Fatalf("profile = LLVM %q, %s/%s, %q/%q; want LLVM %q, %s/%s, %q/%q",
					config.LLVMTarget, config.GOOS, config.GOARCH, config.WasmProfile, config.WasmProvider,
					test.llvmTarget, test.goos, test.goarch, test.profile, test.provider)
			}
			if config.CPU != "generic" {
				t.Errorf("CPU = %q, want generic", config.CPU)
			}
			for _, tag := range test.wantTags {
				if !slices.Contains(config.BuildTags, tag) {
					t.Errorf("build tags %v do not contain %q", config.BuildTags, tag)
				}
			}
			if slices.Contains(config.BuildTags, "tinygo.wasm") {
				t.Errorf("build tags %v retain the obsolete TinyGo compatibility tag", config.BuildTags)
			}
			if test.provider == "emscripten" {
				wantRunner := "emscripten-runner.mjs"
				if test.profile == "j64" {
					wantRunner = "emscripten-memory64-runner.mjs"
				}
				if !strings.Contains(config.Emulator, wantRunner) {
					t.Errorf("emulator %q does not instantiate Emscripten module output", config.Emulator)
				}
			}
			if test.provider == "wasi" {
				// R1 translates LLVM's legacy Wasm SjLj encoding to standardized
				// exnref instructions after Asyncify instrumentation. Wasmtime keeps
				// that proposal opt-in, so the inherited public emulator must enable it.
				if !strings.Contains(config.Emulator, "-W exceptions=y") {
					t.Errorf("emulator %q does not enable the exception proposal", config.Emulator)
				}
			}
		})
	}
}

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

func TestResolveAllRealTargets(t *testing.T) {
	resolver := NewDefaultResolver()
	targetsDir := resolver.GetTargetsDirectory()

	// Check if targets directory exists
	if _, err := os.Stat(targetsDir); os.IsNotExist(err) {
		t.Skipf("Targets directory %s does not exist, skipping resolve all test", targetsDir)
	}

	// Test resolving all targets
	configs, err := resolver.ResolveAll()
	if err != nil {
		t.Fatalf("Failed to resolve all targets: %v", err)
	}

	t.Logf("Successfully resolved %d targets", len(configs))

	// Check that all configs have names
	for name, config := range configs {
		if config.Name != name {
			t.Errorf("Config name mismatch: key=%s, config.Name=%s", name, config.Name)
		}
		if config.IsEmpty() {
			t.Errorf("Config %s appears to be empty", name)
		}
	}

	// Log some statistics
	goosCounts := make(map[string]int)
	goarchCounts := make(map[string]int)

	for _, config := range configs {
		if config.GOOS != "" {
			goosCounts[config.GOOS]++
		}
		if config.GOARCH != "" {
			goarchCounts[config.GOARCH]++
		}
	}

	t.Logf("GOOS distribution: %v", goosCounts)
	t.Logf("GOARCH distribution: %v", goarchCounts)
}
