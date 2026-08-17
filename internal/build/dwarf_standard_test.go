//go:build !llgo

package build

import (
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/internal/optlevel"
	llvmenv "github.com/xgo-dev/llgo/xtool/env/llvm"
)

const dwarfLanguageC = 0x02

type dwarfNode struct {
	entry    *dwarf.Entry
	children []*dwarfNode
}

func TestStandardDWARF(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native DWARF integration test requires Mach-O or ELF")
	}

	for _, level := range []optlevel.Level{
		optlevel.O0,
		optlevel.O1,
		optlevel.O2,
		optlevel.O3,
		optlevel.Os,
		optlevel.Oz,
	} {
		t.Run(level.String(), func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "dwarf-standard")
			conf := NewDefaultConf(ModeBuild)
			conf.OutFile = bin
			conf.OptLevel = level
			conf.LinkOptions.DWARF = DWARFPreserve
			if _, err := Do([]string{"./testdata/dwarf"}, conf); err != nil {
				t.Fatalf("build DWARF fixture at %s: %v", level, err)
			}
			if got := runBinary(t, bin); got != "dwarf-ok\n" {
				t.Fatalf("DWARF fixture output at %s = %q", level, got)
			}

			data, dwarfPath := openNativeDWARF(t, bin)
			verifyDWARF(t, dwarfPath)
			roots := readDWARFTree(t, data)
			cu := findDWARFNode(roots, dwarf.TagCompileUnit, "main")
			if cu == nil {
				t.Fatal("main compile unit not found")
			}
			if got, _ := cu.entry.Val(dwarf.AttrLanguage).(int64); got != dwarfLanguageC {
				t.Fatalf("DW_AT_language = %#x, want DW_LANG_C", got)
			}
			if got, _ := cu.entry.Val(dwarf.AttrProducer).(string); got != "LLGo" {
				t.Fatalf("DW_AT_producer = %q, want LLGo", got)
			}

			assertDWARFCoreTypes(t, data, cu)
			if level == optlevel.O0 {
				assertDWARFProgramStructure(t, data, cu)
			}
			assertDWARFLines(t, data, cu, "helper.go", markerLineInFile(t, "testdata/dwarf/helper.go", "DWARF_LINE_MARKER"))
		})
	}
}

func openNativeDWARF(t *testing.T, bin string) (*dwarf.Data, string) {
	t.Helper()
	switch runtime.GOOS {
	case "linux":
		file, err := elf.Open(bin)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })
		data, err := file.DWARF()
		if err != nil {
			t.Fatal(err)
		}
		return data, bin
	case "darwin":
		dsym := filepath.Join(t.TempDir(), filepath.Base(bin)+".dSYM")
		if out, err := exec.Command("dsymutil", bin, "-o", dsym).CombinedOutput(); err != nil {
			t.Fatalf("dsymutil: %v\n%s", err, out)
		}
		dwarfPath := filepath.Join(dsym, "Contents", "Resources", "DWARF", filepath.Base(bin))
		file, err := macho.Open(dwarfPath)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })
		data, err := file.DWARF()
		if err != nil {
			t.Fatal(err)
		}
		return data, dwarfPath
	default:
		panic("unreachable")
	}
}

func verifyDWARF(t *testing.T, path string) {
	t.Helper()
	binDir := llvmenv.New("").BinDir()
	verifyPath := path
	if runtime.GOOS == "linux" {
		// ELF section GC leaves BFD tombstones in DIEs for discarded functions.
		// Normalize a verifier-only copy; semantic assertions still read the
		// original ELF because dwarfutil intentionally drops global variables.
		verifyPath = filepath.Join(t.TempDir(), filepath.Base(path)+".dwarf")
		tool := filepath.Join(binDir, "llvm-dwarfutil")
		if out, err := exec.Command(tool, "--garbage-collection", path, verifyPath).CombinedOutput(); err != nil {
			t.Fatalf("llvm-dwarfutil: %v\n%s", err, out)
		}
	}
	tool := filepath.Join(binDir, "llvm-dwarfdump")
	if out, err := exec.Command(tool, "--verify", verifyPath).CombinedOutput(); err != nil {
		t.Fatalf("llvm-dwarfdump --verify: %v\n%s", err, out)
	}
}

func readDWARFTree(t *testing.T, data *dwarf.Data) []*dwarfNode {
	t.Helper()
	reader := data.Reader()
	var roots []*dwarfNode
	var stack []*dwarfNode
	for {
		entry, err := reader.Next()
		if err != nil {
			t.Fatal(err)
		}
		if entry == nil {
			break
		}
		if entry.Tag == 0 {
			if len(stack) == 0 {
				t.Fatal("unbalanced DWARF child terminator")
			}
			stack = stack[:len(stack)-1]
			continue
		}
		node := &dwarfNode{entry: entry}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1]
			parent.children = append(parent.children, node)
		}
		if entry.Children {
			stack = append(stack, node)
		}
	}
	return roots
}

func findDWARFNode(nodes []*dwarfNode, tag dwarf.Tag, name string) *dwarfNode {
	for _, node := range nodes {
		if node.entry.Tag == tag {
			if got, _ := node.entry.Val(dwarf.AttrName).(string); got == name || strings.HasSuffix(got, "."+name) {
				return node
			}
		}
		if found := findDWARFNode(node.children, tag, name); found != nil {
			return found
		}
	}
	return nil
}

func findDWARFNodes(nodes []*dwarfNode, tag dwarf.Tag, name string) (found []*dwarfNode) {
	for _, node := range nodes {
		if node.entry.Tag == tag {
			if got, _ := node.entry.Val(dwarf.AttrName).(string); got == name {
				found = append(found, node)
			}
		}
		found = append(found, findDWARFNodes(node.children, tag, name)...)
	}
	return found
}

func dwarfTypeOf(t *testing.T, data *dwarf.Data, entry *dwarf.Entry) dwarf.Type {
	t.Helper()
	offset, ok := entry.Val(dwarf.AttrType).(dwarf.Offset)
	if !ok {
		t.Fatalf("%v has no DW_AT_type", entry)
	}
	typ, err := data.Type(offset)
	if err != nil {
		t.Fatalf("read type at %#x: %v", offset, err)
	}
	return typ
}

func unwrapDWARFTypedef(typ dwarf.Type) dwarf.Type {
	for {
		typedef, ok := typ.(*dwarf.TypedefType)
		if !ok {
			return typ
		}
		typ = typedef.Type
	}
}

func assertDWARFCoreTypes(t *testing.T, data *dwarf.Data, cu *dwarfNode) {
	t.Helper()
	global := findDWARFNode(cu.children, dwarf.TagVariable, "GlobalInt")
	if global == nil {
		t.Fatal("GlobalInt DIE not found")
	}
	if _, ok := dwarfTypeOf(t, data, global.entry).(*dwarf.TypedefType); !ok {
		t.Fatalf("GlobalInt type = %T, want source NamedInt without a storage pointer", dwarfTypeOf(t, data, global.entry))
	}
	namedInt := findDWARFNode(cu.children, dwarf.TagTypedef, "NamedInt")
	if namedInt == nil {
		t.Fatal("NamedInt typedef DIE not found")
	}
	if got, _ := namedInt.entry.Val(dwarf.AttrDeclLine).(int64); got != int64(markerLineInFile(t, "testdata/dwarf/main.go", "type NamedInt")) {
		t.Errorf("NamedInt declaration line = %d, want its type declaration", got)
	}

	sampleNode := findDWARFNode(cu.children, dwarf.TagTypedef, "Sample")
	if sampleNode == nil {
		t.Fatal("Sample typedef DIE not found")
	}
	sample, ok := unwrapDWARFTypedef(dwarfTypeOf(t, data, sampleNode.entry)).(*dwarf.StructType)
	if !ok {
		t.Fatalf("Sample underlying type = %T, want struct", unwrapDWARFTypedef(dwarfTypeOf(t, data, sampleNode.entry)))
	}
	fields := make(map[string]*dwarf.StructField, len(sample.Field))
	for _, field := range sample.Field {
		fields[field.Name] = field
	}
	ptrSize := int64(dataAddressSize())
	checks := map[string]struct {
		kind any
		size int64
	}{
		"Bool":     {(*dwarf.BoolType)(nil), 1},
		"I8":       {(*dwarf.IntType)(nil), 1},
		"I16":      {(*dwarf.IntType)(nil), 2},
		"I32":      {(*dwarf.IntType)(nil), 4},
		"I64":      {(*dwarf.IntType)(nil), 8},
		"U8":       {(*dwarf.UintType)(nil), 1},
		"U16":      {(*dwarf.UintType)(nil), 2},
		"U32":      {(*dwarf.UintType)(nil), 4},
		"U64":      {(*dwarf.UintType)(nil), 8},
		"F32":      {(*dwarf.FloatType)(nil), 4},
		"F64":      {(*dwarf.FloatType)(nil), 8},
		"C64":      {(*dwarf.ComplexType)(nil), 8},
		"C128":     {(*dwarf.ComplexType)(nil), 16},
		"Text":     {(*dwarf.StructType)(nil), 2 * ptrSize},
		"Values":   {(*dwarf.StructType)(nil), 3 * ptrSize},
		"Fixed":    {(*dwarf.ArrayType)(nil), 6},
		"Lookup":   {(*dwarf.PtrType)(nil), ptrSize},
		"Queue":    {(*dwarf.PtrType)(nil), ptrSize},
		"Callback": {(*dwarf.StructType)(nil), 2 * ptrSize},
		"Any":      {(*dwarf.StructType)(nil), 2 * ptrSize},
		"Iface":    {(*dwarf.StructType)(nil), 2 * ptrSize},
		"Pointer":  {(*dwarf.PtrType)(nil), ptrSize},
		"Unsafe":   {(*dwarf.PtrType)(nil), ptrSize},
		"Pair":     {(*dwarf.StructType)(nil), 16},
	}
	for name, check := range checks {
		field := fields[name]
		if field == nil {
			t.Errorf("Sample.%s field not found", name)
			continue
		}
		got := unwrapDWARFTypedef(field.Type)
		if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", check.kind) {
			t.Errorf("Sample.%s type = %T, want %T", name, got, check.kind)
			continue
		}
		if got.Size() != check.size {
			t.Errorf("Sample.%s size = %d, want %d", name, got.Size(), check.size)
		}
	}
	assertDWARFReflectLayout(t, sample, reflect.TypeOf(dwarfFixtureSample{}))

	array := unwrapDWARFTypedef(fields["Fixed"].Type).(*dwarf.ArrayType)
	if array.Count != 3 || array.Type.Size() != 2 {
		t.Errorf("Fixed = count %d, element size %d; want 3 and 2", array.Count, array.Type.Size())
	}
	for _, name := range []string{"Lookup", "Queue"} {
		pointer := unwrapDWARFTypedef(fields[name].Type).(*dwarf.PtrType)
		if _, ok := pointer.Type.(*dwarf.VoidType); pointer.Type != nil && !ok {
			t.Errorf("Sample.%s pointee = %T, want void or unspecified", name, pointer.Type)
		}
	}

	assertDWARFWordStruct(t, "string", fields["Text"].Type, ptrSize, "data", "len")
	assertDWARFWordStruct(t, "slice", fields["Values"].Type, ptrSize, "data", "len", "cap")
	assertDWARFWordStruct(t, "any", fields["Any"].Type, ptrSize, "type", "data")
	assertDWARFWordStruct(t, "interface", fields["Iface"].Type, ptrSize, "type", "data")
	callback := assertDWARFWordStruct(t, "function value", fields["Callback"].Type, ptrSize, "$f", "$data")
	codeType := unwrapDWARFTypedef(callback.Field[0].Type)
	codePointer, ok := codeType.(*dwarf.PtrType)
	if !ok {
		t.Fatalf("function value code field = %T, want pointer", codeType)
	}
	functionType, ok := codePointer.Type.(*dwarf.FuncType)
	if !ok {
		t.Errorf("function value code pointee = %T, want subroutine type", codePointer.Type)
	} else {
		assertDWARFFunctionSignature(t, functionType)
	}

	for _, check := range []struct {
		name string
		want any
	}{
		{"GlobalMap", (*dwarf.PtrType)(nil)},
		{"GlobalChan", (*dwarf.PtrType)(nil)},
		{"GlobalFunc", (*dwarf.StructType)(nil)},
		{"GlobalInterface", (*dwarf.StructType)(nil)},
		{"GlobalUnsafe", (*dwarf.PtrType)(nil)},
	} {
		node := findDWARFNode(cu.children, dwarf.TagVariable, check.name)
		if node == nil {
			t.Errorf("%s DIE not found", check.name)
			continue
		}
		got := unwrapDWARFTypedef(dwarfTypeOf(t, data, node.entry))
		if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", check.want) {
			t.Errorf("%s type = %T, want %T", check.name, got, check.want)
		}
	}

	recursiveNode := findDWARFNode(cu.children, dwarf.TagTypedef, "Recursive")
	if recursiveNode == nil {
		t.Fatal("Recursive typedef DIE not found")
	}
	recursive, ok := unwrapDWARFTypedef(dwarfTypeOf(t, data, recursiveNode.entry)).(*dwarf.StructType)
	if !ok || len(recursive.Field) != 2 {
		t.Fatalf("Recursive type = %#v, want two-field struct", recursive)
	}
	if _, ok := unwrapDWARFTypedef(recursive.Field[1].Type).(*dwarf.PtrType); !ok {
		t.Fatalf("Recursive.Next = %T, want pointer", recursive.Field[1].Type)
	}
}

func assertDWARFProgramStructure(t *testing.T, data *dwarf.Data, cu *dwarfNode) {
	t.Helper()
	inspect := findDWARFNode(cu.children, dwarf.TagSubprogram, "inspect")
	if inspect == nil {
		t.Fatal("inspect subprogram DIE not found")
	}
	if inspect.entry.Val(dwarf.AttrLowpc) == nil || inspect.entry.Val(dwarf.AttrHighpc) == nil {
		t.Fatal("inspect has no code range")
	}
	for _, name := range []string{"input", "pair", "values"} {
		param := findDWARFNode(inspect.children, dwarf.TagFormalParameter, name)
		if param == nil || param.entry.Val(dwarf.AttrLocation) == nil {
			t.Errorf("parameter %s has no located DIE", name)
		}
	}
	for _, name := range []string{"result", "local", "index", "value", "shadow", "closure", "boxed", "typed"} {
		variables := findDWARFNodes(inspect.children, dwarf.TagVariable, name)
		if len(variables) == 0 {
			t.Errorf("local %s DIE not found", name)
			continue
		}
		if variables[0].entry.Val(dwarf.AttrLocation) == nil {
			t.Errorf("local %s has no location", name)
		}
	}
	if countDWARFTag(inspect.children, dwarf.TagLexDwarfBlock) == 0 {
		t.Error("inspect has no lexical block DIE")
	}
	if findDWARFNodeMatching(cu.children, dwarf.TagSubprogram, func(name string) bool {
		return strings.Contains(name, ".identity[") && strings.HasSuffix(name, ".NamedInt]")
	}) == nil {
		t.Error("instantiated generic identity subprogram DIE not found")
	}
	if findDWARFNode(cu.children, dwarf.TagSubprogram, "main.(*Sample).Number") == nil &&
		findDWARFNode(cu.children, dwarf.TagSubprogram, "Number") == nil {
		t.Error("method subprogram DIE not found")
	}
	_ = data
}

func findDWARFNodeMatching(nodes []*dwarfNode, tag dwarf.Tag, match func(string) bool) *dwarfNode {
	for _, node := range nodes {
		if node.entry.Tag == tag {
			if name, _ := node.entry.Val(dwarf.AttrName).(string); match(name) {
				return node
			}
		}
		if found := findDWARFNodeMatching(node.children, tag, match); found != nil {
			return found
		}
	}
	return nil
}

func countDWARFTag(nodes []*dwarfNode, tag dwarf.Tag) int {
	count := 0
	for _, node := range nodes {
		if node.entry.Tag == tag {
			count++
		}
		count += countDWARFTag(node.children, tag)
	}
	return count
}

func assertDWARFLines(t *testing.T, data *dwarf.Data, cu *dwarfNode, fileSuffix string, line int) {
	t.Helper()
	reader, err := data.LineReader(cu.entry)
	if err != nil {
		t.Fatal(err)
	}
	var entry dwarf.LineEntry
	for {
		err := reader.Next(&entry)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
		if entry.File != nil && strings.HasSuffix(entry.File.Name, fileSuffix) && entry.Line == line && entry.Address != 0 {
			return
		}
	}
	t.Errorf("line table has no address for %s:%d", fileSuffix, line)
}

func markerLineInFile(t *testing.T, path, marker string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, marker) {
			return i + 1
		}
	}
	t.Fatalf("marker %q not found in %s", marker, path)
	return 0
}

func assertDWARFReflectLayout(t *testing.T, got *dwarf.StructType, want reflect.Type) {
	t.Helper()
	if got.Size() != int64(want.Size()) {
		t.Errorf("%s size = %d, want %d", want.Name(), got.Size(), want.Size())
	}
	if len(got.Field) != want.NumField() {
		t.Fatalf("%s has %d fields, want %d", want.Name(), len(got.Field), want.NumField())
	}
	for i, field := range got.Field {
		wantField := want.Field(i)
		if field.Name != wantField.Name {
			t.Errorf("%s field %d name = %q, want %q", want.Name(), i, field.Name, wantField.Name)
		}
		if field.ByteOffset != int64(wantField.Offset) {
			t.Errorf("%s.%s offset = %d, want %d", want.Name(), field.Name, field.ByteOffset, wantField.Offset)
		}
	}
}

func assertDWARFWordStruct(t *testing.T, name string, typ dwarf.Type, wordSize int64, fieldNames ...string) *dwarf.StructType {
	t.Helper()
	structure, ok := unwrapDWARFTypedef(typ).(*dwarf.StructType)
	if !ok {
		t.Fatalf("%s type = %T, want struct", name, unwrapDWARFTypedef(typ))
	}
	if structure.Size() != int64(len(fieldNames))*wordSize {
		t.Errorf("%s size = %d, want %d", name, structure.Size(), int64(len(fieldNames))*wordSize)
	}
	if len(structure.Field) != len(fieldNames) {
		t.Fatalf("%s has %d fields, want %d", name, len(structure.Field), len(fieldNames))
	}
	for i, field := range structure.Field {
		if field.Name != fieldNames[i] {
			t.Errorf("%s field %d name = %q, want %q", name, i, field.Name, fieldNames[i])
		}
		if field.ByteOffset != int64(i)*wordSize {
			t.Errorf("%s.%s offset = %d, want %d", name, field.Name, field.ByteOffset, int64(i)*wordSize)
		}
		fieldType := unwrapDWARFTypedef(field.Type)
		if field.Name == "len" || field.Name == "cap" {
			if _, ok := fieldType.(*dwarf.UintType); !ok {
				t.Errorf("%s.%s type = %T, want unsigned word", name, field.Name, fieldType)
			}
		} else if _, ok := fieldType.(*dwarf.PtrType); !ok {
			t.Errorf("%s.%s type = %T, want pointer", name, field.Name, fieldType)
		}
	}
	return structure
}

func assertDWARFFunctionSignature(t *testing.T, function *dwarf.FuncType) {
	t.Helper()
	if len(function.ParamType) != 1 {
		t.Fatalf("function parameter count = %d, want 1", len(function.ParamType))
	}
	parameter := unwrapDWARFTypedef(function.ParamType[0])
	if _, ok := parameter.(*dwarf.IntType); !ok || parameter.Size() != 8 {
		t.Errorf("function parameter = %T size %d, want NamedInt", parameter, parameter.Size())
	}
	results, ok := unwrapDWARFTypedef(function.ReturnType).(*dwarf.StructType)
	if !ok {
		t.Fatalf("function result = %T, want two-value tuple", unwrapDWARFTypedef(function.ReturnType))
	}
	ptrSize := int64(dataAddressSize())
	if results.Size() != 8+2*ptrSize || len(results.Field) != 2 {
		t.Fatalf("function result tuple = size %d, %d fields; want size %d, 2 fields", results.Size(), len(results.Field), 8+2*ptrSize)
	}
	if results.Field[0].ByteOffset != 0 || results.Field[1].ByteOffset != 8 {
		t.Errorf("function result offsets = %d, %d; want 0, 8", results.Field[0].ByteOffset, results.Field[1].ByteOffset)
	}
	assertDWARFWordStruct(t, "error interface", results.Field[1].Type, ptrSize, "type", "data")
}

type dwarfFixtureNamedInt int64

type dwarfFixtureRecursive struct {
	Value dwarfFixtureNamedInt
	Next  *dwarfFixtureRecursive
}

type dwarfFixturePair struct {
	First  dwarfFixtureNamedInt
	Second dwarfFixtureNamedInt
}

type dwarfFixtureFuncValue struct {
	code    unsafe.Pointer
	context unsafe.Pointer
}

type dwarfFixtureNumber interface {
	Number() dwarfFixtureNamedInt
}

type dwarfFixtureSample struct {
	Bool     bool
	I8       int8
	I16      int16
	I32      int32
	I64      int64
	U8       uint8
	U16      uint16
	U32      uint32
	U64      uint64
	F32      float32
	F64      float64
	C64      complex64
	C128     complex128
	Text     string
	Values   []dwarfFixtureNamedInt
	Fixed    [3]uint16
	Lookup   map[string]dwarfFixtureNamedInt
	Queue    chan dwarfFixturePair
	Callback dwarfFixtureFuncValue
	Any      any
	Iface    dwarfFixtureNumber
	Pointer  *dwarfFixtureRecursive
	Unsafe   unsafe.Pointer
	Pair     dwarfFixturePair
}

func dataAddressSize() int {
	if ^uintptr(0)>>32 == 0 {
		return 4
	}
	return 8
}
