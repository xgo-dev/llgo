package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/types"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type goRelocation struct {
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
	Kind   string `json:"kind"`
	Symbol string `json:"symbol"`
	Addend int64  `json:"addend"`
}

type goDataSymbol struct {
	Size        int            `json:"size"`
	Raw         string         `json:"raw"`
	Relocations []goRelocation `json:"relocations"`
	data        []byte
	written     int
}

type goMetadataReference struct {
	Raw        uint64        `json:"raw"`
	Relocation *goRelocation `json:"relocation,omitempty"`
}

type goMetadataHeader struct {
	Raw        string              `json:"raw"`
	Size       uint64              `json:"size"`
	PtrBytes   uint64              `json:"ptr_bytes"`
	Hash       uint32              `json:"hash"`
	TFlag      byte                `json:"tflag"`
	Align      byte                `json:"align"`
	FieldAlign byte                `json:"field_align"`
	Kind       byte                `json:"kind"`
	Equal      goMetadataReference `json:"equal"`
	GCData     goMetadataReference `json:"gc_data"`
	Name       goMetadataReference `json:"name"`
	PtrToThis  goMetadataReference `json:"ptr_to_this"`
}

type goMetadata struct {
	PointerSymbol string            `json:"pointer_symbol"`
	TypeSymbol    string            `json:"type_symbol"`
	PointerHeader *goMetadataHeader `json:"pointer_header"`
	TypeHeader    *goMetadataHeader `json:"type_header"`
	Layout        goStorageLayout   `json:"layout"`
	Fields        []goMetadataField `json:"fields"`
	storage       *goStorageType
}

type goMetadataField struct {
	Name       string `json:"name"`
	Offset     uint64 `json:"offset"`
	TypeSymbol string `json:"type_symbol"`
	storage    *goStorageType
}

// Only inline struct/array edges are followed. Pointer and container fields
// keep their own header, so recursive types cannot create a layout cycle.
type goStorageType struct {
	header *goMetadataHeader
	fields []goMetadataField
	elem   *goStorageType
}

// The model supplies storage facts only. Names, Kind and comparability must
// still come from the original type rather than a synthetic closure struct.
type goStorageLayout struct {
	TypeSymbol string `json:"type_symbol"`
	Size       uint64 `json:"size"`
	PtrBytes   uint64 `json:"ptr_bytes"`
	Align      byte   `json:"align"`
	FieldAlign byte   `json:"field_align"`
}

func readGoSymbols(path, anchorPrefix string, addendBase int) (map[string]*goDataSymbol, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	header := regexp.MustCompile(`^(.*) (?:SRODATA|SDATA|SNOPTRDATA|SBSS|SNOPTRBSS)(?: \S+)* size=([0-9]+)(?: align=0x[0-9a-fA-F]+)?$`)
	relocation := regexp.MustCompile(`^\s*rel ([0-9]+)\+([0-9]+) t=(\S+) (.*)\+(-?[0-9a-fA-F]+)$`)
	symbols := make(map[string]*goDataSymbol)
	var current *goDataSymbol
	reader := bufio.NewReader(file)
	for lineNumber := 1; ; lineNumber++ {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		line = strings.TrimRight(line, "\r\n")
		if match := header.FindStringSubmatch(line); match != nil {
			current = nil
			name := match[1]
			if strings.HasPrefix(name, "type:") || strings.HasPrefix(name, "runtime.gcbits.") || strings.HasPrefix(name, anchorPrefix) {
				if _, exists := symbols[name]; exists {
					return nil, fmt.Errorf("duplicate Go data symbol %s", name)
				}
				size, err := strconv.Atoi(match[2])
				if err != nil {
					return nil, err
				}
				// The compiler may omit trailing zero bytes: a static interface
				// containing (*T)(nil) prints only its first word, for example.
				current = &goDataSymbol{Size: size, data: make([]byte, size)}
				symbols[name] = current
			}
		} else if line != "" && line[0] != ' ' && line[0] != '\t' {
			current = nil
		} else if current != nil {
			text := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(text, "0x"):
				end := strings.IndexByte(text, ' ')
				if end == -1 {
					return nil, fmt.Errorf("Go data line %d has no bytes", lineNumber)
				}
				offset, err := strconv.ParseUint(text[:end], 0, 64)
				if err != nil || offset != uint64(current.written) {
					return nil, fmt.Errorf("Go data line %d has an invalid byte offset", lineNumber)
				}
				// writeSymDebugNamed emits sixteen three-character hex columns,
				// then ASCII. Restrict parsing to the hex columns, even if the
				// ASCII suffix happens to look like hexadecimal (for example ab).
				columns := text[end:min(end+48, len(text))]
				for pos := 0; pos+3 <= len(columns); pos += 3 {
					column := columns[pos : pos+3]
					if strings.TrimSpace(column) == "" {
						break
					}
					value, err := strconv.ParseUint(strings.TrimSpace(column), 16, 8)
					if err != nil || current.written >= len(current.data) {
						return nil, fmt.Errorf("Go data line %d has invalid bytes", lineNumber)
					}
					current.data[current.written] = byte(value)
					current.written++
				}
			case strings.HasPrefix(text, "rel "):
				match := relocation.FindStringSubmatch(line)
				if match == nil {
					return nil, fmt.Errorf("unrecognized Go relocation on line %d", lineNumber)
				}
				offset, err := strconv.Atoi(match[1])
				if err != nil {
					return nil, err
				}
				size, err := strconv.Atoi(match[2])
				if err != nil || offset > current.Size || size > current.Size-offset {
					return nil, fmt.Errorf("Go relocation on line %d exceeds its data symbol", lineNumber)
				}
				addend, err := strconv.ParseInt(match[5], addendBase, 64)
				if err != nil {
					return nil, err
				}
				current.Relocations = append(current.Relocations, goRelocation{offset, size, match[3], match[4], addend})
			}
		}
		if readErr == io.EOF {
			break
		}
	}
	for _, symbol := range symbols {
		symbol.Raw = hex.EncodeToString(symbol.data)
	}
	return symbols, nil
}

func goReferenceAt(symbol *goDataSymbol, offset, size int) (goMetadataReference, error) {
	var result goMetadataReference
	if offset < 0 || size > len(symbol.data)-offset {
		return result, fmt.Errorf("Go reference at %d exceeds descriptor size %d", offset, symbol.Size)
	}
	if size == 4 {
		result.Raw = uint64(binary.NativeEndian.Uint32(symbol.data[offset:]))
	} else {
		result.Raw = binary.NativeEndian.Uint64(symbol.data[offset:])
	}
	for i := range symbol.Relocations {
		reloc := &symbol.Relocations[i]
		// R_USEIFACE and similar zero-width markers do not fill an address.
		if reloc.Offset != offset || reloc.Size == 0 {
			continue
		}
		if reloc.Size != size || result.Relocation != nil {
			return result, fmt.Errorf("ambiguous Go reference at %d", offset)
		}
		result.Relocation = reloc
	}
	return result, nil
}

func decodeGoHeader(symbol *goDataSymbol, wordSize int) (*goMetadataHeader, error) {
	headerSize := 4*wordSize + 16
	if len(symbol.data) < headerSize {
		return nil, fmt.Errorf("Go type descriptor has %d bytes, want at least %d", len(symbol.data), headerSize)
	}
	word := func(data []byte) uint64 {
		if wordSize == 4 {
			return uint64(binary.NativeEndian.Uint32(data))
		}
		return binary.NativeEndian.Uint64(data)
	}
	data := symbol.data
	result := &goMetadataHeader{
		Raw: hex.EncodeToString(data[:headerSize]), Size: word(data), PtrBytes: word(data[wordSize:]),
		Hash:  binary.NativeEndian.Uint32(data[2*wordSize:]),
		TFlag: data[2*wordSize+4], Align: data[2*wordSize+5], FieldAlign: data[2*wordSize+6], Kind: data[2*wordSize+7],
	}
	for i, destination := range []*goMetadataReference{&result.Equal, &result.GCData, &result.Name, &result.PtrToThis} {
		offset, size := 2*wordSize+8+i*wordSize, wordSize
		if i >= 2 {
			offset, size = 4*wordSize+8+(i-2)*4, 4
		}
		reference, err := goReferenceAt(symbol, offset, size)
		if err != nil {
			return nil, err
		}
		*destination = reference
	}
	return result, nil
}

func readGoMetadata(root, directory, baselineDirectory, pkgPath string, checks []metadataCheck, layoutNames []string, dependencies map[string]string) ([]goMetadata, error) {
	addendBase := 10
	if runtime.GOARCH == "arm" || runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" {
		// cmd/internal/obj.writeSymDebugNamed prints these targets' addends
		// in hexadecimal without a 0x prefix.
		addendBase = 16
	}
	prefix := pkgPath + ".layoutlint"
	symbols, err := readGoSymbols(filepath.Join(directory, "go-compile.log"), prefix, addendBase)
	if err != nil {
		return nil, err
	}
	// Go emits basic descriptors in runtime and imported named descriptors in
	// their defining package. Resolve those relocations from compiler listings,
	// never from synthesized Sizeof/Alignof expressions or go/types sizes.
	loaded := make(map[string]bool)
	load := func(owner string) error {
		if loaded[owner] || owner == pkgPath {
			return nil
		}
		path := dependencies[owner]
		if path == "" {
			path = filepath.Join(baselineDirectory, fmt.Sprintf("dependency-%d.log", len(dependencies)))
			log, err := os.Create(path)
			if err != nil {
				return err
			}
			args := []string{"build", "-gcflags=" + owner + "=-S -S", owner}
			fmt.Fprintf(log, "go %q\n", args)
			compiler := exec.Command("go", args...)
			compiler.Dir = root
			compiler.Stdout, compiler.Stderr = log, log
			compileErr := compiler.Run()
			closeErr := log.Close()
			if compileErr != nil || closeErr != nil {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				failurePath := filepath.Join(directory, "go-dependency-error.log")
				if err := os.WriteFile(failurePath, data, 0644); err != nil {
					return err
				}
				return fmt.Errorf("Go metadata dependency %s: compile %v, log %v; see %s", owner, compileErr, closeErr, failurePath)
			}
			dependencies[owner] = path
		}
		imported, err := readGoSymbols(path, owner+".layoutlint", addendBase)
		if err != nil {
			return fmt.Errorf("Go metadata dependency %s: %w", owner, err)
		}
		for name, symbol := range imported {
			if symbols[name] == nil {
				symbols[name] = symbol
			}
		}
		loaded[owner] = true
		return nil
	}
	resolve := func(name string, typ types.Type) (*goDataSymbol, error) {
		if symbol := symbols[name]; symbol != nil {
			return symbol, nil
		}
		base := types.Unalias(typ)
		if pointer, ok := base.(*types.Pointer); ok {
			base = types.Unalias(pointer.Elem())
		}
		owner := "runtime"
		if named, ok := base.(*types.Named); ok && named.Obj().Pkg() != nil {
			owner = named.Obj().Pkg().Path()
		}
		if err := load(owner); err != nil {
			return nil, err
		}
		if symbol := symbols[name]; symbol != nil {
			return symbol, nil
		}
		return nil, fmt.Errorf("missing Go descriptor %s for %s after reading %s", name, typ, owner)
	}
	wordSize := strconv.IntSize / 8
	headerSize := 4*wordSize + 16
	var readStorage func(string, types.Type) (*goStorageType, error)
	readStorage = func(name string, typ types.Type) (*goStorageType, error) {
		symbol, err := resolve(name, typ)
		if err != nil {
			return nil, err
		}
		header, err := decodeGoHeader(symbol, wordSize)
		if err != nil {
			return nil, fmt.Errorf("Go descriptor %s: %w", name, err)
		}
		if header.Size > math.MaxInt64 || header.Align == 0 || header.Align&(header.Align-1) != 0 || header.FieldAlign == 0 || header.FieldAlign&(header.FieldAlign-1) != 0 {
			return nil, fmt.Errorf("invalid Go storage header %s: size %d, align %d, field align %d", name, header.Size, header.Align, header.FieldAlign)
		}
		storage := &goStorageType{header: header}
		switch underlying := types.Unalias(typ).Underlying().(type) {
		case *types.Struct:
			if header.Kind&31 != 25 {
				return nil, fmt.Errorf("Go descriptor %s is not a struct", name)
			}
			storage.fields, err = readGoStructFields(symbols, name, underlying, wordSize)
			if err != nil {
				return nil, err
			}
			for i := range storage.fields {
				field := &storage.fields[i]
				field.storage, err = readStorage(field.TypeSymbol, underlying.Field(i).Type())
				if err != nil {
					return nil, fmt.Errorf("Go field %s.%s: %w", name, field.Name, err)
				}
			}
		case *types.Array:
			if header.Kind&31 != 17 {
				return nil, fmt.Errorf("Go descriptor %s is not an array", name)
			}
			elem, err := goReferenceAt(symbol, headerSize, wordSize)
			if err != nil {
				return nil, err
			}
			if elem.Relocation == nil || elem.Relocation.Addend != 0 {
				return nil, fmt.Errorf("Go array %s has no element descriptor relocation", name)
			}
			length, err := goReferenceAt(symbol, headerSize+2*wordSize, wordSize)
			if err != nil {
				return nil, err
			}
			if length.Raw != uint64(underlying.Len()) {
				return nil, fmt.Errorf("Go array %s length %d, want %d", name, length.Raw, underlying.Len())
			}
			storage.elem, err = readStorage(elem.Relocation.Symbol, underlying.Elem())
			if err != nil {
				return nil, err
			}
		}
		return storage, nil
	}
	readRoot := func(anchorName string, typ types.Type) (goMetadata, error) {
		var result goMetadata
		anchor := symbols[anchorName]
		if anchor == nil {
			return result, fmt.Errorf("missing reference symbol %s", anchorName)
		}
		reference, err := goReferenceAt(anchor, 0, wordSize)
		if err != nil {
			return result, err
		}
		if reference.Relocation == nil || reference.Relocation.Addend != 0 {
			return result, fmt.Errorf("reference %s has no descriptor relocation", anchorName)
		}
		pointer, err := resolve(reference.Relocation.Symbol, types.NewPointer(typ))
		if err != nil {
			return result, err
		}
		elem, err := goReferenceAt(pointer, headerSize, wordSize)
		if err != nil {
			return result, err
		}
		if elem.Relocation == nil || elem.Relocation.Addend != 0 {
			return result, fmt.Errorf("reference %s pointer has no Elem relocation", anchorName)
		}
		if _, err := resolve(elem.Relocation.Symbol, typ); err != nil {
			return result, err
		}
		return readGoReferencedType(symbols, anchorName, wordSize)
	}
	results := make([]goMetadata, len(checks))
	for i, check := range checks {
		original, err := readRoot(fmt.Sprintf("%sABIReference%d<0>", prefix, i), check.typ)
		if err != nil {
			return nil, fmt.Errorf("Go metadata %s: %w", check.name, err)
		}
		storage := original
		if layoutNames[i] != check.identifier {
			storage, err = readRoot(fmt.Sprintf("%sStorageReference%d<0>", prefix, i), check.typ)
			if err != nil {
				return nil, fmt.Errorf("Go storage layout %s: %w", check.name, err)
			}
		}
		header := storage.TypeHeader
		original.Layout = goStorageLayout{storage.TypeSymbol, header.Size, header.PtrBytes, header.Align, header.FieldAlign}
		original.storage, err = readStorage(storage.TypeSymbol, check.typ)
		if err != nil {
			return nil, fmt.Errorf("Go metadata fields %s: %w", check.name, err)
		}
		original.Fields = original.storage.fields
		results[i] = original
	}
	// Include the defining-package descriptors used by fields and array elements.
	data, err := json.MarshalIndent(symbols, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(directory, "go-abi.json"), append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	return results, nil
}

// readGoStructFields follows the compiler's field-table relocation. Field
// indices include blanks; accessibility does not affect metadata coverage.
func readGoStructFields(symbols map[string]*goDataSymbol, name string, typ *types.Struct, wordSize int) ([]goMetadataField, error) {
	symbol := symbols[name]
	if symbol == nil {
		return nil, fmt.Errorf("missing struct descriptor %s", name)
	}
	headerSize := 4*wordSize + 16
	length, err := goReferenceAt(symbol, headerSize+2*wordSize, wordSize)
	if err != nil {
		return nil, err
	}
	if length.Raw != uint64(typ.NumFields()) {
		return nil, fmt.Errorf("field count %d, want %d", length.Raw, typ.NumFields())
	}
	fields := make([]goMetadataField, typ.NumFields())
	if len(fields) == 0 {
		return fields, nil
	}
	reference, err := goReferenceAt(symbol, headerSize+wordSize, wordSize)
	if err != nil {
		return nil, err
	}
	if reference.Relocation == nil {
		return nil, fmt.Errorf("field table has no relocation")
	}
	table := symbols[reference.Relocation.Symbol]
	if table == nil {
		return nil, fmt.Errorf("missing field table %s", reference.Relocation.Symbol)
	}
	for i := range fields {
		// Go StructField is Name, Typ and Offset, each one machine word.
		offset := reference.Relocation.Addend + int64((3*i+2)*wordSize)
		if offset < 0 || offset > int64(len(table.data)) {
			return nil, fmt.Errorf("field %d offset exceeds its data symbol", i)
		}
		value, err := goReferenceAt(table, int(offset), wordSize)
		if err != nil {
			return nil, err
		}
		typeReference, err := goReferenceAt(table, int(offset)-wordSize, wordSize)
		if err != nil {
			return nil, err
		}
		if value.Raw > math.MaxInt64 || typeReference.Relocation == nil || typeReference.Relocation.Addend != 0 {
			return nil, fmt.Errorf("invalid Go field %s.%s offset or type relocation", name, typ.Field(i).Name())
		}
		fields[i] = goMetadataField{Name: typ.Field(i).Name(), Offset: value.Raw, TypeSymbol: typeReference.Relocation.Symbol}
	}
	return fields, nil
}

func readGoReferencedType(symbols map[string]*goDataSymbol, anchorName string, wordSize int) (goMetadata, error) {
	var result goMetadata
	anchor := symbols[anchorName]
	if anchor == nil {
		return result, fmt.Errorf("missing reference symbol %s", anchorName)
	}
	reference, err := goReferenceAt(anchor, 0, wordSize)
	if err != nil {
		return result, err
	}
	if reference.Relocation == nil || reference.Relocation.Addend != 0 {
		return result, fmt.Errorf("reference has no descriptor relocation")
	}
	result.PointerSymbol = reference.Relocation.Symbol
	pointer := symbols[result.PointerSymbol]
	if pointer == nil {
		return result, fmt.Errorf("missing pointer descriptor %s", result.PointerSymbol)
	}
	result.PointerHeader, err = decodeGoHeader(pointer, wordSize)
	if err != nil {
		return result, err
	}
	if result.PointerHeader.Kind&31 != 22 {
		return result, fmt.Errorf("reference is not a pointer type")
	}
	elem, err := goReferenceAt(pointer, 4*wordSize+16, wordSize)
	if err != nil {
		return result, err
	}
	if elem.Relocation == nil || elem.Relocation.Addend != 0 {
		return result, fmt.Errorf("pointer has no Elem relocation")
	}
	result.TypeSymbol = elem.Relocation.Symbol
	typeSymbol := symbols[result.TypeSymbol]
	if typeSymbol == nil {
		return result, fmt.Errorf("missing type descriptor %s", result.TypeSymbol)
	}
	result.TypeHeader, err = decodeGoHeader(typeSymbol, wordSize)
	if err != nil {
		return result, err
	}
	if result.TypeHeader.Kind&31 != 25 {
		return result, fmt.Errorf("Elem is not a struct type")
	}
	return result, nil
}
