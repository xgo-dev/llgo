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
	"os/exec"
	"runtime"
	"strings"
)

// -----------------------------------------------------------------------------

// Cmd represents a clang command.
type Cmd struct {
	app     string
	Env     []string
	Verbose bool
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// New creates a new clang command.
func New(app string) *Cmd {
	if app == "" {
		app = "clang"
	}
	return &Cmd{app, nil, false, nil, os.Stdout, os.Stderr}
}

func (p *Cmd) Compile(args ...string) error {
	return p.execWithFlags([]string{"CFLAGS", "CCFLAGS"}, args...)
}

func (p *Cmd) Link(args ...string) error {
	return p.execWithFlags([]string{"LDFLAGS", "CCFLAGS"}, args...)
}

// execWithFlags executes a clang command with flags from environment variables.
func (p *Cmd) execWithFlags(flags []string, args ...string) error {
	var allFlags []string
	if p.Env != nil {
		for _, env := range p.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key, value := parts[0], parts[1]
			for _, flagName := range flags {
				if key == flagName {
					allFlags = append(allFlags, strings.Fields(value)...)
					break
				}
			}
		}
	} else {
		for _, flagName := range flags {
			envValue := os.Getenv(flagName)
			if envValue != "" {
				allFlags = append(allFlags, strings.Fields(envValue)...)
			}
		}
	}
	cmdArgs := make([]string, 0, len(allFlags)+len(args))
	cmdArgs = append(cmdArgs, allFlags...)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(p.app, cmdArgs...)
	if p.Verbose {
		fmt.Fprintf(os.Stderr, "%v\n", cmd)
	}
	cmd.Stdin = p.Stdin
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	if p.Env != nil {
		cmd.Env = p.Env
	}
	return cmd.Run()
}

// Exec executes a clang command.
func (p *Cmd) Exec(args ...string) error {
	cmd := exec.Command(p.app, args...)
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	if p.Env != nil {
		cmd.Env = p.Env
	}
	return cmd.Run()
}

func (p *Cmd) CheckLinkArgs(cmdArgs []string, wasm bool) error {
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
	p.Stdin = srcIn

	// Execute the command
	return p.execWithFlags([]string{"LDFLAGS", "CCFLAGS"}, args...)
}

// -----------------------------------------------------------------------------
