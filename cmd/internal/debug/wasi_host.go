// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package debug

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func findWASIHost(configured string) (string, error) {
	if configured == "" {
		configured = os.Getenv("LLGO_WASI_HOST")
	}
	if configured == "" {
		configured = "llgo-wasi-debug"
	}
	path, err := exec.LookPath(configured)
	if err != nil {
		return "", fmt.Errorf("llgo debug: find WASI host: %w; build test/debug/wasi/host or set -wasi-host / LLGO_WASI_HOST", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("llgo debug: query WASI host version: %w: %s", err, output)
	}
	if err := wasiHostVersion(string(output)); err != nil {
		return "", err
	}
	return path, nil
}

func wasiHostVersion(output string) error {
	fields := strings.Fields(output)
	if len(fields) < 2 || fields[0] != "llgo-wasi-debug" ||
		!semver.IsValid("v"+fields[1]) || semver.Major("v"+fields[1]) != "v1" {
		return fmt.Errorf("llgo debug: WASI host requires llgo-wasi-debug ABI 1; found %q", strings.TrimSpace(output))
	}
	return nil
}
