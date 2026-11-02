//go:build !llgo

package browserdebug

import (
	"debug/dwarf"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func TestClosureEnvironmentIndex(t *testing.T) {
	// Compile real LLVM DWARF plus an address-taken function, so the table
	// index, artificial variable and pointer/struct layout are read together.
	const source = `target triple = "wasm32-unknown-unknown"
define i32 @add() { %p = ptrtoint ptr @capture to i32
ret i32 %p }
define i32 @capture(ptr %env) !dbg !4 {
call void @llvm.dbg.value(metadata ptr %env, metadata !7, metadata !DIExpression()), !dbg !8
%p = load ptr, ptr %env, !dbg !8
%v = load i32, ptr %p, !dbg !8
ret i32 %v, !dbg !8
}
declare void @llvm.dbg.value(metadata, metadata, metadata)
!llvm.dbg.cu = !{!0}
!llvm.module.flags = !{!2, !3}
!0 = distinct !DICompileUnit(language: DW_LANG_C, file: !1, producer: "LLGo", isOptimized: false, runtimeVersion: 0, emissionKind: FullDebug)
!1 = !DIFile(filename: "closure.go", directory: "/src")
!2 = !{i32 2, !"Dwarf Version", i32 4}
!3 = !{i32 2, !"Debug Info Version", i32 3}
!4 = distinct !DISubprogram(name: "capture", linkageName: "capture", scope: !1, file: !1, line: 1, type: !5, scopeLine: 1, spFlags: DISPFlagDefinition, unit: !0)
!5 = !DISubroutineType(types: !6)
!6 = !{!13, !9}
!7 = !DILocalVariable(name: "$env", scope: !4, file: !1, line: 1, type: !9, flags: DIFlagArtificial)
!8 = !DILocation(line: 1, column: 1, scope: !4)
!9 = !DIDerivedType(tag: DW_TAG_pointer_type, baseType: !10, size: 32)
!10 = !DICompositeType(tag: DW_TAG_structure_type, name: "captures", size: 32, elements: !11)
!11 = !{!12}
!12 = !DIDerivedType(tag: DW_TAG_member, name: "value", scope: !10, baseType: !14, size: 32)
!13 = !DIBasicType(name: "int32", size: 32, encoding: DW_ATE_signed)
!14 = !DIDerivedType(tag: DW_TAG_pointer_type, baseType: !13, size: 32)
`
	dir := t.TempDir()
	ir, wasm := filepath.Join(dir, "closure.ll"), filepath.Join(dir, "closure.wasm")
	if err := os.WriteFile(ir, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	compileWasmFixture(t, ir, wasm)
	raw, err := os.ReadFile(wasm)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wasm, browserTestIdentity(t, raw), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := Load(wasm, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Index.FunctionValues) != 1 {
		t.Fatalf("function values = %+v, diagnostics = %v", bundle.Index.FunctionValues, bundle.Index.Diagnostics)
	}
	value := bundle.Index.FunctionValues[0]
	if value.Index == 0 || value.Name != "capture" || value.Environment == "" {
		t.Fatalf("closure value = %+v", value)
	}
	types := make(map[string]Type)
	for _, typ := range bundle.Index.Types {
		types[typ.ID] = typ
	}
	ptr := types[value.Environment]
	env := types[ptr.Elem]
	if ptr.Kind != "pointer" || env.Kind != "struct" || len(env.Fields) != 1 ||
		env.Fields[0].Name != "value" || env.Fields[0].Offset != 0 || types[env.Fields[0].Type].Kind != "pointer" {
		t.Fatalf("capture layout = %+v -> %+v", ptr, env)
	}
	if hasVariable(bundle.Index.Variables, "$env") {
		t.Fatal("hidden environment leaked into source locals")
	}

	sections, err := wasmdebug.DWARFSections(raw)
	if err != nil {
		t.Fatal(err)
	}
	data, err := dwarfData(sections)
	if err != nil {
		t.Fatal(err)
	}
	reader := data.Reader()
	var envEntry *dwarf.Entry
	var scalarOffset, opaqueOffset dwarf.Offset
	for {
		entry, err := reader.Next()
		if err != nil {
			t.Fatal(err)
		}
		if entry == nil {
			break
		}
		if entry.Val(dwarf.AttrName) == "$env" {
			envEntry = entry
		}
		if entry.Tag == dwarf.TagBaseType {
			scalarOffset = entry.Offset
		}
		if entry.Tag == dwarf.TagPointerType {
			ty, err := data.Type(entry.Offset)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := ty.(*dwarf.PtrType).Type.(*dwarf.IntType); ok {
				opaqueOffset = entry.Offset
			}
		}
	}
	if envEntry == nil || scalarOffset == 0 || opaqueOffset == 0 {
		t.Fatal("fixture lacks required DWARF declarations")
	}
	builder := indexBuilder{data: data, typeByKey: make(map[uintptr]string)}
	for _, attr := range []dwarf.Attr{dwarf.AttrAbstractOrigin, dwarf.AttrSpecification} {
		entry := &dwarf.Entry{Offset: 1, Field: []dwarf.Field{{Attr: attr, Val: envEntry.Offset}}}
		if got := builder.attribute(entry, dwarf.AttrName); got != "$env" {
			t.Fatalf("inherited closure variable name = %v", got)
		}
	}
	for _, offset := range []dwarf.Offset{1, 0xfffffff} {
		entry := &dwarf.Entry{Offset: 1, Field: []dwarf.Field{{Attr: dwarf.AttrAbstractOrigin, Val: offset}}}
		if got := builder.attribute(entry, dwarf.AttrName); got != nil {
			t.Fatalf("invalid/cyclic origin returned %v", got)
		}
	}
	// Reject unmarked user metadata, missing/invalid types, and opaque env
	// pointers. No rejected entry may publish a layout for arbitrary memory.
	for _, fields := range [][]dwarf.Field{
		{{Attr: dwarf.AttrName, Val: "$env"}},
		{{Attr: dwarf.AttrArtificial, Val: true}},
		{{Attr: dwarf.AttrArtificial, Val: true}, {Attr: dwarf.AttrType, Val: dwarf.Offset(0xfffffff)}},
		{{Attr: dwarf.AttrArtificial, Val: true}, {Attr: dwarf.AttrType, Val: scalarOffset}},
		{{Attr: dwarf.AttrArtificial, Val: true}, {Attr: dwarf.AttrType, Val: opaqueOffset}},
	} {
		builder.addClosureEnv(&dwarf.Entry{Field: fields}, "rejected")
		if _, exists := builder.closureEnvs["rejected"]; exists {
			t.Fatalf("invalid closure layout was published: %v", fields)
		}
	}
	builder.addClosureEnv(envEntry, "")
	builder.addClosureEnv(envEntry, "capture")
	if builder.closureEnvs["capture"] == "" {
		t.Fatal("valid closure was rejected")
	}
	builder.closureEnvs["capture"] = "conflicting-layout"
	builder.addClosureEnv(envEntry, "capture")
	builder.addClosureEnv(envEntry, "capture")
	if builder.closureEnvs["capture"] != "" {
		t.Fatal("ambiguous layout was republished")
	}
}

func TestFunctionValuesUnavailableMetadata(t *testing.T) {
	header := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	elements := []byte{1, 0, 0x41, 0, 0x0b, 2, 1, 2}
	module := append(append([]byte(nil), header...), 9)
	module = binary.AppendUvarint(module, uint64(len(elements)))
	module = append(module, elements...)
	for _, raw := range [][]byte{header, module} {
		var builder indexBuilder
		builder.addFunctionValues(raw)
		if len(builder.index.FunctionValues) != 0 || len(builder.index.Diagnostics) != 0 {
			t.Fatalf("absent names produced values or errors: %+v", builder.index)
		}
	}
	for _, tc := range []struct {
		raw    []byte
		prefix string
	}{
		{[]byte("invalid"), "function table: "},
		{appendBrowserCustomSection(module, "name", []byte{1}), "function names: "},
	} {
		var builder indexBuilder
		builder.addFunctionValues(tc.raw)
		if len(builder.index.FunctionValues) != 0 || len(builder.index.Diagnostics) != 1 || !strings.HasPrefix(builder.index.Diagnostics[0], tc.prefix) {
			t.Fatalf("invalid optional metadata = %+v", builder.index)
		}
	}
	// Table slot zero is nil even when it has a name; the other slots are
	// sorted by table index, not by function index or DWARF address.
	elements = []byte{1, 0, 0x41, 0, 0x0b, 3, 1, 3, 2}
	module = append(append([]byte(nil), header...), 9, byte(len(elements)))
	module = append(module, elements...)
	names := []byte{3, 1, 1, 'n', 2, 1, 'a', 3, 1, 'b'}
	module = appendBrowserCustomSection(module, "name", append([]byte{1, byte(len(names))}, names...))
	var builder indexBuilder
	builder.addFunctionValues(module)
	if values := builder.index.FunctionValues; len(values) != 2 || values[0].Index != 1 || values[0].Name != "b" || values[1].Index != 2 || values[1].Name != "a" {
		t.Fatalf("function table identity = %+v", values)
	}
	// Two physical functions may share a tool-conventions display name.
	// That name is insufficient evidence to choose a capture layout.
	names[len(names)-1] = 'a'
	module = append(append([]byte(nil), header...), 9, byte(len(elements)))
	module = append(module, elements...)
	module = appendBrowserCustomSection(module, "name", append([]byte{1, byte(len(names))}, names...))
	builder = indexBuilder{closureEnvs: map[string]string{"a": "env"}}
	builder.addFunctionValues(module)
	for _, value := range builder.index.FunctionValues {
		if value.Environment != "" {
			t.Fatalf("duplicate function name selected a layout: %+v", value)
		}
	}
}
