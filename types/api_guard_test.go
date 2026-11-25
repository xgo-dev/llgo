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

package types_test

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type apiSurface struct {
	types      map[string]*apiType
	vars       map[string]bool
	funcs      map[string]*apiFunc
	deprecated map[string]bool
	goLeaks    []string
}

type apiType struct {
	name     string
	methods  map[string]*apiFunc
	embedded []string
}

type apiFunc struct {
	recvPtr  bool
	params   []string
	results  []string
	variadic bool
	rawSig   string
}

func parsePackageSurface(t *testing.T, dir string, files []string, checkGoTypesLeaks bool) *apiSurface {
	t.Helper()
	s := &apiSurface{
		types:      make(map[string]*apiType),
		vars:       make(map[string]bool),
		funcs:      make(map[string]*apiFunc),
		deprecated: make(map[string]bool),
	}
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		goTypesAlias := ""
		for _, imp := range f.Imports {
			if imp.Path.Value == `"go/types"` {
				if imp.Name != nil {
					goTypesAlias = imp.Name.Name
				} else {
					goTypesAlias = "types"
				}
			}
		}

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch sp := spec.(type) {
					case *ast.TypeSpec:
						if !sp.Name.IsExported() {
							continue
						}
						tname := sp.Name.Name
						if s.types[tname] == nil {
							s.types[tname] = &apiType{name: tname, methods: make(map[string]*apiFunc)}
						}
						if isDeprecatedDoc(sp.Doc) || (len(d.Specs) == 1 && isDeprecatedDoc(d.Doc)) {
							s.deprecated["type:"+tname] = true
						}
						// Record embedded types on structs and interface methods on interfaces.
						if st, ok := sp.Type.(*ast.StructType); ok && st.Fields != nil {
							for _, field := range st.Fields.List {
								if len(field.Names) == 0 {
									if embName, _ := recvTypeName(field.Type); embName != "" {
										s.types[tname].embedded = append(s.types[tname].embedded, embName)
									}
								}
							}
						}
						if iface, ok := sp.Type.(*ast.InterfaceType); ok && iface.Methods != nil {
							for _, field := range iface.Methods.List {
								if len(field.Names) == 0 {
									if embName, _ := recvTypeName(field.Type); embName != "" {
										s.types[tname].embedded = append(s.types[tname].embedded, embName)
									}
									continue
								}
								for _, id := range field.Names {
									if !id.IsExported() {
										continue
									}
									if ft, ok := field.Type.(*ast.FuncType); ok {
										s.types[tname].methods[id.Name] = parseFuncSig(fset, false, ft)
										if isDeprecatedDoc(field.Doc) {
											s.deprecated["method:"+tname+"."+id.Name] = true
										}
									}
								}
							}
						}
						if checkGoTypesLeaks && goTypesAlias != "" && exprRefsAlias(sp.Type, goTypesAlias) {
							if !allowedGoTypesTypeAlias[tname] {
								s.goLeaks = append(s.goLeaks, "type "+tname)
							}
						}
					case *ast.ValueSpec:
						if d.Tok != token.VAR {
							continue
						}
						for _, id := range sp.Names {
							if !id.IsExported() {
								continue
							}
							s.vars[id.Name] = true
							if isDeprecatedDoc(sp.Doc) || (len(d.Specs) == 1 && isDeprecatedDoc(d.Doc)) {
								s.deprecated["var:"+id.Name] = true
							}
							if checkGoTypesLeaks && goTypesAlias != "" && exprRefsAlias(sp.Type, goTypesAlias) {
								s.goLeaks = append(s.goLeaks, "var "+id.Name)
							}
						}
					}
				}
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				if d.Recv == nil || len(d.Recv.List) == 0 {
					sig := parseFuncSig(fset, false, d.Type)
					s.funcs[d.Name.Name] = sig
					if isDeprecatedDoc(d.Doc) {
						s.deprecated["func:"+d.Name.Name] = true
					}
					if checkGoTypesLeaks && goTypesAlias != "" && exprRefsAlias(d.Type, goTypesAlias) {
						s.goLeaks = append(s.goLeaks, "func "+d.Name.Name)
					}
					continue
				}
				recvName, recvPtr := recvTypeName(d.Recv.List[0].Type)
				if recvName == "" || !ast.IsExported(recvName) {
					continue
				}
				if s.types[recvName] == nil {
					s.types[recvName] = &apiType{name: recvName, methods: make(map[string]*apiFunc)}
				}
				s.types[recvName].methods[d.Name.Name] = parseFuncSig(fset, recvPtr, d.Type)
				if isDeprecatedDoc(d.Doc) {
					s.deprecated["method:"+recvName+"."+d.Name.Name] = true
				}
				if checkGoTypesLeaks && goTypesAlias != "" && exprRefsAlias(d.Type, goTypesAlias) {
					key := recvName + "." + d.Name.Name
					if !allowedGoTypesBridgeMethods[key] {
						s.goLeaks = append(s.goLeaks, "method "+key)
					}
				}
			}
		}
	}
	// Also collect exported methods on unexported embedded helper types
	// (e.g., object) so promoted methods on exported structs are visible.
	unexportedMethods := make(map[string]map[string]*apiFunc)
	for _, name := range files {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if !ok || !d.Name.IsExported() || d.Recv == nil || len(d.Recv.List) == 0 {
				continue
			}
			recvName, recvPtr := recvTypeName(d.Recv.List[0].Type)
			if recvName == "" || ast.IsExported(recvName) {
				continue
			}
			if unexportedMethods[recvName] == nil {
				unexportedMethods[recvName] = make(map[string]*apiFunc)
			}
			unexportedMethods[recvName][d.Name.Name] = parseFuncSig(fset, recvPtr, d.Type)
		}
	}
	for _, tp := range s.types {
		for _, emb := range tp.embedded {
			if ms, ok := unexportedMethods[emb]; ok {
				for mname, sig := range ms {
					if tp.methods[mname] == nil {
						tp.methods[mname] = sig
					}
				}
			}
			if et, ok := s.types[emb]; ok {
				for mname, sig := range et.methods {
					if tp.methods[mname] == nil {
						tp.methods[mname] = sig
					}
				}
			}
		}
	}
	return s
}

func isDeprecatedDoc(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	text := doc.Text()
	return strings.HasPrefix(text, "Deprecated:") || strings.Contains(text, "\nDeprecated:")
}

func recvTypeName(expr ast.Expr) (string, bool) {
	isPtr := false
	if star, ok := expr.(*ast.StarExpr); ok {
		isPtr = true
		expr = star.X
	}
	if idx, ok := expr.(*ast.IndexExpr); ok {
		expr = idx.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name, isPtr
	}
	return "", isPtr
}

func parseFuncSig(fset *token.FileSet, recvPtr bool, ft *ast.FuncType) *apiFunc {
	af := &apiFunc{recvPtr: recvPtr}
	if ft.Params != nil {
		for _, field := range ft.Params.List {
			ts := exprTypeStr(fset, field.Type)
			if _, ok := field.Type.(*ast.Ellipsis); ok {
				af.variadic = true
			}
			n := len(field.Names)
			if n == 0 {
				n = 1
			}
			for i := 0; i < n; i++ {
				af.params = append(af.params, ts)
			}
		}
	}
	if ft.Results != nil {
		for _, field := range ft.Results.List {
			ts := exprTypeStr(fset, field.Type)
			n := len(field.Names)
			if n == 0 {
				n = 1
			}
			for i := 0; i < n; i++ {
				af.results = append(af.results, ts)
			}
		}
	}
	af.rawSig = "(" + strings.Join(af.params, ", ") + ") -> (" + strings.Join(af.results, ", ") + ")"
	return af
}

func exprTypeStr(fset *token.FileSet, e ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, e)
	return buf.String()
}

func exprRefsAlias(node ast.Node, alias string) bool {
	if node == nil || alias == "" {
		return false
	}
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		// Only inspect exported struct fields inside StructType.
		if st, ok := n.(*ast.StructType); ok && st.Fields != nil {
			for _, f := range st.Fields.List {
				exported := len(f.Names) == 0 // embedded
				for _, id := range f.Names {
					if id.IsExported() {
						exported = true
						break
					}
				}
				if exported && exprRefsAlias(f.Type, alias) {
					found = true
					return false
				}
			}
			return false
		}
		// Only inspect exported methods and embedded interfaces inside InterfaceType.
		if it, ok := n.(*ast.InterfaceType); ok && it.Methods != nil {
			for _, f := range it.Methods.List {
				exported := len(f.Names) == 0
				for _, id := range f.Names {
					if id.IsExported() {
						exported = true
						break
					}
				}
				if exported && exprRefsAlias(f.Type, alias) {
					found = true
					return false
				}
			}
			return false
		}
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == alias {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// allowedGoTypesTypeAlias lists the ONLY exported type aliases referencing
// go/types (all pure leaf enums with zero go/types.Type/Object methods).
var allowedGoTypesTypeAlias = map[string]bool{
	"BasicKind":     true,
	"BasicInfo":     true,
	"ChanDir":       true,
	"SelectionKind": true,
	"ImportMode":    true,
}

// allowedGoTypesBridgeMethods lists the ONLY exported methods permitted to
// reference go/types in their signatures (the explicit *Context boundary).
var allowedGoTypesBridgeMethods = map[string]bool{
	"Context.FromGo":             true,
	"Context.FromGoWithCallConv": true,
	"Context.FromGoObject":       true,
	"Context.GoSizes":            true,
}

// knownGoTypesOmissions lists the ONLY non-deprecated stdlib go/types symbols
// intentionally omitted in llgo/types (strictly Go 1.24+ iter.Seq methods that
// cannot compile under Go 1.20, plus stdlib's internal DefPredeclaredTestFuncs).
var knownGoTypesOmissions = map[string]bool{
	"func:DefPredeclaredTestFuncs":       true,
	"method:Interface.EmbeddedTypes":     true, // Go 1.24 iter.Seq
	"method:Interface.ExplicitMethods":   true, // Go 1.24 iter.Seq
	"method:Interface.Methods":           true, // Go 1.24 iter.Seq
	"method:MethodSet.Methods":           true, // Go 1.24 iter.Seq
	"method:Named.Methods":               true, // Go 1.24 iter.Seq
	"method:Scope.Children":              true, // Go 1.24 iter.Seq
	"method:Struct.Fields":               true, // Go 1.24 iter.Seq
	"method:Tuple.Variables":             true, // Go 1.24 iter.Seq
	"method:TypeList.Types":              true, // Go 1.24 iter.Seq
	"method:TypeParamList.TypeParams":    true, // Go 1.24 iter.Seq
	"method:Union.Terms":                 true, // Go 1.24 iter.Seq
}

// goTypesForwardCompat lists standard go/types APIs introduced in Go 1.21-1.27
// that llgo/types implements in pure Go 1.20 so the guard passes on older Go.
var goTypesForwardCompat = map[string]bool{
	"type:Alias":                  true,
	"type:VarKind":                true,
	"type:Hasher":                 true,
	"type:HasherIgnoreTags":       true,
	"func:NewAlias":               true,
	"func:Unalias":                true,
	"func:LookupSelection":        true,
	"method:Package.GoVersion":    true,
	"method:Info.PkgNameOf":       true,
	"method:Var.Kind":             true,
	"method:Var.SetKind":          true,
	"method:Alias.Obj":            true,
	"method:Alias.Origin":         true,
	"method:Alias.Rhs":            true,
	"method:Alias.SetTypeParams":  true,
	"method:Alias.String":         true,
	"method:Alias.TypeArgs":       true,
	"method:Alias.TypeParams":     true,
	"method:Alias.Underlying":     true,
	"method:VarKind.String":       true,
	"method:Hasher.Equal":         true,
	"method:Hasher.Hash":          true,
	"method:HasherIgnoreTags.Equal": true,
	"method:HasherIgnoreTags.Hash":  true,
}

// verifiedTypeInterfaceExtensions are the 7 verified compiler methods promoted
// onto the Type interface (implemented by all concrete Type types).
var verifiedTypeInterfaceExtensions = []string{
	"HasPtrData",
	"IsHostLayoutMarker",
	"HasHostLayout",
	"ContainsHostLayout",
	"ContainsSIMDStorage",
	"SIMDTagIndex",
	"SIMDVec",
}

var concreteGoTypeNames = []string{
	"Type",
	"Basic", "Pointer", "Tuple", "Signature", "Slice", "Array",
	"Map", "Chan", "Interface", "Struct", "Named", "Alias", "TypeParam", "Union",
}

// verifiedLLGoExtensions is the strict allowlist of ALL exported llgo/types
// symbols beyond standard go/types. Every entry must have a verified consumer
// in ssa/ or internal/llvmgen/ (do not expand speculatively).
func buildVerifiedLLGoExtensions() map[string]bool {
	m := map[string]bool{
		// 9 verified LLGo-only types.
		"type:CallConv":      true,
		"type:PtrKind":       true,
		"type:SIMDVec":       true,
		"type:ContextConfig": true,
		"type:Layout":        true,
		"type:FieldLayout":   true,
		"type:LayoutPair":    true,
		"type:DataKind":      true,
		"type:NativeIntExt":  true,

		// 5 verified LLGo-only top-level functions.
		"func:NewContextWithConfig":     true,
		"func:NewPointerWithKind":       true,
		"func:NewSignatureWithCallConv": true,
		"func:DataKindOf":               true,
		"func:NativeIntExtForSpec":      true,

		// Verified methods on shared types.
		"method:Pointer.Kind":          true,
		"method:Signature.CallConv":    true,
		"method:Signature.WithCallConv": true,
		"method:Signature.IsClosure":   true,
		"method:Signature.IsNative":    true,
		"method:Named.CallConv":        true,
		"method:Alias.CallConv":        true,

		// Verified *Context compiler/bridge/Sizes methods.
		"method:Context.FromGo":                 true,
		"method:Context.FromGoWithCallConv":     true,
		"method:Context.FromGoObject":           true,
		"method:Context.GoSizes":                true,
		"method:Context.Alignof":                true,
		"method:Context.Offsetsof":              true,
		"method:Context.Sizeof":                 true,
		"method:Context.Layout":                 true,
		"method:Context.EqualName":              true,
		"method:Context.HostUsesNativeGoLayout": true,
		"method:Context.HostFieldLayout":        true,
		"method:Context.HostBasicLayout":        true,
		"method:Context.SIMDAlignment":          true,
		"method:Context.HasNativeMetadata":      true,
		"method:Context.SetTypeCallConv":        true,
		"method:Context.TypeCallConv":           true,
		"method:Context.NativeTypeCallConv":     true,
		"method:Context.SetNativeStructFields":  true,
		"method:Context.MarkNativeField":        true,
		"method:Context.SetSizes":               true,
	}
	for _, tn := range concreteGoTypeNames {
		for _, mn := range verifiedTypeInterfaceExtensions {
			m["method:"+tn+"."+mn] = true
		}
	}
	return m
}

func loadSurfaces(t *testing.T) (goSurf, llSurf *apiSurface) {
	t.Helper()
	goPkg, err := build.Default.Import("go/types", "", 0)
	if err != nil {
		t.Fatalf("import go/types: %v", err)
	}
	llPkg, err := build.Default.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("import .: %v", err)
	}
	return parsePackageSurface(t, goPkg.Dir, goPkg.GoFiles, false),
		parsePackageSurface(t, llPkg.Dir, llPkg.GoFiles, true)
}

func TestGoTypesAPIGuard(t *testing.T) {
	goSurf, llSurf := loadSurfaces(t)

	t.Run("NonDeprecatedGoTypesCoverage", func(t *testing.T) {
		var missing []string
		for name := range goSurf.types {
			if goSurf.deprecated["type:"+name] || knownGoTypesOmissions["type:"+name] {
				continue
			}
			if llSurf.types[name] == nil {
				missing = append(missing, "type "+name)
			}
		}
		for name := range goSurf.vars {
			if goSurf.deprecated["var:"+name] || knownGoTypesOmissions["var:"+name] {
				continue
			}
			if !llSurf.vars[name] {
				missing = append(missing, "var "+name)
			}
		}
		for name := range goSurf.funcs {
			if goSurf.deprecated["func:"+name] || knownGoTypesOmissions["func:"+name] {
				continue
			}
			if llSurf.funcs[name] == nil {
				missing = append(missing, "func "+name)
			}
		}
		for tname, gt := range goSurf.types {
			lt := llSurf.types[tname]
			if lt == nil {
				continue
			}
			for mname := range gt.methods {
				key := "method:" + tname + "." + mname
				if goSurf.deprecated[key] || knownGoTypesOmissions[key] {
					continue
				}
				if lt.methods[mname] == nil {
					missing = append(missing, "method "+tname+"."+mname)
				}
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("llgo/types is missing non-deprecated go/types APIs (implement or justify in knownGoTypesOmissions):\n  %s",
				strings.Join(missing, "\n  "))
		}
	})

	t.Run("MinimalVerifiedExtensions", func(t *testing.T) {
		allowed := buildVerifiedLLGoExtensions()
		var unexpected []string
		for name := range llSurf.types {
			if goSurf.types[name] != nil {
				continue
			}
			key := "type:" + name
			if !allowed[key] && !goTypesForwardCompat[key] {
				unexpected = append(unexpected, "type "+name)
			}
		}
		for name := range llSurf.vars {
			if goSurf.vars[name] {
				continue
			}
			key := "var:" + name
			if !allowed[key] && !goTypesForwardCompat[key] {
				unexpected = append(unexpected, "var "+name)
			}
		}
		for name := range llSurf.funcs {
			if goSurf.funcs[name] != nil {
				continue
			}
			key := "func:" + name
			if !allowed[key] && !goTypesForwardCompat[key] {
				unexpected = append(unexpected, "func "+name)
			}
		}
		for tname, lt := range llSurf.types {
			gt := goSurf.types[tname]
			if gt == nil {
				continue
			}
			for mname := range lt.methods {
				if gt.methods[mname] != nil {
					continue
				}
				key := "method:" + tname + "." + mname
				if !allowed[key] && !goTypesForwardCompat[key] {
					unexpected = append(unexpected, "method "+tname+"."+mname)
				}
			}
		}
		sort.Strings(unexpected)
		if len(unexpected) > 0 {
			t.Errorf("llgo/types introduced unverified non-go/types exports (remove or verify necessity and add to buildVerifiedLLGoExtensions):\n  %s",
				strings.Join(unexpected, "\n  "))
		}

		// Ensure allowlist has no stale entries.
		var stale []string
		for key := range allowed {
			switch {
			case strings.HasPrefix(key, "type:"):
				if llSurf.types[strings.TrimPrefix(key, "type:")] == nil {
					stale = append(stale, key)
				}
			case strings.HasPrefix(key, "func:"):
				if llSurf.funcs[strings.TrimPrefix(key, "func:")] == nil {
					stale = append(stale, key)
				}
			case strings.HasPrefix(key, "method:"):
				parts := strings.SplitN(strings.TrimPrefix(key, "method:"), ".", 2)
				lt := llSurf.types[parts[0]]
				if lt == nil || lt.methods[parts[1]] == nil {
					stale = append(stale, key)
				}
			}
		}
		sort.Strings(stale)
		if len(stale) > 0 {
			t.Errorf("stale entries in buildVerifiedLLGoExtensions (remove them):\n  %s", strings.Join(stale, "\n  "))
		}
	})

	t.Run("SignatureParity", func(t *testing.T) {
		var mismatches []string
		for name, gsig := range goSurf.funcs {
			lsig := llSurf.funcs[name]
			if lsig == nil {
				continue
			}
			if gsig.rawSig != lsig.rawSig || gsig.variadic != lsig.variadic {
				mismatches = append(mismatches, "func "+name+": go="+gsig.rawSig+" llgo="+lsig.rawSig)
			}
		}
		for tname, gt := range goSurf.types {
			lt := llSurf.types[tname]
			if lt == nil {
				continue
			}
			for mname, gsig := range gt.methods {
				lsig := lt.methods[mname]
				if lsig == nil {
					continue
				}
				if gsig.recvPtr != lsig.recvPtr || gsig.rawSig != lsig.rawSig || gsig.variadic != lsig.variadic {
					mismatches = append(mismatches, "method "+tname+"."+mname+": go="+gsig.rawSig+" llgo="+lsig.rawSig)
				}
			}
		}
		sort.Strings(mismatches)
		if len(mismatches) > 0 {
			t.Errorf("signature drift between go/types and llgo/types:\n  %s", strings.Join(mismatches, "\n  "))
		}
	})

	t.Run("NoGoTypesPublicLeakage", func(t *testing.T) {
		if len(llSurf.goLeaks) > 0 {
			sort.Strings(llSurf.goLeaks)
			t.Errorf("unexpected go/types leakage in llgo/types public API:\n  %s",
				strings.Join(llSurf.goLeaks, "\n  "))
		}
	})
}
