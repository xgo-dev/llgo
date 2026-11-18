package browserdebug

import (
	"debug/dwarf"
	"fmt"
	"sort"

	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

// LLVM can move a variable's name and type to an abstract origin after
// inlining. Location/range attributes still belong to the concrete instance.
func (b *indexBuilder) attribute(entry *dwarf.Entry, attr dwarf.Attr) any {
	seen := make(map[dwarf.Offset]bool)
	for entry != nil && !seen[entry.Offset] {
		seen[entry.Offset] = true
		if value := entry.Val(attr); value != nil {
			return value
		}
		offset, ok := entry.Val(dwarf.AttrAbstractOrigin).(dwarf.Offset)
		if !ok {
			offset, ok = entry.Val(dwarf.AttrSpecification).(dwarf.Offset)
		}
		if !ok {
			break
		}
		reader := b.data.Reader()
		reader.Seek(offset)
		entry, _ = reader.Next()
	}
	return nil
}

func (b *indexBuilder) addClosureEnv(entry *dwarf.Entry, name string) {
	if name == "" || b.attribute(entry, dwarf.AttrArtificial) != true {
		return
	}
	offset, ok := b.attribute(entry, dwarf.AttrType).(dwarf.Offset)
	if !ok {
		return
	}
	ty, err := b.data.Type(offset)
	if err != nil {
		return
	}
	ptr, ok := ty.(*dwarf.PtrType)
	if !ok {
		return
	}
	if env, ok := ptr.Type.(*dwarf.StructType); !ok || env.Incomplete {
		return
	}
	id := b.addType(ty)
	if b.closureEnvs == nil {
		b.closureEnvs = make(map[string]string)
	}
	if previous, exists := b.closureEnvs[name]; exists && previous != id {
		id = "" // Conflicting descriptions must never select an arbitrary layout.
	}
	b.closureEnvs[name] = id
}

func (b *indexBuilder) addFunctionValues(module []byte) {
	table, err := wasmdebug.FunctionTable(module)
	if err != nil {
		b.index.Diagnostics = append(b.index.Diagnostics, fmt.Sprintf("function table: %v", err))
		return
	}
	names, err := wasmdebug.FunctionNames(module)
	if err != nil {
		b.index.Diagnostics = append(b.index.Diagnostics, fmt.Sprintf("function names: %v", err))
		return
	}
	counts := make(map[string]int)
	for _, name := range names {
		counts[name]++
	}
	for index, function := range table {
		if name := names[function]; index != 0 && name != "" {
			environment := b.closureEnvs[name]
			if counts[name] != 1 {
				environment = ""
			}
			b.index.FunctionValues = append(b.index.FunctionValues, FunctionValue{
				Index: index, Name: name, Environment: environment,
			})
		}
	}
	sort.Slice(b.index.FunctionValues, func(i, j int) bool {
		return b.index.FunctionValues[i].Index < b.index.FunctionValues[j].Index
	})
}
