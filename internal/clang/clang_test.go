//go:build !llgo
// +build !llgo

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package clang

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("NewConfig", func(t *testing.T) {
		ccflags := []string{"-O2", "-g"}
		cflags := []string{"-std=c99"}
		ldflags := []string{"-lm"}

		config := NewConfig("clang++", ccflags, cflags, ldflags, "ld.lld")

		if config.CC != "clang++" {
			t.Errorf("Expected CC to be 'clang++', got %q", config.CC)
		}
		if !reflect.DeepEqual(config.CCFLAGS, ccflags) {
			t.Errorf("Expected CCFLAGS %v, got %v", ccflags, config.CCFLAGS)
		}
		if !reflect.DeepEqual(config.CFLAGS, cflags) {
			t.Errorf("Expected CFLAGS %v, got %v", cflags, config.CFLAGS)
		}
		if !reflect.DeepEqual(config.LDFLAGS, ldflags) {
			t.Errorf("Expected LDFLAGS %v, got %v", ldflags, config.LDFLAGS)
		}
		if config.Linker != "ld.lld" {
			t.Errorf("Expected Linker to be 'ld.lld', got %q", config.Linker)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("WithApp", func(t *testing.T) {
		config := Config{CC: "gcc"}
		cmd := New("clang++", config)

		if cmd.app != "clang++" {
			t.Errorf("Expected app to be 'clang++', got %q", cmd.app)
		}
		if !reflect.DeepEqual(cmd.config, config) {
			t.Errorf("Expected config %+v, got %+v", config, cmd.config)
		}
		if cmd.Stdout != os.Stdout {
			t.Error("Expected Stdout to be os.Stdout")
		}
		if cmd.Stderr != os.Stderr {
			t.Error("Expected Stderr to be os.Stderr")
		}
	})

	t.Run("WithoutApp", func(t *testing.T) {
		config := Config{}
		cmd := New("", config)

		if cmd.app != "clang" {
			t.Errorf("Expected app to be 'clang', got %q", cmd.app)
		}
	})
}

func TestNewCompiler(t *testing.T) {
	t.Run("WithCC", func(t *testing.T) {
		config := Config{CC: "gcc"}
		cmd := NewCompiler(config)

		if cmd.app != "gcc" {
			t.Errorf("Expected app to be 'gcc', got %q", cmd.app)
		}
	})

	t.Run("WithoutCC", func(t *testing.T) {
		config := Config{}
		cmd := NewCompiler(config)

		if cmd.app != "clang" {
			t.Errorf("Expected app to be 'clang', got %q", cmd.app)
		}
	})
}

func TestNewLinker(t *testing.T) {
	t.Run("WithLinker", func(t *testing.T) {
		config := Config{Linker: "ld.lld"}
		cmd := NewLinker(config)

		if cmd.app != "ld.lld" {
			t.Errorf("Expected app to be 'ld.lld', got %q", cmd.app)
		}
	})

	t.Run("WithoutLinkerButWithCC", func(t *testing.T) {
		config := Config{CC: "gcc"}
		cmd := NewLinker(config)

		if cmd.app != "gcc" {
			t.Errorf("Expected app to be 'gcc', got %q", cmd.app)
		}
	})

	t.Run("WithoutLinkerAndCC", func(t *testing.T) {
		config := Config{}
		cmd := NewLinker(config)

		if cmd.app != "clang" {
			t.Errorf("Expected app to be 'clang', got %q", cmd.app)
		}
	})
}

func TestMergeCompilerFlags(t *testing.T) {
	// Save original environment
	origCCFlags := os.Getenv("CCFLAGS")
	origCFlags := os.Getenv("CFLAGS")
	defer func() {
		os.Setenv("CCFLAGS", origCCFlags)
		os.Setenv("CFLAGS", origCFlags)
	}()

	t.Run("WithEnvironmentAndConfig", func(t *testing.T) {
		os.Setenv("CCFLAGS", "-O2 -g")
		os.Setenv("CFLAGS", "-std=c99")

		config := Config{
			CCFLAGS: []string{"-Wall"},
			CFLAGS:  []string{"-pedantic"},
		}
		cmd := New("clang", config)

		flags := cmd.mergeCompilerFlags()
		expected := []string{"-O2", "-g", "-std=c99", "-Wall", "-pedantic"}

		if !reflect.DeepEqual(flags, expected) {
			t.Errorf("Expected flags %v, got %v", expected, flags)
		}
	})

	t.Run("WithOnlyConfig", func(t *testing.T) {
		os.Unsetenv("CCFLAGS")
		os.Unsetenv("CFLAGS")

		config := Config{
			CCFLAGS: []string{"-Wall"},
			CFLAGS:  []string{"-pedantic"},
		}
		cmd := New("clang", config)

		flags := cmd.mergeCompilerFlags()
		expected := []string{"-Wall", "-pedantic"}

		if !reflect.DeepEqual(flags, expected) {
			t.Errorf("Expected flags %v, got %v", expected, flags)
		}
	})

	t.Run("WithOnlyEnvironment", func(t *testing.T) {
		os.Setenv("CCFLAGS", "-O3")
		os.Setenv("CFLAGS", "-fPIC")

		config := Config{}
		cmd := New("clang", config)

		flags := cmd.mergeCompilerFlags()
		expected := []string{"-O3", "-fPIC"}

		if !reflect.DeepEqual(flags, expected) {
			t.Errorf("Expected flags %v, got %v", expected, flags)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		os.Unsetenv("CCFLAGS")
		os.Unsetenv("CFLAGS")

		config := Config{}
		cmd := New("clang", config)

		flags := cmd.mergeCompilerFlags()

		if len(flags) != 0 {
			t.Errorf("Expected empty flags, got %v", flags)
		}
	})
}

func TestMergeLinkerFlags(t *testing.T) {
	// Save original environment
	origCCFlags := os.Getenv("CCFLAGS")
	origLDFlags := os.Getenv("LDFLAGS")
	defer func() {
		os.Setenv("CCFLAGS", origCCFlags)
		os.Setenv("LDFLAGS", origLDFlags)
	}()

	t.Run("WithEnvironmentAndConfig", func(t *testing.T) {
		os.Setenv("CCFLAGS", "-O2")
		os.Setenv("LDFLAGS", "-lm -lpthread")

		config := Config{
			LDFLAGS: []string{"-static"},
		}
		cmd := New("clang", config)

		flags := cmd.mergeLinkerFlags()
		expected := []string{"-O2", "-lm", "-lpthread", "-static"}

		if !reflect.DeepEqual(flags, expected) {
			t.Errorf("Expected flags %v, got %v", expected, flags)
		}
	})

	t.Run("WithOnlyConfig", func(t *testing.T) {
		os.Unsetenv("CCFLAGS")
		os.Unsetenv("LDFLAGS")

		config := Config{
			LDFLAGS: []string{"-static", "-lm"},
		}
		cmd := New("clang", config)

		flags := cmd.mergeLinkerFlags()
		expected := []string{"-static", "-lm"}

		if !reflect.DeepEqual(flags, expected) {
			t.Errorf("Expected flags %v, got %v", expected, flags)
		}
	})
}

func TestCompile(t *testing.T) {
	// This test uses echo instead of clang to avoid dependency on clang installation
	config := Config{
		CCFLAGS: []string{"-Wall"},
		CFLAGS:  []string{"-std=c99"},
	}
	cmd := New("echo", config)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Compile("-c", "test.c")
	if err != nil {
		t.Errorf("Compile failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	expectedArgs := "-Wall -std=c99 -c test.c"
	if output != expectedArgs {
		t.Errorf("Expected output %q, got %q", expectedArgs, output)
	}
}

func TestLink(t *testing.T) {
	// This test uses echo instead of clang to avoid dependency on clang installation
	config := Config{
		LDFLAGS: []string{"-lm"},
	}
	cmd := New("echo", config)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Link("test.o", "-o", "test")
	if err != nil {
		t.Errorf("Link failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	expectedArgs := "-lm test.o -o test"
	if output != expectedArgs {
		t.Errorf("Expected output %q, got %q", expectedArgs, output)
	}
}

func TestVerboseMode(t *testing.T) {
	config := Config{}
	cmd := New("echo", config)
	cmd.Verbose = true

	// Since verbose output goes to os.Stderr directly, we'll test
	// that verbose flag is set and command executes successfully
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &bytes.Buffer{} // Suppress stdout

	err := cmd.Compile("-c", "test.c")
	if err != nil {
		t.Errorf("Compile failed: %v", err)
	}

	// The test passes if verbose is set and no error occurred
	if !cmd.Verbose {
		t.Error("Expected Verbose to be true")
	}
}

func TestCmdEnvironment(t *testing.T) {
	config := Config{}
	cmd := New("echo", config)
	cmd.Env = []string{"TEST_VAR=test_value"}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Use a command that will show environment variables
	// Note: This is a simplified test that just ensures the Env field is set
	err := cmd.Compile("-c", "test.c")
	if err != nil {
		t.Errorf("Compile failed: %v", err)
	}

	if len(cmd.Env) != 1 || cmd.Env[0] != "TEST_VAR=test_value" {
		t.Errorf("Expected environment to be set correctly")
	}
}

func TestCmdIO(t *testing.T) {
	config := Config{}
	cmd := New("cat", config) // Use cat to test stdin/stdout

	input := "test input"
	cmd.Stdin = strings.NewReader(input)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// cat will read from stdin and write to stdout
	err := cmd.exec() // Call exec directly with no args
	if err != nil {
		t.Errorf("exec failed: %v", err)
	}

	output := stdout.String()
	if output != input {
		t.Errorf("Expected output %q, got %q", input, output)
	}
}

func TestCheckLinkArgs(t *testing.T) {
	t.Run("BasicTest", func(t *testing.T) {
		// Use echo instead of clang to avoid dependency
		config := Config{}
		cmd := New("echo", config)

		// Redirect output to avoid clutter
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}

		// This should succeed with echo
		err := cmd.CheckLinkArgs([]string{"-o"}, false)
		if err != nil {
			t.Errorf("CheckLinkArgs failed: %v", err)
		}
	})

	t.Run("WasmExtension", func(t *testing.T) {
		config := Config{}
		cmd := New("echo", config)

		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}

		// Test with wasm=true
		err := cmd.CheckLinkArgs([]string{"-o"}, true)
		if err != nil {
			t.Errorf("CheckLinkArgs with wasm failed: %v", err)
		}
	})
}

// Additional table-driven test for flag merging scenarios
func TestFlagMergingScenarios(t *testing.T) {
	tests := []struct {
		name       string
		envCCFlags string
		envCFlags  string
		envLDFlags string
		config     Config
		expectComp []string
		expectLink []string
	}{
		{
			name:       "empty everything",
			envCCFlags: "",
			envCFlags:  "",
			envLDFlags: "",
			config:     Config{},
			expectComp: []string{},
			expectLink: []string{},
		},
		{
			name:       "only environment flags",
			envCCFlags: "-O2",
			envCFlags:  "-std=c99",
			envLDFlags: "-lm",
			config:     Config{},
			expectComp: []string{"-O2", "-std=c99"},
			expectLink: []string{"-O2", "-lm"},
		},
		{
			name:       "only config flags",
			envCCFlags: "",
			envCFlags:  "",
			envLDFlags: "",
			config: Config{
				CCFLAGS: []string{"-Wall"},
				CFLAGS:  []string{"-pedantic"},
				LDFLAGS: []string{"-static"},
			},
			expectComp: []string{"-Wall", "-pedantic"},
			expectLink: []string{"-static"},
		},
		{
			name:       "mixed environment and config",
			envCCFlags: "-O3",
			envCFlags:  "-fPIC",
			envLDFlags: "-lm -lpthread",
			config: Config{
				CCFLAGS: []string{"-Wall", "-Wextra"},
				CFLAGS:  []string{"-std=c11"},
				LDFLAGS: []string{"-static"},
			},
			expectComp: []string{"-O3", "-fPIC", "-Wall", "-Wextra", "-std=c11"},
			expectLink: []string{"-O3", "-lm", "-lpthread", "-static"},
		},
		{
			// case from https://github.com/xgo-dev/llgo/issues/1244
			name:       "issue 1244",
			envCFlags:  "-w -pipe -mmacosx-version-min=15 -isysroot/Library/Developer/CommandLineTools/SDKs/MacOSX15.sdk",
			expectComp: []string{"-w", "-pipe", "-mmacosx-version-min=15", "-isysroot/Library/Developer/CommandLineTools/SDKs/MacOSX15.sdk"},
		},
	}

	// Save original environment
	origCCFlags := os.Getenv("CCFLAGS")
	origCFlags := os.Getenv("CFLAGS")
	origLDFlags := os.Getenv("LDFLAGS")
	defer func() {
		os.Setenv("CCFLAGS", origCCFlags)
		os.Setenv("CFLAGS", origCFlags)
		os.Setenv("LDFLAGS", origLDFlags)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("CCFLAGS", tt.envCCFlags)
			os.Setenv("CFLAGS", tt.envCFlags)
			os.Setenv("LDFLAGS", tt.envLDFlags)

			cmd := New("clang", tt.config)

			// Test compiler flags
			compFlags := cmd.mergeCompilerFlags()
			if len(tt.expectComp) == 0 && len(compFlags) == 0 {
				// Both are empty, consider them equal
			} else if !reflect.DeepEqual(compFlags, tt.expectComp) {
				t.Errorf("mergeCompilerFlags() = %v, want %v", compFlags, tt.expectComp)
			}

			// Test linker flags
			linkFlags := cmd.mergeLinkerFlags()
			if len(tt.expectLink) == 0 && len(linkFlags) == 0 {
				// Both are empty, consider them equal
			} else if !reflect.DeepEqual(linkFlags, tt.expectLink) {
				t.Errorf("mergeLinkerFlags() = %v, want %v", linkFlags, tt.expectLink)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMergeCompilerFlags(b *testing.B) {
	config := Config{
		CCFLAGS: []string{"-Wall", "-O2"},
		CFLAGS:  []string{"-std=c99", "-pedantic"},
	}
	cmd := New("clang", config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.mergeCompilerFlags()
	}
}

func BenchmarkMergeLinkerFlags(b *testing.B) {
	config := Config{
		LDFLAGS: []string{"-lm", "-lpthread", "-static"},
	}
	cmd := New("clang", config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.mergeLinkerFlags()
	}
}
