// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package llar provides a small Go wrapper around the llar command.
package llar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

// Cmd represents an llar command.
type Cmd struct {
	bin string

	Stdout io.Writer
	Stderr io.Writer
}

// New creates an llar command wrapper. An empty bin uses llar from PATH.
func New(bin string) *Cmd {
	if bin == "" {
		bin = "llar"
	}
	return &Cmd{bin: bin}
}

// Module identifies an llar module to install.
type Module struct {
	Path    string
	Version string
}

// Config selects the build matrix for an installation.
type Config struct {
	To      string
	OS      string
	Arch    string
	Libc    string
	Options map[string][]string
}

// Dependency is a module returned in an llar install result.
type Dependency struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Dir     string `json:"dir"`
}

// Result is the JSON result returned by llar install.
type Result struct {
	Path       string       `json:"path"`
	Version    string       `json:"version"`
	Dir        string       `json:"dir"`
	Deps       []Dependency `json:"deps,omitempty"`
	BuildFlags string       `json:"metadata"`
}

// Install runs llar install for mod and decodes the command's JSON result.
func (p *Cmd) Install(mod Module, config Config) (Result, error) {
	args := []string{"install", "--json"}
	if config.To != "" {
		args = append(args, "--output", config.To)
	}
	if p.Stderr != nil {
		args = append(args, "--verbose")
	}
	if config.OS != "" {
		args = append(args, "--os", config.OS)
	}
	if config.Arch != "" {
		args = append(args, "--arch", config.Arch)
	}
	if config.Libc != "" {
		args = append(args, "--libc", config.Libc)
	}

	keys := make([]string, 0, len(config.Options))
	for key := range config.Options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range config.Options[key] {
			args = append(args, "--option", key+"="+value)
		}
	}
	target := mod.Path
	if mod.Version != "" {
		target += "@" + mod.Version
	}
	args = append(args, target)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(p.bin, args...)
	cmd.Stdout = io.Writer(&stdout)
	if p.Stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, p.Stdout)
	}
	cmd.Stderr = io.Writer(&stderr)
	if p.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, p.Stderr)
	}
	if err := cmd.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return Result{}, fmt.Errorf("llar install: %w: %s", err, message)
		}
		return Result{}, fmt.Errorf("llar install: %w", err)
	}

	var result Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return Result{}, fmt.Errorf("llar install: decode result: %w", err)
	}
	return result, nil
}
