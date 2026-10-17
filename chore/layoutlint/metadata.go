package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type metadataCheck struct {
	typ        types.Type
	name, file string
	identifier string
	line       int
}

type metadataFieldCheck struct {
	field string
	path  []layoutStep
}

type metadataHeader struct {
	Raw          string    `json:"raw"`
	Size         uint64    `json:"size"`
	PtrBytes     uint64    `json:"ptr_bytes"`
	Hash         uint32    `json:"hash"`
	TFlag        byte      `json:"tflag"`
	Align        byte      `json:"align"`
	FieldAlign   byte      `json:"field_align"`
	Kind         byte      `json:"kind"`
	Equal        [2]string `json:"equal"`
	GCData       string    `json:"gc_data"`
	StringData   string    `json:"string_data"`
	StringLength uint64    `json:"string_length"`
	PtrToThis    string    `json:"ptr_to_this"`
}

type metadataDifference struct {
	Descriptor string `json:"descriptor"`
	Field      string `json:"field"`
	Expected   uint64 `json:"expected"`
	Actual     uint64 `json:"actual"`
}

type metadataResult struct {
	fieldOffsets   map[string]uint64
	Go             *goMetadata           `json:"go"`
	Type           string                `json:"type"`
	File           string                `json:"file"`
	Line           int                   `json:"line"`
	Observed       bool                  `json:"observed"`
	PointerAddress string                `json:"pointer_address,omitempty"`
	PointerSymbol  string                `json:"pointer_symbol,omitempty"`
	TypeAddress    string                `json:"type_address,omitempty"`
	TypeSymbol     string                `json:"type_symbol,omitempty"`
	PointerHeader  *metadataHeader       `json:"pointer_header,omitempty"`
	TypeHeader     *metadataHeader       `json:"type_header,omitempty"`
	Stride         *strideResult         `json:"stride,omitempty"`
	Differences    []metadataDifference  `json:"differences,omitempty"`
	Fields         []metadataFieldResult `json:"fields"`
}

type metadataFieldResult struct {
	Name     string `json:"name"`
	Expected uint64 `json:"expected_offset"`
	Actual   uint64 `json:"actual_offset"`
	Observed bool   `json:"observed"`
}

func readMetadata(session *lldbSession, directory string, index int, pointer uint64, typ types.Type, fields []metadataFieldCheck, result *metadataResult) error {
	// This is LLGo's serialized runtime/abi.Type format, not an expected Go
	// layout: two uintptrs, eight bytes of scalar header, then six words for
	// Equal's code/context, GCData, Str_ and PtrToThis_. PtrType.Elem follows.
	// Function values elsewhere are not assumed to have this representation.
	wordSize := strconv.IntSize / 8
	headerSize := 8*wordSize + 8
	word := func(data []byte) uint64 {
		if wordSize == 4 {
			return uint64(binary.NativeEndian.Uint32(data))
		}
		return binary.NativeEndian.Uint64(data)
	}
	read := func(address uint64, count int, label string) ([]byte, string, error) {
		if address == 0 {
			return nil, "", fmt.Errorf("%s descriptor is missing (nil address)", label)
		}
		lookup, err := session.run(fmt.Sprintf("image lookup --verbose --address 0x%x", address))
		if err != nil {
			return nil, "", err
		}
		// Symbol ranges locate globals without consulting DWARF type layouts.
		symbol := regexp.MustCompile(`(?m)^\s*Symbol:.*range = \[(0x[0-9a-fA-F]+)-(0x[0-9a-fA-F]+)\), name="([^"]+)"`).FindStringSubmatch(lookup)
		if symbol == nil {
			return nil, "", fmt.Errorf("%s descriptor at 0x%x has no symbol", label, address)
		}
		start, err := strconv.ParseUint(symbol[1], 0, 64)
		if err != nil {
			return nil, "", err
		}
		end, err := strconv.ParseUint(symbol[2], 0, 64)
		if err != nil {
			return nil, "", err
		}
		if start != address || end < start || end-start < uint64(count) {
			return nil, "", fmt.Errorf("%s descriptor at 0x%x is outside symbol range [%s,%s)", label, address, symbol[1], symbol[2])
		}
		path := filepath.Join(directory, fmt.Sprintf("metadata-%d-%s.bin", index, label))
		if _, err := session.run(fmt.Sprintf("memory read --binary --force --size 1 --count %d --outfile %q 0x%x", count, path, address)); err != nil {
			return nil, "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, "", err
		}
		if len(data) != count {
			return nil, "", fmt.Errorf("%s descriptor: read %d bytes, want %d", label, len(data), count)
		}
		return data, symbol[3], nil
	}
	decode := func(data []byte) *metadataHeader {
		tail := 2*wordSize + 8
		return &metadataHeader{
			Raw:  hex.EncodeToString(data[:headerSize]),
			Size: word(data), PtrBytes: word(data[wordSize:]),
			Hash:  binary.NativeEndian.Uint32(data[2*wordSize:]),
			TFlag: data[2*wordSize+4], Align: data[2*wordSize+5],
			FieldAlign: data[2*wordSize+6], Kind: data[2*wordSize+7],
			Equal:        [2]string{fmt.Sprintf("0x%x", word(data[tail:])), fmt.Sprintf("0x%x", word(data[tail+wordSize:]))},
			GCData:       fmt.Sprintf("0x%x", word(data[tail+2*wordSize:])),
			StringData:   fmt.Sprintf("0x%x", word(data[tail+3*wordSize:])),
			StringLength: word(data[tail+4*wordSize:]),
			PtrToThis:    fmt.Sprintf("0x%x", word(data[tail+5*wordSize:])),
		}
	}
	result.PointerAddress = fmt.Sprintf("0x%x", pointer)
	data, symbol, err := read(pointer, headerSize+wordSize, "pointer")
	if err != nil {
		return err
	}
	result.PointerSymbol, result.PointerHeader = symbol, decode(data)
	// The retained value is *T, and every eligible T is a concrete struct.
	// These checks verify the capture path, not cross-compiler field policy.
	if result.PointerHeader.Kind&31 != 22 {
		return fmt.Errorf("retained descriptor has kind %d, want pointer", result.PointerHeader.Kind&31)
	}
	typeAddress := word(data[headerSize:])
	result.TypeAddress = fmt.Sprintf("0x%x", typeAddress)
	data, symbol, err = read(typeAddress, headerSize, "type")
	if err != nil {
		return err
	}
	result.TypeSymbol, result.TypeHeader = symbol, decode(data)
	if result.TypeHeader.Kind&31 != 25 {
		return fmt.Errorf("retained pointer's Elem has kind %d, want struct", result.TypeHeader.Kind&31)
	}
	result.Observed = true

	// Decode only the descriptor edges needed by existing field probes. A
	// path such as Rows[1].Inner.X follows Fields, ArrayType.Elem and Fields;
	// it never walks every array element or follows a value's pointer graph.
	type fieldEntry struct {
		typ, offset uint64
	}
	type layout struct {
		kind   byte
		fields []fieldEntry
		elem   uint64
		length uint64
	}
	layouts := make(map[uint64]*layout)
	getLayout := func(address uint64, typ types.Type) (*layout, error) {
		underlying := types.Unalias(typ).Underlying()
		kind, count := byte(25), headerSize+5*wordSize
		if _, array := underlying.(*types.Array); array {
			kind, count = 17, headerSize+3*wordSize
		}
		view := layouts[address]
		if view == nil {
			label := fmt.Sprintf("layout-%d", len(layouts))
			data, _, err := read(address, count, label)
			if err != nil {
				return nil, err
			}
			view = &layout{kind: data[2*wordSize+7] & 31}
			if view.kind != kind {
				return nil, fmt.Errorf("field path type %s: descriptor kind %d, want %d", typ, view.kind, kind)
			}
			switch underlying := underlying.(type) {
			case *types.Struct:
				// StructType appends PkgPath (string) and Fields (slice).
				base := word(data[headerSize+2*wordSize:])
				length, capacity := word(data[headerSize+3*wordSize:]), word(data[headerSize+4*wordSize:])
				if length != uint64(underlying.NumFields()) || capacity < length {
					return nil, fmt.Errorf("%s: metadata field count/capacity %d/%d, want %d fields", typ, length, capacity, underlying.NumFields())
				}
				if length != 0 {
					// StructField is Name (2 words), Typ, Offset, Tag (2 words),
					// Embedded (one byte), then padding to a word boundary.
					data, _, err := read(base, underlying.NumFields()*7*wordSize, label+"-fields")
					if err != nil {
						return nil, err
					}
					view.fields = make([]fieldEntry, underlying.NumFields())
					for i := range view.fields {
						entry := data[i*7*wordSize:]
						view.fields[i] = fieldEntry{typ: word(entry[2*wordSize:]), offset: word(entry[3*wordSize:])}
					}
				}
			case *types.Array:
				// ArrayType appends Elem, Slice and Len to the common header.
				view.elem, view.length = word(data[headerSize:]), word(data[headerSize+2*wordSize:])
			}
			layouts[address] = view
		}
		if view.kind != kind {
			return nil, fmt.Errorf("field path type %s: descriptor kind %d, want %d", typ, view.kind, kind)
		}
		switch underlying := underlying.(type) {
		case *types.Struct:
			if len(view.fields) != underlying.NumFields() {
				return nil, fmt.Errorf("%s: metadata has %d fields, want %d", typ, len(view.fields), underlying.NumFields())
			}
		case *types.Array:
			if view.length != uint64(underlying.Len()) {
				return nil, fmt.Errorf("%s: metadata array length %d, want %d", typ, view.length, underlying.Len())
			}
		}
		return view, nil
	}
	result.fieldOffsets = make(map[string]uint64, len(fields))
	root, err := getLayout(typeAddress, typ)
	if err != nil {
		return err
	}
	for i, field := range root.fields {
		result.Fields[i].Actual = field.offset
		result.Fields[i].Observed = true
	}
	for _, field := range fields {
		address, current := typeAddress, typ
		var offset uint64
		for _, step := range field.path {
			view, err := getLayout(address, current)
			if err != nil {
				return fmt.Errorf("%s: %w", field.field, err)
			}
			if step.field != nil {
				st := types.Unalias(current).Underlying().(*types.Struct)
				fieldIndex := -1
				for i := 0; i < st.NumFields(); i++ {
					if st.Field(i) == step.field {
						fieldIndex = i
						break
					}
				}
				if fieldIndex < 0 {
					return fmt.Errorf("%s: field %s is not in source type %s", field.field, step.field.Name(), current)
				}
				entry := view.fields[fieldIndex]
				offset, address, current = entry.offset, entry.typ, step.field.Type()
			} else {
				address, current = view.elem, types.Unalias(current).Underlying().(*types.Array).Elem()
			}
		}
		result.fieldOffsets[field.field] = offset
	}
	return nil
}
