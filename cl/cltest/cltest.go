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

package cltest

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/littest"
	"github.com/goplus/llgo/internal/llgen"
	"github.com/goplus/llgo/internal/mockable"
	"github.com/goplus/llgo/internal/semmeta"
	"github.com/goplus/llgo/ssa/ssatest"
	"github.com/qiniu/x/test"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	llssa "github.com/goplus/llgo/ssa"
)

func init() {
	llssa.Initialize(llssa.InitAll | llssa.InitNative)
}

func InitDebug() {
	cl.SetDebug(cl.DbgFlagAll)
	llssa.SetDebug(llssa.DbgFlagAll)
}

func FromDir(t *testing.T, sel, relDir string) {
	RunAndTestFromDir(t, sel, relDir, nil, WithOutputCheck(false))
}

type runOptions struct {
	conf        *build.Config
	filter      func(string) string
	checkIR     bool
	checkOutput bool
	checkMeta   bool
}

// RunOption customizes directory-based test behavior.
type RunOption func(*runOptions)

// WithRunConfig uses the provided build config for test runs.
func WithRunConfig(conf *build.Config) RunOption {
	return func(opts *runOptions) {
		opts.conf = conf
	}
}

// WithOutputFilter applies a filter to output before comparison.
func WithOutputFilter(filter func(string) string) RunOption {
	return func(opts *runOptions) {
		opts.filter = filter
	}
}

// WithOutputCheck enables or disables runtime output golden checks in RunAndTestFromDir.
func WithOutputCheck(enabled bool) RunOption {
	return func(opts *runOptions) {
		opts.checkOutput = enabled
	}
}

// WithIRCheck enables or disables IR golden checks in RunAndTestFromDir.
func WithIRCheck(enabled bool) RunOption {
	return func(opts *runOptions) {
		opts.checkIR = enabled
	}
}

// WithMetaCheck enables or disables semantic metadata golden checks.
func WithMetaCheck(enabled bool) RunOption {
	return func(opts *runOptions) {
		opts.checkMeta = enabled
	}
}

// FilterEmulatorOutput strips emulator boot logs by returning output after "entry 0x...".
func FilterEmulatorOutput(output string) string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "")
	lines := strings.Split(output, "\n")
	entryPattern := regexp.MustCompile(`^entry 0x[0-9a-fA-F]+$`)
	for i, line := range lines {
		if entryPattern.MatchString(strings.TrimSpace(line)) {
			if i+1 < len(lines) {
				return strings.Join(lines[i+1:], "\n")
			}
			return ""
		}
	}
	return output
}

// RunAndTestFromDir executes tests under relDir and validates both runtime
// output and the pre-transform package IR snapshot when the corresponding
// golden files exist.
func RunAndTestFromDir(t *testing.T, sel, relDir string, ignore []string, opts ...RunOption) {
	rootDir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir := filepath.Join(rootDir, relDir)
	ignoreSet := make(map[string]struct{}, len(ignore))
	for _, item := range ignore {
		ignoreSet[item] = struct{}{}
	}
	options := runOptions{checkIR: true, checkOutput: true}
	for _, opt := range opts {
		opt(&options)
	}
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		pkgDir := filepath.Join(dir, name)
		relPkg, err := filepath.Rel(rootDir, pkgDir)
		if err != nil {
			t.Fatal("Rel failed:", err)
		}
		relPkg = "./" + filepath.ToSlash(relPkg)
		if _, ok := ignoreSet[relPkg]; ok {
			t.Run(name, func(t *testing.T) {
				t.Skip("skip platform-specific output mismatch")
			})
			continue
		}
		t.Run(name, func(t *testing.T) {
			testRunAndTestFrom(t, pkgDir, relPkg, sel, options)
		})
	}
}

// *.ll => *.lla
func decodeLinkFile(llFile string) (data []byte, err error) {
	zipFile := llFile + "a"
	zipf, err := zip.OpenReader(zipFile)
	if err != nil {
		return
	}
	defer zipf.Close()
	f, err := zipf.Open(filepath.Base(llFile))
	if err != nil {
		return
	}
	defer f.Close()
	data, err = io.ReadAll(f)
	if err == nil {
		os.WriteFile(llFile, data, 0644)
	}
	return
}

func Pkg(t *testing.T, pkgPath, outFile string) {
	b, err := os.ReadFile(outFile)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal("ReadFile failed:", err)
		}
		if b, err = decodeLinkFile(outFile); err != nil {
			t.Fatal("decodeLinkFile failed:", err)
		}
	}
	expected := string(b)
	if v := llgen.GenFrom(pkgPath); v != expected {
		t.Fatalf("\n==> got:\n%s\n==> expected:\n%s\n", v, expected)
	}
}

func testFrom(t *testing.T, pkgDir, sel string, opts runOptions) {
	t.Helper()
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}
	if !opts.checkIR && !opts.checkMeta {
		return
	}

	var err error
	var metaExpected []byte
	if opts.checkMeta {
		var hasMetaExpect bool
		metaExpected, hasMetaExpect, err = loadMetaExpect(pkgDir)
		if err != nil {
			t.Fatal("loadMetaExpect failed:", err)
		}
		if !hasMetaExpect {
			t.Fatal("missing meta-expect.txt")
		}
	}

	var spec littest.Spec
	if opts.checkIR {
		var err error
		spec, err = littest.LoadSpec(pkgDir)
		if err != nil {
			t.Fatal("LoadSpec failed:", err)
		}
		if spec.Mode == littest.ModeSkip && !opts.checkMeta {
			return
		}
	}

	mod, err := llgen.GenModuleFrom(pkgDir)
	if err != nil {
		t.Fatal("GenModuleFrom failed:", err)
	}
	defer mod.Dispose()

	if opts.checkIR {
		v := mod.String()
		if spec.Mode == littest.ModeFileCheck {
			if err := littest.Check(spec, v); err != nil {
				_ = os.WriteFile(pkgDir+"/result.txt", []byte(v), 0644)
				t.Fatal(err)
			}
		} else if spec.Mode == littest.ModeLiteral && test.Diff(t, pkgDir+"/result.txt", []byte(v), []byte(spec.Text)) {
			t.Fatal("llgen.GenFrom: unexpected result")
		}
	}

	if opts.checkMeta {
		info := semmeta.Read(mod)
		if test.Diff(t, filepath.Join(pkgDir, "meta-expect.txt.new"), []byte(FormatSemMeta(info)), metaExpected) {
			t.Fatal("semmeta.Read: unexpected result")
		}
	}
}

func loadMetaExpect(pkgDir string) ([]byte, bool, error) {
	path := filepath.Join(pkgDir, "meta-expect.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func testRunAndTestFrom(t *testing.T, pkgDir, relPkg, sel string, opts runOptions) {
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}

	var (
		expectedOutput []byte
		checkOutput    bool
		err            error
	)
	if opts.checkOutput {
		expectedOutput, checkOutput, err = readGolden(filepath.Join(pkgDir, "expect.txt"))
		if err != nil {
			t.Fatal("ReadFile failed:", err)
		}
	}
	if !checkOutput {
		// IR-only or metadata-only mode: when expect.txt is not checked, use
		// testFrom to compare this package's generated IR and/or semantic
		// metadata against the corresponding golden files.
		if opts.checkIR || opts.checkMeta {
			testFrom(t, pkgDir, sel, opts)
		}
		return
	}

	var irSpec littest.Spec
	conf := opts.conf
	var capturedIR *string
	var checkIR bool
	if opts.checkIR {
		irSpec, checkIR, err = readIRSpec(pkgDir)
		if err != nil {
			t.Fatal("LoadSpec failed:", err)
		}
		if checkIR {
			conf, capturedIR = withModuleCapture(opts.conf, pkgDir)
		}
	}

	output, err := runWithConf(relPkg, pkgDir, conf)
	if err != nil {
		t.Logf("raw output:\n%s", string(output))
		t.Fatalf("run failed: %v\noutput: %s", err, string(output))
	}

	assertExpectedOutput(t, pkgDir, expectedOutput, output, opts)
	if !checkIR {
		return
	}
	if capturedIR == nil || *capturedIR == "" {
		t.Fatalf("module snapshot missing for file %s", irSpec.Path)
	}
	if err := littest.Check(irSpec, *capturedIR); err != nil {
		_ = os.WriteFile(filepath.Join(pkgDir, "result.txt"), []byte(*capturedIR), 0644)
		t.Fatal(err)
	}
}

func RunAndCapture(relPkg, pkgDir string) ([]byte, error) {
	conf := build.NewDefaultConf(build.ModeRun)
	return RunAndCaptureWithConf(relPkg, pkgDir, conf)
}

// RunAndCaptureWithConf runs llgo with a custom build config and captures output.
func RunAndCaptureWithConf(relPkg, pkgDir string, conf *build.Config) ([]byte, error) {
	return runWithConf(relPkg, pkgDir, conf)
}

func withModuleCapture(conf *build.Config, pkgDir string) (*build.Config, *string) {
	if conf == nil {
		conf = build.NewDefaultConf(build.ModeRun)
	}
	localConf := *conf
	var module string
	prevHook := localConf.ModuleHook
	localConf.ModuleHook = func(pkg build.Package) {
		if prevHook != nil {
			prevHook(pkg)
		}
		if slices.ContainsFunc(pkg.Package.GoFiles, func(file string) bool {
			return filepath.Dir(file) == pkgDir
		}) {
			module = pkg.LPkg.String()
		}
	}
	return &localConf, &module
}

func runWithConf(relPkg, pkgDir string, conf *build.Config) ([]byte, error) {
	if conf == nil {
		conf = build.NewDefaultConf(build.ModeRun)
	}
	cacheDir, err := os.MkdirTemp("", "llgo-gocache-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(cacheDir)
	oldCache := os.Getenv("GOCACHE")
	if err := os.Setenv("GOCACHE", cacheDir); err != nil {
		return nil, err
	}
	defer func() {
		if oldCache == "" {
			_ = os.Unsetenv("GOCACHE")
		} else {
			_ = os.Setenv("GOCACHE", oldCache)
		}
	}()

	localConf := *conf

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	outputCh := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		outputCh <- buf.Bytes()
	}()

	origDir, err := os.Getwd()
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	if pkgDir != "" {
		if err := os.Chdir(pkgDir); err != nil {
			_ = w.Close()
			return nil, err
		}
		defer os.Chdir(origDir)
		relPkg = "."
	}

	mockable.EnableMock()
	defer mockable.DisableMock()

	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if s, ok := r.(string); ok && s == "exit" {
					return
				}
				panic(r)
			}
		}()
		_, runErr = build.Do([]string{relPkg}, &localConf)
	}()
	_ = w.Close()
	output := <-outputCh
	output = filterRunOutput(output)
	if runErr != nil {
		return output, fmt.Errorf("run failed: %w", runErr)
	}
	return output, nil
}

func assertExpectedOutput(t *testing.T, pkgDir string, expectedOutput, output []byte, opts runOptions) {
	t.Helper()
	if opts.filter != nil {
		output = []byte(opts.filter(string(output)))
	}
	if test.Diff(t, filepath.Join(pkgDir, "expect.txt.new"), output, expectedOutput) {
		t.Fatal("unexpected output")
	}
}

func readGolden(file string) ([]byte, bool, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if bytes.Equal(data, []byte{';'}) {
		return data, false, nil
	}
	return data, true, nil
}

func readIRSpec(pkgDir string) (littest.Spec, bool, error) {
	spec, err := littest.LoadSpec(pkgDir)
	if err != nil {
		var pathErr *os.PathError
		if errors.Is(err, os.ErrNotExist) && errors.As(err, &pathErr) && filepath.Clean(pathErr.Path) == filepath.Join(pkgDir, "out.ll") {
			return littest.Spec{}, false, nil
		}
		return littest.Spec{}, false, err
	}
	return spec, true, nil
}

func filterRunOutput(in []byte) []byte {
	// Tests compare output with expect.txt. Some toolchain/environment warnings are
	// inherently machine-specific and should not be part of the golden output.
	parts := bytes.SplitAfter(in, []byte{'\n'})
	if len(parts) == 0 {
		return in
	}
	var out bytes.Buffer
	for _, p := range parts {
		line := bytes.TrimRight(p, "\r\n")
		trim := bytes.TrimLeft(line, " \t")
		switch {
		case bytes.HasPrefix(trim, []byte("WARNING: Using LLGO root for devel: ")):
			continue
		case bytes.HasPrefix(trim, []byte("WARNING: LLGO_ROOT is not a valid LLGO root: ")):
			continue
		case bytes.HasPrefix(trim, []byte("ld64.lld: warning: ")):
			continue
		case bytes.HasPrefix(trim, []byte("ld.lld: warning: ")):
			continue
		case bytes.HasPrefix(trim, []byte("ld: warning: ")):
			continue
		}
		out.Write(p)
	}
	if out.Len() == 0 {
		return nil
	}
	return out.Bytes()
}

func TestCompileEx(t *testing.T, src any, fname, expected string, dbg bool) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, src, parser.ParseComments)
	if err != nil {
		t.Fatal("ParseFile failed:", err)
	}
	files := []*ast.File{f}
	name := f.Name.Name
	pkg := types.NewPackage(name, name)
	imp := packages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	if dbg {
		mode |= ssa.GlobalDebug
	}
	foo, _, err := ssautil.BuildPackage(
		&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal("BuildPackage failed:", err)
	}
	foo.WriteTo(os.Stderr)
	prog := ssatest.NewProgramEx(t, nil, imp)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))

	ret, err := cl.NewPackage(prog, foo, files)
	if err != nil {
		t.Fatal("cl.NewPackage failed:", err)
	}

	if v := ret.String(); llssa.StripModuleTarget(v) != expected && expected != ";" { // expected == ";" means skipping out.ll
		t.Fatalf("\n==> got:\n%s\n==> expected:\n%s\n", v, expected)
	}
}
