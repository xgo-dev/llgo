package plan9asm

import (
	"context"
	_ "embed"
	"errors"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

//go:embed testdata/amd64-sdk-frame/copy_amd64.s
var amd64SDKFrameAssembly string

func amd64SDKFrameFixture(t *testing.T) (*packages.Package, string) {
	t.Helper()
	directory := t.TempDir()
	for _, name := range []string{"go.mod", "value.go", "copy_amd64.s", "cmd/main.go"} {
		inputName := name
		if name == "go.mod" {
			inputName += ".txt"
		}
		contents, err := os.ReadFile(filepath.Join("testdata", "amd64-sdk-frame", inputName))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fset := token.NewFileSet()
	path := filepath.Join(directory, "value.go")
	syntax, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object)}
	sizes := types.SizesFor("gc", "amd64")
	typed, err := (&types.Config{Sizes: sizes, Importer: importer.Default()}).Check("example.com/sdkframe", fset, []*ast.File{syntax}, info)
	if err != nil {
		t.Fatal(err)
	}
	imports := make(map[string]*packages.Package)
	for _, dependency := range typed.Imports() {
		imports[dependency.Path()] = &packages.Package{PkgPath: dependency.Path(), Types: dependency}
	}
	return &packages.Package{
		ID: typed.Path(), PkgPath: typed.Path(), Name: typed.Name(), Dir: directory,
		Fset: fset, Types: typed, TypesInfo: info, TypesSizes: sizes,
		Syntax: []*ast.File{syntax}, GoFiles: []string{path}, CompiledGoFiles: []string{path},
		OtherFiles: []string{filepath.Join(directory, "copy_amd64.s")},
		Imports:    imports,
	}, directory
}

func TestTranslateAMD64SDKABI0OriginalGo(t *testing.T) {
	_, directory := amd64SDKFrameFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-p=1", "-mod=readonly", "-o", filepath.Join(directory, "original"), "./cmd")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOROOT="+runtime.GOROOT(), "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOPROXY=off", "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("actual stock-Go assembly and complete ELF link: %v\n%s", err, output)
	}
}

func TestTranslateAMD64SDKABI0PhysicalFrame(t *testing.T) {
	pkg, _ := amd64SDKFrameFixture(t)
	opt := TranslateOptions{
		GOROOT: runtime.GOROOT(), GOAMD64: "v1",
		// Runtime patches in another package do not replace this original TU.
		Overlay: map[string][]byte{filepath.Join(t.TempDir(), "other.go"): []byte("package other\n")},
	}
	module, err := TranslateSourceModuleForPkgWithOptions(pkg, "copy_amd64.s", []byte(amd64SDKFrameAssembly), "linux", "amd64", opt)
	if err != nil {
		t.Fatal("actual SDK ABI0 outgoing arguments and physical spill:", err)
	}
	defer module.Module.Dispose()
	if !strings.Contains(module.Module.String(), "runtime.memmove") || strings.Contains(module.Module.String(), "@memmove(") {
		t.Fatal("SDK void ABI0 call must retain its runtime-owned Go hook")
	}
	if err := llvm.VerifyModule(module.Module, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	callee := module.Signatures["runtime.memmove"]
	if callee.Name != "runtime.memmove" || callee.Ret != extplan9asm.Void || len(callee.Frame.Params) != 3 {
		t.Fatalf("lost actual SDK ABI0 signature: %#v", callee)
	}
	for index, typ := range []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64} {
		if slot := callee.Frame.Params[index]; slot.Offset != int64(8*index) || slot.Type != typ {
			t.Fatalf("SDK outgoing slot %d: %#v", index, slot)
		}
	}
}

func TestTranslateAMD64SDKABI0SourceGuards(t *testing.T) {
	for _, mutation := range []string{"overlay", "changed expansion", "unknown input", "private escape"} {
		t.Run(mutation, func(t *testing.T) {
			pkg, directory := amd64SDKFrameFixture(t)
			opt := TranslateOptions{GOROOT: runtime.GOROOT(), GOAMD64: "v1"}
			source := amd64SDKFrameAssembly
			switch mutation {
			case "overlay":
				opt.Overlay = map[string][]byte{filepath.Join(directory, "value.go"): []byte("package changed\n")}
			case "changed expansion":
				opt.SourcePreprocessed = true
				source = strings.Replace(source, "#include \"textflag.h\"", "", 1)
				source = strings.Replace(source, "NOSPLIT", "4", 1)
				source = strings.Replace(source, "ADDQ $37", "ADDQ $39", 1)
			case "unknown input":
				source = strings.Replace(source, "MOVQ n+16(FP), CX", "MOVQ DX, CX", 1)
			case "private escape":
				source = strings.Replace(source, "MOVQ AX, 0(SP)", "MOVQ SP, AX\n\tMOVQ AX, 0(SP)", 1)
			}
			if mutation == "unknown input" || mutation == "private escape" {
				if err := os.WriteFile(filepath.Join(directory, "copy_amd64.s"), []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "copy_amd64.s", []byte(source), "linux", "amd64", opt)
			if tr != nil {
				tr.Module.Dispose()
			}
			if !errors.Is(err, extplan9asm.ErrProbeNeedsContext) {
				t.Fatalf("unproved source/frame acquired authority: %v", err)
			}
		})
	}
}

func TestTranslateAMD64SDKABI0LogicalAUTOHasNoObjectOwner(t *testing.T) {
	pkg, _ := amd64SDKFrameFixture(t)
	logical := []byte("TEXT ·Logical(SB),4,$16-8\nMOVQ $1,local-8(SP)\nMOVQ local-8(SP),AX\nMOVQ AX,ret+0(FP)\nRET\n")
	sigs, finish, err := bindAMD64Frame(pkg, logical, "missing.s", "linux", "amd64",
		TranslateOptions{GOROOT: "not-an-SDK"}, extplan9asm.GoPackage{}, "", nil)
	if err != nil || sigs != nil || finish != nil {
		t.Fatalf("logical AUTO unnecessarily queried or acquired physical source: %v", err)
	}
}

func TestTranslateAMD64SDKABI0FrameLeaseRevocation(t *testing.T) {
	for _, mutation := range []string{"close", "Go source", "assembly source"} {
		t.Run(mutation, func(t *testing.T) {
			pkg, directory := amd64SDKFrameFixture(t)
			goPkg := extplan9asm.GoPackage{Path: pkg.PkgPath, Types: pkg.Types, Syntax: pkg.Syntax,
				Imports: map[string]*types.Package{"runtime": pkg.Imports["runtime"].Types}}
			opt := TranslateOptions{GOROOT: runtime.GOROOT(), GOAMD64: "v1"}
			source, err := PreprocessFileForPkg(pkg, "copy_amd64.s", "linux", "amd64", opt)
			if err != nil {
				t.Fatal(err)
			}
			resolve := ResolveSymFunc(pkg.PkgPath)
			sigs, finish, err := bindAMD64Frame(pkg, source, "copy_amd64.s", "linux", "amd64", opt, goPkg,
				"x86_64-unknown-linux-gnu", resolve)
			if err != nil {
				t.Fatal(err)
			}
			if finish == nil {
				t.Fatal("physical frame did not retain an owner")
			}
			finished := false
			t.Cleanup(func() {
				if !finished {
					if err := finish(); err != nil {
						t.Error(err)
					}
				}
			})
			if mutation != "close" {
				path, contents := filepath.Join(directory, "value.go"), "package sdkframe\nfunc Changed() {}\n"
				if mutation == "assembly source" {
					path, contents = filepath.Join(directory, "copy_amd64.s"), strings.Replace(amd64SDKFrameAssembly, "$37", "$39", 1)
				}
				if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err = finish()
			finished = true
			if mutation == "close" && err != nil || mutation != "close" && !errors.Is(err, extplan9asm.ErrProbeNeedsContext) {
				t.Fatalf("source lifetime verification: %v", err)
			}
			tr, err := extplan9asm.TranslateGoModule(goPkg, source, extplan9asm.GoModuleOptions{
				FileName: "copy_amd64.s", SourcePreprocessed: true,
				GOOS: "linux", GOARCH: "amd64", TargetTriple: "x86_64-unknown-linux-gnu", ResolveSym: resolve,
				ManualSig: func(name string) (extplan9asm.FuncSig, bool) { sig, ok := sigs[name]; return sig, ok },
			})
			if tr != nil {
				tr.Module.Dispose()
			}
			if !errors.Is(err, extplan9asm.ErrProbeNeedsContext) {
				t.Fatalf("revoked source still authorized a physical frame: %v", err)
			}
		})
	}
}
