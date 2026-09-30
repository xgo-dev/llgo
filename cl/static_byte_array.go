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

	llssa "github.com/xgo-dev/llgo/ssa"
)

// minStripStaticByteArray is the number of composite-literal elements at which
// a package-level [N]byte / [...]byte initializer is taken out of go/ssa init
// (IndexAddr+Store per element) and restored as an LLVM constant. cmd/compile
// places these in writable static data via staticinit; go/ssa does not.
const minStripStaticByteArray = 4096

// StaticByteArrayKey is the lookup key for a stripped package-level byte array.
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
// array literals and clears CompositeLit.Elts so ssa.Package.Build does not
// emit one store per element. The returned map is keyed by StaticByteArrayKey
// of llssa.PathOf(pkg) and the variable name, matching initStaticByteArrayGlobal
// lookup. Call this after type checking and before ssa.Package.Build.
// Already-stripped literals (empty Elts) are skipped so a second call does not
// replace a previous payload with zeros.
func StripLargeStaticByteArrays(pkg *types.Package, files []*ast.File) map[string][]byte {
	out := make(map[string][]byte)
	pkgPath := llssa.PathOf(pkg)
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
				for i, name := range vs.Names {
					if name == nil || name.Name == "_" || i >= len(vs.Values) {
						continue
					}
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok || len(cl.Elts) == 0 {
						continue
					}
					data, ok := staticByteArrayLit(cl)
					if !ok {
						continue
					}
					cl.Elts = nil
					out[StaticByteArrayKey(pkgPath, name.Name)] = data
				}
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
			if idx+1 > next {
				next = idx + 1
			}
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
				if idx+1 > next {
					next = idx + 1
				}
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
