package build

import (
	"fmt"
	"go/ast"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"

	gllvm "github.com/xgo-dev/llvm"
)

type cgoImportDynamicDecl struct {
	local   string
	alias   string
	library string
}

func collectGoCgoPragmas(files []*ast.File) (ldflags []string, dynimports []cgoImportDynamicDecl) {
	for _, f := range files {
		if f == nil {
			continue
		}
		for _, cg := range f.Comments {
			if cg == nil {
				continue
			}
			for _, c := range cg.List {
				if c == nil {
					continue
				}
				for _, line := range splitCommentLines(c.Text) {
					switch {
					case strings.HasPrefix(line, "go:cgo_ldflag"):
						rest := strings.TrimSpace(strings.TrimPrefix(line, "go:cgo_ldflag"))
						for _, tok := range splitDirectiveArgs(rest) {
							ldflags = append(ldflags, tok)
						}
					case strings.HasPrefix(line, "go:cgo_import_dynamic"):
						rest := strings.TrimSpace(strings.TrimPrefix(line, "go:cgo_import_dynamic"))
						toks := splitDirectiveArgs(rest)
						if len(toks) == 0 {
							continue
						}
						local := toks[0]
						alias := local
						if len(toks) > 1 && toks[1] != "" {
							alias = toks[1]
						}
						if local == "" || alias == "" {
							continue
						}
						library := ""
						if len(toks) > 2 {
							library = toks[2]
						}
						dynimports = append(dynimports, cgoImportDynamicDecl{
							local:   local,
							alias:   alias,
							library: library,
						})
					}
				}
			}
		}
	}
	return
}

func goCgoLinkArgs(files []*ast.File, goos string) []string {
	ldflags, imports := collectGoCgoPragmas(files)
	if goos != "darwin" {
		return ldflags
	}
	seen := make(map[string]bool)
	for _, imp := range imports {
		lib := imp.library
		if lib == "" || seen[lib] {
			continue
		}
		seen[lib] = true
		// System libraries may exist only in dyld's shared cache. Let the native
		// compiler resolve their SDK stubs instead of opening the runtime path.
		if strings.HasPrefix(lib, "/System/Library/Frameworks/") {
			rest := strings.TrimPrefix(lib, "/System/Library/Frameworks/")
			if name, _, ok := strings.Cut(rest, ".framework/"); ok && !strings.Contains(name, "/") {
				ldflags = append(ldflags, "-framework", name)
				continue
			}
		}
		if path.Dir(lib) == "/usr/lib" && strings.HasPrefix(path.Base(lib), "lib") && strings.HasSuffix(lib, ".dylib") {
			name := strings.TrimSuffix(strings.TrimPrefix(path.Base(lib), "lib"), ".dylib")
			ldflags = append(ldflags, "-l"+name)
		} else {
			ldflags = append(ldflags, lib)
		}
	}
	return ldflags
}

// lowerWindowsCgoImportPointers connects pointer variables emitted for
// //go:cgo_import_dynamic declarations to COFF dllimport functions. The Go
// Windows syscall sources call these variables after the linker resolves the
// imported address. Leaving the generated variables nil faults during package
// initialization before main can run.
func lowerWindowsCgoImportPointers(goos, goarch, pkgPath string, files []*ast.File, mod gllvm.Module) error {
	if goos != "windows" || mod.IsNil() {
		return nil
	}
	_, dynimports := collectGoCgoPragmas(files)
	type aliasShape struct {
		alias       string
		argc        int
		hasArgCount bool
	}
	seen := make(map[string]string, len(dynimports))
	seenAliases := make(map[string]aliasShape, len(dynimports))
	for _, d := range dynimports {
		if prev, ok := seen[d.local]; ok {
			if prev == d.alias {
				continue
			}
			return fmt.Errorf("%s: conflicting go:cgo_import_dynamic for %q: %q vs %q", pkgPath, d.local, prev, d.alias)
		}
		seen[d.local] = d.alias
		name, argc, hasArgCount, err := splitWindowsCgoImportAlias(d.alias)
		if err != nil {
			return fmt.Errorf("%s: invalid go:cgo_import_dynamic alias %q: %w", pkgPath, d.alias, err)
		}
		if prev, ok := seenAliases[name]; ok &&
			(prev.argc != argc || prev.hasArgCount != hasArgCount) {
			return fmt.Errorf("%s: conflicting go:cgo_import_dynamic argument counts for %q: %q vs %q", pkgPath, name, prev.alias, d.alias)
		}
		seenAliases[name] = aliasShape{d.alias, argc, hasArgCount}

		global := mod.NamedGlobal(d.local)
		if global.IsNil() {
			// Function declarations already name the import directly and do not
			// require an address-variable initializer.
			continue
		}
		if global.GlobalValueType().TypeKind() != gllvm.PointerTypeKind {
			return fmt.Errorf("%s: go:cgo_import_dynamic local %q is not a pointer variable", pkgPath, d.local)
		}
		fn := mod.NamedFunction(name)
		if fn.IsNil() {
			argType := mod.Context().Int32Type()
			params := make([]gllvm.Type, argc)
			for i := range params {
				params[i] = argType
			}
			fn = gllvm.AddFunction(mod, name, gllvm.FunctionType(mod.Context().VoidType(), params, false))
		} else if !fn.IsDeclaration() {
			return fmt.Errorf("%s: go:cgo_import_dynamic alias %q collides with a defined function", pkgPath, name)
		}
		if goarch == "386" && hasArgCount {
			fn.SetFunctionCallConv(gllvm.X86StdcallCallConv)
		}
		fn.SetDLLStorageClass(gllvm.DLLImportStorageClass)
		global.SetInitializer(fn)
	}
	return nil
}

func splitWindowsCgoImportAlias(alias string) (name string, argc int, hasArgCount bool, err error) {
	name = alias
	percent := strings.IndexByte(alias, '%')
	if percent < 0 {
		return name, 0, false, nil
	}
	name = alias[:percent]
	if name == "" || percent+1 == len(alias) {
		return "", 0, false, fmt.Errorf("missing symbol name or argument count")
	}
	argc64, parseErr := strconv.ParseUint(alias[percent+1:], 10, 31)
	if parseErr != nil {
		return "", 0, false, fmt.Errorf("invalid argument count: %w", parseErr)
	}
	return name, int(argc64), true, nil
}

func buildGoCgoAliasObjects(ctx *context, pkgPath string, files []*ast.File, verbose bool) ([]string, error) {
	if ctx == nil || ctx.buildConf == nil || ctx.buildConf.Goos != "darwin" {
		return nil, nil
	}
	_, dynimports := collectGoCgoPragmas(files)
	if len(dynimports) == 0 {
		return nil, nil
	}

	seen := make(map[string]string, len(dynimports))
	var b strings.Builder
	for _, d := range dynimports {
		if d.local == "" || d.alias == "" || d.local == d.alias {
			continue
		}
		if prev, ok := seen[d.local]; ok {
			if prev == d.alias {
				continue
			}
			return nil, fmt.Errorf("%s: conflicting go:cgo_import_dynamic for %q: %q vs %q", pkgPath, d.local, prev, d.alias)
		}
		seen[d.local] = d.alias
		emitDarwinDynimportTrampoline(&b, ctx.buildConf.Goarch, d.local, d.alias)
	}
	if len(seen) == 0 {
		return nil, nil
	}

	asmFile, err := os.CreateTemp("", "cgoimportalias-*.s")
	if err != nil {
		return nil, err
	}
	asmPath := asmFile.Name()
	if _, err := asmFile.WriteString(b.String()); err != nil {
		asmFile.Close()
		os.Remove(asmPath)
		return nil, err
	}
	if err := asmFile.Close(); err != nil {
		os.Remove(asmPath)
		return nil, err
	}
	defer os.Remove(asmPath)

	objFile, err := os.CreateTemp("", "cgoimportalias-*.o")
	if err != nil {
		return nil, err
	}
	objPath := objFile.Name()
	objFile.Close()
	args := []string{"-o", objPath, "-c", asmPath}
	if ctx.shouldPrintCommands(verbose) {
		fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", objPath, pkgPath)
		fmt.Fprintln(os.Stderr, "clang", args)
	}
	if err := ctx.compiler().Compile(args...); err != nil {
		os.Remove(objPath)
		return nil, err
	}
	return []string{objPath}, nil
}

func emitDarwinDynimportTrampoline(b *strings.Builder, goarch string, local string, alias string) {
	if b == nil || local == "" || alias == "" {
		return
	}
	trampoline := local + "_trampoline"
	addr := local + "_trampoline_addr"
	// Mach-O external C symbols use a leading underscore.
	switch goarch {
	case "arm64":
		b.WriteString(".text\n")
		b.WriteString(".globl _")
		b.WriteString(local)
		b.WriteString("\n.p2align 2\n_")
		b.WriteString(local)
		b.WriteString(":\n\tb _")
		b.WriteString(alias)
		b.WriteString("\n")
		b.WriteString(".globl _")
		b.WriteString(trampoline)
		b.WriteString("\n.p2align 2\n_")
		b.WriteString(trampoline)
		b.WriteString(":\n\tb _")
		b.WriteString(local)
		b.WriteString("\n.data\n.globl _")
		b.WriteString(addr)
		b.WriteString("\n.p2align 3\n_")
		b.WriteString(addr)
		b.WriteString(":\n\t.quad _")
		b.WriteString(trampoline)
		b.WriteString("\n")
	case "amd64":
		b.WriteString(".text\n")
		b.WriteString(".globl _")
		b.WriteString(local)
		b.WriteString("\n.p2align 4, 0x90\n_")
		b.WriteString(local)
		b.WriteString(":\n\tjmp _")
		b.WriteString(alias)
		b.WriteString("\n")
		b.WriteString(".globl _")
		b.WriteString(trampoline)
		b.WriteString("\n.p2align 4, 0x90\n_")
		b.WriteString(trampoline)
		b.WriteString(":\n\tjmp _")
		b.WriteString(local)
		b.WriteString("\n.data\n.globl _")
		b.WriteString(addr)
		b.WriteString("\n.p2align 3\n_")
		b.WriteString(addr)
		b.WriteString(":\n\t.quad _")
		b.WriteString(trampoline)
		b.WriteString("\n")
	default:
		// Keep unsupported arch silent for now; caller already gates by darwin.
	}
}

func splitCommentLines(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitDirectiveArgs(s string) []string {
	// Match cmd/compile's pragma field boundaries: quoted regions are single
	// fields, and an unterminated trailing quoted region is ignored. This also
	// handles the extra trailing quote in Go 1.27 runtime/sys_darwin.go.
	var fields []string
	start, quoted := -1, false
	for i, ch := range s {
		if ch == '"' {
			if quoted {
				fields = append(fields, s[start:i+1])
				start = -1
			} else {
				if start >= 0 {
					fields = append(fields, s[start:i])
				}
				start = i
			}
			quoted = !quoted
		} else if !quoted && unicode.IsSpace(ch) {
			if start >= 0 {
				fields = append(fields, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if !quoted && start >= 0 {
		fields = append(fields, s[start:])
	}
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if u, err := strconv.Unquote(f); err == nil {
			out = append(out, u)
		} else {
			out = append(out, f)
		}
	}
	return out
}
