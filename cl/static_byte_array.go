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
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/locality"
	"golang.org/x/tools/go/ssa"
)

// minStripStaticByteArray is the number of composite-literal elements at which
// a package-level [N]byte / [...]byte initializer is taken out of go/ssa init
// (IndexAddr+Store per element) and restored as an LLVM constant. cmd/compile
// places these in writable static data via staticinit; go/ssa does not.
const minStripStaticByteArray = 4096

// StaticByteArrayKey is the lookup key for a stripped package-level byte array.
// pkgPath must be types.Package.Path(), not llssa.PathOf: PathOf maps every
// ordinary main package to "main" and would collide across commands.
func StaticByteArrayKey(pkgPath, name string) string {
	if pkgPath == "" {
		return name
	}
	return pkgPath + "." + name
}

// MergeStaticByteArrays copies src into dst. Empty src is ignored.
func MergeStaticByteArrays(dst, src map[string][]byte) map[string][]byte {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string][]byte, len(src))
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// StripLargeStaticByteArrays extracts payload from large package-level byte
// array literals and replaces `var x = [N]byte{...}` with `var x [N]byte`.
// Clearing only CompositeLit.Elts leaves `= [N]byte{}`, and go/ssa then
// memclr's large arrays in init, wiping the restored LLVM initializer.
// The returned map is keyed by StaticByteArrayKey of pkg.Path() and the
// variable name, matching initStaticByteArrayGlobal lookup. Call this after
// type checking and before ssa.Package.Build. Already-stripped specs (no
// values) are skipped so a second call does not replace a previous payload
// with zeros. info.InitOrder entries for stripped variables are removed so
// go/ssa does not rebuild the initializer from the type checker.
func StripLargeStaticByteArrays(pkg *types.Package, files []*ast.File, info *types.Info) map[string][]byte {
	return StripLargeStaticByteArraysExcept(pkg, files, info, nil)
}

// StripLargeStaticByteArraysExcept is StripLargeStaticByteArrays, leaving
// names in skipped untouched. Partial patches collect skipped from the
// alternate package via SkipNamesRetainingInit: processPkg omits those
// original globals, so their InitOrder stores are the only initialization
// of the replacement when original init is retained.
func StripLargeStaticByteArraysExcept(pkg *types.Package, files []*ast.File, info *types.Info, skipped map[string]bool) map[string][]byte {
	out := make(map[string][]byte)
	pkgPath := ""
	if pkg != nil {
		pkgPath = pkg.Path()
	}
	aliases := linknameAliasNames(files)
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if len(vs.Names) != 1 || len(vs.Values) != 1 {
					continue
				}
				name := vs.Names[0]
				if name == nil || name.Name == "_" {
					continue
				}
				if skipped[name.Name] || hasLocalityDirective(gd, vs) || aliases[name.Name] {
					continue
				}
				cl, ok := vs.Values[0].(*ast.CompositeLit)
				if !ok || len(cl.Elts) == 0 {
					continue
				}
				data, ok := staticByteArrayLit(cl)
				if !ok {
					continue
				}
				if !byteArrayVar(info, name, len(data)) {
					continue
				}
				vs.Type = &ast.ArrayType{
					Len: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(len(data))},
					Elt: &ast.Ident{Name: "byte"},
				}
				vs.Values = nil
				if info != nil {
					if v, _ := info.Defs[name].(*types.Var); v != nil {
						dropStaticByteArrayInit(info, v)
					}
				}
				out[StaticByteArrayKey(pkgPath, name.Name)] = data
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SkipNamesRetainingInit returns original globals that must keep executable
// initialization. The alternate package skips them, but original init still
// runs; processPkg then omits the original global, so restoring via
// ConstByteArray never happens. skipall and //llgo:skip init drop original
// init, so stripping those arrays remains valid.
func SkipNamesRetainingInit(altFiles []*ast.File) map[string]bool {
	names, skipall := patchSkipNames(altFiles)
	if skipall || names["init"] {
		return nil
	}
	return names
}

func patchSkipNames(files []*ast.File) (map[string]bool, bool) {
	ctx := &context{skips: make(map[string]none)}
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			switch gd.Tok {
			case token.CONST, token.TYPE:
				ctx.collectSkipNamesByDoc(gd.Doc)
			case token.IMPORT:
				if doc := gd.Doc; doc != nil {
					if n := len(doc.List); n > 0 {
						ctx.collectSkipNames(doc.List[n-1].Text)
					}
				}
			}
		}
	}
	if len(ctx.skips) == 0 && !ctx.skipall {
		return nil, false
	}
	names := make(map[string]bool, len(ctx.skips))
	for name := range ctx.skips {
		names[name] = true
	}
	return names, ctx.skipall
}

func dropStaticByteArrayInit(info *types.Info, v *types.Var) {
	if info == nil || v == nil {
		return
	}
	kept := info.InitOrder[:0]
	for _, init := range info.InitOrder {
		if len(init.Lhs) == 1 && init.Lhs[0] == v {
			continue
		}
		kept = append(kept, init)
	}
	info.InitOrder = kept
}

func staticByteArrayLit(cl *ast.CompositeLit) ([]byte, bool) {
	if cl == nil {
		return nil, false
	}
	at, ok := cl.Type.(*ast.ArrayType)
	if !ok || at.Len == nil || !isByteIdent(at.Elt) {
		return nil, false
	}
	if len(cl.Elts) < minStripStaticByteArray {
		return nil, false
	}
	n, ok := staticByteArrayLen(at, cl)
	if !ok || n < minStripStaticByteArray {
		return nil, false
	}
	buf := make([]byte, n)
	next := 0
	for _, elt := range cl.Elts {
		idx := next
		val := elt
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			idx, ok = astIntLit(kv.Key)
			if !ok {
				return nil, false
			}
			val = kv.Value
		} else {
			next++
		}
		b, ok := astByteLit(val)
		if !ok || idx < 0 || idx >= n {
			return nil, false
		}
		buf[idx] = b
		if _, isKV := elt.(*ast.KeyValueExpr); isKV {
			next = idx + 1
		}
	}
	return buf, true
}

func staticByteArrayLen(at *ast.ArrayType, cl *ast.CompositeLit) (int, bool) {
	if _, ok := at.Len.(*ast.Ellipsis); ok {
		n := 0
		next := 0
		for _, elt := range cl.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				idx, ok := astIntLit(kv.Key)
				if !ok {
					return 0, false
				}
				if idx+1 > n {
					n = idx + 1
				}
				next = idx + 1
				continue
			}
			next++
			if next > n {
				n = next
			}
		}
		return n, n > 0
	}
	return astIntLit(at.Len)
}

func staticByteArrayDeclKey(gbl *ssa.Global) string {
	if gbl == nil {
		return ""
	}
	if obj := gbl.Object(); obj != nil {
		if pkg := obj.Pkg(); pkg != nil {
			return StaticByteArrayKey(pkg.Path(), obj.Name())
		}
	}
	if gbl.Pkg != nil && gbl.Pkg.Pkg != nil {
		return StaticByteArrayKey(gbl.Pkg.Pkg.Path(), gbl.Name())
	}
	return ""
}

func linknameAliasNames(files []*ast.File) map[string]bool {
	out := make(map[string]bool)
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, group := range file.Comments {
			collectLinknameAliasNames(group, out)
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			collectLinknameAliasNames(gd.Doc, out)
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					collectLinknameAliasNames(vs.Doc, out)
					collectLinknameAliasNames(vs.Comment, out)
				}
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func collectLinknameAliasNames(group *ast.CommentGroup, out map[string]bool) {
	for _, item := range directive.ParseGroup(group) {
		if item.Name != "go:linkname" && item.Name != "llgo:link" {
			continue
		}
		fields := strings.Fields(item.Args)
		if len(fields) < 2 {
			continue
		}
		out[fields[0]] = true
	}
}

func hasLocalityDirective(decl *ast.GenDecl, spec *ast.ValueSpec) bool {
	groups := []*ast.CommentGroup{decl.Doc}
	if spec != nil {
		groups = append(groups, spec.Doc, spec.Comment)
	}
	for _, group := range groups {
		kind, _, err := locality.FromDoc(nil, group)
		if err == nil && kind != locality.None {
			return true
		}
	}
	return false
}

func byteArrayVar(info *types.Info, name *ast.Ident, n int) bool {
	if info == nil || name == nil {
		return true
	}
	v, _ := info.Defs[name].(*types.Var)
	if v == nil {
		return false
	}
	arr, ok := v.Type().Underlying().(*types.Array)
	if !ok || arr.Len() != int64(n) {
		return false
	}
	basic, ok := arr.Elem().Underlying().(*types.Basic)
	return ok && (basic.Kind() == types.Byte || basic.Kind() == types.Uint8)
}

func isByteIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && (id.Name == "byte" || id.Name == "uint8")
}

func astIntLit(e ast.Expr) (int, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	n, err := strconv.ParseInt(lit.Value, 0, 32)
	if err != nil || n < 0 {
		return 0, false
	}
	return int(n), true
}

func astByteLit(e ast.Expr) (byte, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok {
		return 0, false
	}
	switch lit.Kind {
	case token.INT:
		n, err := strconv.ParseUint(lit.Value, 0, 8)
		if err != nil {
			return 0, false
		}
		return byte(n), true
	case token.CHAR:
		s, err := strconv.Unquote(lit.Value)
		if err != nil || len(s) != 1 {
			return 0, false
		}
		return s[0], true
	default:
		return 0, false
	}
}
