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
						if local == "" || alias == "" || local == alias {
							continue
						}
						dynimports = append(dynimports, cgoImportDynamicDecl{
							local: local,
							alias: alias,
						})
					}
				}
			}
		}
	}
	return
}

func goCgoLinkArgs(goos string, files []*ast.File) []string {
	ldflags, _ := collectGoCgoPragmas(files)
	return ldflags
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
	fields := strings.Fields(strings.TrimSpace(s))
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
