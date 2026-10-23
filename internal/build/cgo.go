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

package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/buildtags"
	llclang "github.com/xgo-dev/llgo/internal/clang"
	llssa "github.com/xgo-dev/llgo/ssa"
	xenv "github.com/xgo-dev/llgo/xtool/env"
	"github.com/xgo-dev/llgo/xtool/safesplit"
)

type cgoDecl struct {
	tag      string
	cflags   []string
	cxxflags []string
	ldflags  []string
}

type cgoSrcFile struct {
	path  string
	isCXX bool
}

type cgoPreamble struct {
	goFile string
	src    string
}

const (
	cgoHeader = `
#include <stdlib.h>
static void* _Cmalloc(size_t size) {
	return malloc(size);
}
`
)

var cgoExternRE = regexp.MustCompile(`^(_cgo_[^_]+_(C2func|Cfunc|Cmacro)_)(.*)$`)

func buildCgo(ctx *context, pkg *aPackage, files []*ast.File, externs []string, verbose bool) (llfiles, cgoLdflags []string, err error) {
	defer func() {
		if err != nil {
			removeFiles(llfiles)
			llfiles = nil
		}
	}()
	buildCtx := cgoBuildContext(ctx)

	srcFiles, preambles, cdecls, err := parseCgoWithCommandEnv(ctx.commands, &buildCtx, pkg, files)
	if err != nil {
		return
	}
	cflags, cxxflags, ldflags := cgoCompilationFlags(preambles, cdecls)
	for _, src := range srcFiles {
		args := slices.Clone(cflags)
		if src.isCXX {
			args = append(args, cxxflags...)
		}
		linkFile, compileErr := clFile(ctx, args, src.path, pkg.ExportFile, pkg.PkgPath, verbose)
		if compileErr != nil {
			return nil, nil, compileErr
		}
		llfiles = append(llfiles, linkFile)
	}
	cgoSymbols := collectCgoSymbols(externs)
	for _, preamble := range preambles {
		tmpFile, err := os.CreateTemp("", "-cgo-*.c")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp file: %v", err)
		}
		tmpName := tmpFile.Name()
		defer os.Remove(tmpName)
		code := cgoHeader + "\n\n" + preamble.src
		externDecls, err := genExternDeclsByClang(ctx.compiler(), pkg, code, cflags, cgoSymbols, verbose)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate extern decls: %v", err)
		}
		if err = os.WriteFile(tmpName, []byte(code+"\n\n"+externDecls), 0644); err != nil {
			return nil, nil, fmt.Errorf("failed to write temp file: %v", err)
		}
		linkFile, compileErr := clFile(ctx, cflags, tmpName, pkg.ExportFile, pkg.PkgPath, verbose)
		if compileErr != nil {
			return nil, nil, compileErr
		}
		llfiles = append(llfiles, linkFile)
	}
	// Flags were already split when parsing the directives/pkg-config output.
	// Re-splitting individual arguments corrupts standalone framework names.
	cgoLdflags = ldflags
	return
}

func collectCgoFlags(decls []cgoDecl) (cflags, cxxflags, ldflags []string) {
	for _, decl := range decls {
		cflags = append(cflags, decl.cflags...)
		cxxflags = append(cxxflags, decl.cxxflags...)
		ldflags = append(ldflags, decl.ldflags...)
	}
	return
}

func cgoBuildContext(ctx *context) build.Context {
	buildCtx := build.Default
	if ctx.buildConf != nil {
		if ctx.buildConf.Goos != "" {
			buildCtx.GOOS = ctx.buildConf.Goos
		}
		if ctx.buildConf.Goarch != "" {
			buildCtx.GOARCH = ctx.buildConf.Goarch
		}
		if ctx.buildConf.toolTags != nil {
			buildCtx.ToolTags = slices.Clone(ctx.buildConf.toolTags)
		}
	}
	if ctx.conf != nil {
		buildCtx.BuildTags = parseSourcePatchBuildTags(ctx.conf.BuildFlags)
	}
	return buildCtx
}

func cgoCompilationFlags(preambles []cgoPreamble, decls []cgoDecl) (cflags, cxxflags, ldflags []string) {
	cflags, cxxflags, ldflags = collectCgoFlags(decls)
	incDirs := make(map[string]none)
	for _, preamble := range preambles {
		dir, _ := filepath.Split(preamble.goFile)
		if _, ok := incDirs[dir]; !ok {
			incDirs[dir] = none{}
			cflags = append(cflags, "-I"+dir)
		}
	}
	return
}

func collectCgoSymbols(externs []string) map[string]string {
	cgoSymbols := make(map[string]string, len(externs))
	mallocFix := false
	for _, symbolName := range externs {
		lastPart := symbolName
		lastDot := strings.LastIndex(symbolName, ".")
		if lastDot != -1 {
			lastPart = symbolName[lastDot+1:]
		}
		if strings.HasPrefix(lastPart, "__cgo_") {
			// Func pointer vars are looked up in the Go package by their
			// qualified names, but emitted C globals must use bare _cgo_* names.
			cgoSymbols[symbolName] = lastPart
		} else if m := cgoExternRE.FindStringSubmatch(lastPart); len(m) > 0 {
			prefix := m[1] // _cgo_hash_(Cfunc|Cmacro)_
			name := m[3]   // remaining part
			cgoSymbols[lastPart] = name
			// fix missing _cgo_9113e32b6599_Cfunc__Cmalloc
			if !mallocFix && m[2] == "Cfunc" {
				mallocName := prefix + "_Cmalloc"
				cgoSymbols[mallocName] = "_Cmalloc"
				mallocFix = true
			}
		}
	}
	return cgoSymbols
}

// clangASTNode represents a node in clang's AST
type clangASTNode struct {
	Kind  string         `json:"kind"`
	Name  string         `json:"name,omitempty"`
	Inner []clangASTNode `json:"inner,omitempty"`
}

func genExternDeclsByClang(compiler *llclang.Cmd, pkg *aPackage, src string, cflags []string, cgoSymbols map[string]string, verbose bool) (string, error) {
	tmpSrc, err := os.CreateTemp("", "cgo-src-*.c")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpSrc.Name())
	if err := os.WriteFile(tmpSrc.Name(), []byte(src), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}
	symbolNames := make(map[string]bool)
	compiler.Verbose = compiler.Verbose || verbose
	if err := getFuncNames(compiler, tmpSrc.Name(), cflags, symbolNames); err != nil {
		return "", fmt.Errorf("failed to get func names: %v", err)
	}
	macroNames := make(map[string]bool)
	if err := getMacroNames(compiler, tmpSrc.Name(), cflags, macroNames); err != nil {
		return "", fmt.Errorf("failed to get macro names: %v", err)
	}

	b := strings.Builder{}
	var toRemove []string
	for cgoName, symbolName := range cgoSymbols {
		if strings.HasPrefix(symbolName, "__cgo_") {
			// These globals are generated address placeholders, not C storage.
			// Delete them after replacement so DWARF cannot retain a relocation
			// to their undefined symbols. Alternate cgo files may revisit them.
			toRemove = append(toRemove, cgoName)
			if pkg.LPkg.VarOf(cgoName) == nil {
				continue
			}
			gofuncName := strings.Replace(cgoName, ".__cgo_", ".", 1)
			cfuncName := symbolName[len("__cgo_"):]
			gofn := pkg.LPkg.FuncOf(gofuncName)
			// C takes the address of the public export, whose final-link wrapper
			// registers foreign threads. A same-named Go implementation (or a
			// private Go function shadowing a renamed export) must not bypass it.
			for _, exportName := range pkg.LPkg.ExportFuncs() {
				if exportName == cfuncName {
					gofn = pkg.LPkg.FuncOf(cfuncName)
					break
				}
			}
			if gofn != nil {
				pkg.LPkg.ReplaceVarWith(cgoName, gofn.Expr)
			} else {
				cfn := pkg.LPkg.NewFunc(cfuncName, types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.CDecl)
				pkg.LPkg.ReplaceVarWith(cgoName, cfn.Expr)
			}
		} else {
			usePtr := ""
			if symbolNames[symbolName] {
				usePtr = "*"
			} else if !macroNames[symbolName] {
				continue
			}
			/* template:
			typeof(fputs)* _cgo_1574167f3838_Cfunc_fputs;

			__attribute__((constructor))
			static void _init__cgo_1574167f3838_Cfunc_fputs() {
				_cgo_1574167f3838_Cfunc_fputs = fputs;
			}*/
			b.WriteString(fmt.Sprintf(`
typeof(%s)%s %s;

__attribute__((constructor))
static void _init_%s() {
	%s = %s;
}
`,
				symbolName, usePtr, cgoName,
				cgoName,
				cgoName, symbolName))
			toRemove = append(toRemove, cgoName)
		}
	}
	for _, funcName := range toRemove {
		delete(cgoSymbols, funcName)
	}
	return b.String(), nil
}

func getMacroNames(compiler *llclang.Cmd, file string, cflags []string, macroNames map[string]bool) error {
	args := append([]string{"-x", "c", "-dM", "-E"}, cflags...)
	args = append(args, file)
	var output bytes.Buffer
	compiler.Stdout = &output
	compiler.Stderr = nil
	if err := compiler.Compile(args...); err != nil {
		return err
	}
	for _, line := range strings.Split(output.String(), "\n") {
		if strings.HasPrefix(line, "#define ") {
			define := strings.TrimPrefix(line, "#define ")
			parts := strings.SplitN(define, " ", 2)
			if len(parts) > 1 {
				macroNames[parts[0]] = true
			}
		}
	}
	return nil
}

func getFuncNames(compiler *llclang.Cmd, file string, cflags []string, symbolNames map[string]bool) error {
	args := append([]string{"-x", "c", "-Xclang", "-ast-dump=json", "-fsyntax-only"}, cflags...)
	args = append(args, file)
	var output bytes.Buffer
	compiler.Stdout = &output
	compiler.Stderr = os.Stderr
	if err := compiler.Compile(args...); err != nil {
		dump := "dump failed"
		if tmpFile, err := os.CreateTemp("", "llgo-clang-ast-dump*.log"); err == nil {
			dump = "dump saved to " + tmpFile.Name()
			tmpFile.Write(output.Bytes())
			tmpFile.Close()
		}
		return fmt.Errorf("failed to run clang: %v, %s", err, dump)
	}
	var astRoot clangASTNode
	if err := json.Unmarshal(output.Bytes(), &astRoot); err != nil {
		dump := "dump failed"
		if tmpFile, err := os.CreateTemp("", "llgo-clang-ast-dump*.log"); err == nil {
			dump = "dump saved to " + tmpFile.Name()
			tmpFile.Write(output.Bytes())
			tmpFile.Close()
		}
		return fmt.Errorf("failed to unmarshal AST: %v, %s", err, dump)
	}

	extractFuncNames(&astRoot, symbolNames)
	return nil
}

func execCommandVerbose(commands commandEnv, verbose bool, name string, arg ...string) *exec.Cmd {
	if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", name, strings.Join(arg, " "))
	}
	cmd := exec.Command(name, arg...)
	return commands.configure(cmd)
}

func extractFuncNames(node *clangASTNode, funcNames map[string]bool) {
	for _, inner := range node.Inner {
		if inner.Kind == "FunctionDecl" && inner.Name != "" {
			funcNames[inner.Name] = true
		}
	}
}

func parseCgo_(buildCtx *build.Context, pkg *aPackage, files []*ast.File) (srcFiles []cgoSrcFile, preambles []cgoPreamble, cdecls []cgoDecl, err error) {
	return parseCgoWithCommandEnv(commandEnv{}, buildCtx, pkg, files)
}

func parseCgoWithCommandEnv(commands commandEnv, buildCtx *build.Context, pkg *aPackage, files []*ast.File) (srcFiles []cgoSrcFile, preambles []cgoPreamble, cdecls []cgoDecl, err error) {
	dirs := make(map[string]none)
	for _, file := range files {
		pos := pkg.Fset.Position(file.Name.NamePos)
		dir, _ := filepath.Split(pos.Filename)
		dirs[dir] = none{}
	}
	addFile := func(dir, match string, isCXX bool) error {
		fi, statErr := os.Stat(match)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil
			}
			return statErr
		}
		if fi.IsDir() {
			return nil
		}
		name := filepath.Base(match)
		switch {
		case strings.HasSuffix(name, "_test.c"),
			strings.HasSuffix(name, "_test.m"),
			strings.HasSuffix(name, "_test.cc"),
			strings.HasSuffix(name, "_test.cpp"),
			strings.HasSuffix(name, "_test.cxx"):
			return nil
		}
		if buildCtx != nil {
			matchFile, err := buildCtx.MatchFile(dir, name)
			if err != nil {
				return fmt.Errorf("match cgo file %s: %w", match, err)
			}
			if !matchFile {
				return nil
			}
		}
		srcFiles = append(srcFiles, cgoSrcFile{path: match, isCXX: isCXX})
		return nil
	}
	for dir := range dirs {
		for _, pattern := range []struct {
			glob  string
			isCXX bool
		}{
			{glob: "*.c"},
			{glob: "*.m"},
			{glob: "*.cc", isCXX: true},
			{glob: "*.cpp", isCXX: true},
			{glob: "*.cxx", isCXX: true},
		} {
			matches, err := filepath.Glob(filepath.Join(dir, pattern.glob))
			if err != nil {
				continue
			}
			for _, match := range matches {
				if err := addFile(dir, match, pattern.isCXX); err != nil {
					return nil, nil, nil, err
				}
			}
		}
	}

	preambles, cdecls, err = parseCgoFilePreambles(commands, buildCtx, pkg.Fset, files, "unsafe")
	return
}

func parseCgoFilePreambles(commands commandEnv, buildCtx *build.Context, fset *token.FileSet, files []*ast.File, importPath string) (preambles []cgoPreamble, decls []cgoDecl, err error) {
	for _, file := range files {
		for _, node := range file.Decls {
			decl, ok := node.(*ast.GenDecl)
			if !ok || decl.Tok != token.IMPORT {
				continue
			}
			for _, item := range decl.Specs {
				spec := item.(*ast.ImportSpec)
				if spec.Path.Value != strconv.Quote(importPath) {
					continue
				}
				doc := spec.Doc
				if importPath == "unsafe" {
					// Keep the rewritten-source path's declaration-only rule.
					// An unsafe spec comment cannot establish C ownership:
					// ordinary grouped imports may have Go documentation or
					// imports commented out by source patches attached here.
					if len(decl.Specs) != 1 {
						continue
					}
					doc = decl.Doc
				} else if doc == nil && len(decl.Specs) == 1 {
					doc = decl.Doc
				}
				if doc == nil {
					continue
				}
				preamble, flags, err := parseCgoPreambleWithCommandEnv(commands, buildCtx, fset.Position(doc.Pos()), doc.Text())
				if err != nil {
					return nil, nil, err
				}
				preambles = append(preambles, preamble)
				decls = append(decls, flags...)
			}
		}
	}
	return
}

func parseCgoPreamble(pos token.Position, text string) (preamble cgoPreamble, decls []cgoDecl, err error) {
	return parseCgoPreambleWithCommandEnv(commandEnv{}, nil, pos, text)
}

func parseCgoPreambleWithCommandEnv(commands commandEnv, buildCtx *build.Context, pos token.Position, text string) (preamble cgoPreamble, decls []cgoDecl, err error) {
	b := strings.Builder{}
	fline := pos.Line
	fname := pos.Filename
	b.WriteString(fmt.Sprintf("#line %d %q\n", fline, fname))

	for _, line := range strings.Split(text, "\n") {
		fline++
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#cgo ") {
			var cgoDecls []cgoDecl
			cgoDecls, err = parseCgoDeclWithCommandEnv(commands, buildCtx, line)
			if err != nil {
				return
			}
			// Original #cgo flags are package-relative even when the native
			// compiler runs in the invocation directory. pkg-config output
			// has its own path convention and must not be rewritten here.
			header, _, _ := strings.Cut(line, ":")
			fields := strings.Fields(header)
			if len(cgoDecls) != 0 && len(fields) != 0 {
				switch fields[len(fields)-1] {
				case "CPPFLAGS", "CFLAGS", "CXXFLAGS", "LDFLAGS":
					sourceDir, pathErr := filepath.Abs(filepath.Dir(resolvePath(commands.dir, fname)))
					if pathErr != nil {
						err = fmt.Errorf("resolve #cgo source directory: %w", pathErr)
						return
					}
					for i := range cgoDecls {
						normalizeCgoDirectivePaths(cgoDecls[i].cflags, sourceDir)
						normalizeCgoDirectivePaths(cgoDecls[i].cxxflags, sourceDir)
						normalizeCgoDirectivePaths(cgoDecls[i].ldflags, sourceDir)
					}
				}
			}
			decls = append(decls, cgoDecls...)
			b.WriteString(fmt.Sprintf("#line %d %q\n", fline, fname))
		} else {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	preamble = cgoPreamble{
		goFile: pos.Filename,
		src:    b.String(),
	}
	return
}

// Parse cgo directive like:
// #cgo pkg-config: python3
// #cgo windows CFLAGS: -IC:/Python312/include
// #cgo windows LDFLAGS: -LC:/Python312/libs -lpython312
// #cgo linux CPPFLAGS: -I/usr/lib/llvm-22/include -D_GNU_SOURCE
// #cgo CFLAGS: -I/usr/include/python3.12
// #cgo CXXFLAGS: -I/usr/include/c++/v1
// #cgo LDFLAGS: -L/usr/lib/python3.12/config-3.12-x86_64-linux-gnu -lpython3.12
func parseCgoDecl(line string) (cgoDecls []cgoDecl, err error) {
	return parseCgoDeclWithCommandEnv(commandEnv{}, nil, line)
}

func parseCgoDeclWithCommandEnv(commands commandEnv, buildCtx *build.Context, line string) (cgoDecls []cgoDecl, err error) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		err = fmt.Errorf("invalid cgo format: %v", line)
		return
	}

	decl := strings.TrimSpace(line[:idx])
	arg := strings.TrimSpace(line[idx+1:])

	// Split on first space to remove #cgo
	parts := strings.SplitN(decl, " ", 2)
	if len(parts) < 2 {
		err = fmt.Errorf("invalid cgo directive: %v", line)
		return
	}

	// Process remaining part
	remaining := strings.TrimSpace(parts[1])
	var tag, flag string

	// Split on last space to get flag
	if lastSpace := strings.LastIndex(remaining, " "); lastSpace != -1 {
		tag = strings.TrimSpace(remaining[:lastSpace])
		flag = strings.TrimSpace(remaining[lastSpace+1:])
	} else {
		flag = remaining
	}
	// Resolve only directives that apply to the requested target. In
	// particular, do not start pkg-config for an inactive directive.
	if tag != "" && buildCtx != nil && !buildtags.Match(buildCtx, tag) {
		return nil, nil
	}

	switch flag {
	case "pkg-config":
		pkgConfig := xenv.PkgConfigCommand(commands.dir, commands.environ)
		libsCmd := exec.Command(pkgConfig, "--libs", arg)
		commands.configure(libsCmd)
		ldflags, e := libsCmd.Output()
		if e != nil {
			err = fmt.Errorf("pkg-config: %v", e)
			return
		}
		flagsCmd := exec.Command(pkgConfig, "--cflags", arg)
		commands.configure(flagsCmd)
		cflags, e := flagsCmd.Output()
		if e != nil {
			err = fmt.Errorf("pkg-config: %v", e)
			return
		}
		cgoDecls = append(cgoDecls, cgoDecl{
			tag:     tag,
			cflags:  safesplit.SplitPkgConfigFlags(string(cflags)),
			ldflags: safesplit.SplitPkgConfigFlags(string(ldflags)),
		})
	case "CPPFLAGS", "CFLAGS", "CXXFLAGS", "LDFLAGS":
		args, parseErr := splitCgoDirectiveFlags(arg)
		if parseErr != nil {
			return nil, parseErr
		}
		parsed := cgoDecl{tag: tag}
		switch flag {
		case "CPPFLAGS", "CFLAGS":
			parsed.cflags = args
		case "CXXFLAGS":
			parsed.cxxflags = args
		case "LDFLAGS":
			parsed.ldflags = args
		}
		cgoDecls = append(cgoDecls, parsed)
	default:
		err = fmt.Errorf("unsupported cgo flag type: %s", flag)
	}
	return
}
