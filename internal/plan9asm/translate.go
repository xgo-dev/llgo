package plan9asm

import (
	"fmt"
	"go/types"
	"os"
	"regexp"
	"strings"

	"github.com/goplus/llgo/internal/packages"
	intllvm "github.com/goplus/llgo/internal/xtool/llvm"
	gllvm "github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

type FunctionInfo struct {
	TextSymbol     string `json:"text_symbol"`
	ResolvedSymbol string `json:"resolved_symbol"`
}

type FileTranslation struct {
	LLVMIR     string                         `json:"-"`
	Signatures map[string]extplan9asm.FuncSig `json:"signatures"`
	Functions  []FunctionInfo                 `json:"functions"`
}

type ModuleTranslation struct {
	Module     gllvm.Module
	Signatures map[string]extplan9asm.FuncSig
	Functions  []FunctionInfo
}

type TranslateOptions struct {
	AnnotateSource bool
}

func TranslateFileForPkg(pkg *packages.Package, sfile string, goos string, goarch string, overlay map[string][]byte) (*FileTranslation, error) {
	return TranslateFileForPkgWithOptions(pkg, sfile, goos, goarch, overlay, TranslateOptions{})
}

func TranslateFileForPkgWithOptions(pkg *packages.Package, sfile string, goos string, goarch string, overlay map[string][]byte, opt TranslateOptions) (*FileTranslation, error) {
	if pkg == nil {
		return nil, fmt.Errorf("nil package")
	}
	src, err := ReadFileWithOverlay(overlay, sfile)
	if err != nil {
		return nil, err
	}
	return TranslateSourceForPkgWithOptions(pkg, sfile, src, goos, goarch, opt)
}

func TranslateSourceForPkg(pkg *packages.Package, sfile string, src []byte, goos string, goarch string) (*FileTranslation, error) {
	return TranslateSourceForPkgWithOptions(pkg, sfile, src, goos, goarch, TranslateOptions{})
}

func TranslateSourceForPkgWithOptions(pkg *packages.Package, sfile string, src []byte, goos string, goarch string, opt TranslateOptions) (*FileTranslation, error) {
	tr, err := TranslateSourceModuleForPkgWithOptions(pkg, sfile, src, goos, goarch, opt)
	if err != nil {
		return nil, err
	}
	defer tr.Module.Dispose()
	return &FileTranslation{LLVMIR: tr.Module.String(), Signatures: tr.Signatures, Functions: tr.Functions}, nil
}

func TranslateSourceModuleForPkg(pkg *packages.Package, sfile string, src []byte, goos string, goarch string) (*ModuleTranslation, error) {
	return TranslateSourceModuleForPkgWithOptions(pkg, sfile, src, goos, goarch, TranslateOptions{})
}

func TranslateSourceModuleForPkgWithOptions(pkg *packages.Package, sfile string, src []byte, goos string, goarch string, opt TranslateOptions) (*ModuleTranslation, error) {
	if pkg == nil {
		return nil, fmt.Errorf("nil package")
	}
	if pkg.PkgPath == "" {
		return nil, fmt.Errorf("empty package path")
	}
	if pkg.Types == nil || pkg.Types.Scope() == nil {
		return nil, fmt.Errorf("%s: missing types (needed for asm signatures)", pkg.PkgPath)
	}

	resolve := resolveSymFuncForTarget(pkg.PkgPath, goos, goarch)
	keep := func(textSym, resolved string) bool {
		return shouldKeepResolvedFunc(pkg.PkgPath, goos, goarch, resolved)
	}
	manualSig := func(resolved string) (extplan9asm.FuncSig, bool) {
		fs, ok := extraAsmSigsAndDeclMap(pkg.PkgPath, goarch)[resolved]
		if ok && fs.Name == "" {
			fs.Name = resolved
		}
		return fs, ok
	}

	imports := make(map[string]*types.Package, len(pkg.Imports))
	for path, imp := range pkg.Imports {
		if imp != nil && imp.Types != nil {
			imports[path] = imp.Types
		}
	}

	tr, err := extplan9asm.TranslateGoModule(extplan9asm.GoPackage{
		Path:    pkg.PkgPath,
		Types:   pkg.Types,
		Imports: imports,
		Syntax:  pkg.Syntax,
	}, src, extplan9asm.GoModuleOptions{
		FileName:       sfile,
		GOOS:           goos,
		GOARCH:         goarch,
		TargetTriple:   intllvm.GetTargetTriple(goos, goarch),
		AnnotateSource: opt.AnnotateSource,
		ResolveSym:     resolve,
		KeepFunc:       keep,
		ManualSig:      manualSig,
	})
	if err != nil {
		return nil, err
	}

	funcs := make([]FunctionInfo, 0, len(tr.Functions))
	for _, fn := range tr.Functions {
		funcs = append(funcs, FunctionInfo{TextSymbol: fn.TextSymbol, ResolvedSymbol: fn.ResolvedSymbol})
	}
	return &ModuleTranslation{Module: tr.Module, Signatures: tr.Signatures, Functions: funcs}, nil
}

func shouldKeepResolvedFunc(pkgPath, goos, goarch, resolved string) bool {
	if pkgPath == "syscall" && goos == "linux" && (goarch == "arm64" || goarch == "amd64") && strings.HasSuffix(resolved, "rawVforkSyscall") {
		return false
	}
	return true
}

func FilterFuncs(pkgPath, goos, goarch string, funcs []extplan9asm.Func, resolve func(sym string) string) []extplan9asm.Func {
	if len(funcs) == 0 {
		return funcs
	}
	keep := funcs[:0]
	for _, fn := range funcs {
		resolved := resolve(StripABISuffix(fn.Sym))
		if shouldKeepResolvedFunc(pkgPath, goos, goarch, resolved) {
			keep = append(keep, fn)
		}
	}
	return keep
}

func ResolveSymFunc(pkgPath string) func(sym string) string {
	return resolveSymFuncForTarget(pkgPath, "", "")
}

func resolveSymFuncForTarget(pkgPath, goos, goarch string) func(sym string) string {
	return func(sym string) string {
		sym = StripABISuffix(sym)
		if strings.HasPrefix(sym, "·") {
			return pkgPath + "." + strings.TrimPrefix(sym, "·")
		}
		if !strings.Contains(sym, "·") && !strings.Contains(sym, ".") && !strings.Contains(sym, "/") {
			return pkgPath + "." + sym
		}
		sym = strings.ReplaceAll(sym, "∕", "/")
		return strings.ReplaceAll(sym, "·", ".")
	}
}

func ReadFileWithOverlay(overlay map[string][]byte, path string) ([]byte, error) {
	if overlay != nil {
		if b, ok := overlay[path]; ok {
			return b, nil
		}
	}
	return os.ReadFile(path)
}

func HasAnyTextAsm(overlay map[string][]byte, files []string) (bool, error) {
	for _, f := range files {
		src, err := ReadFileWithOverlay(overlay, f)
		if err != nil {
			return false, err
		}
		if reTextDirective.Match(src) {
			return true, nil
		}
	}
	return false, nil
}

var abiSuffixRe = regexp.MustCompile(`<ABI[^>]*>$`)
var reTextDirective = regexp.MustCompile(`(?m)^\s*TEXT\b`)

func StripABISuffix(sym string) string {
	sym = abiSuffixRe.ReplaceAllString(sym, "")
	return strings.TrimSuffix(sym, "<>")
}

func extraAsmSigsAndDeclMap(pkgPath string, goarch string) map[string]extplan9asm.FuncSig {
	manual := map[string]extplan9asm.FuncSig{}
	if pkgPath == "internal/bytealg" {
		switch goarch {
		case "arm64":
			manual["internal/bytealg.cmpbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr, extplan9asm.I64}, Ret: extplan9asm.I64}
			manual["internal/bytealg.memeqbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64}, Ret: extplan9asm.I1}
			manual["internal/bytealg.countbytebody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.LLVMType("i8"), extplan9asm.Ptr}, Ret: extplan9asm.Void, ArgRegs: []extplan9asm.Reg{"R0", "R2", "R1", "R8"}}
			manual["internal/bytealg.indexbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr}, Ret: extplan9asm.Void, ArgRegs: []extplan9asm.Reg{"R0", "R1", "R2", "R3", "R9"}}
			manual["internal/bytealg.indexbytebody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.LLVMType("i8"), extplan9asm.I64, extplan9asm.Ptr}, Ret: extplan9asm.Void, ArgRegs: []extplan9asm.Reg{"R0", "R1", "R2", "R8"}}
		case "amd64":
			manual["internal/bytealg.cmpbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64, extplan9asm.I64}, Ret: extplan9asm.I64, ArgRegs: []extplan9asm.Reg{extplan9asm.SI, extplan9asm.DI, extplan9asm.BX, extplan9asm.DX}}
			manual["internal/bytealg.countbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.LLVMType("i8"), extplan9asm.Ptr}, Ret: extplan9asm.Void, ArgRegs: []extplan9asm.Reg{extplan9asm.SI, extplan9asm.BX, extplan9asm.AX, extplan9asm.Reg("R8")}}
			manual["internal/bytealg.indexbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr, extplan9asm.Ptr}, Ret: extplan9asm.Void, ArgRegs: []extplan9asm.Reg{extplan9asm.DI, extplan9asm.DX, extplan9asm.Reg("R8"), extplan9asm.AX, extplan9asm.Reg("R10"), extplan9asm.Reg("R11")}}
			manual["internal/bytealg.indexbytebody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.LLVMType("i8"), extplan9asm.Ptr}, Ret: extplan9asm.Void, ArgRegs: []extplan9asm.Reg{extplan9asm.SI, extplan9asm.BX, extplan9asm.AX, extplan9asm.Reg("R8")}}
			manual["internal/bytealg.memeqbody"] = extplan9asm.FuncSig{Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64}, Ret: extplan9asm.I1, ArgRegs: []extplan9asm.Reg{extplan9asm.SI, extplan9asm.DI, extplan9asm.BX}}
		}
	}
	if pkgPath == "internal/runtime/gc/scan" && goarch == "amd64" {
		// Go 1.26 emits local AVX512 scan expanders with no Go declarations.
		// They still need signatures so the translator can keep the asm entry
		// points and wire the packed mask argument through AX.
		for _, n := range []int{
			1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32,
			36, 40, 44, 48, 52, 56, 60, 64,
		} {
			name := fmt.Sprintf("internal/runtime/gc/scan.expandAVX512_%d", n)
			manual[name] = extplan9asm.FuncSig{
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{extplan9asm.AX},
			}
		}
	}
	return manual
}
