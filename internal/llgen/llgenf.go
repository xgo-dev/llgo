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

package llgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/goflags"
	"github.com/xgo-dev/llgo/target"
)

// GeneratedIR is an LLVM IR snapshot together with its effective target.
type GeneratedIR struct {
	Text   string
	GOOS   string
	GOARCH string
	Target string
}

// Phase identifies the compiler phase at which LLVM IR is captured.
type Phase string

const (
	PhasePreABI  Phase = "pre-abi"
	PhasePostABI Phase = "post-abi"
)

func GenFrom(fileOrPkg string) string {
	return Generate(fileOrPkg).Text
}

// Generate returns the historical pre-target-ABI module together with its
// effective target. GenFrom is the text-only compatibility wrapper.
func Generate(fileOrPkg string) GeneratedIR {
	return GenerateWithConf(fileOrPkg, nil)
}

// GenerateWithConf is Generate with caller-provided target and frontend
// settings. Any ModuleHook on input is ignored because generation manages the
// hook used to capture the requested phase.
func GenerateWithConf(fileOrPkg string, input *build.Config) GeneratedIR {
	generated, _ := generateWithConf(fileOrPkg, input, PhasePreABI)
	return generated
}

// GeneratePostABI returns the module after aggregate and target ABI lowering
// and before LLVM optimization. GenFrom intentionally retains its historical
// pre-target-ABI behavior.
func GeneratePostABI(fileOrPkg string) GeneratedIR {
	return GeneratePostABIWithConf(fileOrPkg, nil)
}

// GeneratePostABIWithConf is GeneratePostABI with caller-provided target and
// frontend settings. Any ModuleHook on input is ignored because generation
// manages phase capture internally.
func GeneratePostABIWithConf(fileOrPkg string, input *build.Config) GeneratedIR {
	generated, _ := generateWithConf(fileOrPkg, input, PhasePostABI)
	return generated
}

func generateWithConf(fileOrPkg string, input *build.Config, phase Phase) (GeneratedIR, []string) {
	conf := &build.Config{}
	if input != nil {
		*conf = *input
	}
	conf.Mode = build.ModeGen
	conf.GenLL = true
	// Cache hits can skip the module production needed by snapshot generation.
	conf.ForceRebuild = true

	var preABI GeneratedIR
	var preABICaptured bool
	switch phase {
	case PhasePostABI:
		conf.ModuleHook = nil
	case PhasePreABI:
		target := resolveGenerationTarget(fileOrPkg)
		conf.ModuleHook = func(pkg build.Package) {
			if !target.matches(pkg.PkgPath, pkg.GoFiles) {
				return
			}
			preABI = snapshotGeneratedIR(pkg)
			preABICaptured = true
		}
	default:
		panic(fmt.Sprintf("unknown llgen phase %q", phase))
	}
	pkg, err := genFromConf(fileOrPkg, conf)
	check(err)
	defer pkg.LPkg.Prog.Dispose()

	goFiles := append([]string(nil), pkg.GoFiles...)
	var generated GeneratedIR
	if phase == PhasePostABI {
		generated = snapshotGeneratedIR(pkg)
	} else {
		if !preABICaptured {
			panic(fmt.Sprintf("pre-ABI module for %q was not generated", pkg.PkgPath))
		}
		generated = preABI
	}
	return generated, goFiles
}

type generationTarget struct {
	pattern    string
	sourcePath string
	sourceDir  bool
}

func resolveGenerationTarget(pattern string) generationTarget {
	target := generationTarget{pattern: pattern}
	abs, err := filepath.Abs(pattern)
	if err != nil {
		return target
	}
	info, err := os.Stat(abs)
	if err != nil {
		return target
	}
	target.sourcePath = filepath.Clean(abs)
	target.sourceDir = info.IsDir()
	return target
}

func (target generationTarget) matches(pkgPath string, goFiles []string) bool {
	if pkgPath == target.pattern {
		return true
	}
	if target.sourcePath == "" {
		return false
	}
	for _, goFile := range goFiles {
		path, err := filepath.Abs(goFile)
		if err != nil {
			continue
		}
		path = filepath.Clean(path)
		if target.sourceDir {
			path = filepath.Dir(path)
		}
		if path == target.sourcePath {
			return true
		}
	}
	return false
}

func snapshotGeneratedIR(pkg build.Package) GeneratedIR {
	target := pkg.LPkg.Prog.Target()
	return GeneratedIR{
		Text:   pkg.LPkg.String(),
		GOOS:   target.GOOS,
		GOARCH: target.GOARCH,
		Target: target.Target,
	}
}

func genFromConf(pkgPath string, conf *build.Config) (build.Package, error) {
	if err := applyFlagsFile(conf, filepath.Join(pkgPath, "flags.txt")); err != nil {
		return nil, err
	}
	pkgs, err := build.Do([]string{pkgPath}, conf)
	if err != nil {
		return nil, err
	}
	return pkgs[0], nil
}

func DoFile(fileOrPkg, outFile string) {
	ret := GenFrom(fileOrPkg)
	err := os.WriteFile(outFile, []byte(ret), 0644)
	check(err)
}

func readFlags(flagsFile string) ([]string, error) {
	data, err := os.ReadFile(flagsFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	flags, err := goflags.ParseFlagFile(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", flagsFile, err)
	}
	return flags, nil
}

func applyFlagsFile(conf *build.Config, flagsFile string) error {
	flags, err := readFlags(flagsFile)
	if err != nil {
		return err
	}
	next := *conf
	goFlags := make([]string, 0, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		switch {
		case strings.HasPrefix(flag, "GOOS="):
			next.Goos = strings.TrimPrefix(flag, "GOOS=")
		case strings.HasPrefix(flag, "GOARCH="):
			next.Goarch = strings.TrimPrefix(flag, "GOARCH=")
		case flag == "-target" || flag == "--target":
			if i+1 == len(flags) {
				return fmt.Errorf("apply %s: %s requires a value", flagsFile, flag)
			}
			i++
			next.Target = flags[i]
		case strings.HasPrefix(flag, "-target="):
			next.Target = strings.TrimPrefix(flag, "-target=")
		case strings.HasPrefix(flag, "--target="):
			next.Target = strings.TrimPrefix(flag, "--target=")
		default:
			goFlags = append(goFlags, flag)
		}
	}
	if next.Target != "" {
		cfg, err := target.NewResolver(env.LLGoTargetsDir()).Resolve(next.Target)
		if err != nil {
			return fmt.Errorf("apply %s: %w", flagsFile, err)
		}
		next.Goos = cfg.GOOS
		next.Goarch = cfg.GOARCH
	}
	if err := goflags.ApplyBuildFlags(&next, goFlags); err != nil {
		return fmt.Errorf("apply %s: %w", flagsFile, err)
	}
	*conf = next
	return nil
}

func SmartDoFile(pkgPath string) {
	SmartDoFileAtPhase(pkgPath, PhasePreABI)
}

// SmartDoFileAtPhase writes the package module captured at phase.
func SmartDoFileAtPhase(pkgPath string, phase Phase) {
	generated, goFiles := generateWithConf(pkgPath, nil, phase)
	if len(goFiles) == 0 {
		panic(fmt.Sprintf("package %q has no Go files", pkgPath))
	}

	const autgenFile = "llgo_autogen.ll"
	dir, _ := filepath.Split(goFiles[0])
	outFile := dir + autgenFile

	b, err := os.ReadFile(outFile)
	if err == nil && len(b) == 1 && b[0] == ';' {
		return // skip to gen
	}

	if err := os.WriteFile(outFile, []byte(generated.Text), 0644); err != nil {
		panic(err)
	}
}
