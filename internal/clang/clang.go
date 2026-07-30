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
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/goplus/llgo/internal/processenv"
	"github.com/goplus/llgo/xtool/safesplit"
)

// Config represents clang configuration parameters.
type Config struct {
	CC      string   // Compiler to use (e.g., "clang", "clang++")
	CCFLAGS []string // Compiler flags for C/C++ compilation
	CFLAGS  []string // C-specific flags
	LDFLAGS []string // Linker flags
	Linker  string   // Linker to use (e.g., "ld.lld", "avr-ld")
}

// NewConfig creates a new Config with the specified parameters.
func NewConfig(cc string, ccflags, cflags, ldflags []string, linker string) Config {
	return Config{
		CC:      cc,
		CCFLAGS: ccflags,
		CFLAGS:  cflags,
		LDFLAGS: ldflags,
		Linker:  linker,
	}
}

// Cmd represents a clang command with environment and configuration support.
type Cmd struct {
	app     string
	config  Config
	Dir     string
	Env     []string
	Verbose bool
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// New creates a new clang command with configuration.
func New(app string, config Config) *Cmd {
	if app == "" {
		app = "clang"
	}
	return &Cmd{
		app:     app,
		config:  config,
		Dir:     "",
		Env:     nil,
		Verbose: false,
		Stdin:   nil,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// NewCompiler creates a compiler command with proper flag merging.
func NewCompiler(config Config) *Cmd {
	app := "clang"
	if config.CC != "" {
		app = config.CC
	}
	return New(app, config)
}

// NewLinker creates a linker command with proper flag merging.
func NewLinker(config Config) *Cmd {
	app := "clang"
	if config.Linker != "" {
		app = config.Linker
	} else if config.CC != "" {
		app = config.CC
	}
	return New(app, config)
}

// Compile executes a compilation command with merged flags.
func (c *Cmd) Compile(args ...string) error {
	flags := c.mergeCompilerFlags()
	allArgs := make([]string, 0, len(flags)+len(args))
	allArgs = append(allArgs, flags...)
	allArgs = append(allArgs, args...)
	return c.exec(allArgs...)
}

// Link executes a linking command with merged flags.
func (c *Cmd) Link(args ...string) error {
	flags := c.mergeLinkerFlags()
	allArgs := make([]string, 0, len(flags)+len(args))
	allArgs = append(allArgs, flags...)
	allArgs = append(allArgs, args...)
	return c.exec(allArgs...)
}

// mergeCompilerFlags merges environment CCFLAGS/CFLAGS with config flags.
func (c *Cmd) mergeCompilerFlags() []string {
	var flags []string

	// Add environment CCFLAGS
	if envCCFlags := c.getenv("CCFLAGS"); envCCFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envCCFlags)...)
	}

	// Add environment CFLAGS
	if envCFlags := c.getenv("CFLAGS"); envCFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envCFlags)...)
	}

	// Add config CCFLAGS
	flags = append(flags, c.config.CCFLAGS...)

	// Add config CFLAGS
	flags = append(flags, c.config.CFLAGS...)

	return flags
}

// mergeLinkerFlags merges environment CCFLAGS/LDFLAGS with config flags.
func (c *Cmd) mergeLinkerFlags() []string {
	var flags []string

	// Add environment CCFLAGS (for linker)
	if envCCFlags := c.getenv("CCFLAGS"); envCCFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envCCFlags)...)
	}

	// Add environment LDFLAGS
	if envLDFlags := c.getenv("LDFLAGS"); envLDFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envLDFlags)...)
	}

	// Add config LDFLAGS
	flags = append(flags, c.config.LDFLAGS...)

	return flags
}

// exec executes the clang command with given arguments.
func (c *Cmd) exec(args ...string) error {
	cmd := processenv.Command(c.Env, c.Dir, c.app, args...)
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "%v\n", cmd)
	}
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	return cmd.Run()
}

func (c *Cmd) getenv(key string) string {
	if c.Env == nil {
		return os.Getenv(key)
	}
	return processenv.Get(c.Env, key)
}

// CheckLinkArgs validates linking arguments by attempting a test compile.
func (c *Cmd) CheckLinkArgs(cmdArgs []string, wasm bool) error {
	// Create a temporary file with appropriate extension
	extension := ""
	if wasm {
		extension = ".wasm"
	} else if runtime.GOOS == "windows" {
		extension = ".exe"
	}

	tmpFile, err := os.CreateTemp("", "llgo_check*"+extension)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpFile.Close()
	tmpPath := tmpFile.Name()

	// Make sure to delete the temporary file when done
	defer os.Remove(tmpPath)

	// Set up compilation arguments
	args := append([]string{}, cmdArgs...)
	args = append(args, []string{"-x", "c", "-o", tmpPath, "-"}...)
	src := "int main() {return 0;}"
	srcIn := strings.NewReader(src)
	c.Stdin = srcIn

	// Execute the command with linker flags
	return c.Link(args...)
}
