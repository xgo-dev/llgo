package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
	"github.com/xgo-dev/llgo/ssa/abi"
	llvm "github.com/xgo-dev/llvm"
)

var nativeTextRE = regexp.MustCompile(`(?m)^\s*TEXT\s+([^\s(),]+)<>\(SB\),\s*([^,]+),\s*\$0(?:-0)?\s*(?://[^\n]*)?$`)

// foreignARM64Functions selects files made entirely of raw foreign-ABI
// callbacks/trampolines. Go-declared functions continue through typed LLVM
// lowering. NOSPLIT and zero Go frames are required; the callback may manage
// its own native frame explicitly with NOFRAME.
func foreignARM64Functions(src []byte) map[string]bool {
	matches := nativeTextRE.FindAllSubmatchIndex(src, -1)
	if len(matches) == 0 || len(reTextLines.FindAll(src, -1)) != len(matches) {
		return nil
	}
	result := make(map[string]bool)
	for i, m := range matches {
		flags := string(src[m[4]:m[5]])
		if !hasAsmFlag(flags, "NOSPLIT") {
			return nil
		}
		end := len(src)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		if !hasAsmFlag(flags, "NOFRAME") && nativeCallRE.Match(src[m[1]:end]) {
			return nil
		}
		result[string(src[m[2]:m[3]])] = true
	}
	return result
}

var reTextLines = regexp.MustCompile(`(?m)^\s*TEXT\b`)
var nativeCallRE = regexp.MustCompile(`(?m)^\s*(?:CALL|BL)\s`)

func hasAsmFlag(flags, want string) bool {
	for _, f := range strings.Split(flags, "|") {
		if strings.TrimSpace(f) == want {
			return true
		}
	}
	return false
}

func compileForeignARM64Asm(ctx *context, aPkg *aPackage, pkg *packages.Package, sfile string, src []byte) (string, bool, error) {
	if ctx.buildConf.Goos != "darwin" || ctx.buildConf.Goarch != "arm64" {
		return "", false, nil
	}
	_, decls := collectGoCgoPragmas(pkg.Syntax)
	if len(decls) == 0 {
		return "", false, nil
	}
	funcs := foreignARM64Functions(src)
	if len(funcs) == 0 {
		return "", false, nil
	}
	imports := make(map[string]string)
	for _, d := range decls {
		if prev, ok := imports[d.local]; ok && prev != d.alias {
			return "", true, fmt.Errorf("conflicting dynamic import %s", d.local)
		}
		imports[d.local] = d.alias
	}
	root, _, err := env.GOROOTAndGOVERSIONWithEnv(ctx.commands.environ)
	if err != nil {
		return "", true, err
	}
	dir, err := os.MkdirTemp("", "llgo-native-asm-")
	if err != nil {
		return "", true, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "input.s")
	if err = os.WriteFile(input, src, 0600); err != nil {
		return "", true, err
	}
	obj := filepath.Join(dir, "input.o")
	pkgPath := abi.PathOf(pkg.Types)
	cmd := exec.Command(filepath.Join(root, "bin", "go"), "tool", "asm", "-p", pkgPath, "-I", filepath.Join(root, "pkg", "include"), "-I", filepath.Dir(sfile), "-o", obj, input)
	cmd.Env = withEnv(ctx.commands.environ, "GOOS=darwin", "GOARCH=arm64", "GOROOT="+root)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", true, fmt.Errorf("assemble foreign ABI code: %w\n%s", err, out)
	}
	object, err := os.ReadFile(obj)
	if err != nil {
		return "", true, err
	}
	assembly, data, err := llplan9asm.NativeARM64Object(object, funcs, imports, pkgPath)
	if err != nil {
		return "", true, err
	}
	gas := filepath.Join(dir, "native.s")
	if err = os.WriteFile(gas, []byte(assembly), 0600); err != nil {
		return "", true, err
	}
	output, err := os.CreateTemp("", "llgo-native-asm-*.o")
	if err != nil {
		return "", true, err
	}
	outpath := output.Name()
	output.Close()
	if err = ctx.irCompiler().Compile("-c", gas, "-o", outpath); err != nil {
		os.Remove(outpath)
		return "", true, err
	}
	// Reuse the normal DATA binding checks before changing the Go definitions.
	mod := aPkg.LPkg.Module().Context().NewModule("native-data")
	defer mod.Dispose()
	for _, d := range data {
		g := llvm.AddGlobal(mod, llvm.ArrayType(mod.Context().Int8Type(), int(d.Size)), d.Name)
		g.SetInitializer(llvm.ConstNull(g.GlobalValueType()))
	}
	if err = externalizePlan9DataGlobals(aPkg.LPkg.Module(), mod, ctx.prog.TargetData()); err != nil {
		os.Remove(outpath)
		return "", true, err
	}
	return outpath, true, nil
}
