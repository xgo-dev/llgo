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
	"go/token"
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
// multiple calling-convention directives in the same doc group.
// On error, the returned TypeCallConvResult carries the offending directive Pos.
func ParseTypeCallConv(doc *ast.CommentGroup) (TypeCallConvResult, error) {
	var res TypeCallConvResult
	for _, d := range ParseGroup(doc) {
		var next CallConv
		var name, replacement string
		var deprecated bool
		switch d.Name {
		case "llgo:cdecl":
			if d.Args != "" {
				return TypeCallConvResult{Pos: d.Pos}, fmt.Errorf("//llgo:cdecl does not accept arguments")
			}
			next, name = CallConvC, "//llgo:cdecl"
		case "llgo:stdcall":
			if d.Args != "" {
				return TypeCallConvResult{Pos: d.Pos}, fmt.Errorf("//llgo:stdcall does not accept arguments")
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
		if res.CallConv != CallConvNone {
			return TypeCallConvResult{Pos: d.Pos}, fmt.Errorf("conflicting calling-convention directives on one type: %s and %s", res.Directive, name)
		}
		res = TypeCallConvResult{
			CallConv:    next,
			Directive:   name,
			Replacement: replacement,
			Deprecated:  deprecated,
			Pos:         d.Pos,
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

// ParseLocality extracts a locality directive from doc. Legacy //llgo:threadlocal
// and //llgo:goroutinelocal spellings, arguments, and conflicting directives
// return an error along with the offending directive position.
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
		case "llgo:threadlocal":
			return LocalityNone, d.Pos, fmt.Errorf("//llgo:threadlocal is not supported; use //llgointernal:tls")
		case "llgo:goroutinelocal":
			return LocalityNone, d.Pos, fmt.Errorf("//llgo:goroutinelocal is not supported; use //llgointernal:gls")
		default:
			continue
		}
		if d.Args != "" {
			return LocalityNone, d.Pos, fmt.Errorf("//%s does not accept arguments", d.Name)
		}
		if kind != LocalityNone && kind != next {
			return LocalityNone, d.Pos, fmt.Errorf("//llgointernal:tls and //llgointernal:gls cannot apply to the same variable declaration")
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

// ParseWasmImport inspects the last //go:wasmimport directive in doc.
//
// Mirroring the compiler frontend, the last //go:wasmimport directive always
// wins even if it is malformed:
//   - Absent: returns (WasmImport{}, false, nil).
//   - Valid: returns (WasmImport{...}, true, nil).
//   - Malformed: returns (WasmImport{Pos: pos}, true, err).
func ParseWasmImport(doc *ast.CommentGroup) (WasmImport, bool, error) {
	directives := ParseGroup(doc)
	for i := len(directives) - 1; i >= 0; i-- {
		d := directives[i]
		if d.Name != "go:wasmimport" {
			continue
		}
		fields := strings.Fields(d.Args)
		if len(fields) != 2 {
			return WasmImport{Pos: d.Pos}, true, fmt.Errorf("//go:wasmimport expects 2 arguments (module name), got %d", len(fields))
		}
		return WasmImport{Module: fields[0], Name: fields[1], Pos: d.Pos}, true, nil
	}
	return WasmImport{}, false, nil
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

// ParseWasmExport extracts a //go:wasmexport <name> or //llgo:export <name>
// directive from doc.
func ParseWasmExport(doc *ast.CommentGroup) (string, token.Pos, bool) {
	directives := ParseGroup(doc)
	for i := len(directives) - 1; i >= 0; i-- {
		d := directives[i]
		if (d.Name == "go:wasmexport" || d.Name == "llgo:export") && d.Args != "" {
			fields := strings.Fields(d.Args)
			if len(fields) > 0 {
				return fields[0], d.Pos, true
			}
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

// HasNoInline reports whether doc contains a //go:noinline directive.
func HasNoInline(doc *ast.CommentGroup) bool {
	for _, d := range ParseGroup(doc) {
		if d.Name == "go:noinline" && d.Args == "" {
			return true
		}
	}
	return false
}

// HasNoInterface reports whether doc contains a //go:nointerface directive.
func HasNoInterface(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for n := len(doc.List) - 1; n >= 0; n-- {
		d, ok := Parse(doc.List[n])
		if !ok {
			return false
		}
		if d.Name == "go:nointerface" && d.Args == "" {
			return true
		}
		if !strings.HasPrefix(d.Name, "go:") && !strings.HasPrefix(d.Name, "llgo:") && d.Name != "export" {
			return false
		}
	}
	return false
}

// HasNoEscape reports whether doc contains a //go:noescape directive.
func HasNoEscape(doc *ast.CommentGroup) bool {
	for _, d := range ParseGroup(doc) {
		if d.Name == "go:noescape" && d.Args == "" {
			return true
		}
	}
	return false
}

// HasUintptrEscapes reports whether doc contains a //go:uintptrescapes directive.
func HasUintptrEscapes(doc *ast.CommentGroup) bool {
	for _, d := range ParseGroup(doc) {
		if d.Name == "go:uintptrescapes" && d.Args == "" {
			return true
		}
	}
	return false
}

// HasNoSplit reports whether doc contains a //go:nosplit directive.
func HasNoSplit(doc *ast.CommentGroup) bool {
	for _, d := range ParseGroup(doc) {
		if d.Name == "go:nosplit" && d.Args == "" {
			return true
		}
	}
	return false
}

// LinkNamespace classifies the target namespace of a //go:linkname or
// //llgo:link target string.
type LinkNamespace uint8

const (
	LinkGo LinkNamespace = iota
	LinkC
	LinkStdcall
	LinkPy
	LinkLLGo
	LinkLLVM
)

func (n LinkNamespace) String() string {
	switch n {
	case LinkC:
		return "C"
	case LinkStdcall:
		return "stdcall"
	case LinkPy:
		return "py"
	case LinkLLGo:
		return "llgo"
	case LinkLLVM:
		return "llvm"
	default:
		return "go"
	}
}

// ParseLinkTarget splits a linkname target into its namespace and symbol name.
// For LinkLLVM, the full "llvm.*" intrinsic name is preserved as symbol.
func ParseLinkTarget(target string) (LinkNamespace, string) {
	switch {
	case strings.HasPrefix(target, "C."):
		return LinkC, target[len("C."):]
	case strings.HasPrefix(target, "stdcall."):
		return LinkStdcall, target[len("stdcall."):]
	case strings.HasPrefix(target, "py."):
		return LinkPy, target[len("py."):]
	case strings.HasPrefix(target, "llgo."):
		return LinkLLGo, target[len("llgo."):]
	case strings.HasPrefix(target, "llvm."):
		return LinkLLVM, target
	default:
		return LinkGo, target
	}
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

// ParsePackageKind parses the string value of an LLGoPackage constant,
// matching the compiler's package classification rules verbatim.
func ParsePackageKind(v string) (PackageKind, string) {
	switch v {
	case "link":
		return PkgLinkIR, ""
	case "decl":
		return PkgDeclOnly, ""
	case "noinit":
		return PkgNoInit, ""
	default:
		if strings.HasPrefix(v, "link:") { // "link: <libpath>"
			return PkgLinkExtern, v[5:]
		} else if strings.HasPrefix(v, "py.") { // "py.<module>"
			return PkgPyModule, v[3:]
		} else if strings.HasPrefix(v, "decl:") { // "decl: <param>"
			return PkgDeclOnly, v[5:]
		}
	}
	return PkgLLGo, ""
}
