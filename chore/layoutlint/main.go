// Command layoutlint checks field offsets against the Go compiler using LLDB.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goplus/gogen"
	"golang.org/x/tools/go/packages"
)

type wrapperCall struct {
	owner, field types.Type
	read, write  *gogen.Func
}

func main() {
	var root, output string
	var tests bool
	flag.StringVar(&root, "root", ".", "working directory for input package patterns or Go files")
	flag.StringVar(&output, "output", "", "new directory for source, Go offsets and LLDB reports (default: temporary directory)")
	flag.BoolVar(&tests, "test", false, "include types declared in test files")
	flag.Parse()
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	if err := run(root, output, tests, patterns); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, output string, tests bool, patterns []string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if output == "" {
		output, err = os.MkdirTemp("", "layoutlint-wrappers-*")
	} else {
		output, err = filepath.Abs(output)
		if err == nil {
			err = os.Mkdir(output, 0755)
		}
	}
	if err != nil {
		return err
	}
	fmt.Println("Artifacts:", output)
	baselineDirectory, err := os.MkdirTemp("", "layoutlint-go-dependencies-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(baselineDirectory)
	dependencies := make(map[string]string)

	// Listing does not type-check the whole input set. A broken dependency in
	// one package must not prevent the remaining packages from being checked.
	listed, err := packages.Load(&packages.Config{Dir: root, Mode: packages.NeedName | packages.NeedFiles}, patterns...)
	if err != nil {
		return err
	}
	if len(listed) == 0 {
		return fmt.Errorf("no packages matched %q", patterns)
	}
	failed := false
	seen := make(map[string]bool)
	for _, input := range listed {
		if seen[input.PkgPath] {
			continue
		}
		seen[input.PkgPath] = true
		selected := []string{input.PkgPath}
		if input.PkgPath == "command-line-arguments" {
			selected = patterns
		}
		loaded, loadErr := packages.Load(&packages.Config{
			Dir: root, Tests: tests,
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedForTest | packages.NeedModule,
		}, selected...)
		if loadErr == nil && packages.PrintErrors(loaded) != 0 {
			loadErr = fmt.Errorf("input package could not be loaded")
		}
		if loadErr == nil && len(loaded) == 0 {
			loadErr = fmt.Errorf("no packages matched %q", selected)
		}
		if loadErr != nil {
			failed = true
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", input.PkgPath, loadErr)
			continue
		}
		for _, pkg := range loaded {
			if strings.HasSuffix(pkg.PkgPath, ".test") {
				continue
			}
			// packages.Load returns both the ordinary and test-augmented package.
			if tests && pkg.ID == pkg.PkgPath && slices.ContainsFunc(loaded, func(other *packages.Package) bool {
				return other.PkgPath == pkg.PkgPath && other.ID != pkg.ID
			}) {
				continue
			}
			if len(pkg.Syntax) == 0 {
				continue
			}
			if err := checkPackage(root, output, baselineDirectory, tests, selected, pkg, dependencies); err != nil {
				failed = true
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", pkg.PkgPath, err)
			}
		}
	}
	if failed {
		return fmt.Errorf("layout checks failed; see %s", output)
	}
	return nil
}

// checkPackage owns one package's generation, observation and failure artifacts.
func checkPackage(root, output, baselineDirectory string, tests bool, patterns []string, pkg *packages.Package, dependencies map[string]string) error {
	var generated *gogen.Package
	var probe *gogen.Func
	var checks []fieldCheck
	var calls []wrapperCall
	metadata, err := discoverSubjects(pkg)
	if err != nil {
		return err
	}
	for subjectIndex, subject := range metadata {
		st := types.Unalias(subject.typ).Underlying().(*types.Struct)
		if generated == nil {
			generated = gogen.NewPackage(pkg.PkgPath, pkg.Name, &gogen.Config{Types: pkg.Types, Fset: pkg.Fset})
			if pkg.Types.Scope().Lookup("layoutlintCheckpoint") != nil {
				return fmt.Errorf("%s: layoutlintCheckpoint already exists", pkg.PkgPath)
			}
			base := generated.NewParam(token.NoPos, "base", types.Typ[types.UnsafePointer], false)
			address := generated.NewParam(token.NoPos, "address", types.Typ[types.UnsafePointer], false)
			id := generated.NewParam(token.NoPos, "id", types.Typ[types.Uintptr], false)
			probe = generated.NewFunc(nil, "layoutlintCheckpoint", types.NewTuple(base, address, id), nil, false)
			probe.SetComments(generated, &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}})
			// Printing keeps the primitive arguments observable. LLDB
			// reads the entry registers, never these printed values.
			probe.BodyStart(generated).Val(types.Universe.Lookup("println")).Val("layoutlint").Val(base).Val(address).Val(id).Call(4).EndStmt().End()
		}
		owner := types.NewPointer(subject.typ)
		for index := 0; index < st.NumFields(); index++ {
			field := st.Field(index)
			if field.Name() == "_" || !field.Exported() && field.Pkg() != nil && field.Pkg() != pkg.Types {
				fmt.Printf("INFO %s.%s: no direct field access (%q, line %d)\n", subject.name, field.Name(), subject.file, subject.line)
				continue
			}
			switch types.Unalias(field.Type()).Underlying().(type) {
			case *types.Array, *types.Struct:
				// Inline aggregates use the address observations below. Passing
				// a field such as [1 << 21]byte by value adds an unverified copy
				// that can expand into millions of backend operations.
				continue
			}
			call := wrapperCall{owner: subject.typ, field: field.Type()}
			kind := types.Invalid
			if basic, ok := field.Type().Underlying().(*types.Basic); ok && basic.Info()&(types.IsBoolean|types.IsNumeric) != 0 {
				kind = basic.Kind()
			}
			for _, write := range []bool{false, true} {
				operation := "Read"
				if write {
					operation = "Write"
				}
				name := "layout" + operation + "_" + subject.identifier + "_" + field.Name()
				if pkg.Types.Scope().Lookup(name) != nil {
					return fmt.Errorf("%s: generated name %s already exists", subject.name, name)
				}
				p := generated.NewParam(token.NoPos, "p", owner, false)
				params := types.NewTuple(p)
				var results *types.Tuple
				var v *types.Var
				if write {
					v = generated.NewParam(token.NoPos, "v", field.Type(), false)
					params = types.NewTuple(p, v)
				} else {
					result := generated.NewParam(token.NoPos, "", field.Type(), false)
					results = types.NewTuple(result)
				}
				fn := generated.NewFunc(nil, name, params, results, false)
				fn.SetComments(generated, &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}})
				cb := fn.BodyStart(generated)
				// Pass the actual pointer that this access will dereference.
				// Neither the wrapper nor the probe computes an offset.
				cb.DefineVarStart(token.NoPos, "field").Val(&gogen.Element{
					Val:  &ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: ast.NewIdent("p"), Sel: ast.NewIdent(field.Name())}},
					Type: types.NewPointer(field.Type()),
				}).EndInit(1)
				cb.Val(probe).Typ(types.Typ[types.UnsafePointer]).Val(p).Call(1).
					Typ(types.Typ[types.UnsafePointer]).VarVal("field").Call(1).Val(2 * len(checks)).Call(3).EndStmt()
				if write {
					cb.VarVal("field").ElemRef().Val(v).Assign(1)
					if kind != types.Invalid {
						cb.Val(probe).Typ(types.Typ[types.UnsafePointer]).Val(p).Call(1).
							Typ(types.Typ[types.UnsafePointer]).VarVal("field").Call(1).Val(2*len(checks) + 1).Call(3).EndStmt()
					}
					cb.End()
					call.write = fn
				} else {
					if kind != types.Invalid {
						cb.DefineVarStart(token.NoPos, "value").VarVal("field").Elem().EndInit(1)
						cb.Val(probe).Typ(types.Typ[types.UnsafePointer]).Val(p).Call(1).
							Typ(types.Typ[types.UnsafePointer]).VarVal("value").UnaryOp(token.AND).Call(1).Val(2*len(checks) + 1).Call(3).EndStmt()
						cb.VarVal("value").Return(1).End()
					} else {
						cb.VarVal("field").Elem().Return(1).End()
					}
					call.read = fn
				}
				location := pkg.Fset.PositionFor(field.Pos(), false)
				checks = append(checks, fieldCheck{owner: subjectIndex, typeName: subject.name, field: field.Name(), operation: strings.ToLower(operation), wrapper: name, file: location.Filename, line: location.Line, kind: kind})
			}
			calls = append(calls, call)
		}
	}
	if len(metadata) == 0 {
		fmt.Printf("SKIP %s: no eligible package-level struct types\n", pkg.PkgPath)
		return nil
	}
	var nested []nestedCheck
	var walk func(int, types.Type, []layoutStep)
	walk = func(owner int, typ types.Type, path []layoutStep) {
		// Walk inline storage only. Pointer edges end the traversal, so a
		// recursive named type cannot turn this into a runtime object walk.
		switch typ := types.Unalias(typ).Underlying().(type) {
		case *types.Struct:
			for i := 0; i < typ.NumFields(); i++ {
				field := typ.Field(i)
				if field.Name() == "_" || !field.Exported() && field.Pkg() != nil && field.Pkg() != pkg.Types {
					if len(path) != 0 {
						location := pkg.Fset.PositionFor(field.Pos(), true)
						fmt.Printf("INFO %s.%s: no direct access in the input package (%q, line %d)\n", metadata[owner].name, field.Name(), location.Filename, location.Line)
					}
					continue
				}
				child := append(slices.Clone(path), layoutStep{field: field})
				observe := len(path) != 0
				switch types.Unalias(field.Type()).Underlying().(type) {
				case *types.Array, *types.Struct:
					// Direct aggregate fields have no value wrappers. Measure
					// their placement before descending into inline storage.
					observe = true
				}
				if observe {
					nested = append(nested, nestedCheck{owner: owner, path: child})
				}
				walk(owner, field.Type(), child)
			}
		case *types.Array:
			elem := types.Unalias(typ.Elem()).Underlying()
			for {
				array, ok := elem.(*types.Array)
				if !ok {
					break
				}
				elem = types.Unalias(array.Elem()).Underlying()
			}
			if _, ok := elem.(*types.Struct); !ok {
				return
			}
			for index := int64(0); index < min(typ.Len(), 2); index++ {
				child := append(slices.Clone(path), layoutStep{index: index})
				nested = append(nested, nestedCheck{owner: owner, path: child})
				walk(owner, typ.Elem(), child)
			}
		}
	}
	for i, check := range metadata {
		walk(i, check.typ, nil)
	}
	directory := filepath.Join(output, filepath.FromSlash(pkg.PkgPath))
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	layouts, goMetadata, err := goOffsets(root, directory, baselineDirectory, pkg, tests, patterns, checks, metadata, nested, dependencies, false)
	if err != nil {
		return err
	}
	private, err := preparePrivateLayouts(root, directory, baselineDirectory, pkg, metadata, dependencies)
	if err != nil {
		return err
	}
	var sizes []fieldSizeCheck
	// Share a physical stride observation between identical field types.
	// For example, point.x and point.y both use the same [2]int probe.
	fieldSize := func(typ types.Type, width int64) int {
		index := slices.IndexFunc(sizes, func(check fieldSizeCheck) bool { return types.Identical(check.typ, typ) })
		if index == -1 {
			index = len(sizes)
			sizes = append(sizes, fieldSizeCheck{typ: typ, width: width})
		}
		return index
	}
	for i, call := range calls {
		index := fieldSize(call.field, layouts[2*i].width)
		checks[2*i].sizeIndex, checks[2*i+1].sizeIndex = index, index
	}
	for i := range nested {
		check := &nested[i]
		typ := metadata[check.owner].typ
		for _, step := range check.path {
			if step.field != nil {
				typ = step.field.Type()
			} else {
				typ = types.Unalias(typ).Underlying().(*types.Array).Elem()
			}
		}
		check.sizeIndex = fieldSize(typ, check.width)
	}
	// The init runs a fixed set of accesses, then exits before the original
	// main/tests. Scalar accesses run at both Go-legal address cases.
	exit := generated.Import("os").Ref("Exit")
	driver := generated.NewFunc(nil, "init", nil, nil, false).BodyStart(generated)
	for i, check := range metadata {
		// A typed nil pointer keeps *T and its Elem descriptor without
		// constructing T (which may contain a large array). Only the
		// interface's type pointer is passed to the existing checkpoint.
		keep := fmt.Sprintf("metadataValue%d", i)
		descriptor := fmt.Sprintf("metadataPointer%d", i)
		driver.NewVarStart(types.NewInterfaceType(nil, nil).Complete(), keep).
			Typ(types.NewPointer(check.typ)).Val(nil).Call(1).EndInit(1)
		driver.DefineVarStart(token.NoPos, descriptor).Typ(types.NewPointer(types.Typ[types.UnsafePointer])).
			Typ(types.Typ[types.UnsafePointer]).VarVal(keep).UnaryOp(token.AND).Call(1).Call(1).Elem().EndInit(1)
		driver.Val(probe).VarVal(descriptor).Val(nil).Val(2*len(checks) + i).Call(3).EndStmt()
	}
	for i, check := range metadata {
		// Observe the array indexing stride, including trailing padding.
		// LLDB subtracts these addresses; LLGo does not report a size.
		items := fmt.Sprintf("layoutItems%d", i)
		driver.NewVar(types.NewArray(check.typ, 2), items)
		driver.Val(probe)
		for index := 0; index < 2; index++ {
			driver.Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{
				Val: &ast.UnaryExpr{Op: token.AND, X: &ast.IndexExpr{
					X: ast.NewIdent(items), Index: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(index)},
				}},
				Type: types.NewPointer(check.typ),
			}).Call(1)
		}
		driver.Val(2*len(checks) + len(metadata) + i).Call(3).EndStmt()
	}
	for i, check := range sizes {
		items := fmt.Sprintf("layoutFieldItems%d", i)
		driver.NewVar(types.NewArray(check.typ, 2), items)
		driver.Val(probe)
		for index := 0; index < 2; index++ {
			driver.Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{
				Val: &ast.UnaryExpr{Op: token.AND, X: &ast.IndexExpr{
					X: ast.NewIdent(items), Index: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(index)},
				}},
				Type: types.NewPointer(check.typ),
			}).Call(1)
		}
		driver.Val(2*len(checks) + 2*len(metadata) + len(nested) + i).Call(3).EndStmt()
	}
	for i, check := range nested {
		root := &ast.IndexExpr{
			X: ast.NewIdent(fmt.Sprintf("layoutItems%d", check.owner)), Index: &ast.BasicLit{Kind: token.INT, Value: "0"},
		}
		var base, selected ast.Expr = root, root
		baseType, selectedType := metadata[check.owner].typ, metadata[check.owner].typ
		for _, step := range check.path {
			base, baseType = selected, selectedType
			if step.field != nil {
				selected = &ast.SelectorExpr{X: selected, Sel: ast.NewIdent(step.field.Name())}
				selectedType = step.field.Type()
			} else {
				selected = &ast.IndexExpr{X: selected, Index: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(step.index)}}
				selectedType = types.Unalias(selectedType).Underlying().(*types.Array).Elem()
			}
		}
		driver.Val(probe).
			Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: base}, Type: types.NewPointer(baseType)}).Call(1).
			Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: selected}, Type: types.NewPointer(selectedType)}).Call(1).
			Val(2*len(checks) + 2*len(metadata) + i).Call(3).EndStmt()
	}
	var current types.Type
	var subject string
	for i, call := range calls {
		if checks[2*i].kind != types.Invalid {
			// Reserve room to round up to 2*A and then shift by A. For A=4,
			// the second address is four-byte aligned but not eight-byte aligned.
			storage := fmt.Sprintf("storage%d", i)
			start := fmt.Sprintf("start%d", i)
			aligned := fmt.Sprintf("aligned%d", i)
			alignment := layouts[2*i].objectAlign
			windowBytes := (layouts[2*i].size + 2*accessGuard + 7) / 8 * 8
			words := (windowBytes + 3*alignment - 1 + 7) / 8
			driver.NewVar(types.NewArray(types.Typ[types.Uint64], words), storage)
			driver.DefineVarStart(token.NoPos, start).Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{
				Val:  &ast.UnaryExpr{Op: token.AND, X: &ast.IndexExpr{X: ast.NewIdent(storage), Index: &ast.BasicLit{Kind: token.INT, Value: "2"}}},
				Type: types.NewPointer(types.Typ[types.Uint64]),
			}).Call(1).EndInit(1)
			driver.DefineVarStart(token.NoPos, aligned).Val(generated.Unsafe().Ref("Add")).VarVal(start).
				Typ(types.Typ[types.Uintptr]).VarVal(start).Call(1).UnaryOp(token.SUB).
				Val(2*alignment - 1).BinaryOp(token.AND).Call(2).EndInit(1)
			for addressCase := 0; addressCase < 2; addressCase++ {
				pointer := fmt.Sprintf("pointer%d_%d", i, addressCase)
				driver.DefineVarStart(token.NoPos, pointer).Typ(types.NewPointer(call.owner)).
					Val(generated.Unsafe().Ref("Add")).VarVal(aligned).Val(int64(addressCase) * alignment).Call(2).Call(1).EndInit(1)
				driver.Val(call.read).VarVal(pointer).Call(1).EndStmt()
				value, _, err := scalarPattern(checks[2*i].kind, layouts[2*i].width)
				if err != nil {
					return err
				}
				driver.Val(call.write).VarVal(pointer).Val(value).Call(2).EndStmt()
			}
			continue
		}
		if current != call.owner {
			current = call.owner
			subject = fmt.Sprintf("subject%d", i)
			driver.NewVar(current, subject)
		}
		driver.Val(call.read).VarVal(subject).UnaryOp(token.AND).Call(1).EndStmt()
		value := fmt.Sprintf("value%d", i)
		driver.NewVar(call.field, value)
		driver.Val(call.write).VarVal(subject).UnaryOp(token.AND).VarVal(value).Call(2).EndStmt()
	}
	driver.Val(exit).Val(0).Call(1).EndStmt().End()
	var source bytes.Buffer
	source.WriteString("// Code generated by layoutlint. DO NOT EDIT.\n\n")
	if err := generated.WriteTo(&source); err != nil {
		return fmt.Errorf("generate %s: %w", pkg.PkgPath, err)
	}
	filename := "layoutlint_generated.go"
	if tests {
		filename = "layoutlint_generated_test.go"
	}
	path := filepath.Join(directory, filename)
	if err := os.WriteFile(path, source.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Printf("%s: %d wrappers -> %s\n", pkg.PkgPath, len(checks), path)
	if err := checkOffsets(root, directory, path, pkg, tests, checks, layouts, metadata, goMetadata, nested, sizes, private); err != nil {
		return err
	}
	for _, plan := range private {
		if err := os.RemoveAll(plan.directory); err != nil {
			return err
		}
	}
	// Every file at this level belongs to this run. Subdirectories can
	// contain failures from another input package and must be retained.
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
