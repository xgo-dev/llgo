package build

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Whole reviewed Go1.27.0/1.27.2 sources. These entries have a native Windows
// C ABI tail, not an ordinary Go tail call. Keep all other package sources.
var windowsStdCallReviewedSources = map[string]string{
	"syscall_windows.go":  "0926c1e15eb3a44cd32a2a4529055486f76f9601e9968adbb6702e5059ef29cd",
	"asm_windows_386.s":   "0e06d52318dba3d45f60940239840b99c739f6e9990f6194ee8d9e52186a64be",
	"asm_windows_amd64.s": "4fc92d43006d2ee3862bc1d058c4dc53a5fe0d92ace8b9e6f3e97d95f1630c87",
	"asm_windows_arm64.s": "f6c4de146be5017d35267502eba02cbc6eb7105b5b925acd17e96a3f93eabae2",
}

func windowsStdCallSourcePatchView(dir string, entries []os.DirEntry, ctx build.Context,
	read func(string) ([]byte, error), base map[string][]byte, body []byte, skips map[string]struct{},
) (map[string][]byte, error) {
	const marker = "// replaced by LLGo source patch\n"
	filename := filepath.Join(dir, "asm_windows_"+ctx.GOARCH+".s")
	assembly, err := read(filename)
	if err != nil {
		return nil, err
	}
	reused := string(assembly) == marker
	if reused && !bytes.Equal(base[filepath.Join(dir, "z_llgo_patch_syscall_windows_go127.go")], body) {
		return nil, fmt.Errorf("Windows syscall marker lacks its exact generated body owner")
	}
	wantSkips := map[string]struct{}{
		"StdCall": {}, "AsmStdCallAddr": {}, "asmstdcall": {}, "llgoStdCall": {}, "llgoStdCallAddr": {},
	}
	if !reflect.DeepEqual(skips, wantSkips) {
		return nil, fmt.Errorf("Windows syscall declaration selection changed")
	}
	var originals [][]byte
	seen := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_test.s") ||
			(!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s")) {
			continue
		}
		match, err := ctx.MatchFile(dir, name)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}
		if strings.HasSuffix(name, ".s") && filepath.Join(dir, name) != filename {
			return nil, fmt.Errorf("unreviewed Windows syscall assembly owner %s", name)
		}
		path := filepath.Join(dir, name)
		source, err := read(path)
		if err != nil {
			return nil, err
		}
		if hash := windowsStdCallReviewedSources[name]; hash != "" {
			if reused {
				original, err := os.ReadFile(path)
				if err != nil {
					return nil, err
				}
				if strings.HasSuffix(name, ".go") {
					filtered, _, err := filterSourcePatchFile(original, skips)
					if err != nil || !bytes.Equal(source, filtered) {
						return nil, fmt.Errorf("changed generated Windows declaration view")
					}
				}
				source = original
			}
			if fmt.Sprintf("%x", sha256.Sum256(source)) != hash {
				return nil, fmt.Errorf("unreviewed complete Windows syscall source %s", name)
			}
			seen++
		}
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		originals = append(originals, source)
		filtered, _, err := filterSourcePatchFile(source, skips)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, filtered, 0)
		if err != nil {
			return nil, err
		}
		usesPrivate := false
		ast.Inspect(file, func(node ast.Node) bool {
			if name, ok := node.(*ast.Ident); ok && name.Name == "asmstdcall" {
				usesPrivate = true
			}
			return true
		})
		if usesPrivate {
			return nil, fmt.Errorf("remaining SDK Go consumer of native asmstdcall")
		}
	}
	if seen != 2 {
		return nil, fmt.Errorf("incomplete selected Windows syscall source group")
	}
	before, err := atomicSourcePatchFunctions(originals)
	if err != nil {
		return nil, err
	}
	after, err := atomicSourcePatchFunctions([][]byte{body})
	if err != nil {
		return nil, err
	}
	if len(after) != 4 || before["llgoStdCall"] != nil || before["llgoStdCallAddr"] != nil {
		return nil, fmt.Errorf("Windows syscall replacement body group changed")
	}
	for _, name := range []string{"StdCall", "AsmStdCallAddr"} {
		original, replacement := before[name], after[name]
		if original == nil || replacement == nil || replacement.Body == nil || len(replacement.Body.List) == 0 {
			return nil, fmt.Errorf("Windows syscall %s lacks its original signature and real Go body", name)
		}
		want, err := atomicSourcePatchSignature(original.Type)
		if err != nil {
			return nil, err
		}
		got, err := atomicSourcePatchSignature(replacement.Type)
		if err != nil || !reflect.DeepEqual(got, want) {
			return nil, fmt.Errorf("Windows syscall %s signature changed", name)
		}
	}
	return map[string][]byte{filename: []byte(marker)}, nil
}
