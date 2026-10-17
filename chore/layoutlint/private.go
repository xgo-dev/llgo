package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/goplus/gogen"
	llruntime "github.com/xgo-dev/llgo/runtime"
	"golang.org/x/tools/go/packages"
)

const privateCheckpointBase = 1 << 30

type privateLayout struct {
	pkg       *packages.Package
	subject   metadataCheck
	paths     []nestedCheck
	baseline  goMetadata
	directory string
	source    string
	logical   []string
	firstID   int
	Metadata  metadataResult `json:"metadata"`
	Fields    []offsetResult `json:"fields"`
}

// preparePrivateLayouts generates within the owner package. The emitted
// expressions name original fields; no expected offset is used as an address.
func preparePrivateLayouts(root, directory, baselineDirectory string, pkg *packages.Package, subjects []metadataCheck, dependencies map[string]string) ([]privateLayout, error) {
	var targets []*types.Named
	visited := make(map[types.Type]bool)
	var containsPrivate func(types.Type, *types.Package) bool
	containsPrivate = func(typ types.Type, owner *types.Package) bool {
		typ = types.Unalias(typ)
		if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != owner {
			return false // That type is probed in its own defining package.
		}
		switch typ := typ.Underlying().(type) {
		case *types.Struct:
			for i := 0; i < typ.NumFields(); i++ {
				field := typ.Field(i)
				if field.Name() != "_" && (!field.Exported() && field.Pkg() == owner || containsPrivate(field.Type(), owner)) {
					return true
				}
			}
		case *types.Array:
			return typ.Len() != 0 && containsPrivate(typ.Elem(), owner)
		}
		return false
	}
	var visit func(types.Type)
	visit = func(typ types.Type) {
		typ = types.Unalias(typ)
		if visited[typ] {
			return
		}
		visited[typ] = true
		switch underlying := typ.Underlying().(type) {
		case *types.Struct:
			if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg() != pkg.Types {
				if containsPrivate(named, named.Obj().Pkg()) && !slices.ContainsFunc(targets, func(previous *types.Named) bool { return types.Identical(previous, named) }) {
					targets = append(targets, named)
				}
			}
			for i := 0; i < underlying.NumFields(); i++ {
				visit(underlying.Field(i).Type())
			}
		case *types.Array:
			if underlying.Len() != 0 {
				visit(underlying.Elem())
			}
		}
	}
	for _, subject := range subjects {
		visit(subject.typ)
	}
	slices.SortFunc(targets, func(a, b *types.Named) int { return strings.Compare(a.String(), b.String()) })
	var plans []privateLayout
	nextID := privateCheckpointBase
	for _, target := range targets {
		owner := target.Obj().Pkg().Path()
		loaded, err := packages.Load(&packages.Config{
			Dir:  root,
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule,
		}, owner)
		if err != nil {
			return nil, err
		}
		if packages.PrintErrors(loaded) != 0 || len(loaded) != 1 {
			return nil, fmt.Errorf("private layout owner %s could not be loaded", owner)
		}
		ownerPkg := loaded[0]
		object := ownerPkg.Types.Scope().Lookup(target.Obj().Name())
		if object == nil {
			return nil, fmt.Errorf("private layout owner %s has no type %s", owner, target.Obj().Name())
		}
		typ := object.Type()
		if target.TypeArgs().Len() != 0 {
			var arguments []types.Type
			for i := 0; i < target.TypeArgs().Len(); i++ {
				arguments = append(arguments, target.TypeArgs().At(i))
			}
			typ, err = types.Instantiate(nil, typ, arguments, true)
			if err != nil {
				return nil, err
			}
		}
		position := ownerPkg.Fset.PositionFor(object.Pos(), true)
		name := object.Name()
		identifier := name
		if target.TypeArgs().Len() != 0 {
			name = types.TypeString(typ, func(p *types.Package) string {
				if p == ownerPkg.Types {
					return ""
				}
				return p.Path()
			})
			identifier = "layoutlintPrivateSubject"
		}
		plan := privateLayout{pkg: ownerPkg, subject: metadataCheck{typ: typ, name: name, identifier: identifier, file: position.Filename, line: position.Line}, firstID: nextID}
		var walk func(types.Type, []layoutStep)
		walk = func(typ types.Type, path []layoutStep) {
			switch typ := types.Unalias(typ).Underlying().(type) {
			case *types.Struct:
				for i := 0; i < typ.NumFields(); i++ {
					field := typ.Field(i)
					if field.Name() == "_" || !field.Exported() && field.Pkg() != ownerPkg.Types {
						continue
					}
					child := append(slices.Clone(path), layoutStep{field: field})
					plan.paths = append(plan.paths, nestedCheck{path: child})
					walk(field.Type(), child)
				}
			case *types.Array:
				elem := typ.Elem()
				for {
					array, ok := types.Unalias(elem).Underlying().(*types.Array)
					if !ok {
						break
					}
					elem = array.Elem()
				}
				if _, ok := types.Unalias(elem).Underlying().(*types.Struct); !ok {
					return
				}
				for i := int64(0); i < min(typ.Len(), 2); i++ {
					child := append(slices.Clone(path), layoutStep{index: i})
					plan.paths = append(plan.paths, nestedCheck{path: child})
					walk(typ.Elem(), child)
				}
			}
		}
		walk(typ, nil)
		plan.directory = filepath.Join(directory, fmt.Sprintf("private-%d", len(plans)))
		if err := os.Mkdir(plan.directory, 0755); err != nil {
			return nil, err
		}
		_, baseline, err := goOffsets(root, plan.directory, baselineDirectory, ownerPkg, false, []string{owner}, nil, []metadataCheck{plan.subject}, plan.paths, dependencies, true)
		if err != nil {
			return nil, err
		}
		plan.baseline = baseline[0]
		generated := gogen.NewPackage(ownerPkg.PkgPath, ownerPkg.Name, &gogen.Config{Types: ownerPkg.Types, Fset: ownerPkg.Fset})
		probeName := fmt.Sprintf("layoutlintPrivateCheckpoint%d", len(plans))
		sizeName := fmt.Sprintf("layoutlintPrivateFieldSize%d", len(plans))
		for _, reserved := range []string{probeName, sizeName} {
			if ownerPkg.Types.Scope().Lookup(reserved) != nil {
				return nil, fmt.Errorf("%s: generated name %s already exists", owner, reserved)
			}
		}
		base := generated.NewParam(token.NoPos, "base", types.Typ[types.UnsafePointer], false)
		address := generated.NewParam(token.NoPos, "address", types.Typ[types.UnsafePointer], false)
		id := generated.NewParam(token.NoPos, "id", types.Typ[types.Uintptr], false)
		probe := generated.NewFunc(nil, probeName, types.NewTuple(base, address, id), nil, false)
		probe.SetComments(generated, &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}})
		probe.BodyStart(generated).Val(types.Universe.Lookup("println")).Val("layoutlint").Val(base).Val(address).Val(id).Call(4).EndStmt().End()
		// Infer F from the actual field expression, so replacement packages do
		// not have to import the Go package's private field-type dependencies.
		parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, ownerPkg.Types, "F", nil), types.Universe.Lookup("any").Type())
		value := generated.NewParam(token.NoPos, "value", types.NewPointer(parameter), false)
		sizeID := generated.NewParam(token.NoPos, "id", types.Typ[types.Uintptr], false)
		sizeProbe := generated.NewFuncDecl(token.NoPos, sizeName, types.NewSignatureType(nil, nil, []*types.TypeParam{parameter}, types.NewTuple(value, sizeID), nil, false))
		cb := sizeProbe.BodyStart(generated)
		cb.NewVar(types.NewArray(parameter, 2), "items")
		cb.Val(probe)
		for index := 0; index < 2; index++ {
			cb.Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: &ast.IndexExpr{X: ast.NewIdent("items"), Index: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(index)}}}, Type: types.NewPointer(parameter)}).Call(1)
		}
		cb.Val(sizeID).Call(3).EndStmt().End()
		driver := generated.NewFunc(nil, "init", nil, nil, false).BodyStart(generated)
		driver.NewVarStart(types.NewInterfaceType(nil, nil).Complete(), "metadataValue").Typ(types.NewPointer(typ)).Val(nil).Call(1).EndInit(1)
		driver.DefineVarStart(token.NoPos, "descriptor").Typ(types.NewPointer(types.Typ[types.UnsafePointer])).Typ(types.Typ[types.UnsafePointer]).VarVal("metadataValue").UnaryOp(token.AND).Call(1).Call(1).Elem().EndInit(1)
		driver.Val(probe).VarVal("descriptor").Val(nil).Val(nextID).Call(3).EndStmt()
		driver.NewVar(types.NewArray(typ, 2), "subjects")
		driver.Val(probe)
		for index := 0; index < 2; index++ {
			driver.Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: &ast.IndexExpr{X: ast.NewIdent("subjects"), Index: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(index)}}}, Type: types.NewPointer(typ)}).Call(1)
		}
		driver.Val(nextID + 1).Call(3).EndStmt()
		for i, check := range plan.paths {
			var expression ast.Expr = &ast.IndexExpr{X: ast.NewIdent("subjects"), Index: &ast.BasicLit{Kind: token.INT, Value: "0"}}
			var parent ast.Expr
			fieldType, parentType := typ, typ
			for _, step := range check.path {
				parent, parentType = expression, fieldType
				if step.field != nil {
					expression = &ast.SelectorExpr{X: expression, Sel: ast.NewIdent(step.field.Name())}
					fieldType = step.field.Type()
				} else {
					expression = &ast.IndexExpr{X: expression, Index: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(step.index)}}
					fieldType = types.Unalias(fieldType).Underlying().(*types.Array).Elem()
				}
			}
			driver.Val(probe).Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: parent}, Type: types.NewPointer(parentType)}).Call(1).
				Typ(types.Typ[types.UnsafePointer]).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: expression}, Type: types.NewPointer(fieldType)}).Call(1).Val(nextID + 2 + 2*i).Call(3).EndStmt()
			driver.Val(sizeProbe).Val(&gogen.Element{Val: &ast.UnaryExpr{Op: token.AND, X: expression}, Type: types.NewPointer(fieldType)}).Val(nextID + 3 + 2*i).Call(2).EndStmt()
		}
		driver.End()
		var source strings.Builder
		if err := generated.WriteTo(&source); err != nil {
			return nil, err
		}
		plan.source = filepath.Join(plan.directory, "layoutlint_private.go")
		if err := os.WriteFile(plan.source, []byte(source.String()), 0644); err != nil {
			return nil, err
		}
		filename := fmt.Sprintf("layoutlint_private_%d.go", len(plans))
		plan.logical = []string{filepath.Join(filepath.Dir(ownerPkg.GoFiles[0]), filename)}
		if llruntime.HasAltPkgForGOARCH(owner, runtime.GOARCH) {
			compiler := os.Getenv("LLGO")
			if compiler == "" {
				compiler = "llgo"
			}
			data, err := command(root, filepath.Join(plan.directory, "llgo-env.log"), compiler, "env", "LLGO_RUNTIME_DIR")
			if err != nil {
				return nil, err
			}
			actual := filepath.Join(strings.TrimSpace(string(data)), "internal", "lib", filepath.FromSlash(owner))
			if info, err := os.Stat(actual); err != nil || !info.IsDir() {
				return nil, fmt.Errorf("LLGo replacement source for %s is unavailable at %s", owner, actual)
			}
			plan.logical = append(plan.logical, filepath.Join(actual, filename))
		}
		nextID += 2 + 2*len(plan.paths)
		plans = append(plans, plan)
	}
	return plans, nil
}

// Dependency initialization order is determined by the compiler. Match these
// checkpoints by ID rather than imposing the input package's discovery order.
func observePrivateLayouts(session *lldbSession, plans []privateLayout, readArguments func() ([3]uint64, error)) (bool, error) {
	type observation struct{ group, event int }
	pending := make(map[uint64]observation)
	for i, plan := range plans {
		for event := 0; event < 2+2*len(plan.paths); event++ {
			pending[uint64(plan.firstID+event)] = observation{i, event}
		}
	}
	failed := false
	for len(pending) != 0 {
		arguments, err := readArguments()
		if err != nil {
			return failed, err
		}
		current, ok := pending[arguments[2]]
		if !ok {
			return failed, fmt.Errorf("unexpected checkpoint %d with %d private observations still missing", arguments[2], len(pending))
		}
		plan := &plans[current.group]
		switch current.event {
		case 0:
			if arguments[1] != 0 {
				return failed, fmt.Errorf("private metadata %s has unexpected payload", plan.Metadata.Type)
			}
			var fields []metadataFieldCheck
			for i, path := range plan.paths {
				if path.path[len(path.path)-1].field != nil {
					fields = append(fields, metadataFieldCheck{field: plan.Fields[i].Field, path: path.path})
				}
			}
			captureErr := readMetadata(session, plan.directory, 0, arguments[0], plan.subject.typ, fields, &plan.Metadata)
			for i := range plan.Fields {
				field := &plan.Fields[i]
				if offset, ok := plan.Metadata.fieldOffsets[field.Field]; ok {
					field.MetadataOffset = &offset
				}
			}
			expected := *plan.baseline.TypeHeader
			expected.Size, expected.PtrBytes = plan.baseline.Layout.Size, plan.baseline.Layout.PtrBytes
			expected.Align, expected.FieldAlign = plan.baseline.Layout.Align, plan.baseline.Layout.FieldAlign
			for _, pair := range []struct {
				label string
				want  *goMetadataHeader
				got   *metadataHeader
			}{{"pointer", plan.baseline.PointerHeader, plan.Metadata.PointerHeader}, {"type", &expected, plan.Metadata.TypeHeader}} {
				if pair.got == nil {
					continue
				}
				for _, field := range []struct {
					name      string
					want, got uint64
				}{
					{"Size_", pair.want.Size, pair.got.Size}, {"PtrBytes", pair.want.PtrBytes, pair.got.PtrBytes},
					{"Align_", uint64(pair.want.Align), uint64(pair.got.Align)}, {"FieldAlign_", uint64(pair.want.FieldAlign), uint64(pair.got.FieldAlign)},
					{"Kind", uint64(pair.want.Kind & 31), uint64(pair.got.Kind & 31)},
				} {
					if field.want != field.got {
						failed = true
						plan.Metadata.Differences = append(plan.Metadata.Differences, metadataDifference{pair.label, field.name, field.want, field.got})
						fmt.Printf("FAIL %s: %s.%s: expected %d, got %d\n", plan.Metadata.Type, pair.label, field.name, field.want, field.got)
					}
				}
			}
			for index, field := range plan.Metadata.Fields {
				if field.Observed && field.Expected != field.Actual {
					failed = true
					name := fmt.Sprintf("Fields[%d](%s).Offset", index, field.Name)
					plan.Metadata.Differences = append(plan.Metadata.Differences, metadataDifference{"type", name, field.Expected, field.Actual})
					fmt.Printf("FAIL %s: %s: expected %d, got %d\n", plan.Metadata.Type, name, field.Expected, field.Actual)
				}
			}
			if captureErr != nil {
				return failed, fmt.Errorf("private metadata %s: %w", plan.Metadata.Type, captureErr)
			}
		case 1:
			if plan.Metadata.TypeHeader == nil || arguments[1] < arguments[0] {
				return failed, fmt.Errorf("invalid private array observation for %s", plan.Metadata.Type)
			}
			stride := arguments[1] - arguments[0]
			plan.Metadata.Stride = &strideResult{FirstAddress: fmt.Sprintf("0x%x", arguments[0]), SecondAddress: fmt.Sprintf("0x%x", arguments[1]), Bytes: stride, GoMatches: stride == plan.baseline.Layout.Size, MetadataMatches: stride == plan.Metadata.TypeHeader.Size}
			if !plan.Metadata.Stride.GoMatches || !plan.Metadata.Stride.MetadataMatches {
				failed = true
				fmt.Printf("FAIL %s: actual stride %d, Go %d, metadata %d\n", plan.Metadata.Type, stride, plan.baseline.Layout.Size, plan.Metadata.TypeHeader.Size)
			}
		default:
			index := (current.event - 2) / 2
			field, expected := &plan.Fields[index], plan.paths[index]
			if arguments[1] < arguments[0] {
				return failed, fmt.Errorf("private field %s.%s has a reversed address range", plan.Metadata.Type, field.Field)
			}
			if current.event%2 == 0 {
				field.Observed, field.Actual = true, int64(arguments[1]-arguments[0])
				field.Alignment = &alignmentResult{ObjectRequired: expected.baseAlign, FieldRequired: expected.alignment, ObjectAddress: fmt.Sprintf("0x%x", arguments[0]), FieldAddress: fmt.Sprintf("0x%x", arguments[1]), ObjectMatches: arguments[0]%uint64(expected.baseAlign) == 0, FieldMatches: arguments[1]%uint64(expected.alignment) == 0}
				if field.Actual != field.Expected || !field.Alignment.ObjectMatches || !field.Alignment.FieldMatches {
					failed = true
					fmt.Printf("FAIL %s.%s: offset/alignment: GoOffset %d, LLGoOffset %d\n", plan.Metadata.Type, field.Field, field.Expected, field.Actual)
				}
				if field.Operation != "array-element" && (field.MetadataOffset == nil || *field.MetadataOffset != uint64(field.Expected) || *field.MetadataOffset != uint64(field.Actual)) {
					failed = true
					fmt.Printf("FAIL %s.%s: metadata offset missing or inconsistent\n", plan.Metadata.Type, field.Field)
				}
			} else {
				field.Size.Actual, field.Size.Observed = arguments[1]-arguments[0], true
				field.Size.FirstAddress, field.Size.SecondAddress = fmt.Sprintf("0x%x", arguments[0]), fmt.Sprintf("0x%x", arguments[1])
				if field.Size.Actual != field.Size.Expected {
					failed = true
					fmt.Printf("FAIL %s.%s: GoSize %d, LLGoSize %d\n", plan.Metadata.Type, field.Field, field.Size.Expected, field.Size.Actual)
				}
			}
		}
		delete(pending, arguments[2])
		if _, err := session.run("process continue --continue-to-bkpt 1"); err != nil {
			return failed, err
		}
	}
	return failed, nil
}
