// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// loadPathMu serializes PATH pins so concurrent loads cannot restore another
// caller's toolchain into the process environment. GoCommand takes the same
// lock so LookPath never observes a pin.
var loadPathMu sync.Mutex

// LoadMetadata runs go/packages with the selected GOROOT/bin first on the
// process PATH for the duration of the load. Setting Config.Env alone cannot
// do this: x/tools resolves exec.Command("go") using the process environment,
// including for GO111MODULE=off release-tag probes.
//
// The caller's Config.Env slice is not mutated. A copy is used, with
// GOROOT/bin prepended to its PATH so go list subprocesses inherit the same
// toolchain. The parent process PATH is restored after the load. Concurrent
// LoadMetadata calls are serialized.
//
// Other process-wide go invocations should use GoCommand: it resolves `go`
// while no pin is active and keeps an absolute Path, so a later pin cannot
// retarget that command. If GOROOT/bin/go is missing and GOPACKAGESDRIVER
// names an external driver, the load proceeds without a pin. This loader is
// for metadata only; LoadEx supplies LLGo's parser and typechecker.
func LoadMetadata(cfg *Config, patterns ...string) ([]*Package, error) {
	if cfg == nil {
		return loadPackages(cfg, patterns...)
	}
	if cfg.Mode&(NeedTypes|NeedTypesInfo|NeedSyntax) != 0 {
		return nil, fmt.Errorf("package driver requires a metadata-only load mode")
	}
	goroot := environmentValue(cfg.Env, "GOROOT")
	if goroot == "" {
		return loadPackages(cfg, patterns...)
	}
	goBin, err := selectedGoBin(goroot)
	if err != nil {
		if usesExternalPackageDriver(cfg.Env) {
			return loadPackages(cfg, patterns...)
		}
		return nil, fmt.Errorf("selected Go executable: %w", err)
	}

	loadPathMu.Lock()
	defer loadPathMu.Unlock()

	oldPath := os.Getenv("PATH")
	pinned := goBin + string(os.PathListSeparator) + oldPath
	if err := os.Setenv("PATH", pinned); err != nil {
		return nil, err
	}
	defer os.Setenv("PATH", oldPath)

	// TODO(golang/go#68495): Drop the PATH pin once x/tools can select the Go
	// executable per load, including its GO111MODULE=off probes.
	loadCfg := *cfg
	loadCfg.Env = replaceEnvironment(cfg.Env, "PATH", goBin+string(os.PathListSeparator)+environmentValue(cfg.Env, "PATH"))
	return loadPackages(&loadCfg, patterns...)
}

// GoCommand is exec.Command("go", args...) with Path resolved while
// LoadMetadata is not pinning PATH. The absolute Path is immune to a later pin.
func GoCommand(args ...string) *exec.Cmd {
	loadPathMu.Lock()
	defer loadPathMu.Unlock()
	path, err := exec.LookPath("go")
	if err != nil {
		return exec.Command("go", args...)
	}
	return exec.Command(path, args...)
}

func loadPackages(cfg *Config, patterns ...string) ([]*Package, error) {
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil && cfg != nil && cfg.Context != nil && cfg.Context.Err() != nil {
		return nil, cfg.Context.Err()
	}
	return pkgs, err
}

func selectedGoBin(goroot string) (string, error) {
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	goBin := filepath.Join(goroot, "bin")
	if _, err := exec.LookPath(filepath.Join(goBin, goName)); err != nil {
		return "", err
	}
	return goBin, nil
}

func usesExternalPackageDriver(env []string) bool {
	driver := environmentValue(env, "GOPACKAGESDRIVER")
	return driver != "" && driver != "off"
}

func environmentKeyEqual(a, b string) bool {
	return a == b || runtime.GOOS == "windows" && strings.EqualFold(a, b)
}

func environmentValue(environ []string, key string) string {
	var value string
	for _, entry := range environ {
		if name, v, ok := strings.Cut(entry, "="); ok && environmentKeyEqual(name, key) {
			value = v
		}
	}
	return value
}

func replaceEnvironment(environ []string, key, value string) []string {
	result := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if !environmentKeyEqual(name, key) {
			result = append(result, entry)
		}
	}
	return append(result, key+"="+value)
}
