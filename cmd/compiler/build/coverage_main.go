// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import "fmt"

const coverageTestMain = `package main

import (
	"internal/coverage/cfile"
	"internal/runtime/exithook"
	"runtime"
	"testing/internal/testdeps"
	_ "unsafe"
)

func init() {
	testdeps.CoverMode = %q
	testdeps.Covered = %q
	testdeps.CoverSelectedPackages = %#v
	testdeps.CoverSnapshotFunc = cfile.Snapshot
	testdeps.CoverProcessTestDirFunc = cfile.ProcessCoverTestDir
	testdeps.CoverMarkProfileEmittedFunc = cfile.MarkProfileEmitted
	exithook.Gosched = runtime.Gosched
	exithook.Goid = llgoCoverGoid
	exithook.Throw = llgoCoverThrow
	llgoCoverSetExitHook(exithook.Run)
	cfile.InitHook(true)
}
`

// Applications use the same runtime as tests, but emit metadata at the end of
// main-package initialization and counters on exit through GOCOVERDIR.
const coverageBuildMain = `package main

import (
	"internal/coverage/cfile"
	"internal/runtime/exithook"
	"runtime"
	_ "unsafe"
)

func init() {
	exithook.Gosched = runtime.Gosched
	exithook.Goid = llgoCoverGoid
	exithook.Throw = llgoCoverThrow
	llgoCoverSetExitHook(exithook.Run)
	cfile.InitHook(false)
}
`

// Install exit support only in covered executables. Ordinary binaries must not
// acquire coverage writers, their imports, or their initialization cost.
const coverageExitSupport = `
//go:linkname llgoCoverSetExitHook runtime.setCoverageExitHook
func llgoCoverSetExitHook(func(int))

//go:linkname llgoCoverGoid github.com/xgo-dev/llgo/runtime/internal/runtime.goid
func llgoCoverGoid() uint64

//go:linkname llgoCoverThrow runtime.throw
func llgoCoverThrow(string)
`

func legacyCoverageMain(mode, covered string, go120 bool) string {
	snapshotParam := "\tsnapshot func() float64,\n"
	snapshotArg := ", llgoCoverSnapshot"
	if go120 {
		// testing.Coverage snapshots were added to the redesigned protocol
		// in Go 1.21. The Go 1.20 callback accepts only mode and teardown.
		snapshotParam = ""
		snapshotArg = ""
	}
	return fmt.Sprintf(coverageTestMainLegacy, mode, covered, snapshotParam, snapshotArg) + coverageLegacyExitSupport
}

const coverageBuildMainLegacy = `package main

import (
	_ "runtime/coverage"
	_ "unsafe"
)

//go:linkname llgoCoverInitHook runtime/coverage.initHook
func llgoCoverInitHook(bool)

func init() {
	llgoCoverSetExitHook(llgoCoverRunExitHooks)
	llgoCoverInitHook(false)
}
`

// Mirrors cmd/go's Go 1.20–1.22 testmain contract. These versions predate
// internal/coverage/cfile and the coverage callbacks in testing/internal/testdeps.
const coverageTestMainLegacy = `package main

import (
	"os"
	_ "runtime/coverage"
	_ "unsafe"
)

//go:linkname llgoCoverProcess runtime/coverage.processCoverTestDir
func llgoCoverProcess(dir, profile, mode, covered string) error

//go:linkname llgoCoverRegister testing.registerCover2
func llgoCoverRegister(
	mode string,
	tearDown func(profile, dir string) (string, error),
%[3]s
)

//go:linkname llgoCoverMarkEmitted runtime/coverage.markProfileEmitted
func llgoCoverMarkEmitted(bool)

//go:linkname llgoCoverSnapshot runtime/coverage.snapshot
func llgoCoverSnapshot() float64

//go:linkname llgoCoverInitHook runtime/coverage.initHook
func llgoCoverInitHook(bool)

func llgoCoverTearDown(profile, dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.MkdirTemp("", "gocoverdir")
		if err != nil {
			return "error setting GOCOVERDIR: bad os.MkdirTemp return", err
		}
		defer os.RemoveAll(dir)
	}
	llgoCoverMarkEmitted(true)
	if err := llgoCoverProcess(dir, profile, %[1]q, %[2]q); err != nil {
		return "error generating coverage report", err
	}
	return "", nil
}

func init() {
	llgoCoverRegister(%[1]q, llgoCoverTearDown%[4]s)
	llgoCoverSetExitHook(llgoCoverRunExitHooks)
	llgoCoverInitHook(true)
}
`

// Old Go releases keep the hooks in runtime instead of internal/runtime/exithook.
// Both test and application mains use the same reverse-order exit dispatch.
const coverageLegacyExitSupport = `

var llgoCoverExitHooks []struct {
	f func()
	onFailure bool
}

//go:linkname llgoCoverAddExitHook runtime.addExitHook
func llgoCoverAddExitHook(f func(), onFailure bool) {
	llgoCoverExitHooks = append(llgoCoverExitHooks, struct {
		f func()
		onFailure bool
	}{f, onFailure})
}

var llgoCoverExiting bool

func llgoCoverRunExitHooks(code int) {
	if llgoCoverExiting {
		llgoCoverThrow("exit hook invoked exit")
	}
	llgoCoverExiting = true
	defer func() {
		if recover() != nil {
			llgoCoverThrow("exit hook invoked panic")
		}
		llgoCoverExiting = false
	}()
	for len(llgoCoverExitHooks) != 0 {
		n := len(llgoCoverExitHooks) - 1
		hook := llgoCoverExitHooks[n]
		llgoCoverExitHooks = llgoCoverExitHooks[:n]
		if code == 0 || hook.onFailure {
			hook.f()
		}
	}
}
`
