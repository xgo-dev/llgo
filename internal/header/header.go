/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package header

import (
	"bytes"
	"fmt"
	"go/types"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/ssa"
)

// cheaderWriter handles C header generation with type definition management
type cheaderWriter struct {
	p             ssa.Program
	typeBuf       *bytes.Buffer   // buffer for type definitions
	funcBuf       *bytes.Buffer   // buffer for function declarations
	declaredTypes map[string]bool // track declared types to avoid duplicates
}

// newCHeaderWriter creates a new C header writer
func newCHeaderWriter(p ssa.Program) *cheaderWriter {
	return &cheaderWriter{
		p:             p,
		typeBuf:       &bytes.Buffer{},
		funcBuf:       &bytes.Buffer{},
		declaredTypes: make(map[string]bool),
	}
}

// writeTypedef writes a C typedef for the given Go type if not already declared
func (hw *cheaderWriter) writeTypedef(t types.Type) error {
	return hw.writeTypedefRecursive(t, make(map[string]bool))
}

// writeTypedefRecursive writes typedefs recursively, handling dependencies
func (hw *cheaderWriter) writeTypedefRecursive(t types.Type, visiting map[string]bool) error {
	// Handle container types that only need element processing
	switch typ := t.(type) {
	case *types.Array:
		return hw.writeTypedefRecursive(typ.Elem(), visiting)
	case *types.Slice:
		return hw.writeTypedefRecursive(typ.Elem(), visiting)
	case *types.Map:
		if err := hw.writeTypedefRecursive(typ.Key(), visiting); err != nil {
			return err
		}
		return hw.writeTypedefRecursive(typ.Elem(), visiting)
	case *types.Chan:
		return hw.writeTypedefRecursive(typ.Elem(), visiting)
	}

	cType := hw.goCTypeName(t)
	if cType == "" || hw.declaredTypes[cType] {
		return nil
	}

	// Prevent infinite recursion for self-referential types
	if visiting[cType] {
		return nil
	}
	visiting[cType] = true
	defer delete(visiting, cType)

	// Process dependent types for complex types
	if err := hw.processDependentTypes(t, visiting); err != nil {
		return err
	}

	// Then write the typedef for this type
	typedef := hw.generateTypedef(t)
	if typedef != "" {
		fmt.Fprintln(hw.typeBuf, typedef)
		// Add empty line after each type definition
		fmt.Fprintln(hw.typeBuf)
		hw.declaredTypes[cType] = true
	}
	return nil
}

// processDependentTypes processes dependent types for composite types
func (hw *cheaderWriter) processDependentTypes(t types.Type, visiting map[string]bool) error {
	switch typ := t.(type) {
	case *types.Pointer:
		return hw.writeTypedefRecursive(typ.Elem(), visiting)
	case *types.Struct:
		// For anonymous structs, handle field dependencies
		for i := 0; i < typ.NumFields(); i++ {
			field := typ.Field(i)
			if err := hw.writeTypedefRecursive(field.Type(), visiting); err != nil {
				return err
			}
		}
	case *types.Named:
		// For named types, handle the underlying type dependencies
		underlying := typ.Underlying()
		if structType, ok := underlying.(*types.Struct); ok {
			if ssa.IsClosure(structType) {
				return fmt.Errorf("closure type %s can't export to C header", typ.Obj().Name())
			}
			// For named struct types, handle field dependencies directly
			for i := 0; i < structType.NumFields(); i++ {
				field := structType.Field(i)
				if err := hw.writeTypedefRecursive(field.Type(), visiting); err != nil {
					return err
				}
			}
		} else {
			// For other named types, handle the underlying type
			return hw.writeTypedefRecursive(underlying, visiting)
		}
	case *types.Signature:
		return hw.processSignatureTypes(typ, visiting)
	}
	return nil
}

// processSignatureTypes processes function signature parameter and result types
func (hw *cheaderWriter) processSignatureTypes(sig *types.Signature, visiting map[string]bool) error {
	// Handle function parameters
	if sig.Params() != nil {
		for i := 0; i < sig.Params().Len(); i++ {
			param := sig.Params().At(i)
			if err := hw.writeTypedefRecursive(param.Type(), visiting); err != nil {
				return err
			}
		}
	}
	// Handle function results
	if sig.Results() != nil {
		for i := 0; i < sig.Results().Len(); i++ {
			result := sig.Results().At(i)
			if err := hw.writeTypedefRecursive(result.Type(), visiting); err != nil {
				return err
			}
		}
	}
	return nil
}

// goCTypeName returns the C type name for a Go type
func (hw *cheaderWriter) goCTypeName(t types.Type) string {
retry:
	switch typ := t.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Invalid:
			return ""
		case types.Bool:
			return "_Bool"
		case types.Int8:
			return "int8_t"
		case types.Uint8:
			return "uint8_t"
		case types.Int16:
			return "int16_t"
		case types.Uint16:
			return "uint16_t"
		case types.Int32:
			return "int32_t"
		case types.Uint32:
			return "uint32_t"
		case types.Int64:
			return "int64_t"
		case types.Uint64:
			return "uint64_t"
		case types.Int:
			return "intptr_t"
		case types.Uint:
			return "uintptr_t"
		case types.Uintptr:
			return "uintptr_t"
		case types.Float32:
			return "float"
		case types.Float64:
			return "double"
		case types.Complex64:
			return "GoComplex64"
		case types.Complex128:
			return "GoComplex128"
		case types.String:
			return "GoString"
		case types.UnsafePointer:
			return "void*"
		}
	case *types.Pointer:
		elemType := hw.goCTypeName(typ.Elem())
		if elemType == "" {
			return "void*"
		}
		return elemType + "*"
	case *types.Slice:
		return "GoSlice"
	case *types.Array:
		// For arrays, we return just the element type
		// The array size will be handled in field generation
		return hw.goCTypeName(typ.Elem())
	case *types.Map:
		return "GoMap"
	case *types.Chan:
		return "GoChan"
	case *types.Interface:
		return "GoInterface"
	case *types.Struct:
		if ssa.IsClosure(typ) {
			panic("closure type can't export to C header")
		}
		// For anonymous structs, generate a descriptive name
		var fields []string
		for i := 0; i < typ.NumFields(); i++ {
			field := typ.Field(i)
			fieldType := hw.goCTypeName(field.Type())
			fields = append(fields, fmt.Sprintf("%s_%s", fieldType, field.Name()))
		}
		return fmt.Sprintf("struct_%s", strings.Join(fields, "_"))
	case *types.Named:
		// For named types, always use the named type
		pkg := typ.Obj().Pkg()
		return fmt.Sprintf("%s_%s", pkg.Name(), typ.Obj().Name())
	case *types.Signature:
		// Function types are represented as function pointers in C
		// Generate proper function pointer syntax
		return hw.generateFunctionPointerType(typ)
	case *types.Alias:
		t = types.Unalias(typ)
		goto retry
	}
	panic(fmt.Errorf("unsupported type: %v", t))
}

// generateFunctionPointerType generates C function pointer type for Go function signatures
func (hw *cheaderWriter) generateFunctionPointerType(sig *types.Signature) string {
	// Generate return type
	var returnType string
	results := sig.Results()
	if results == nil || results.Len() == 0 {
		returnType = "void"
	} else if results.Len() == 1 {
		returnType = hw.goCTypeName(results.At(0).Type())
	} else {
		panic("multiple return values can't export to C header")
	}

	// Generate parameter types
	var paramTypes []string
	params := sig.Params()
	if params == nil || params.Len() == 0 {
		paramTypes = []string{"void"}
	} else {
		for i := 0; i < params.Len(); i++ {
			paramType := hw.goCTypeName(params.At(i).Type())
			paramTypes = append(paramTypes, paramType)
		}
	}

	// Return function pointer type: returnType (*)(paramType1, paramType2, ...)
	return fmt.Sprintf("%s (*)(%s)", returnType, strings.Join(paramTypes, ", "))
}

// generateTypedef generates C typedef declaration for complex types
func (hw *cheaderWriter) generateTypedef(t types.Type) string {
	switch typ := t.(type) {
	case *types.Struct:
		// Only generate typedef for anonymous structs
		return hw.generateStructTypedef(typ)
	case *types.Named:
		underlying := typ.Underlying()
		if structType, ok := underlying.(*types.Struct); ok {
			// For named struct types, generate the typedef directly
			return hw.generateNamedStructTypedef(typ, structType)
		}

		cTypeName := hw.goCTypeName(typ)

		// Special handling for function types
		if sig, ok := underlying.(*types.Signature); ok {
			// Generate return type
			var returnType string
			results := sig.Results()
			if results == nil || results.Len() == 0 {
				returnType = "void"
			} else if results.Len() == 1 {
				returnType = hw.goCTypeName(results.At(0).Type())
			} else {
				panic("multiple return values can't export to C header")
			}

			// Generate parameter types
			var paramTypes []string
			params := sig.Params()
			if params == nil || params.Len() == 0 {
				paramTypes = []string{"void"}
			} else {
				for i := 0; i < params.Len(); i++ {
					paramType := hw.goCTypeName(params.At(i).Type())
					paramTypes = append(paramTypes, paramType)
				}
			}

			// Generate proper function pointer typedef: typedef returnType (*typeName)(params);
			return fmt.Sprintf("typedef %s (*%s)(%s);", returnType, cTypeName, strings.Join(paramTypes, ", "))
		}

		// For other named types, create a typedef to the underlying type
		underlyingCType := hw.goCTypeName(underlying)
		if underlyingCType != "" {
			return fmt.Sprintf("typedef %s %s;", underlyingCType, cTypeName)
		}
	}
	return ""
}

// generateReturnType generates C return type, converting arrays to struct wrappers
func (hw *cheaderWriter) generateReturnType(retType types.Type) string {
	switch typ := retType.(type) {
	case *types.Array:
		// For array return values, generate a struct wrapper
		return hw.ensureArrayStruct(typ)
	default:
		// For non-array types, use regular type conversion
		return hw.goCTypeName(retType)
	}
}

// ensureArrayStruct generates array struct name and ensures its typedef is declared
func (hw *cheaderWriter) ensureArrayStruct(arr *types.Array) string {
	// Generate struct name
	var dimensions []int64
	baseType := types.Type(arr)

	// Traverse all array dimensions
	for {
		if a, ok := baseType.(*types.Array); ok {
			dimensions = append(dimensions, a.Len())
			baseType = a.Elem()
		} else {
			break
		}
	}

	// Get base element type
	elemType := hw.goCTypeName(baseType)

	// Generate struct name: Array_int32_t_4 for [4]int32, Array_int32_t_3_4 for [3][4]int32
	var name strings.Builder
	name.WriteString("Array_")
	name.WriteString(strings.ReplaceAll(elemType, "*", "_ptr"))
	for _, dim := range dimensions {
		name.WriteString(fmt.Sprintf("_%d", dim))
	}

	structName := name.String()

	// Ensure typedef is declared
	if !hw.declaredTypes[structName] {
		hw.declaredTypes[structName] = true
		// Generate field declaration for the array
		fieldDecl := hw.generateFieldDeclaration(arr, "data")
		// Write the typedef
		typedef := fmt.Sprintf("typedef struct {\n%s\n} %s;", fieldDecl, structName)
		fmt.Fprintf(hw.typeBuf, "%s\n\n", typedef)
	}

	return structName
}

// generateParameterDeclaration generates C parameter declaration for function parameters
func (hw *cheaderWriter) generateParameterDeclaration(paramType types.Type, paramName string) string {
	var cType string

	switch typ := paramType.(type) {
	case *types.Array:
		// C array parameters decay to pointers and therefore do not preserve
		// Go's by-value array ABI on architectures such as amd64. Use the same
		// generated wrapper struct as array return values so C passes the whole
		// value with the platform aggregate ABI.
		cType = hw.ensureArrayStruct(typ)
	case *types.Pointer:
		pointeeType := hw.goCTypeName(typ.Elem())
		cType = pointeeType + "*"
	default:
		// Regular types
		cType = hw.goCTypeName(paramType)
	}

	if paramName == "" {
		return cType
	}
	return cType + " " + paramName
}

// generateFieldDeclaration generates C field declaration with correct array syntax
func (hw *cheaderWriter) generateFieldDeclaration(fieldType types.Type, fieldName string) string {
	switch fieldType.(type) {
	case *types.Array:
		// Handle multidimensional arrays by collecting all dimensions
		var dimensions []int64
		baseType := fieldType

		// Traverse all array dimensions
		for {
			if arr, ok := baseType.(*types.Array); ok {
				dimensions = append(dimensions, arr.Len())
				baseType = arr.Elem()
			} else {
				break
			}
		}

		// Get base element type
		elemType := hw.goCTypeName(baseType)

		// Build array dimensions string [d1][d2][d3]...
		var dimStr strings.Builder
		for _, dim := range dimensions {
			dimStr.WriteString(fmt.Sprintf("[%d]", dim))
		}

		return fmt.Sprintf("    %s %s%s;", elemType, fieldName, dimStr.String())
	default:
		cType := hw.goCTypeName(fieldType)
		return fmt.Sprintf("    %s %s;", cType, fieldName)
	}
}

// generateStructTypedef generates typedef for anonymous struct
func (hw *cheaderWriter) generateStructTypedef(s *types.Struct) string {
	// Generate descriptive type name inline
	var nameFields []string
	var declFields []string

	for i := 0; i < s.NumFields(); i++ {
		field := s.Field(i)
		fieldType := hw.goCTypeName(field.Type())
		nameFields = append(nameFields, fmt.Sprintf("%s_%s", fieldType, field.Name()))
		declFields = append(declFields, hw.generateFieldDeclaration(field.Type(), field.Name()))
	}

	typeName := fmt.Sprintf("struct_%s", strings.Join(nameFields, "_"))
	return fmt.Sprintf("typedef struct {\n%s\n} %s;", strings.Join(declFields, "\n"), typeName)
}

// generateNamedStructTypedef generates typedef for named struct
func (hw *cheaderWriter) generateNamedStructTypedef(named *types.Named, s *types.Struct) string {
	typeName := hw.goCTypeName(named)

	// Check if this is a self-referential struct
	needsForwardDecl := hw.needsForwardDeclaration(s, typeName)
	var result string

	if needsForwardDecl {
		// Add forward declaration
		result = fmt.Sprintf("typedef struct %s %s;\n", typeName, typeName)
	}

	var fields []string
	for i := 0; i < s.NumFields(); i++ {
		field := s.Field(i)
		fields = append(fields, hw.generateFieldDeclaration(field.Type(), field.Name()))
	}

	if needsForwardDecl {
		// Use struct tag in definition
		result += fmt.Sprintf("struct %s {\n%s\n};", typeName, strings.Join(fields, "\n"))
	} else {
		result = fmt.Sprintf("typedef struct {\n%s\n} %s;", strings.Join(fields, "\n"), typeName)
	}

	return result
}

// needsForwardDeclaration checks if a struct needs forward declaration due to self-reference
func (hw *cheaderWriter) needsForwardDeclaration(s *types.Struct, typeName string) bool {
	for i := 0; i < s.NumFields(); i++ {
		field := s.Field(i)
		if hw.typeReferencesSelf(field.Type(), typeName) {
			return true
		}
	}
	return false
}

// typeReferencesSelf checks if a type references the given type name
func (hw *cheaderWriter) typeReferencesSelf(t types.Type, selfTypeName string) bool {
	switch typ := t.(type) {
	case *types.Pointer:
		elemTypeName := hw.goCTypeName(typ.Elem())
		return elemTypeName == selfTypeName
	case *types.Slice:
		elemTypeName := hw.goCTypeName(typ.Elem())
		return elemTypeName == selfTypeName
	case *types.Array:
		elemTypeName := hw.goCTypeName(typ.Elem())
		return elemTypeName == selfTypeName
	case *types.Named:
		return hw.goCTypeName(typ) == selfTypeName
	}
	return false
}

// writeFunctionDecl writes C function declaration for exported Go function
// fullName: the C function name to display in header
// linkName: the actual Go function name for linking
func (hw *cheaderWriter) writeFunctionDecl(fullName, linkName string, fn ssa.Function) error {
	if fn.IsNil() {
		return nil
	}

	// Get Go signature from LLVM function type
	goType := fn.Type.RawType()
	sig, ok := goType.(*types.Signature)
	if !ok {
		return fmt.Errorf("function %s does not have signature type", fullName)
	}

	// Generate return type
	var returnType string
	if sig.Results().Len() == 0 {
		returnType = "void"
	} else if sig.Results().Len() == 1 {
		retType := sig.Results().At(0).Type()
		if err := hw.writeTypedef(retType); err != nil {
			return err
		}
		returnType = hw.generateReturnType(retType)
	} else {
		return fmt.Errorf("function %s has more than one result", fullName)
	}

	// Generate parameters
	var params []string
	for i := 0; i < sig.Params().Len(); i++ {
		param := sig.Params().At(i)
		paramType := param.Type()

		if err := hw.writeTypedef(paramType); err != nil {
			return err
		}

		paramName := param.Name()

		// Generate parameter declaration
		paramDecl := hw.generateParameterDeclaration(paramType, paramName)
		params = append(params, paramDecl)
	}

	paramStr := strings.Join(params, ", ")
	if paramStr == "" {
		paramStr = "void"
	}
	// Write function declaration with return type on separate line for normal functions
	fmt.Fprintln(hw.funcBuf, returnType)
	// Generate function declaration using cross-platform macro when names differ
	var funcDecl string
	if fullName != linkName {
		funcDecl = fmt.Sprintf("%s(%s) GO_SYMBOL_RENAME(\"%s\")", fullName, paramStr, linkName)
	} else {
		funcDecl = fmt.Sprintf("%s(%s);", fullName, paramStr)
	}

	fmt.Fprintln(hw.funcBuf, funcDecl)
	// Add empty line after each function declaration
	fmt.Fprintln(hw.funcBuf)

	return nil
}

// writeCommonIncludes writes common C header includes and Go runtime type definitions
func (hw *cheaderWriter) writeCommonIncludes() error {
	includes := `
// Platform-specific symbol renaming macro
#ifdef __APPLE__
    #define GO_SYMBOL_RENAME(go_name) __asm("_" go_name);
#else
    #define GO_SYMBOL_RENAME(go_name) __asm(go_name);
#endif

// Go runtime types
typedef struct { const char *p; intptr_t n; } GoString;
typedef struct { void *data; intptr_t len; intptr_t cap; } GoSlice;
typedef struct { void *data; } GoMap;
typedef struct { void *data; } GoChan;
typedef struct { void *data; void *type; } GoInterface;
typedef struct { float real; float imag; } GoComplex64;
typedef struct { double real; double imag; } GoComplex128;

`

	if _, err := hw.typeBuf.WriteString(includes); err != nil {
		return err
	}
	return nil
}

// writeTo writes all generated content to the output writer
func (hw *cheaderWriter) writeTo(w io.Writer) error {
	// Write type definitions first
	if hw.typeBuf.Len() > 0 {
		if _, err := hw.typeBuf.WriteTo(w); err != nil {
			return err
		}
	}

	// Then write function declarations
	if hw.funcBuf.Len() > 0 {
		if _, err := hw.funcBuf.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}

func genHeader(p ssa.Program, pkgs []ssa.Package, w io.Writer) error {
	hw := newCHeaderWriter(p)

	// Write common header includes and type definitions
	if err := hw.writeCommonIncludes(); err != nil {
		return err
	}

	// Mark predefined Go types as declared
	hw.declaredTypes["GoString"] = true
	hw.declaredTypes["GoSlice"] = true
	hw.declaredTypes["GoMap"] = true
	hw.declaredTypes["GoChan"] = true
	hw.declaredTypes["GoInterface"] = true
	hw.declaredTypes["GoComplex64"] = true
	hw.declaredTypes["GoComplex128"] = true

	// Process all exported functions
	for _, pkg := range pkgs {
		exports := pkg.ExportFuncs()
		// Sort functions for testing
		exportNames := make([]string, 0, len(exports))
		for name := range exports {
			// Imported //export metadata may be recorded on the importing
			// package as well. Only declarations owned by this package belong
			// in its public header; otherwise internal runtime callbacks can
			// leak into user headers or fail type generation entirely.
			if !exportBelongsToPackage(pkg.Path(), name) {
				continue
			}
			exportNames = append(exportNames, name)
		}
		sort.Strings(exportNames)

		for _, name := range exportNames { // name is goName
			link := exports[name] // link is cName
			fn := pkg.FuncOf(link)
			if fn == nil {
				return fmt.Errorf("function %s not found in package %s", link, pkg.Path())
			}

			// Write function declaration with proper C types
			if err := hw.writeFunctionDecl(link, link, fn); err != nil {
				return fmt.Errorf("failed to write declaration for function %s: %w", name, err)
			}
		}

		initFnName := pkg.Path() + ".init"
		initFn := pkg.FuncOf(initFnName)
		if initFn != nil {
			cInitFnName := cIdentifier(initFnName)
			if err := hw.writeFunctionDecl(cInitFnName, initFnName, initFn); err != nil {
				return fmt.Errorf("failed to write declaration for function %s: %w", initFnName, err)
			}
		}
	}

	// Write all content to output in the correct order
	return hw.writeTo(w)
}

func cIdentifier(name string) string {
	var out strings.Builder
	out.Grow(len(name))
	for i, r := range name {
		valid := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if !valid {
			r = '_'
		}
		if i == 0 && r >= '0' && r <= '9' {
			out.WriteByte('_')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func exportBelongsToPackage(pkgPath, name string) bool {
	if !strings.Contains(name, ".") {
		// Preserve support for callers that register the historical
		// unqualified form via ssa.Package.SetExport.
		return true
	}
	return strings.HasPrefix(name, pkgPath+".")
}

func GenHeaderFile(p ssa.Program, pkgs []ssa.Package, libName, headerPath string, verbose bool) error {
	// Write header file
	w, err := os.Create(headerPath)
	if err != nil {
		return fmt.Errorf("failed to write header file %s: %w", headerPath, err)
	}
	defer w.Close()

	if verbose {
		fmt.Fprintf(os.Stderr, "Generated C header: %s\n", headerPath)
	}

	headerIdent := strings.ToUpper(strings.ReplaceAll(libName, "-", "_"))
	headerContent := fmt.Sprintf(`/* Code generated by llgo; DO NOT EDIT. */

#ifndef __%s_H_
#define __%s_H_

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif
`, headerIdent, headerIdent)

	w.Write([]byte(headerContent))

	if err = genHeader(p, pkgs, w); err != nil {
		return fmt.Errorf("failed to generate header content for %s: %w", libName, err)
	}

	footerContent := fmt.Sprintf(`

#ifdef __cplusplus
}
#endif

#endif /* __%s_H_ */
`, headerIdent)

	_, err = w.Write([]byte(footerContent))
	return err
}
