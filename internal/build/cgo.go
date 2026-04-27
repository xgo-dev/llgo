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
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/goplus/llgo/internal/buildtags"
	llssa "github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/xtool/safesplit"
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

type pkgConfigResult struct {
	cflags  []string
	ldflags []string
}

var (
	pkgConfigCache  sync.Map
	pkgConfigOutput = func(arg ...string) ([]byte, error) {
		return exec.Command("pkg-config", arg...).Output()
	}
)

func clonePkgConfigResult(result pkgConfigResult) pkgConfigResult {
	return pkgConfigResult{
		cflags:  append([]string(nil), result.cflags...),
		ldflags: append([]string(nil), result.ldflags...),
	}
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

func buildCgo(ctx *context, pkg *aPackage, files []*ast.File, externs []string, verbose bool) (llfiles, cgoLdflags []string, err error) {
	var buildCtx *build.Context
	if _, ok := pkgSrcFilesFromMetadata(pkg); !ok {
		ctxVal := build.Default
		if ctx.buildConf.Goos != "" {
			ctxVal.GOOS = ctx.buildConf.Goos
		}
		if ctx.buildConf.Goarch != "" {
			ctxVal.GOARCH = ctx.buildConf.Goarch
		}
		ctxVal.BuildTags = parseSourcePatchBuildTags(ctx.conf.BuildFlags)
		buildCtx = &ctxVal
	}

	srcFiles, preambles, cdecls, err := parseCgo_(buildCtx, pkg, files)
	if err != nil {
		return
	}
	if len(srcFiles) == 0 && len(preambles) == 0 && len(cdecls) == 0 {
		return
	}
	tagUsed := make(map[string]bool)
	for _, cdecl := range cdecls {
		if cdecl.tag != "" {
			tagUsed[cdecl.tag] = false
		}
	}
	buildtags.CheckTags(ctx.conf.BuildFlags, tagUsed)
	cflags := []string{}
	cxxflags := []string{}
	ldflags := []string{}
	for _, cdecl := range cdecls {
		if cdecl.tag == "" || tagUsed[cdecl.tag] {
			if len(cdecl.cflags) > 0 {
				cflags = append(cflags, cdecl.cflags...)
			}
			if len(cdecl.cxxflags) > 0 {
				cxxflags = append(cxxflags, cdecl.cxxflags...)
			}
			if len(cdecl.ldflags) > 0 {
				ldflags = append(ldflags, cdecl.ldflags...)
			}
		}
	}
	incDirs := make(map[string]none)
	for _, preamble := range preambles {
		dir, _ := filepath.Split(preamble.goFile)
		if _, ok := incDirs[dir]; !ok {
			incDirs[dir] = none{}
			cflags = append(cflags, "-I"+dir)
		}
	}
	for _, src := range srcFiles {
		args := slices.Clone(cflags)
		if src.isCXX {
			args = append(args, cxxflags...)
		}
		clFile(ctx, args, src.path, pkg.ExportFile, pkg.PkgPath, func(linkFile string) {
			llfiles = append(llfiles, linkFile)
		}, verbose)
	}
	var cgoSymbols map[string]string
	if len(externs) > 0 {
		cgoSymbols = collectCgoSymbols(externs)
	}
	for _, preamble := range preambles {
		tmpFile, err := os.CreateTemp("", "-cgo-*.c")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp file: %v", err)
		}
		tmpName := tmpFile.Name()
		defer os.Remove(tmpName)
		code := cgoHeader + "\n\n" + preamble.src
		externDecls, err := cgoExternDecls(pkg, code, cflags, cgoSymbols, verbose)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate extern decls: %v", err)
		}
		if err = os.WriteFile(tmpName, []byte(code+"\n\n"+externDecls), 0644); err != nil {
			return nil, nil, fmt.Errorf("failed to write temp file: %v", err)
		}
		clFile(ctx, cflags, tmpName, pkg.ExportFile, pkg.PkgPath, func(linkFile string) {
			llfiles = append(llfiles, linkFile)
		}, verbose)
	}
	for _, ldflag := range ldflags {
		cgoLdflags = append(cgoLdflags, safesplit.SplitPkgConfigFlags(ldflag)...)
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
		} else if prefix, kind, name, ok := parseCgoExternSymbol(lastPart); ok {
			cgoSymbols[lastPart] = name
			// fix missing _cgo_9113e32b6599_Cfunc__Cmalloc
			if !mallocFix && kind == "Cfunc" {
				mallocName := prefix + "_Cmalloc"
				cgoSymbols[mallocName] = "_Cmalloc"
				mallocFix = true
			}
		}
	}
	return cgoSymbols
}

func parseCgoExternSymbol(lastPart string) (prefix, kind, name string, ok bool) {
	if !strings.HasPrefix(lastPart, "_cgo_") {
		return "", "", "", false
	}
	rest := lastPart[len("_cgo_"):]
	hashEnd := strings.IndexByte(rest, '_')
	if hashEnd <= 0 {
		return "", "", "", false
	}
	kindStart := len("_cgo_") + hashEnd + 1
	suffix := lastPart[kindStart:]
	switch {
	case strings.HasPrefix(suffix, "C2func_"):
		prefixEnd := kindStart + len("C2func_")
		return lastPart[:prefixEnd], "C2func", lastPart[prefixEnd:], true
	case strings.HasPrefix(suffix, "Cfunc_"):
		prefixEnd := kindStart + len("Cfunc_")
		return lastPart[:prefixEnd], "Cfunc", lastPart[prefixEnd:], true
	case strings.HasPrefix(suffix, "Cmacro_"):
		prefixEnd := kindStart + len("Cmacro_")
		return lastPart[:prefixEnd], "Cmacro", lastPart[prefixEnd:], true
	}
	return "", "", "", false
}

// clangASTNode represents a node in clang's AST
type clangASTNode struct {
	Kind  string         `json:"kind"`
	Name  string         `json:"name,omitempty"`
	Inner []clangASTNode `json:"inner,omitempty"`
}

var genExternDeclsByClangFunc = genExternDeclsByClang

func cgoExternDecls(pkg *aPackage, src string, cflags []string, cgoSymbols map[string]string, verbose bool) (string, error) {
	if len(cgoSymbols) == 0 {
		return "", nil
	}
	return genExternDeclsByClangFunc(pkg, src, cflags, cgoSymbols, verbose)
}

func genExternDeclsByClang(pkg *aPackage, src string, cflags []string, cgoSymbols map[string]string, verbose bool) (string, error) {
	tmpSrc, err := os.CreateTemp("", "cgo-src-*.c")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpSrc.Name())
	if err := os.WriteFile(tmpSrc.Name(), []byte(src), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}
	symbolNames := make(map[string]bool)
	if err := getFuncNames(tmpSrc.Name(), cflags, symbolNames, verbose); err != nil {
		return "", fmt.Errorf("failed to get func names: %v", err)
	}
	macroNames := make(map[string]bool)
	if err := getMacroNames(tmpSrc.Name(), cflags, macroNames, verbose); err != nil {
		return "", fmt.Errorf("failed to get macro names: %v", err)
	}

	b := strings.Builder{}
	var toRemove []string
	for cgoName, symbolName := range cgoSymbols {
		if strings.HasPrefix(symbolName, "__cgo_") {
			gofuncName := strings.Replace(cgoName, ".__cgo_", ".", 1)
			gofn := pkg.LPkg.FuncOf(gofuncName)
			cgoVar := pkg.LPkg.VarOf(cgoName)
			if gofn != nil {
				cgoVar.ReplaceAllUsesWith(gofn.Expr)
			} else {
				cfuncName := symbolName[len("__cgo_"):]
				cfn := pkg.LPkg.NewFunc(cfuncName, types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.InC)
				cgoVar.ReplaceAllUsesWith(cfn.Expr)
			}
			toRemove = append(toRemove, cgoName)
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

func getMacroNames(file string, cflags []string, macroNames map[string]bool, verbose bool) error {
	args := append([]string{"-dM", "-E"}, cflags...)
	args = append(args, file)
	cmd := execCommandVerbose(verbose, "clang", args...)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(output), "\n") {
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

func getFuncNames(file string, cflags []string, symbolNames map[string]bool, verbose bool) error {
	args := append([]string{"-Xclang", "-ast-dump=json", "-fsyntax-only"}, cflags...)
	args = append(args, file)
	cmd := execCommandVerbose(verbose, "clang", args...)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		dump := "dump failed"
		if tmpFile, err := os.CreateTemp("", "llgo-clang-ast-dump*.log"); err == nil {
			dump = "dump saved to " + tmpFile.Name()
			tmpFile.Write(output)
			tmpFile.Close()
		}
		return fmt.Errorf("failed to run clang: %v, %s", err, dump)
	}
	var astRoot clangASTNode
	if err := json.Unmarshal(output, &astRoot); err != nil {
		dump := "dump failed"
		if tmpFile, err := os.CreateTemp("", "llgo-clang-ast-dump*.log"); err == nil {
			dump = "dump saved to " + tmpFile.Name()
			tmpFile.Write(output)
			tmpFile.Close()
		}
		return fmt.Errorf("failed to unmarshal AST: %v, %s", err, dump)
	}

	extractFuncNames(&astRoot, symbolNames)
	return nil
}

func execCommandVerbose(verbose bool, name string, arg ...string) *exec.Cmd {
	if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", name, strings.Join(arg, " "))
	}
	return exec.Command(name, arg...)
}

func extractFuncNames(node *clangASTNode, funcNames map[string]bool) {
	for _, inner := range node.Inner {
		if inner.Kind == "FunctionDecl" && inner.Name != "" {
			funcNames[inner.Name] = true
		}
	}
}

func parseCgo_(buildCtx *build.Context, pkg *aPackage, files []*ast.File) (srcFiles []cgoSrcFile, preambles []cgoPreamble, cdecls []cgoDecl, err error) {
	if metadataFiles, ok := pkgSrcFilesFromMetadata(pkg); ok {
		srcFiles = metadataFiles
	} else {
		dirs := make(map[string]none)
		for _, file := range files {
			pos := pkg.Fset.Position(file.Name.NamePos)
			dir, _ := filepath.Split(pos.Filename)
			dirs[dir] = none{}
		}
		for dir := range dirs {
			matches, err := dirCgoSrcFiles(buildCtx, dir)
			if err != nil {
				return nil, nil, nil, err
			}
			srcFiles = append(srcFiles, matches...)
		}
	}

	for _, file := range files {
		if !fileImportsUnsafe(file) {
			continue
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if decl.Tok == token.IMPORT {
					if doc := decl.Doc; doc != nil && len(decl.Specs) == 1 {
						spec := decl.Specs[0].(*ast.ImportSpec)
						if spec.Path.Value == "\"unsafe\"" {
							pos := pkg.Fset.Position(doc.Pos())
							preamble, flags, err := parseCgoPreamble(pos, doc.Text())
							if err != nil {
								panic(err)
							}
							preambles = append(preambles, preamble)
							cdecls = append(cdecls, flags...)
						}
					}
				}
			}
		}
	}
	return
}

func fileImportsUnsafe(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == "\"unsafe\"" {
			return true
		}
	}
	return false
}

func pkgSrcFilesFromMetadata(pkg *aPackage) ([]cgoSrcFile, bool) {
	if len(pkg.CompiledGoFiles) == 0 || len(pkg.IgnoredFiles) > 0 {
		return nil, false
	}
	if len(pkg.OtherFiles) == 0 {
		return nil, true
	}
	var srcFiles []cgoSrcFile
	for i, file := range pkg.OtherFiles {
		isCXX, ok := cgoSourceKind(file)
		if !ok {
			continue
		}
		if srcFiles == nil {
			capHint := len(pkg.OtherFiles) - i
			if i > 0 && capHint > 4 {
				capHint = 4
			}
			srcFiles = make([]cgoSrcFile, 0, capHint)
		}
		srcFiles = append(srcFiles, cgoSrcFile{path: file, isCXX: isCXX})
	}
	if len(srcFiles) > 1 {
		sort.Slice(srcFiles, func(i, j int) bool { return srcFiles[i].path < srcFiles[j].path })
	}
	return srcFiles, true
}

func dirCgoSrcFiles(buildCtx *build.Context, dir string) ([]cgoSrcFile, error) {
	openDir := dir
	if openDir == "" {
		openDir = "."
	}
	f, err := os.Open(openDir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	var files []cgoSrcFile
	for _, name := range names {
		isCXX, ok := cgoSourceKind(name)
		if !ok {
			continue
		}
		path := name
		if dir != "" {
			path = dir + name
			if !os.IsPathSeparator(dir[len(dir)-1]) {
				path = dir + string(os.PathSeparator) + name
			}
		}
		fi, statErr := os.Stat(path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, statErr
		}
		if fi.IsDir() {
			continue
		}
		if buildCtx != nil {
			matchFile, err := buildCtx.MatchFile(openDir, name)
			if err != nil {
				return nil, fmt.Errorf("match cgo file %s: %w", path, err)
			}
			if !matchFile {
				continue
			}
		}
		files = append(files, cgoSrcFile{path: path, isCXX: isCXX})
	}
	if len(files) > 1 {
		sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	}
	return files, nil
}

func cgoSourceKind(name string) (isCXX bool, ok bool) {
	switch filepath.Ext(name) {
	case ".c":
		if strings.HasSuffix(name, "_test.c") {
			return false, false
		}
		return false, true
	case ".cc":
		if strings.HasSuffix(name, "_test.cc") {
			return false, false
		}
		return true, true
	case ".cpp":
		if strings.HasSuffix(name, "_test.cpp") {
			return false, false
		}
		return true, true
	case ".cxx":
		if strings.HasSuffix(name, "_test.cxx") {
			return false, false
		}
		return true, true
	}
	return false, false
}

func parseCgoPreamble(pos token.Position, text string) (preamble cgoPreamble, decls []cgoDecl, err error) {
	b := strings.Builder{}
	b.Grow(len(text) + 64)
	fline := pos.Line
	fname := pos.Filename
	writeCgoLineDirective(&b, fline, fname)

	for start := 0; start <= len(text); {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		fline++
		line := strings.TrimSpace(text[start:end])
		if strings.HasPrefix(line, "#cgo ") {
			var cgoDecls []cgoDecl
			cgoDecls, err = parseCgoDecl(line)
			if err != nil {
				return
			}
			decls = append(decls, cgoDecls...)
			writeCgoLineDirective(&b, fline, fname)
		} else {
			b.WriteString(line)
			b.WriteString("\n")
		}
		if end == len(text) {
			break
		}
		start = end + 1
	}
	preamble = cgoPreamble{
		goFile: pos.Filename,
		src:    b.String(),
	}
	return
}

func writeCgoLineDirective(b *strings.Builder, line int, file string) {
	b.WriteString("#line ")
	b.WriteString(strconv.Itoa(line))
	b.WriteByte(' ')
	b.WriteString(strconv.Quote(file))
	b.WriteByte('\n')
}

func cachedPkgConfig(arg string) (pkgConfigResult, error) {
	keyParts := []string{arg, os.Getenv("PATH")}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PKG_CONFIG") {
			keyParts = append(keyParts, env)
		}
	}
	sort.Strings(keyParts[2:])
	key := strings.Join(keyParts, "\x00")
	if cached, ok := pkgConfigCache.Load(key); ok {
		return clonePkgConfigResult(cached.(pkgConfigResult)), nil
	}
	ldflags, err := pkgConfigOutput("--libs", arg)
	if err != nil {
		return pkgConfigResult{}, fmt.Errorf("pkg-config: %v", err)
	}
	cflags, err := pkgConfigOutput("--cflags", arg)
	if err != nil {
		return pkgConfigResult{}, fmt.Errorf("pkg-config: %v", err)
	}
	result := pkgConfigResult{
		cflags:  safesplit.SplitPkgConfigFlags(string(cflags)),
		ldflags: safesplit.SplitPkgConfigFlags(string(ldflags)),
	}
	actual, _ := pkgConfigCache.LoadOrStore(key, result)
	return clonePkgConfigResult(actual.(pkgConfigResult)), nil
}

// Parse cgo directive like:
// #cgo pkg-config: python3
// #cgo windows CFLAGS: -IC:/Python312/include
// #cgo windows LDFLAGS: -LC:/Python312/libs -lpython312
// #cgo linux CPPFLAGS: -I/usr/lib/llvm-19/include -D_GNU_SOURCE
// #cgo CFLAGS: -I/usr/include/python3.12
// #cgo CXXFLAGS: -I/usr/include/c++/v1
// #cgo LDFLAGS: -L/usr/lib/python3.12/config-3.12-x86_64-linux-gnu -lpython3.12
func parseCgoDecl(line string) (cgoDecls []cgoDecl, err error) {
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

	switch flag {
	case "pkg-config":
		result, e := cachedPkgConfig(arg)
		if e != nil {
			err = e
			return
		}
		cgoDecls = append(cgoDecls, cgoDecl{
			tag:     tag,
			cflags:  result.cflags,
			ldflags: result.ldflags,
		})
	case "CPPFLAGS", "CFLAGS":
		cgoDecls = append(cgoDecls, cgoDecl{
			tag:    tag,
			cflags: safesplit.SplitPkgConfigFlags(arg),
		})
	case "CXXFLAGS":
		cgoDecls = append(cgoDecls, cgoDecl{
			tag:      tag,
			cxxflags: safesplit.SplitPkgConfigFlags(arg),
		})
	case "LDFLAGS":
		cgoDecls = append(cgoDecls, cgoDecl{
			tag:     tag,
			ldflags: safesplit.SplitPkgConfigFlags(arg),
		})
	default:
		err = fmt.Errorf("unsupported cgo flag type: %s", flag)
	}
	return
}
