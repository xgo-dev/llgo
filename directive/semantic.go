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

package directive

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
)

// CallConv identifies a native calling convention declared via //llgo:cdecl,
// //llgo:stdcall, or legacy //llgo:type C / //llgo:type stdcall.
type CallConv uint8

const (
	CallConvNone CallConv = iota
	CallConvC
	CallConvStdcall
)

func (c CallConv) String() string {
	switch c {
	case CallConvC:
		return "C"
	case CallConvStdcall:
		return "stdcall"
	default:
		return ""
	}
}

// TypeCallConvResult captures a parsed calling-convention directive on a type
// declaration along with deprecation metadata for legacy //llgo:type spellings.
type TypeCallConvResult struct {
	CallConv    CallConv
	Directive   string
	Replacement string
	Deprecated  bool
	Pos         token.Pos
}

// ParseTypeCallConv extracts the calling-convention directive attached to a
// type declaration, rejecting arguments on //llgo:cdecl / //llgo:stdcall and
// conflicting calling conventions in the same doc group.
func ParseTypeCallConv(doc *ast.CommentGroup) (TypeCallConvResult, error) {
	var res TypeCallConvResult
	for _, d := range ParseGroup(doc) {
		var next CallConv
		var name, replacement string
		var deprecated bool
		switch d.Name {
		case "llgo:cdecl":
			if d.Args != "" {
				return TypeCallConvResult{}, fmt.Errorf("//llgo:cdecl does not accept arguments")
			}
			next, name = CallConvC, "//llgo:cdecl"
		case "llgo:stdcall":
			if d.Args != "" {
				return TypeCallConvResult{}, fmt.Errorf("//llgo:stdcall does not accept arguments")
			}
			next, name = CallConvStdcall, "//llgo:stdcall"
		case "llgo:type":
			switch d.Args {
			case "C":
				next, name, replacement, deprecated = CallConvC, "//llgo:type C", "//llgo:cdecl", true
			case "stdcall":
				next, name, replacement, deprecated = CallConvStdcall, "//llgo:type stdcall", "//llgo:stdcall", true
			default:
				continue
			}
		default:
			continue
		}
		if res.CallConv != CallConvNone && res.CallConv != next {
			return TypeCallConvResult{}, fmt.Errorf("conflicting calling convention directives %s and %s", res.Directive, name)
		}
		if res.CallConv == CallConvNone {
			res = TypeCallConvResult{
				CallConv:    next,
				Directive:   name,
				Replacement: replacement,
				Deprecated:  deprecated,
				Pos:         d.Pos,
			}
		}
	}
	return res, nil
}

// Locality identifies the execution context that owns a package variable
// annotated with //llgointernal:tls or //llgointernal:gls.
type Locality uint8

const (
	LocalityNone Locality = iota
	LocalityThread
	LocalityGoroutine
)

func (l Locality) String() string {
	switch l {
	case LocalityThread:
		return "tls"
	case LocalityGoroutine:
		return "gls"
	default:
		return ""
	}
}

// ParseLocality extracts a locality directive from doc. Legacy //llgo:tls and
// //llgo:gls spellings and conflicting directives return an error.
func ParseLocality(doc *ast.CommentGroup) (Locality, token.Pos, error) {
	kind := LocalityNone
	pos := token.NoPos
	for _, d := range ParseGroup(doc) {
		var next Locality
		switch d.Name {
		case "llgointernal:tls":
			next = LocalityThread
		case "llgointernal:gls":
			next = LocalityGoroutine
		case "llgo:tls":
			return LocalityNone, d.Pos, fmt.Errorf("//llgo:tls is not supported; use //llgointernal:tls")
		case "llgo:gls":
			return LocalityNone, d.Pos, fmt.Errorf("//llgo:gls is not supported; use //llgointernal:gls")
		default:
			continue
		}
		if d.Args != "" {
			return LocalityNone, d.Pos, fmt.Errorf("//%s does not accept arguments", d.Name)
		}
		if kind != LocalityNone && kind != next {
			return LocalityNone, d.Pos, fmt.Errorf("conflicting locality directives")
		}
		kind, pos = next, d.Pos
	}
	return kind, pos, nil
}

// LinkSpec describes a //go:linkname or //llgo:link directive.
type LinkSpec struct {
	Directive string // "go:linkname" or "llgo:link"
	Local     string
	Target    string
	Pos       token.Pos
}

// ParseLinks extracts all //go:linkname and //llgo:link directives from doc.
func ParseLinks(doc *ast.CommentGroup) []LinkSpec {
	var out []LinkSpec
	for _, d := range ParseGroup(doc) {
		if d.Name != "go:linkname" && d.Name != "llgo:link" {
			continue
		}
		local, target := ParseLinknameArgs(d.Args)
		out = append(out, LinkSpec{
			Directive: d.Name,
			Local:     local,
			Target:    target,
			Pos:       d.Pos,
		})
	}
	return out
}

// WasmImport describes a //go:wasmimport <module> <name> directive.
type WasmImport struct {
	Module string
	Name   string
	Pos    token.Pos
}

// ParseWasmImport extracts the last //go:wasmimport directive in doc, if any.
func ParseWasmImport(doc *ast.CommentGroup) (WasmImport, bool) {
	directives := ParseGroup(doc)
	for i := len(directives) - 1; i >= 0; i-- {
		d := directives[i]
		if d.Name != "go:wasmimport" {
			continue
		}
		fields := strings.Fields(d.Args)
		if len(fields) == 2 {
			return WasmImport{Module: fields[0], Name: fields[1], Pos: d.Pos}, true
		}
	}
	return WasmImport{}, false
}

// ParseExport extracts a //export <name> directive from doc.
func ParseExport(doc *ast.CommentGroup) (string, token.Pos, bool) {
	directives := ParseGroup(doc)
	for i := len(directives) - 1; i >= 0; i-- {
		d := directives[i]
		if d.Name == "export" && d.Args != "" {
			return d.Args, d.Pos, true
		}
	}
	return "", token.NoPos, false
}

// HasClosureEnv reports whether doc contains a //llgo:env directive.
func HasClosureEnv(doc *ast.CommentGroup) bool {
	for _, d := range ParseGroup(doc) {
		if d.Name == "llgo:env" {
			return true
		}
	}
	return false
}

// SkipSpec describes //llgo:skip and //llgo:skipall directives on a package.
type SkipSpec struct {
	All   bool
	Names []string
}

// ParseSkip extracts //llgo:skip and //llgo:skipall directives from doc.
func ParseSkip(doc *ast.CommentGroup) SkipSpec {
	var spec SkipSpec
	for _, d := range ParseGroup(doc) {
		switch d.Name {
		case "llgo:skipall":
			spec.All = true
		case "llgo:skip":
			spec.Names = append(spec.Names, strings.Fields(d.Args)...)
		}
	}
	return spec
}

// PackageKind classifies a package's LLGoPackage declaration.
type PackageKind uint8

const (
	PkgNormal PackageKind = iota
	PkgLLGo
	PkgPyModule
	PkgNoInit
	PkgDeclOnly
	PkgLinkIR
	PkgLinkExtern
)

// ParsePackageKind parses the string value of an LLGoPackage constant.
func ParsePackageKind(v string) (PackageKind, string) {
	if pos := strings.IndexByte(v, ':'); pos > 0 {
		kind, param := v[:pos], strings.TrimSpace(v[pos+1:])
		switch kind {
		case "link":
			return PkgLinkExtern, param
		case "decl":
			return PkgDeclOnly, param
		case "py":
			return PkgPyModule, param
		case "llgo":
			return PkgLinkIR, param
		}
	}
	switch v {
	case "decl":
		return PkgDeclOnly, ""
	case "noinit":
		return PkgNoInit, ""
	default:
		return PkgLLGo, v
	}
}

// PackageKindFromScope inspects a package scope for an LLGoPackage constant.
func PackageKindFromScope(scope *types.Scope) (PackageKind, string) {
	if scope == nil {
		return PkgNormal, ""
	}
	if c, ok := scope.Lookup("LLGoPackage").(*types.Const); ok {
		if v := c.Val(); v != nil && v.Kind() == constant.String {
			return ParsePackageKind(constant.StringVal(v))
		}
		return PkgLLGo, ""
	}
	return PkgNormal, ""
}

// LLGoFiles returns the string value of the LLGoFiles constant in scope, if set.
func LLGoFiles(scope *types.Scope) (string, bool) {
	if scope == nil {
		return "", false
	}
	if c, ok := scope.Lookup("LLGoFiles").(*types.Const); ok {
		if v := c.Val(); v != nil && v.Kind() == constant.String {
			return constant.StringVal(v), true
		}
	}
	return "", false
}
