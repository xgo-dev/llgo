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

package cl

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type pkgAPISurface struct {
	types      map[string]bool
	funcs      map[string]string // funcName -> normalized signature
	vars       map[string]bool
	consts     map[string]bool
	methods    map[string]map[string]string // typeName -> methodName -> normalized signature
	fields     map[string]map[string]string // structTypeName -> fieldName -> normalized type
	deprecated map[string]bool
}

func loadPkgAPISurface(t *testing.T, importPath string) *pkgAPISurface {
	t.Helper()
	out, err := exec.Command("go", "list", "-json", importPath).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", importPath, err)
	}
	var info struct {
		Dir     string
		GoFiles []string
	}
	if err := json.Unmarshal(out, &info); err != nil {
		t.Fatalf("unmarshal go list %s: %v", importPath, err)
	}

	s := &pkgAPISurface{
		types:      make(map[string]bool),
		funcs:      make(map[string]string),
		vars:       make(map[string]bool),
		consts:     make(map[string]bool),
		methods:    make(map[string]map[string]string),
		fields:     make(map[string]map[string]string),
		deprecated: make(map[string]bool),
	}
	embedded := make(map[string][]string)
	fset := token.NewFileSet()

	ensureMethods := func(tname string) map[string]string {
		if s.methods[tname] == nil {
			s.methods[tname] = make(map[string]string)
		}
		return s.methods[tname]
	}
	ensureFields := func(tname string) map[string]string {
		if s.fields[tname] == nil {
			s.fields[tname] = make(map[string]string)
		}
		return s.fields[tname]
	}

	for _, name := range info.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(info.Dir, name), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s/%s: %v", importPath, name, err)
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch sp := spec.(type) {
					case *ast.TypeSpec:
						tname := sp.Name.Name
						if isDocDeprecated(sp.Doc) || (len(d.Specs) == 1 && isDocDeprecated(d.Doc)) {
							s.deprecated["type:"+tname] = true
						}
						if sp.Name.IsExported() {
							s.types[tname] = true
						}
						if st, ok := sp.Type.(*ast.StructType); ok && st.Fields != nil {
							for _, field := range st.Fields.List {
								if len(field.Names) == 0 {
									if emb := localTypeName(field.Type); emb != "" {
										embedded[tname] = append(embedded[tname], emb)
									}
									continue
								}
								ftype := normalizeExpr(fset, field.Type)
								fdep := isDocDeprecated(field.Doc) || isDocDeprecated(field.Comment)
								for _, id := range field.Names {
									if id.IsExported() {
										ensureFields(tname)[id.Name] = ftype
										if fdep {
											s.deprecated["field:"+tname+"."+id.Name] = true
										}
									}
								}
							}
						}
						if iface, ok := sp.Type.(*ast.InterfaceType); ok && iface.Methods != nil {
							for _, field := range iface.Methods.List {
								if len(field.Names) == 0 {
									if emb := localTypeName(field.Type); emb != "" {
										embedded[tname] = append(embedded[tname], emb)
									}
									continue
								}
								ft, ok := field.Type.(*ast.FuncType)
								if !ok {
									continue
								}
								sig := normalizeFuncType(fset, ft)
								for _, id := range field.Names {
									if id.IsExported() {
										ensureMethods(tname)[id.Name] = sig
									}
								}
							}
						}
					case *ast.ValueSpec:
						dep := isDocDeprecated(sp.Doc) || (len(d.Specs) == 1 && isDocDeprecated(d.Doc))
						for _, id := range sp.Names {
							if !id.IsExported() {
								continue
							}
							if d.Tok == token.CONST {
								s.consts[id.Name] = true
								if dep {
									s.deprecated["const:"+id.Name] = true
								}
							} else if d.Tok == token.VAR {
								s.vars[id.Name] = true
								if dep {
									s.deprecated["var:"+id.Name] = true
								}
							}
						}
					}
				}
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				dep := isDocDeprecated(d.Doc)
				sig := normalizeFuncType(fset, d.Type)
				if d.Recv == nil || len(d.Recv.List) == 0 {
					s.funcs[d.Name.Name] = sig
					if dep {
						s.deprecated["func:"+d.Name.Name] = true
					}
					continue
				}
				if tname := localTypeName(d.Recv.List[0].Type); tname != "" {
					ensureMethods(tname)[d.Name.Name] = sig
					if dep {
						s.deprecated["method:"+tname+"."+d.Name.Name] = true
					}
				}
			}
		}
	}

	// Propagate promoted methods and exported fields from embedded types.
	for changed := true; changed; {
		changed = false
		for tname, embs := range embedded {
			dstM := ensureMethods(tname)
			dstF := ensureFields(tname)
			for _, emb := range embs {
				for m, sig := range s.methods[emb] {
					if _, exists := dstM[m]; !exists {
						dstM[m] = sig
						changed = true
					}
				}
				for f, ftyp := range s.fields[emb] {
					if _, exists := dstF[f]; !exists {
						dstF[f] = ftyp
						changed = true
					}
				}
			}
		}
	}
	return s
}

func normalizeExpr(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, expr)
	s := strings.Join(strings.Fields(buf.String()), " ")
	// Normalize Go 1.18+ 'any' alias and Go 1.20 'interface{}' identically.
	s = strings.ReplaceAll(s, "interface{}", "any")
	s = strings.ReplaceAll(s, "interface{ }", "any")
	return s
}

func normalizeFieldList(fset *token.FileSet, fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return "()"
	}
	var parts []string
	for _, f := range fl.List {
		typ := normalizeExpr(fset, f.Type)
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			parts = append(parts, typ)
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func normalizeFuncType(fset *token.FileSet, ft *ast.FuncType) string {
	return "func" + normalizeFieldList(fset, ft.Params) + " " + normalizeFieldList(fset, ft.Results)
}

func localTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return localTypeName(e.X)
	case *ast.IndexExpr:
		return localTypeName(e.X)
	case *ast.IndexListExpr:
		return localTypeName(e.X)
	case *ast.Ident:
		return e.Name
	}
	return ""
}

func isDocDeprecated(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if strings.Contains(c.Text, "Deprecated:") {
			return true
		}
	}
	return false
}

func assertXToolsParity(
	t *testing.T,
	llgoPkg, upstreamPkg string,
	allowedOmitted map[string]bool,
	allowedAdded map[string]bool,
) {
	t.Helper()
	got := loadPkgAPISurface(t, llgoPkg)
	want := loadPkgAPISurface(t, upstreamPkg)

	var missing, extra, mismatches []string

	checkBoolMap := func(kind string, wantMap, gotMap map[string]bool) {
		for name := range wantMap {
			key := kind + ":" + name
			if want.deprecated[key] {
				continue
			}
			if !gotMap[name] && !allowedOmitted[key] {
				missing = append(missing, key)
			}
		}
		for name := range gotMap {
			key := kind + ":" + name
			if !wantMap[name] && !allowedAdded[key] {
				extra = append(extra, key)
			}
		}
	}

	checkBoolMap("type", want.types, got.types)
	checkBoolMap("var", want.vars, got.vars)
	checkBoolMap("const", want.consts, got.consts)

	// Check package-level functions AND exact signatures.
	for name, wantSig := range want.funcs {
		key := "func:" + name
		if want.deprecated[key] {
			continue
		}
		gotSig, ok := got.funcs[name]
		if !ok {
			if !allowedOmitted[key] {
				missing = append(missing, key)
			}
			continue
		}
		if gotSig != wantSig {
			mismatches = append(mismatches, key+" got="+gotSig+" want="+wantSig)
		}
	}
	for name := range got.funcs {
		key := "func:" + name
		if _, ok := want.funcs[name]; !ok && !allowedAdded[key] {
			extra = append(extra, key)
		}
	}

	// Check methods (names + exact signatures) and exported struct fields (names + exact types).
	for tname := range want.types {
		if want.deprecated["type:"+tname] || !got.types[tname] {
			continue
		}
		for m, wantSig := range want.methods[tname] {
			key := "method:" + tname + "." + m
			if want.deprecated[key] {
				continue
			}
			gotSig, ok := got.methods[tname][m]
			if !ok {
				if !allowedOmitted[key] {
					missing = append(missing, key)
				}
				continue
			}
			if gotSig != wantSig {
				mismatches = append(mismatches, key+" got="+gotSig+" want="+wantSig)
			}
		}
		for m := range got.methods[tname] {
			key := "method:" + tname + "." + m
			if _, ok := want.methods[tname][m]; !ok && !allowedAdded[key] {
				extra = append(extra, key)
			}
		}
		for f, wantType := range want.fields[tname] {
			key := "field:" + tname + "." + f
			if want.deprecated[key] {
				continue
			}
			gotType, ok := got.fields[tname][f]
			if !ok {
				if !allowedOmitted[key] {
					missing = append(missing, key)
				}
				continue
			}
			if gotType != wantType {
				mismatches = append(mismatches, key+" got="+gotType+" want="+wantType)
			}
		}
		for f := range got.fields[tname] {
			key := "field:" + tname + "." + f
			if _, ok := want.fields[tname][f]; !ok && !allowedAdded[key] {
				extra = append(extra, key)
			}
		}
	}

	var staleOmit, staleAdd []string
	for key := range allowedOmitted {
		if hasSurfaceKey(got, key) {
			staleOmit = append(staleOmit, key)
		}
	}
	for key := range allowedAdded {
		if !hasSurfaceKey(got, key) || hasSurfaceKey(want, key) {
			staleAdd = append(staleAdd, key)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(mismatches)
	sort.Strings(staleOmit)
	sort.Strings(staleAdd)

	if len(missing) > 0 {
		t.Errorf("%s vs %s — MISSING non-deprecated upstream APIs (%d): %v", llgoPkg, upstreamPkg, len(missing), missing)
	}
	if len(extra) > 0 {
		t.Errorf("%s vs %s — UNAPPROVED extra APIs (%d): %v", llgoPkg, upstreamPkg, len(extra), extra)
	}
	if len(mismatches) > 0 {
		t.Errorf("%s vs %s — SIGNATURE/TYPE MISMATCHES (%d):\n  %s", llgoPkg, upstreamPkg, len(mismatches), strings.Join(mismatches, "\n  "))
	}
	if len(staleOmit) > 0 {
		t.Errorf("%s vs %s — stale allowedOmitted entries: %v", llgoPkg, upstreamPkg, staleOmit)
	}
	if len(staleAdd) > 0 {
		t.Errorf("%s vs %s — stale allowedAdded entries: %v", llgoPkg, upstreamPkg, staleAdd)
	}
}

func hasSurfaceKey(s *pkgAPISurface, key string) bool {
	kind, rest, ok := strings.Cut(key, ":")
	if !ok {
		return false
	}
	switch kind {
	case "type":
		return s.types[rest]
	case "func":
		_, ok := s.funcs[rest]
		return ok
	case "var":
		return s.vars[rest]
	case "const":
		return s.consts[rest]
	case "method":
		tname, mname, ok := strings.Cut(rest, ".")
		if !ok {
			return false
		}
		_, exists := s.methods[tname][mname]
		return exists
	case "field":
		tname, fname, ok := strings.Cut(rest, ".")
		if !ok {
			return false
		}
		_, exists := s.fields[tname][fname]
		return exists
	}
	return false
}

// TestXToolsPackageParityGuards enforces 1:1 type, field, and method-signature
// compatibility with golang.org/x/tools/go/* across every upstream-mirrored
// root pure-Go package and rejects unapproved new exports.
func TestXToolsPackageParityGuards(t *testing.T) {
	t.Run("types/typeutil", func(t *testing.T) {
		assertXToolsParity(t,
			"github.com/xgo-dev/llgo/types/typeutil",
			"golang.org/x/tools/go/types/typeutil",
			nil,
			nil,
		)
	})

	t.Run("exportdata", func(t *testing.T) {
		assertXToolsParity(t,
			"github.com/xgo-dev/llgo/exportdata",
			"golang.org/x/tools/go/gcexportdata",
			map[string]bool{
				// gc toolchain .a archive header helpers do not apply to LLGo's binary format.
				"func:Find":      true,
				"func:NewReader": true,
			},
			nil,
		)
	})

	t.Run("callgraph", func(t *testing.T) {
		assertXToolsParity(t,
			"github.com/xgo-dev/llgo/callgraph",
			"golang.org/x/tools/go/callgraph",
			nil,
			nil,
		)
	})

	t.Run("callgraph/rta", func(t *testing.T) {
		assertXToolsParity(t,
			"github.com/xgo-dev/llgo/callgraph/rta",
			"golang.org/x/tools/go/callgraph/rta",
			nil,
			nil,
		)
	})

	t.Run("ssa", func(t *testing.T) {
		assertXToolsParity(t,
			"github.com/xgo-dev/llgo/ssa",
			"golang.org/x/tools/go/ssa",
			nil,
			map[string]bool{
				// Verified LLGo-specific metadata fields:
				"field:Alloc.NoScan":             true,
				"field:CallCommon.RecvNilCheck":  true,
				"field:Function.ExportName":      true,
				"field:Function.HasClosureEnv":   true,
				"field:Function.LinkName":        true,
				"field:Function.NoEscape":        true,
				"field:Function.NoInline":        true,
				"field:Function.NoInterface":     true,
				"field:Function.WasmExport":      true,
				"field:Function.WasmImport":      true,
				"field:Global.ExportName":        true,
				"field:Global.LinkName":          true,
				"field:Global.Locality":          true,
				"field:Global.StaticBytes":       true,
				"field:MakeClosure.RecvNilCheck": true,
				"field:Package.Kind":             true,
				"field:Package.KindParam":        true,
				"field:Package.RetainInit":       true,
				"field:Package.SkipAll":          true,
				"field:Package.Skips":            true,
				"field:Program.Target":           true,
				// Verified LLGo-specific semantic types, constants, and methods:
				"type:CallKind":                       true,
				"type:Intrinsic":                      true,
				"type:ReceiverNilCheck":               true,
				"func:LookupIntrinsic":                true,
				"method:CallCommon.CallKind":          true,
				"method:CallCommon.Intrinsic":         true,
				"method:CallCommon.IsCVariadic":       true,
				"method:Function.CallConv":            true,
				"method:Function.CallKind":            true,
				"method:Function.Intrinsic":           true,
				"method:Function.LLVMIntrinsic":       true,
				"method:Function.LinkSymbol":          true,
				"method:Global.LinkTarget":            true,
				"method:Package.IsCPackage":           true,
				"method:Package.PyModule":             true,
				"method:Package.Skipped":              true,
				"method:Package.SkipsInit":            true,
				"const:CallGo":                        true,
				"const:CallC":                         true,
				"const:CallStdcall":                   true,
				"const:CallPy":                        true,
				"const:CallWasm":                      true,
				"const:CallLLGoIntrinsic":             true,
				"const:CallLLVMIntrinsic":             true,
				"const:IntrinsicNone":                 true,
				"const:IntrinsicUnreachable":          true,
				"const:IntrinsicCStr":                 true,
				"const:IntrinsicAlloca":               true,
				"const:IntrinsicAllocaCStr":           true,
				"const:IntrinsicAllocaCStrs":          true,
				"const:IntrinsicAllocCStr":            true,
				"const:IntrinsicAdvance":              true,
				"const:IntrinsicIndex":                true,
				"const:IntrinsicStringData":           true,
				"const:IntrinsicString":               true,
				"const:IntrinsicDeferData":            true,
				"const:IntrinsicSigjmpbuf":            true,
				"const:IntrinsicSigsetjmp":            true,
				"const:IntrinsicSiglongjmp":           true,
				"const:IntrinsicSetjmp":               true,
				"const:IntrinsicLongjmp":              true,
				"const:IntrinsicFuncAddr":             true,
				"const:IntrinsicFuncPCABI0":           true,
				"const:IntrinsicPyList":               true,
				"const:IntrinsicPyStr":                true,
				"const:IntrinsicPyTuple":              true,
				"const:IntrinsicAtomicLoad":           true,
				"const:IntrinsicAtomicStore":          true,
				"const:IntrinsicAtomicStoreUnordered": true,
				"const:IntrinsicAtomicAddRelaxed":     true,
				"const:IntrinsicAtomicCmpXchg":        true,
				"const:IntrinsicAtomicCmpXchgOK":      true,
				"const:IntrinsicAtomicAddReturnNew":   true,
				"const:IntrinsicAtomicXchg":           true,
				"const:IntrinsicAtomicAdd":            true,
				"const:IntrinsicAtomicSub":            true,
				"const:IntrinsicAtomicAnd":            true,
				"const:IntrinsicAtomicNand":           true,
				"const:IntrinsicAtomicOr":             true,
				"const:IntrinsicAtomicXor":            true,
				"const:IntrinsicAtomicMax":            true,
				"const:IntrinsicAtomicMin":            true,
				"const:IntrinsicAtomicUMax":           true,
				"const:IntrinsicAtomicUMin":           true,
				"const:IntrinsicCgoCString":           true,
				"const:IntrinsicCgoCBytes":            true,
				"const:IntrinsicCgoGoString":          true,
				"const:IntrinsicCgoGoStringN":         true,
				"const:IntrinsicCgoGoBytes":           true,
				"const:IntrinsicCgoCMalloc":           true,
				"const:IntrinsicCgoCheckPointer":      true,
				"const:IntrinsicCgoCgocall":           true,
				"const:IntrinsicAsm":                  true,
				"const:IntrinsicStackSave":            true,
				"const:IntrinsicSkip":                 true,
				"const:IntrinsicSyscall":              true,
				"const:IntrinsicBoolToUint8":          true,
				"const:IntrinsicClosureEnv":           true,
				"const:IntrinsicFloat32FromBits":      true,
				"const:IntrinsicFloat32Bits":          true,
				"const:IntrinsicFloat64FromBits":      true,
				"const:IntrinsicFloat64Bits":          true,
				"const:IntrinsicUMulOverflow":         true,
				"const:IntrinsicSIMD":                 true,
				"const:IntrinsicInterfaceKindUsed":    true,
				"const:IntrinsicAESRound":             true,
				"const:IntrinsicAESSubWord":           true,
				"const:IntrinsicAESAlloca":            true,
				"const:IntrinsicKeepAlive":            true,
				"const:IntrinsicNoEscape":             true,
				// Minimal verified IR construction & verification hooks required by
				// frontend lowering (cmd/compiler/cl/lower_ssa.go) and pure-Go tests:
				"func:NewBuiltin":               true,
				"func:VerifyFunction":           true,
				"func:VerifyPackage":            true,
				"method:BasicBlock.AddEdge":     true,
				"method:BasicBlock.Emit":        true,
				"method:Function.AddAnonFunc":   true,
				"method:Function.AddFreeVar":    true,
				"method:Function.AddParam":      true,
				"method:Function.Finish":        true,
				"method:Function.NewBasicBlock": true,
				"method:Package.AddConst":       true,
				"method:Package.AddFunction":    true,
				"method:Package.AddGlobal":      true,
				"method:Package.AddType":        true,
				"method:Program.AddRuntimeType": true,
				"method:Program.RegisterMethod": true,
			},
		)
	})

	t.Run("ssa/ssautil", func(t *testing.T) {
		assertXToolsParity(t,
			"github.com/xgo-dev/llgo/ssa/ssautil",
			"golang.org/x/tools/go/ssa/ssautil",
			map[string]bool{
				// Packages and AllPackages accept golang.org/x/tools/go/packages.Package,
				// which root pure-Go core packages intentionally do not import.
				"func:Packages":    true,
				"func:AllPackages": true,
			},
			nil,
		)
	})
}
