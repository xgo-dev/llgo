// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

const (
	metadataDriverArg = "--llgo-internal-package-driver"
	metadataDriverEnv = "__LLGO_PACKAGE_DRIVER_WORKER"
)

// Re-exec also works for programs embedding LLGo and for Go test binaries.
// init inspects os.Args[1], but only the private argument plus the child-only
// environment marker takes this path before the caller's main or TestMain.
func init() {
	if len(os.Args) != 2 || os.Args[1] != metadataDriverArg || os.Getenv(metadataDriverEnv) != "1" {
		return
	}
	if err := runMetadataDriver(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

type metadataRequest struct {
	packages.DriverRequest
	Log bool
}

// The go/packages JSON protocol omits Module, Dir, Target, ForTest and sizes.
// Keep them alongside the standard flattened package/import representation.
type packageMetadata struct {
	Package *Package
	Module  *packages.Module
	Dir     string
	Target  string
	ForTest string
	Sizes   bool
	Imports bool
}

type metadataResponse struct {
	Roots    []string
	Packages []packageMetadata
	Compiler string
	Arch     string
	StdSizes *types.StdSizes
	Error    string
}

// LoadMetadata runs the unmodified go/packages driver with the selected
// GOROOT/bin first on its process PATH. Setting Config.Env alone cannot do this:
// x/tools resolves exec.Command("go") using the process environment, including
// for GO111MODULE=off release-tag probes. The parent's environment is untouched.
// This loader is for metadata only; LoadEx supplies LLGo's parser and typechecker.
func LoadMetadata(cfg *Config, patterns ...string) ([]*Package, error) {
	if cfg == nil || environmentValue(cfg.Env, "GOROOT") == "" {
		return packages.Load(cfg, patterns...)
	}
	if cfg.Mode&(NeedTypes|NeedTypesInfo|NeedSyntax) != 0 {
		return nil, fmt.Errorf("package driver requires a metadata-only load mode")
	}
	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	request := metadataRequest{
		DriverRequest: packages.DriverRequest{
			Mode: cfg.Mode, Env: cfg.Env, BuildFlags: cfg.BuildFlags,
			Tests: cfg.Tests, Overlay: cfg.Overlay,
		},
		Log: cfg.Logf != nil,
	}
	// Preserve the caller's external driver selection before changing the
	// child's lookup path. A NotHandled response still falls back to selected go.
	driver := environmentValue(request.Env, "GOPACKAGESDRIVER")
	if driver == "" {
		if driver, err := exec.LookPath("gopackagesdriver"); err == nil {
			request.Env = replaceEnvironment(request.Env, "GOPACKAGESDRIVER", driver)
		}
	} else if driver != "off" && !strings.ContainsAny(driver, `/\`) {
		path, err := exec.LookPath(driver)
		if err != nil {
			return nil, err
		}
		request.Env = replaceEnvironment(request.Env, "GOPACKAGESDRIVER", path)
	}
	goBin := filepath.Join(environmentValue(cfg.Env, "GOROOT"), "bin")
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	if _, err := exec.LookPath(filepath.Join(goBin, goName)); err != nil {
		return nil, fmt.Errorf("selected Go executable: %w", err)
	}
	request.Env = replaceEnvironment(request.Env, "PATH", goBin+string(os.PathListSeparator)+environmentValue(cfg.Env, "PATH"))
	// TODO(golang/go#68495): Use packages.Load directly once x/tools can select
	// the Go executable per load, including its GO111MODULE=off probes. Remove
	// the worker init dispatch and metadata round trip together at that point.
	// CLI-only setup misses direct build callers; changing the parent PATH
	// would let concurrent invocations interfere with each other's toolchain.
	cmd := exec.CommandContext(ctx, executable, metadataDriverArg)
	cmd.Dir = cfg.Dir
	// This is a re-exec protocol marker, not a user-facing build/test switch.
	// Keep it out of request.Env, which is passed on to external package drivers.
	cmd.Env = replaceEnvironment(request.Env, metadataDriverEnv, "1")
	if cfg.Dir != "" {
		cmd.Env = replaceEnvironment(cmd.Env, "PWD", cfg.Dir)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()
	// EOF cancels the worker's go/packages context and its Go subprocesses.
	// WaitDelay bounds cleanup if the worker cannot finish cancellation.
	cmd.Cancel = stdin.Close
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := json.NewEncoder(stdin).Encode(requestWithPatterns{metadataRequest: request, Patterns: patterns}); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("Go package driver: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stderr.Len() != 0 {
		if cfg.Logf != nil {
			cfg.Logf("%s", strings.TrimSpace(stderr.String()))
		} else {
			_, _ = os.Stderr.Write(stderr.Bytes())
		}
	}
	var response metadataResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("decode Go package metadata: %w", err)
	}
	return response.packages()
}

type requestWithPatterns struct {
	metadataRequest
	Patterns []string
}

func runMetadataDriver(input io.Reader, output io.Writer) error {
	var request requestWithPatterns
	if err := json.NewDecoder(input).Decode(&request); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_, _ = io.Copy(io.Discard, input)
		cancel()
	}()
	cfg := Config{
		Context: ctx, Mode: request.Mode, Env: request.Env,
		BuildFlags: request.BuildFlags, Tests: request.Tests, Overlay: request.Overlay,
	}
	if request.Log {
		cfg.Logf = func(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }
	}
	response := loadDriverMetadata(&cfg, request.Patterns)
	return json.NewEncoder(output).Encode(response)
}

func loadDriverMetadata(cfg *Config, patterns []string) metadataResponse {
	initial, err := packages.Load(cfg, patterns...)
	if err != nil {
		return metadataResponse{Error: err.Error()}
	}
	var response metadataResponse
	for _, pkg := range initial {
		response.Roots = append(response.Roots, pkg.ID)
	}
	Visit(initial, func(pkg *Package) bool {
		response.Packages = append(response.Packages, packageMetadata{
			Package: pkg, Module: pkg.Module, Dir: pkg.Dir, Target: pkg.Target,
			ForTest: pkg.ForTest, Sizes: pkg.TypesSizes != nil, Imports: pkg.Imports != nil,
		})
		return true
	}, nil)
	for _, metadata := range response.Packages {
		if sizes := metadata.Package.TypesSizes; sizes != nil {
			response.Compiler, response.Arch, response.StdSizes = describeSizes(sizes)
			if response.Compiler == "" && response.StdSizes == nil {
				response.Error = fmt.Sprintf("unsupported package sizes %T", sizes)
			}
			break
		}
	}
	return response
}

func describeSizes(sizes types.Sizes) (compiler, arch string, std *types.StdSizes) {
	if std, ok := sizes.(*types.StdSizes); ok {
		return "", "", std
	}
	// go/packages obtains sizes from types.SizesFor. Match its cached gc
	// implementations rather than substituting StdSizes (whose struct padding
	// differs). These are the architectures documented by types.SizesFor.
	for _, arch := range strings.Fields("386 amd64 amd64p32 arm arm64 loong64 mips mipsle mips64 mips64le ppc64 ppc64le riscv64 s390x sparc64 wasm") {
		if sizes == types.SizesFor("gc", arch) {
			return "gc", arch, nil
		}
	}
	return "", "", nil
}

func (response metadataResponse) packages() ([]*Package, error) {
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	byID := make(map[string]*Package, len(response.Packages))
	for _, metadata := range response.Packages {
		pkg := metadata.Package
		pkg.Module, pkg.Dir, pkg.Target, pkg.ForTest = metadata.Module, metadata.Dir, metadata.Target, metadata.ForTest
		if metadata.Imports && pkg.Imports == nil {
			pkg.Imports = make(map[string]*Package)
		}
		if metadata.Sizes {
			pkg.TypesSizes = types.SizesFor(response.Compiler, response.Arch)
			if response.StdSizes != nil {
				pkg.TypesSizes = response.StdSizes
			}
		}
		byID[pkg.ID] = pkg
	}
	for _, pkg := range byID {
		for path, imported := range pkg.Imports {
			if resolved := byID[imported.ID]; resolved != nil {
				pkg.Imports[path] = resolved
			}
		}
	}
	initial := make([]*Package, 0, len(response.Roots))
	for _, id := range response.Roots {
		initial = append(initial, byID[id])
	}
	return initial, nil
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
