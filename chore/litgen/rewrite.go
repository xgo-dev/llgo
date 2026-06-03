/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/goplus/llgo/internal/llgen"
	"github.com/goplus/mod"
	"golang.org/x/mod/modfile"
)

type resolvedTarget struct {
	sourceFile string
	genTarget  string
	pkgDir     string
	modulePath string
	pkgPath    string
}

type irProgram struct {
	globals []irGlobal
	funcs   []irFunction
}

type irGlobal struct {
	symbol string
	line   string
}

type irFunction struct {
	symbol string
	lines  []string
}

var (
	defineQuotedRE  = regexp.MustCompile(`^define\b.* @"([^"]+)"\(`)
	definePlainRE   = regexp.MustCompile(`^define\b.* @([^\s(]+)\(`)
	globalQuotedRE  = regexp.MustCompile(`^@"([^"]+)"\s*=`)
	globalPlainRE   = regexp.MustCompile(`^@([A-Za-z0-9$._-]+)\s*=`)
	globalRefRE     = regexp.MustCompile(`@"([^"]+)"|@([A-Za-z0-9$._-]+)`)
	checkLineRE     = regexp.MustCompile(`^\s*//\s*CHECK(?:-[A-Z]+)?:`)
	debugMetaRE     = regexp.MustCompile(`, ![A-Za-z0-9_.-]+ ![0-9]+`)
	attrGroupTailRE = regexp.MustCompile(`\s+#\d+$`)
	numericNameRE   = regexp.MustCompile(`^\d+$`)
)

func generateFile(target resolvedTarget) error {
	data, err := os.ReadFile(target.sourceFile)
	if err != nil {
		return err
	}
	cleaned := stripCheckDirectives(string(data))
	ir, err := genIR(target.genTarget)
	if err != nil {
		return err
	}
	updated, err := rewriteSource(cleaned, target.sourceFile, target.pkgPath, target.modulePath, ir)
	if err != nil {
		return err
	}
	formatted, err := format.Source([]byte(updated))
	if err != nil {
		return fmt.Errorf("%s: gofmt failed: %w", target.sourceFile, err)
	}
	return writeFileAtomically(target.sourceFile, formatted, 0644)
}

func resolveTarget(sourceFile, genTarget string) (resolvedTarget, error) {
	pkgDir := filepath.Dir(sourceFile)
	root, goMod, err := mod.FindGoMod(pkgDir)
	if err != nil {
		return resolvedTarget{}, err
	}
	modulePath, err := readModulePath(goMod)
	if err != nil {
		return resolvedTarget{}, err
	}
	pkgPath, err := packagePath(modulePath, root, pkgDir)
	if err != nil {
		return resolvedTarget{}, err
	}
	return resolvedTarget{
		sourceFile: sourceFile,
		genTarget:  genTarget,
		pkgDir:     pkgDir,
		modulePath: modulePath,
		pkgPath:    pkgPath,
	}, nil
}

func genIR(target string) (ret string, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = fmt.Errorf("llgen failed for %s: %w", target, v)
			case string:
				err = fmt.Errorf("llgen failed for %s: %s", target, v)
			default:
				_, _ = os.Stderr.Write(debug.Stack())
				panic(r)
			}
		}
	}()
	return llgen.GenFrom(target), nil
}

func readModulePath(goMod string) (string, error) {
	data, err := os.ReadFile(goMod)
	if err != nil {
		return "", err
	}
	modulePath := modfile.ModulePath(data)
	if modulePath == "" {
		return "", fmt.Errorf("%s: module directive not found", goMod)
	}
	return modulePath, nil
}

func packagePath(modulePath, root, pkgDir string) (string, error) {
	rel, err := filepath.Rel(root, pkgDir)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return modulePath, nil
	}
	return path.Join(modulePath, filepath.ToSlash(rel)), nil
}

func rewriteSource(src, srcPath, pkgPath, modulePath, ir string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, src, parser.ParseComments)
	if err != nil {
		return "", err
	}
	prog := parseIR(ir)
	anchors, topPos := collectAnchors(src, fset, file)
	injections := make(map[int][]string)

	if globals := buildGlobalChecks(prog, modulePath); len(globals) != 0 {
		injections[topPos] = append(injections[topPos], formatDirectiveBlock(indentAt(src, topPos), globals))
	}
	eof := len(src)
	lastOffset := topPos
	for _, fn := range prog.funcs {
		if shouldSkipFunctionCheck(fn.symbol) {
			continue
		}
		lines := buildFunctionChecks(fn, modulePath)
		if len(lines) == 0 {
			continue
		}
		offset := eof
		if name, ok := trimPkgPrefix(fn.symbol, pkgPath); ok {
			if pos, found := anchors[name]; found {
				offset = pos
			}
		}
		if offset < lastOffset {
			offset = lastOffset
		}
		injections[offset] = append(injections[offset], formatDirectiveBlock(indentAt(src, offset), lines))
		lastOffset = offset
	}
	return applyInjections(src, injections), nil
}

func collectAnchors(src string, fset *token.FileSet, file *ast.File) (map[string]int, int) {
	anchors := make(map[string]int)
	counts := make(map[string]int)
	topPos := topInsertPos(src, fset, file)
	if initPos, ok := syntheticInitPos(src, fset, file); ok {
		anchors["init"] = initPos
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name := inPkgFuncName(d)
			anchors[name] = declInsertPos(src, fset, d.Pos(), d.Doc)
			collectFuncLitAnchors(src, fset, d.Body, name, anchors, counts)
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			collectFuncLitAnchors(src, fset, d, "init", anchors, counts)
		}
	}
	return anchors, topPos
}

func topInsertPos(src string, fset *token.FileSet, file *ast.File) int {
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		return lineStart(src, offsetOf(fset, decl.Pos()))
	}
	return len(src)
}

func syntheticInitPos(src string, fset *token.FileSet, file *ast.File) (int, bool) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "init" {
				return declInsertPos(src, fset, d.Pos(), d.Doc), true
			}
		case *ast.GenDecl:
			if d.Tok != token.IMPORT {
				return declInsertPos(src, fset, d.Pos(), d.Doc), true
			}
		}
	}
	return 0, false
}

func declInsertPos(src string, fset *token.FileSet, pos token.Pos, doc *ast.CommentGroup) int {
	if doc != nil {
		pos = doc.Pos()
	}
	return lineStart(src, offsetOf(fset, pos))
}

func collectFuncLitAnchors(src string, fset *token.FileSet, node ast.Node, parent string, anchors map[string]int, counts map[string]int) {
	if isNilNode(node) {
		return
	}
	var walk func(ast.Node, string)
	walk = func(root ast.Node, current string) {
		if isNilNode(root) {
			return
		}
		ast.Inspect(root, func(n ast.Node) bool {
			lit, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}
			counts[current]++
			name := fmt.Sprintf("%s$%d", current, counts[current])
			anchors[name] = lineStart(src, offsetOf(fset, lit.Pos()))
			walk(lit.Body, name)
			return false
		})
	}
	walk(node, parent)
}

func isNilNode(node ast.Node) bool {
	if node == nil {
		return true
	}
	v := reflect.ValueOf(node)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func inPkgFuncName(fn *ast.FuncDecl) string {
	name := fn.Name.Name
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return name
	}
	recv := fn.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		return "(*" + recvTypeName(star.X) + ")." + name
	}
	return recvTypeName(recv) + "." + name
}

func recvTypeName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return recvTypeName(v.X) + "." + v.Sel.Name
	case *ast.IndexExpr:
		return recvTypeName(v.X)
	case *ast.IndexListExpr:
		return recvTypeName(v.X)
	default:
		return ""
	}
}

func parseIR(ir string) irProgram {
	lines := splitIRLines(ir)
	var prog irProgram
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "define ") {
			j := i + 1
			for j < len(lines) {
				if lines[j] == "}" {
					j++
					break
				}
				j++
			}
			block := append([]string(nil), lines[i:j]...)
			prog.funcs = append(prog.funcs, irFunction{
				symbol: extractDefineSymbol(line),
				lines:  block,
			})
			i = j - 1
			continue
		}
		if symbol, ok := extractGlobalSymbol(line); ok {
			prog.globals = append(prog.globals, irGlobal{symbol: symbol, line: line})
		}
	}
	return prog
}

func splitIRLines(ir string) []string {
	ir = strings.ReplaceAll(ir, "\r\n", "\n")
	ir = strings.TrimSuffix(ir, "\n")
	if ir == "" {
		return nil
	}
	return strings.Split(ir, "\n")
}

func extractDefineSymbol(line string) string {
	if m := defineQuotedRE.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	if m := definePlainRE.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return ""
}

func extractGlobalSymbol(line string) (string, bool) {
	if m := globalQuotedRE.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	if m := globalPlainRE.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	return "", false
}

func buildGlobalChecks(prog irProgram, modulePath string) []string {
	defs := make(map[string]string, len(prog.globals))
	order := make([]string, 0, len(prog.globals))
	for _, g := range prog.globals {
		if !shouldCheckGlobal(g.symbol) {
			continue
		}
		defs[g.symbol] = g.line
		order = append(order, g.symbol)
	}
	if len(defs) == 0 {
		return nil
	}
	needed := make(map[string]bool)
	for _, fn := range prog.funcs {
		if shouldSkipFunctionCheck(fn.symbol) {
			continue
		}
		for _, line := range fn.lines[1:] {
			for _, ref := range collectRefs(line) {
				if _, ok := defs[ref]; ok {
					needed[ref] = true
				}
			}
		}
	}
	var lines []string
	for _, symbol := range order {
		if !needed[symbol] {
			continue
		}
		lines = append(lines, "// CHECK-LINE: "+generalizeIRLine(defs[symbol], modulePath))
	}
	return lines
}

func buildFunctionChecks(fn irFunction, modulePath string) []string {
	if len(fn.lines) == 0 {
		return nil
	}
	lines := make([]string, 0, len(fn.lines))
	lines = append(lines, "// CHECK-LABEL: "+generalizeDefineLine(fn.lines[0], modulePath))
	for _, line := range fn.lines[1:] {
		if strings.TrimSpace(line) == "" {
			lines = append(lines, "// CHECK-EMPTY:")
			continue
		}
		lines = append(lines, "// CHECK-NEXT: "+generalizeIRLine(line, modulePath))
	}
	return lines
}

func generalizeDefineLine(line, modulePath string) string {
	line = scrubIRLine(line)
	if idx := strings.LastIndex(line, " {"); idx >= 0 {
		head := attrGroupTailRE.ReplaceAllString(line[:idx], "")
		line = head + line[idx:]
	}
	return generalizeModulePath(line, modulePath)
}

func generalizeIRLine(line, modulePath string) string {
	return generalizeModulePath(scrubIRLine(line), modulePath)
}

func scrubIRLine(line string) string {
	line = debugMetaRE.ReplaceAllString(line, "")
	return strings.TrimRight(line, " \t")
}

func generalizeModulePath(line, modulePath string) string {
	if modulePath == "" {
		return line
	}
	var b strings.Builder
	start := 0
	inQuote := false
	for i := 0; i < len(line); i++ {
		if line[i] != '"' || isEscapedQuote(line, i) {
			continue
		}
		if !inQuote {
			b.WriteString(line[start : i+1])
			start = i + 1
			inQuote = true
			continue
		}
		b.WriteString(strings.ReplaceAll(line[start:i], modulePath, "{{.*}}"))
		b.WriteByte('"')
		start = i + 1
		inQuote = false
	}
	b.WriteString(line[start:])
	return b.String()
}

func isEscapedQuote(line string, idx int) bool {
	backslashes := 0
	for i := idx - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func shouldCheckGlobal(symbol string) bool {
	return numericNameRE.MatchString(symbol)
}

func collectRefs(line string) []string {
	matches := globalRefRE.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	refs := make([]string, 0, len(matches))
	for _, m := range matches {
		if m[1] != "" {
			refs = append(refs, m[1])
			continue
		}
		if m[2] != "" {
			refs = append(refs, m[2])
		}
	}
	return refs
}

func shouldSkipFunctionCheck(symbol string) bool {
	base := strings.TrimPrefix(symbol, "__llgo_stub.")
	return strings.HasSuffix(base, "/runtime/internal/runtime.memequal32") ||
		strings.HasSuffix(base, "/runtime/internal/runtime.memequalptr") ||
		strings.HasSuffix(base, "/runtime/internal/runtime.strequal")
}

func trimPkgPrefix(symbol, pkgPath string) (string, bool) {
	prefix := pkgPath + "."
	if strings.HasPrefix(symbol, prefix) {
		return strings.TrimPrefix(symbol, prefix), true
	}
	return "", false
}

func applyInjections(src string, injections map[int][]string) string {
	if len(injections) == 0 {
		return src
	}
	offsets := make([]int, 0, len(injections))
	for offset := range injections {
		offsets = append(offsets, offset)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(offsets)))
	out := src
	for _, offset := range offsets {
		block := strings.Join(injections[offset], "")
		out = out[:offset] + block + out[offset:]
	}
	return out
}

func formatDirectiveBlock(indent string, lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(indent)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}

func stripCheckDirectives(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.SplitAfter(src, "\n")
	if len(lines) == 0 {
		return src
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n")
		if checkLineRE.MatchString(trimmed) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "")
}

func lineStart(src string, offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(src) {
		return len(src)
	}
	for offset > 0 && src[offset-1] != '\n' {
		offset--
	}
	return offset
}

func indentAt(src string, offset int) string {
	if offset >= len(src) {
		return ""
	}
	start := lineStart(src, offset)
	end := start
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return src[start:end]
}

func offsetOf(fset *token.FileSet, pos token.Pos) int {
	return fset.PositionFor(pos, false).Offset
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".litgen-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
