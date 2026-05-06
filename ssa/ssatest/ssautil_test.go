//go:build !llgo
// +build !llgo

package ssatest

import (
	"go/types"
	"testing"

	"github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func init() {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()
}

type mockImporter struct {
	pkgs map[string]*types.Package
}

func newMockImporter() *mockImporter {
	return &mockImporter{
		pkgs: make(map[string]*types.Package),
	}
}

func (m *mockImporter) Import(path string) (*types.Package, error) {
	if pkg, ok := m.pkgs[path]; ok {
		return pkg, nil
	}
	pkg := types.NewPackage(path, path)
	m.pkgs[path] = pkg
	return pkg, nil
}

func TestNewProgram(t *testing.T) {
	target := &ssa.Target{
		GOOS:   "linux",
		GOARCH: "amd64",
		GOARM:  "7",
	}

	prog := NewProgram(t, target)
	if prog == nil {
		t.Fatal("NewProgram returned nil")
	}

	// Set runtime package
	rtPkg := types.NewPackage(ssa.PkgRuntime, ssa.PkgRuntime)
	prog.SetRuntime(rtPkg)

	// Set python package
	pyPkg := types.NewPackage(ssa.PkgPython, ssa.PkgPython)
	prog.SetRuntime(pyPkg)
}

func TestNewProgramEx(t *testing.T) {
	target := &ssa.Target{
		GOOS:   "linux",
		GOARCH: "amd64",
		GOARM:  "7",
	}

	imp := newMockImporter()
	prog := NewProgramEx(t, target, imp)
	if prog == nil {
		t.Fatal("NewProgramEx returned nil")
	}

	// Set runtime package
	rtPkg := types.NewPackage(ssa.PkgRuntime, ssa.PkgRuntime)
	prog.SetRuntime(rtPkg)

	// Set python package
	pyPkg := types.NewPackage(ssa.PkgPython, ssa.PkgPython)
	prog.SetRuntime(pyPkg)
}

func TestAssert(t *testing.T) {
	target := &ssa.Target{
		GOOS:   "linux",
		GOARCH: "amd64",
		GOARM:  "7",
	}

	prog := NewProgram(t, target)
	if prog == nil {
		t.Fatal("NewProgram returned nil")
	}

	tests := []struct {
		name     string
		pkg      ssa.Package
		expected string
	}{
		{
			name:     "test package path",
			pkg:      prog.NewPackage("test", "test/path"),
			expected: "; ModuleID = 'test/path'\nsource_filename = \"test/path\"\n",
		},
		{
			name:     "another package path",
			pkg:      prog.NewPackage("another", "another/path"),
			expected: "; ModuleID = 'another/path'\nsource_filename = \"another/path\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Assert(t, tt.pkg, tt.expected)
		})
	}
}
