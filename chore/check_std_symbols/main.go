// Package main implements check_std_symbols, a tool that verifies test coverage
// for all exported symbols in Go standard library packages.
//
// Usage:
//
//	go run ./chore/check_std_symbols [-modfile test.mod] -pkg math [-pkg strings ...]
//
// The tool compares exported symbols from a package (via go doc) with
// symbols used in test files, reporting any uncovered exported identifiers.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

type pkgSpec struct {
	pkgPath string
	testDir string
}

type pkgSpecs []pkgSpec

func (p *pkgSpecs) String() string {
	var parts []string
	for _, spec := range *p {
		parts = append(parts, fmt.Sprintf("%s=%s", spec.pkgPath, spec.testDir))
	}
	return strings.Join(parts, ", ")
}

func (p *pkgSpecs) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("invalid pkg specification %q (empty)", value)
	}

	var pkgPath, testDir string
	if strings.Contains(value, "=") {
		parts := strings.SplitN(value, "=", 2)
		pkgPath = strings.TrimSpace(parts[0])
		testDir = strings.TrimSpace(parts[1])
		if pkgPath == "" || testDir == "" {
			return fmt.Errorf("invalid pkg specification %q (empty component)", value)
		}
	} else {
		pkgPath = value
		testDir = filepath.Join("test", "std", value)
	}
	*p = append(*p, pkgSpec{pkgPath: pkgPath, testDir: testDir})
	return nil
}

type symbolKind string

type symbol struct {
	kind     symbolKind
	name     string
	receiver string
}

var pkgPathPattern = regexp.MustCompile(`^[A-Za-z0-9_/.\-]+$`)

const (
	kindConst  symbolKind = "const"
	kindVar    symbolKind = "var"
	kindFunc   symbolKind = "func"
	kindType   symbolKind = "type"
	kindMethod symbolKind = "method"
)

const (
	testingPkg        = "testing"
	testingCommonType = "common"
)

var testingCommonEmbeddedIn = []string{"T", "B", "F"}

func main() {
	var specs pkgSpecs
	var modfile string
	var verbose bool
	flag.Var(&specs, "pkg", "package coverage check in the form <import path>=<test dir>")
	flag.StringVar(&modfile, "modfile", "", "alternate go.mod used to load version-tagged tests")
	flag.BoolVar(&verbose, "v", false, "display coverage status for each exported symbol")
	flag.Parse()

	debugUsed := os.Getenv("DEBUG_USED_SYMBOLS") != ""

	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./chore/check_std_symbols [-modfile test.mod] -pkg math [-pkg strings ...]")
		os.Exit(2)
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to determine working directory: %v\n", err)
		os.Exit(2)
	}
	if modfile != "" {
		modfile, err = filepath.Abs(modfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve modfile: %v\n", err)
			os.Exit(2)
		}
	}

	var failures int

	for _, spec := range specs {
		if err := validatePkgPath(spec.pkgPath); err != nil {
			fmt.Fprintf(os.Stderr, "invalid package path %s: %v\n", spec.pkgPath, err)
			failures++
			continue
		}

		symbols, err := exportedSymbols(spec.pkgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to inspect package %s: %v\n", spec.pkgPath, err)
			failures++
			continue
		}

		testDir := filepath.Join(root, filepath.Clean(spec.testDir))
		if !pathWithin(root, testDir) {
			fmt.Fprintf(os.Stderr, "test directory %s escapes repository root\n", spec.testDir)
			failures++
			continue
		}
		used, err := usedSymbols(testDir, spec.pkgPath, modfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to inspect tests in %s: %v\n", spec.testDir, err)
			failures++
			continue
		}
		if debugUsed {
			fmt.Println("-- used symbols for", spec.pkgPath)
			var keys []string
			for k := range used {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Println("used:", k)
			}
		}

		missing := collectMissing(symbols, used, spec.pkgPath, verbose)
		if len(missing) > 0 {
			sort.Strings(missing)
			fmt.Fprintf(os.Stderr, "package %s missing coverage for %d exported identifiers:\n", spec.pkgPath, len(missing))
			for _, name := range missing {
				fmt.Fprintf(os.Stderr, "  - %s\n", name)
			}
			failures++
			continue
		}

		fmt.Printf("package %s: %d exported identifiers covered by tests in %s\n", spec.pkgPath, len(symbols), spec.testDir)
	}

	if failures > 0 {
		os.Exit(1)
	}
}

func symbolKey(sym symbol) string {
	if sym.kind == kindMethod {
		return fmt.Sprintf("%s.%s", sym.receiver, sym.name)
	}
	return sym.name
}

func collectMissing(symbols []symbol, used map[string]bool, pkgPath string, verbose bool) []string {
	var missing []string
	if verbose && len(symbols) > 0 {
		fmt.Printf("package %s symbols:\n", pkgPath)
	}
	for _, sym := range symbols {
		key := symbolKey(sym)

		// Skip intrinsic symbols that are compiled away
		if shouldSkipSymbol(pkgPath, key) {
			if verbose {
				fmt.Printf("  SKIP %s (intrinsic)\n", key)
			}
			continue
		}

		if used[key] {
			if verbose {
				fmt.Printf("  OK   %s\n", key)
			}
			continue
		}
		missing = append(missing, key)
		if verbose {
			fmt.Printf("  MISS %s\n", key)
		}
	}
	if verbose && len(symbols) > 0 {
		fmt.Println()
	}
	return missing
}

func shouldSkipSymbol(pkgPath, symbolName string) bool {
	// unsafe package has special intrinsic symbols that are compiled away
	if pkgPath == "unsafe" {
		intrinsics := map[string]bool{
			"Alignof":       true,
			"ArbitraryType": true,
			"IntegerType":   true,
			"Offsetof":      true,
			"Sizeof":        true,
			"String":        true,
			"StringData":    true,
		}
		return intrinsics[symbolName]
	}
	return false
}

func exportedSymbols(pkgPath string) ([]symbol, error) {
	out, err := runGoDoc(pkgPath)
	if err != nil {
		return nil, err
	}

	parser := newDocParser(out)
	if err := parser.parse(); err != nil {
		return nil, err
	}

	symbols := parser.symbols()
	sort.Slice(symbols, func(i, j int) bool {
		return symbolKey(symbols[i]) < symbolKey(symbols[j])
	})
	return symbols, nil
}

func exportedName(name string) bool {
	if name == "" {
		return false
	}
	r := rune(name[0])
	return unicode.IsUpper(r)
}

func parseIdentifier(input string) string {
	for i, r := range input {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			if i == 0 {
				return ""
			}
			return input[:i]
		}
	}
	return input
}

func receiverTypeName(recv string) string {
	fields := strings.Fields(recv)
	if len(fields) == 0 {
		return ""
	}
	typ := fields[len(fields)-1]
	typ = strings.TrimPrefix(typ, "*")
	return parseIdentifier(typ)
}

func usedSymbols(testDir, targetPkg, modfile string) (map[string]bool, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:   testDir,
		Tests: true,
	}
	if modfile != "" {
		cfg.BuildFlags = []string{"-modfile=" + modfile}
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, errors.New("packages.Load reported errors")
	}

	dirAbs, err := filepath.Abs(testDir)
	if err != nil {
		return nil, fmt.Errorf("resolve test directory: %w", err)
	}

	if err := rejectDotImports(pkgs, targetPkg, dirAbs); err != nil {
		return nil, err
	}

	used := make(map[string]bool)

	for _, pkg := range pkgs {
		fset := pkg.Fset
		if pkg.TypesInfo == nil {
			continue
		}
		for ident, obj := range pkg.TypesInfo.Uses {
			if obj == nil {
				continue
			}
			if !isIdentifierInDir(fset, ident, dirAbs) {
				continue
			}
			markIfFromTarget(used, obj, targetPkg)
		}
		for selExpr, sel := range pkg.TypesInfo.Selections {
			if !isSelectorInDir(fset, selExpr, dirAbs) {
				continue
			}
			markSelectionUsage(used, sel, targetPkg)
		}
	}

	return used, nil
}

func rejectDotImports(pkgs []*packages.Package, targetPkg, dirAbs string) error {
	for _, pkg := range pkgs {
		if pkg.Fset == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, imp := range file.Imports {
				if imp.Name == nil || imp.Name.Name != "." {
					continue
				}
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if importPath != targetPkg {
					continue
				}
				pos := pkg.Fset.PositionFor(imp.Pos(), false)
				if pos.Filename == "" {
					continue
				}
				if !pathWithinAbs(dirAbs, pos.Filename) || !strings.HasSuffix(pos.Filename, "_test.go") {
					continue
				}
				return fmt.Errorf("dot imports of %s are not supported (found in %s)", targetPkg, pos.Filename)
			}
		}
	}
	return nil
}

func isIdentifierInDir(fset *token.FileSet, ident *ast.Ident, dirAbs string) bool {
	pos := fset.PositionFor(ident.Pos(), false)
	if pos.Filename == "" {
		return false
	}
	if !pathWithinAbs(dirAbs, pos.Filename) {
		return false
	}
	return strings.HasSuffix(pos.Filename, "_test.go")
}

func isSelectorInDir(fset *token.FileSet, sel *ast.SelectorExpr, dirAbs string) bool {
	if sel == nil {
		return false
	}
	pos := fset.PositionFor(sel.Sel.Pos(), false)
	if pos.Filename == "" {
		return false
	}
	if !pathWithinAbs(dirAbs, pos.Filename) {
		return false
	}
	return strings.HasSuffix(pos.Filename, "_test.go")
}

// markIfFromTarget marks obj as used if it belongs to targetPkg.
// For methods, it handles both direct methods and methods from embedded types,
// marking both the method and its receiver type.
func markIfFromTarget(used map[string]bool, obj types.Object, targetPkg string) {
	pkg := obj.Pkg()
	if pkg == nil {
		// For methods, package is obtained from the receiver type.
		if fn, ok := obj.(*types.Func); ok {
			markMethodUsage(used, fn, targetPkg)
		}
		return
	}

	// Check if it's a method (even if from different package)
	if fn, ok := obj.(*types.Func); ok {
		sig, isSig := fn.Type().(*types.Signature)
		if isSig && sig.Recv() != nil {
			// Try to find which type from targetPkg this method belongs to
			// This handles embedded methods
			recv := sig.Recv().Type()
			recvNames := findReceiversInPackage(recv, fn.Name(), targetPkg)
			for _, recvName := range recvNames {
				used[recvName] = true
				used[fmt.Sprintf("%s.%s", recvName, fn.Name())] = true
			}
			if len(recvNames) > 0 {
				return
			}
		}
	}

	if pkg.Path() != targetPkg {
		return
	}
	switch obj := obj.(type) {
	case *types.Const:
		used[obj.Name()] = true
	case *types.Var:
		if obj.IsField() {
			return
		}
		used[obj.Name()] = true
	case *types.TypeName:
		used[obj.Name()] = true
	case *types.Func:
		sig := obj.Type().(*types.Signature)
		if sig.Recv() == nil {
			used[obj.Name()] = true
		} else {
			recv := receiverFromType(sig.Recv().Type(), targetPkg)
			if recv != "" {
				used[recv] = true
				used[fmt.Sprintf("%s.%s", recv, obj.Name())] = true
			}
		}
	}
}

// findReceiversInPackage finds all types in targetPkg that have the given method
// This handles both direct methods and methods from embedded types
func findReceiversInPackage(recvType types.Type, methodName string, targetPkg string) []string {
	var result []string

	// Get the actual receiver type (unwrap pointer)
	actualRecv := recvType
	if ptr, ok := recvType.(*types.Pointer); ok {
		actualRecv = ptr.Elem()
	}

	named, ok := actualRecv.(*types.Named)
	if !ok {
		return nil
	}

	// Check if the method exists on types from targetPkg
	// We need to check the package that defines the receiver
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return nil
	}

	recvPkg := obj.Pkg().Path()

	// If the receiver is directly from targetPkg
	if recvPkg == targetPkg {
		recvName := obj.Name()
		// Special case: testing.common is embedded in T, B, and F
		if recvPkg == testingPkg && recvName == testingCommonType {
			result = append(result, testingCommonEmbeddedIn...)
		}
	}

	// Otherwise, the method might be embedded in types from targetPkg
	// We need to load the target package and check which types embed this receiver
	// For now, let's check if it's an internal package of targetPkg
	if strings.HasPrefix(recvPkg, targetPkg+"/") || strings.HasSuffix(recvPkg, "/"+targetPkg) {
		// It's an internal type, check which exported types from targetPkg might use it
		// This is a simplified check for nested packages
		recvName := obj.Name()
		if exportedName(recvName) {
			result = append(result, recvName)
		}
	}

	return result
}

func markMethodUsage(used map[string]bool, fn *types.Func, targetPkg string) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return
	}
	recv := receiverFromType(sig.Recv().Type(), targetPkg)
	if recv == "" {
		return
	}
	obj := fn
	// Accept methods from target package or builtin (pkg == nil)
	if obj.Pkg() != nil && obj.Pkg().Path() != targetPkg {
		return
	}
	// Also mark usage if receiver type is from target package, even if method pkg is nil
	if obj.Pkg() == nil {
		// For methods with nil package, verify the receiver is from target package
		recvObj := getReceiverTypeObj(sig.Recv().Type())
		if recvObj == nil || recvObj.Pkg() == nil || recvObj.Pkg().Path() != targetPkg {
			return
		}
	}
	used[recv] = true
	used[fmt.Sprintf("%s.%s", recv, fn.Name())] = true
}

func getReceiverTypeObj(t types.Type) *types.TypeName {
	switch tt := t.(type) {
	case *types.Pointer:
		return getReceiverTypeObj(tt.Elem())
	case *types.Named:
		return tt.Obj()
	}
	return nil
}

func markSelectionUsage(used map[string]bool, sel *types.Selection, targetPkg string) {
	obj := sel.Obj()
	if obj == nil {
		return
	}
	recvName := typeNameFromType(sel.Recv(), targetPkg)
	if recvName == "" {
		return
	}
	used[recvName] = true
	used[fmt.Sprintf("%s.%s", recvName, obj.Name())] = true
}

func typeNameFromType(t types.Type, targetPkg string) string {
	switch tt := t.(type) {
	case *types.Pointer:
		return typeNameFromType(tt.Elem(), targetPkg)
	case *types.Named:
		if tt.Obj() != nil && tt.Obj().Pkg() != nil && tt.Obj().Pkg().Path() == targetPkg {
			return tt.Obj().Name()
		}
	}
	return ""
}

func receiverFromType(t types.Type, targetPkg string) string {
	switch tt := t.(type) {
	case *types.Pointer:
		return receiverFromType(tt.Elem(), targetPkg)
	case *types.Named:
		if tt.Obj() != nil && tt.Obj().Pkg() != nil && tt.Obj().Pkg().Path() == targetPkg {
			return tt.Obj().Name()
		}
	}
	return ""
}

func validatePkgPath(path string) error {
	if !pkgPathPattern.MatchString(path) {
		return fmt.Errorf("package path must match pattern %s", pkgPathPattern.String())
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("package path cannot contain '..'")
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("package path cannot be absolute")
	}
	return nil
}

func pathWithin(base, target string) bool {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	return pathWithinAbs(baseAbs, target)
}

func pathWithinAbs(baseAbs, target string) bool {
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func runGoDoc(pkgPath string) ([]byte, error) {
	cmd := exec.Command("go", "doc", "-all", pkgPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go doc %q: %w\n%s", pkgPath, err, out)
	}
	return out, nil
}

type docSection int

const (
	sectionNone docSection = iota
	sectionConst
	sectionVar
	sectionFunc
	sectionType
)

type docParser struct {
	scanner     *bufio.Scanner
	section     docSection
	inConst     bool
	inVar       bool
	seen        map[string]struct{}
	accumulated []symbol
}

func newDocParser(out []byte) *docParser {
	return &docParser{
		scanner: bufio.NewScanner(bytes.NewReader(out)),
		section: sectionNone,
		seen:    make(map[string]struct{}),
	}
}

func (p *docParser) parse() error {
	for p.scanner.Scan() {
		raw := p.scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if p.switchSection(line) {
			continue
		}
		switch p.section {
		case sectionConst:
			p.handleConst(line)
		case sectionVar:
			p.handleVar(line)
		case sectionFunc:
			p.handleFunc(line, raw)
		case sectionType:
			p.handleType(line, raw)
		}
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("scan go doc output: %w", err)
	}
	return nil
}

func (p *docParser) switchSection(line string) bool {
	switch line {
	case "CONSTANTS":
		p.section = sectionConst
		p.inConst = false
		p.inVar = false
		return true
	case "VARIABLES":
		p.section = sectionVar
		p.inConst = false
		p.inVar = false
		return true
	case "FUNCTIONS":
		p.section = sectionFunc
		p.inConst = false
		p.inVar = false
		return true
	case "TYPES":
		p.section = sectionType
		p.inConst = false
		p.inVar = false
		return true
	}
	return false
}

func (p *docParser) handleConst(line string) {
	switch {
	case strings.HasPrefix(line, "const ("):
		p.inConst = true
	case line == ")":
		p.inConst = false
	case strings.HasPrefix(line, "const "):
		name := parseIdentifier(strings.TrimSpace(strings.TrimPrefix(line, "const ")))
		p.addSymbol(symbol{kind: kindConst, name: name})
	case p.inConst:
		name := parseIdentifier(line)
		p.addSymbol(symbol{kind: kindConst, name: name})
	}
}

func (p *docParser) handleVar(line string) {
	switch {
	case strings.HasPrefix(line, "var ("):
		p.inVar = true
	case line == ")":
		p.inVar = false
	case strings.HasPrefix(line, "var "):
		name := parseIdentifier(strings.TrimSpace(strings.TrimPrefix(line, "var ")))
		p.addSymbol(symbol{kind: kindVar, name: name})
	case p.inVar:
		name := parseIdentifier(line)
		p.addSymbol(symbol{kind: kindVar, name: name})
	}
}

func (p *docParser) handleFunc(line, raw string) {
	if !strings.HasPrefix(line, "func ") {
		return
	}
	if len(raw) > 0 {
		if first := raw[0]; first == ' ' || first == '\t' {
			return
		}
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "func "))
	if strings.HasPrefix(rest, "(") {
		return
	}
	name := parseIdentifier(rest)
	p.addSymbol(symbol{kind: kindFunc, name: name})
}

func (p *docParser) handleType(line, raw string) {
	switch {
	case strings.HasPrefix(line, "type "):
		// Only parse type definitions that are not indented (not in doc comments)
		if len(raw) > 0 {
			if first := raw[0]; first == ' ' || first == '\t' {
				return
			}
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "type "))
		name := parseIdentifier(rest)
		p.addSymbol(symbol{kind: kindType, name: name})
	case strings.HasPrefix(line, "func ("):
		if len(raw) > 0 {
			if first := raw[0]; first == ' ' || first == '\t' {
				return
			}
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "func "))
		recvEnd := strings.Index(rest, ")")
		if recvEnd == -1 {
			return
		}
		recvPart := rest[1:recvEnd]
		methodPart := strings.TrimSpace(rest[recvEnd+1:])
		methodName := parseIdentifier(methodPart)
		if methodName == "" {
			return
		}
		receiver := receiverTypeName(recvPart)
		if receiver == "" {
			return
		}
		p.addSymbol(symbol{kind: kindMethod, name: methodName, receiver: receiver})
	}
}

func (p *docParser) addSymbol(sym symbol) {
	if sym.name == "" {
		return
	}
	if sym.kind == kindMethod && sym.receiver == "" {
		return
	}
	if sym.kind != kindMethod && !exportedName(sym.name) {
		return
	}
	if sym.kind == kindMethod {
		if !exportedName(sym.name) || !exportedName(sym.receiver) {
			return
		}
	}
	key := symbolKey(sym)
	if key == "" {
		return
	}
	if _, exists := p.seen[key]; exists {
		return
	}
	p.seen[key] = struct{}{}
	p.accumulated = append(p.accumulated, sym)
}

func (p *docParser) symbols() []symbol {
	return p.accumulated
}
