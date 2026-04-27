package build

import (
	"fmt"
	"go/ast"
	"os"
	"strconv"
	"strings"
)

type cgoImportDynamicDecl struct {
	local string
	alias string
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
				if strings.HasPrefix(c.Text, "//go:cgo_") {
					if dynimports == nil && strings.HasPrefix(c.Text, "//go:cgo_import_dynamic") {
						dynimports = make([]cgoImportDynamicDecl, 0, len(cg.List))
					}
					ldflags, dynimports = collectGoCgoPragmaLine(c.Text[2:], ldflags, dynimports)
					continue
				}
				if !strings.Contains(c.Text, "go:cgo_") {
					continue
				}
				if dynimports == nil && strings.Contains(c.Text, "go:cgo_import_dynamic") {
					dynimports = make([]cgoImportDynamicDecl, 0, len(cg.List))
				}
				forEachCommentLine(c.Text, func(line string) {
					ldflags, dynimports = collectGoCgoPragmaLine(line, ldflags, dynimports)
				})
			}
		}
	}
	return
}

func collectGoCgoPragmaLine(line string, ldflags []string, dynimports []cgoImportDynamicDecl) ([]string, []cgoImportDynamicDecl) {
	switch {
	case strings.HasPrefix(line, "go:cgo_ldflag"):
		rest := strings.TrimSpace(line[len("go:cgo_ldflag"):])
		for _, tok := range splitDirectiveArgs(rest) {
			ldflags = append(ldflags, tok)
		}
	case strings.HasPrefix(line, "go:cgo_import_dynamic"):
		rest := strings.TrimSpace(line[len("go:cgo_import_dynamic"):])
		local, rest, ok := nextDirectiveArg(rest)
		if !ok {
			return ldflags, dynimports
		}
		alias := local
		if next, _, ok := nextDirectiveArg(rest); ok && next != "" {
			alias = next
		}
		if local == "" || alias == "" || local == alias {
			return ldflags, dynimports
		}
		dynimports = append(dynimports, cgoImportDynamicDecl{
			local: local,
			alias: alias,
		})
	}
	return ldflags, dynimports
}

func nextDirectiveArg(s string) (arg string, rest string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", false
	}
	end := len(s)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\r':
			end = i
			i = len(s)
		}
	}
	arg = s[:end]
	if len(arg) >= 2 {
		switch arg[0] {
		case '`', '\'', '"':
			if u, err := strconv.Unquote(arg); err == nil {
				arg = u
			}
		}
	}
	return arg, s[end:], true
}

func goCgoLinkArgs(goos string, files []*ast.File) []string {
	ldflags, _ := collectGoCgoPragmas(files)
	return ldflags
}

func buildGoCgoAliasObjects(ctx *context, pkgPath string, dynimports []cgoImportDynamicDecl, verbose bool) ([]string, error) {
	if ctx == nil || ctx.buildConf == nil || ctx.buildConf.Goos != "darwin" || len(dynimports) == 0 {
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

func forEachCommentLine(text string, fn func(string)) {
	for start := 0; start <= len(text); {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		line := strings.TrimSpace(text[start:end])
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		line = strings.TrimSpace(line)
		if line != "" {
			fn(line)
		}
		if end == len(text) {
			break
		}
		start = end + 1
	}
}

func splitDirectiveArgs(s string) []string {
	fields := strings.Fields(s)
	for i, f := range fields {
		if len(f) < 2 {
			continue
		}
		switch f[0] {
		case '`', '\'', '"':
			if u, err := strconv.Unquote(f); err == nil {
				fields[i] = u
			}
		}
	}
	return fields
}
