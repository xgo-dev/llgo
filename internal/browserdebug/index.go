// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package browserdebug converts standards-based WebAssembly DWARF into the
// compact query index used by LLGo's Chrome Language Extension. DWARF remains
// the source of truth; the index is generated for a debug session and is not a
// compiler-owned replacement debug format.
package browserdebug

import (
	"bytes"
	"debug/dwarf"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

const IndexVersion = 1

// PathMapping maps a source prefix recorded by the compiler to a local source
// prefix. The longest matching From prefix wins.
type PathMapping struct {
	From string
	To   string
}

// ParsePathMapping parses the debugger spelling FROM=TO.
func ParsePathMapping(value string) (PathMapping, error) {
	from, to, ok := strings.Cut(value, "=")
	if !ok || from == "" || to == "" {
		return PathMapping{}, fmt.Errorf("source mapping %q must be FROM=TO", value)
	}
	return PathMapping{From: filepath.Clean(from), To: filepath.Clean(to)}, nil
}

// Index is the transport-neutral state consumed by the browser extension.
type Index struct {
	Contract       string          `json:"contract"`
	Version        int             `json:"version"`
	BuildID        string          `json:"build_id"`
	Artifact       string          `json:"artifact"`
	Record         Record          `json:"record"`
	Sources        []Source        `json:"sources"`
	Lines          []LineRange     `json:"lines"`
	Functions      []Function      `json:"functions"`
	FunctionValues []FunctionValue `json:"function_values,omitempty"`
	Variables      []Variable      `json:"variables"`
	Types          []Type          `json:"types"`
	Diagnostics    []string        `json:"diagnostics,omitempty"`
}

type Record struct {
	RecordVersion        uint8              `json:"record_version"`
	SchemaVersion        uint8              `json:"schema_version"`
	RuntimeLayoutVersion uint8              `json:"runtime_layout_version"`
	LLGoABIVersion       uint8              `json:"llgo_abi_version"`
	PointerSize          uint8              `json:"pointer_size"`
	ByteOrder            debugabi.ByteOrder `json:"byte_order"`
}

type Source struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	URL   string `json:"url"`
	Local bool   `json:"local"`
}

type AddressRange struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type LineRange struct {
	Source string `json:"source"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Start  uint64 `json:"start"`
	End    uint64 `json:"end"`
}

type Function struct {
	Name   string         `json:"name"`
	Ranges []AddressRange `json:"ranges"`
}

// FunctionValue joins a module-owned indirect-call index to its DWARF name
// and optional, compiler-described closure environment pointer type.
type FunctionValue struct {
	Index       uint32 `json:"index"`
	Name        string `json:"name"`
	Environment string `json:"environment,omitempty"`
}

type Location struct {
	Start      uint64 `json:"start,omitempty"`
	End        uint64 `json:"end,omitempty"`
	Expression string `json:"expression"`
}

type Constant struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Variable struct {
	Name      string         `json:"name"`
	Scope     string         `json:"scope"`
	Type      string         `json:"type"`
	Ranges    []AddressRange `json:"ranges,omitempty"`
	Locations []Location     `json:"locations,omitempty"`
	FrameBase []Location     `json:"frame_base,omitempty"`
	Constant  *Constant      `json:"constant,omitempty"`
	Depth     int            `json:"depth"`
}

type Type struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Kind     string      `json:"kind"`
	Size     int64       `json:"size"`
	Signed   bool        `json:"signed,omitempty"`
	Elem     string      `json:"elem,omitempty"`
	Count    int64       `json:"count,omitempty"`
	Fields   []TypeField `json:"fields,omitempty"`
	Enum     []EnumValue `json:"enum,omitempty"`
	Complete bool        `json:"complete"`
}

type TypeField struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Offset int64  `json:"offset"`
}

type EnumValue struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// Bundle retains the index and the local files needed to serve a browser
// session. SourceFiles is keyed by Source.ID.
type Bundle struct {
	Index       Index
	MainPath    string
	SymbolsPath string
	SourceFiles map[string]string
	sourceRoots []string
}

// ReadSource reads an indexed source within its trusted filesystem root.
// Root.ReadFile also prevents a symlink replacement after indexing from
// redirecting an HTTP request outside that root.
func (b *Bundle) ReadSource(id string) ([]byte, error) {
	path, ok := b.SourceFiles[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	for _, directory := range b.sourceRoots {
		if relative, ok := pathPrefix(path, directory); ok {
			root, err := os.OpenRoot(directory)
			if err != nil {
				return nil, err
			}
			defer root.Close()
			return root.ReadFile(relative)
		}
	}
	return nil, os.ErrPermission
}

// MissingSymbolsError reports an unavailable external_debug_info target.
type MissingSymbolsError struct {
	URL  string
	Path string
	Err  error
}

func (e *MissingSymbolsError) Error() string {
	return fmt.Sprintf("external WebAssembly DWARF %q is unavailable at %q: %v", e.URL, e.Path, e.Err)
}

func (e *MissingSymbolsError) Unwrap() error { return e.Err }

// Load reads an embedded or external LLGo WebAssembly artifact and builds its
// browser query index. External sidecars must carry the same standard build_id
// as the main module. Sources are exposed only within the module directory,
// explicit mapping destinations, or sourceRoots supplied by a trusted build.
func Load(mainPath string, mappings []PathMapping, sourceRoots ...string) (*Bundle, error) {
	mainPath, err := filepath.Abs(mainPath)
	if err != nil {
		return nil, err
	}
	main, err := os.ReadFile(mainPath)
	if err != nil {
		return nil, fmt.Errorf("read browser WebAssembly artifact: %w", err)
	}
	record, ok, err := wasmdebug.DebuggerRecord(main)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("browser WebAssembly artifact has no LLGo debugger ABI record")
	}
	buildID, ok, err := wasmdebug.BuildID(main)
	if err != nil {
		return nil, err
	}
	if !ok || len(buildID) == 0 {
		return nil, errors.New("browser WebAssembly artifact has no build_id")
	}

	symbolsPath := mainPath
	symbols := main
	hasDWARF, err := wasmdebug.HasDWARF(main)
	if err != nil {
		return nil, err
	}
	external, hasExternal, err := wasmdebug.ExternalURL(main)
	if err != nil {
		return nil, err
	}
	if hasDWARF && hasExternal {
		return nil, errors.New("browser WebAssembly artifact contains both embedded and external DWARF")
	}
	artifactMode := "embedded"
	if !hasDWARF {
		if !hasExternal {
			return nil, errors.New("browser WebAssembly artifact contains no DWARF")
		}
		artifactMode = "external"
		symbolsPath, err = localExternalPath(mainPath, external)
		if err != nil {
			return nil, err
		}
		root, openErr := os.OpenRoot(filepath.Dir(mainPath))
		if openErr != nil {
			return nil, openErr
		}
		relative, _ := filepath.Rel(filepath.Dir(mainPath), symbolsPath)
		symbols, err = root.ReadFile(relative)
		root.Close()
		if err != nil {
			return nil, &MissingSymbolsError{URL: external, Path: symbolsPath, Err: err}
		}
		sidecarID, sidecarHasID, err := wasmdebug.BuildID(symbols)
		if err != nil {
			return nil, fmt.Errorf("read external WebAssembly DWARF build ID: %w", err)
		}
		if !sidecarHasID {
			return nil, errors.New("external WebAssembly DWARF has no build_id")
		}
		if !bytes.Equal(buildID, sidecarID) {
			return nil, fmt.Errorf("stale external WebAssembly DWARF: main build_id %x does not match sidecar %x", buildID, sidecarID)
		}
		sidecarRecord, sidecarHasRecord, err := wasmdebug.DebuggerRecord(symbols)
		if err != nil {
			return nil, fmt.Errorf("read external WebAssembly debugger record: %w", err)
		}
		if !sidecarHasRecord || sidecarRecord != record {
			return nil, errors.New("external WebAssembly debugger record does not match main module")
		}
		if sidecarHasDWARF, err := wasmdebug.HasDWARF(symbols); err != nil {
			return nil, err
		} else if !sidecarHasDWARF {
			return nil, errors.New("external WebAssembly DWARF sidecar contains no DWARF")
		}
	}

	sections, err := wasmdebug.DWARFSections(symbols)
	if err != nil {
		return nil, err
	}
	data, err := dwarfData(sections)
	if err != nil {
		return nil, fmt.Errorf("read WebAssembly DWARF: %w", err)
	}
	roots := append([]string{filepath.Dir(mainPath)}, sourceRoots...)
	for _, mapping := range mappings {
		roots = append(roots, mapping.To)
	}
	var trustedRoots []string
	for _, root := range roots {
		if canonical, err := filepath.EvalSymlinks(root); err == nil {
			if absolute, err := filepath.Abs(canonical); err == nil {
				trustedRoots = append(trustedRoots, absolute)
			}
		}
	}
	builder := indexBuilder{
		sourceRoots: trustedRoots,
		data:        data,
		sections:    sections,
		mappings:    normalizeMappings(mappings),
		index: Index{
			Sources:   []Source{},
			Lines:     []LineRange{},
			Functions: []Function{},
			Variables: []Variable{},
			Types:     []Type{},
		},
		sourceByPath: make(map[string]string),
		sourceFiles:  make(map[string]string),
		typeByKey:    make(map[uintptr]string),
	}
	if err := builder.build(); err != nil {
		return nil, err
	}
	// Externalize preserves element/name sections; the build_id check above
	// ensures the sidecar describes the same linked module as main.
	builder.addFunctionValues(symbols)
	builder.index.Contract = "llgo.browser.debug"
	builder.index.Version = IndexVersion
	builder.index.BuildID = hex.EncodeToString(buildID)
	builder.index.Artifact = artifactMode
	builder.index.Record = Record{
		RecordVersion:        record.RecordVersion,
		SchemaVersion:        record.SchemaVersion,
		RuntimeLayoutVersion: record.RuntimeLayoutVersion,
		LLGoABIVersion:       record.LLGoABIVersion,
		PointerSize:          record.PointerSize,
		ByteOrder:            record.ByteOrder,
	}
	builder.sort()
	return &Bundle{
		Index:       builder.index,
		MainPath:    mainPath,
		SymbolsPath: symbolsPath,
		SourceFiles: builder.sourceFiles,
		sourceRoots: trustedRoots,
	}, nil
}

func localExternalPath(mainPath, reference string) (string, error) {
	parsed, err := url.Parse(reference)
	if err != nil {
		return "", fmt.Errorf("parse external WebAssembly DWARF URL: %w", err)
	}
	if parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("external WebAssembly DWARF URL %q is not a local relative URL", reference)
	}
	decoded, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", fmt.Errorf("decode external WebAssembly DWARF URL: %w", err)
	}
	if decoded == "" {
		return "", errors.New("external WebAssembly DWARF URL is empty")
	}
	// URL paths use forward slashes on every host. Reject Windows separators
	// too, so a module cannot change meaning when moved between hosts.
	if strings.Contains(decoded, "\\") || !filepath.IsLocal(filepath.FromSlash(decoded)) {
		return "", fmt.Errorf("external WebAssembly DWARF URL %q escapes the module directory", reference)
	}
	root := filepath.Dir(mainPath)
	path := filepath.Join(root, filepath.FromSlash(decoded))
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, nil // Preserve MissingSymbolsError for a valid missing sidecar.
		}
		return "", err
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	if _, ok := pathPrefix(canonical, canonicalRoot); !ok {
		return "", fmt.Errorf("external WebAssembly DWARF URL %q escapes the module directory", reference)
	}
	return path, nil
}

func dwarfData(sections map[string][]byte) (*dwarf.Data, error) {
	data, err := dwarf.New(
		sections[".debug_abbrev"], sections[".debug_aranges"],
		sections[".debug_frame"], sections[".debug_info"],
		sections[".debug_line"], sections[".debug_pubnames"],
		sections[".debug_ranges"], sections[".debug_str"],
	)
	if err != nil {
		return nil, err
	}
	for name, contents := range sections {
		switch name {
		case ".debug_abbrev", ".debug_aranges", ".debug_frame", ".debug_info",
			".debug_line", ".debug_pubnames", ".debug_ranges", ".debug_str":
			continue
		}
		if err := data.AddSection(name, contents); err != nil {
			return nil, err
		}
	}
	return data, nil
}

type scopeState struct {
	function     bool
	functionName string
	ranges       []AddressRange
	frameBase    []Location
}

type indexBuilder struct {
	addressSize  int
	data         *dwarf.Data
	sections     map[string][]byte
	mappings     []PathMapping
	index        Index
	sourceRoots  []string
	sourceByPath map[string]string
	sourceFiles  map[string]string
	typeByKey    map[uintptr]string
	closureEnvs  map[string]string
}

func (b *indexBuilder) build() error {
	reader := b.data.Reader()
	stack := []scopeState{{}}
	for {
		entry, err := reader.Next()
		if err != nil {
			return fmt.Errorf("read WebAssembly DWARF entry: %w", err)
		}
		if entry == nil {
			return nil
		}
		if entry.Tag == 0 {
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		current := stack[len(stack)-1]
		ranges := b.ranges(entry)
		switch entry.Tag {
		case dwarf.TagCompileUnit:
			b.addressSize = reader.AddressSize()
			if err := b.addLines(entry); err != nil {
				b.index.Diagnostics = append(b.index.Diagnostics, err.Error())
			}
		case dwarf.TagSubprogram, dwarf.TagInlinedSubroutine:
			name, _ := b.attribute(entry, dwarf.AttrName).(string)
			if linkage, ok := b.attribute(entry, dwarf.AttrLinkageName).(string); ok && linkage != "" {
				name = linkage
			}
			if name != "" && len(ranges) != 0 {
				b.index.Functions = append(b.index.Functions, Function{Name: name, Ranges: ranges})
			}
			current.function = true
			current.functionName = name
			// An abstract or eliminated function has no execution scope. Do not
			// inherit the enclosing function's ranges for its local declarations.
			current.ranges = ranges
			if entry.Tag == dwarf.TagSubprogram {
				current.frameBase = nil
			}
			if raw := entry.Val(dwarf.AttrFrameBase); raw != nil {
				current.frameBase = b.locations(raw, "frame base")
			}
		case dwarf.TagLexDwarfBlock, dwarf.TagTryDwarfBlock, dwarf.TagCatchDwarfBlock:
			if len(ranges) != 0 {
				current.ranges = ranges
			}
		case dwarf.TagVariable, dwarf.TagFormalParameter:
			b.addVariable(entry, stack[len(stack)-1], len(stack)-1)
		}
		if entry.Children {
			stack = append(stack, current)
		}
	}
}

func (b *indexBuilder) ranges(entry *dwarf.Entry) []AddressRange {
	raw, err := b.data.Ranges(entry)
	if err != nil {
		b.index.Diagnostics = append(b.index.Diagnostics,
			fmt.Sprintf("DWARF ranges at %#x: %v", entry.Offset, err))
		return nil
	}
	result := make([]AddressRange, 0, len(raw))
	for _, item := range raw {
		if item[1] > item[0] {
			result = append(result, AddressRange{Start: item[0], End: item[1]})
		}
	}
	return result
}

func (b *indexBuilder) addLines(unit *dwarf.Entry) error {
	reader, err := b.data.LineReader(unit)
	if err != nil {
		return fmt.Errorf("read line table at %#x: %w", unit.Offset, err)
	}
	if reader == nil {
		return nil
	}
	var previous dwarf.LineEntry
	havePrevious := false
	for {
		var current dwarf.LineEntry
		err := reader.Next(&current)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read line table at %#x: %w", unit.Offset, err)
		}
		if havePrevious && !previous.EndSequence && current.Address > previous.Address && previous.File != nil {
			source := b.addSource(previous.File.Name)
			if source != "" {
				line, column := previous.Line-1, previous.Column
				if line < 0 {
					line = 0
				}
				if column > 0 {
					column--
				}
				b.index.Lines = append(b.index.Lines, LineRange{
					Source: source, Line: line, Column: column,
					Start: previous.Address, End: current.Address,
				})
			}
		}
		previous = current
		havePrevious = !current.EndSequence
	}
	return nil
}

func (b *indexBuilder) addSource(recorded string) string {
	if recorded == "" {
		return ""
	}
	recorded = filepath.Clean(recorded)
	if id, ok := b.sourceByPath[recorded]; ok {
		return id
	}
	idBytes := []byte(recorded)
	// FNV-sized stable IDs are sufficient here; retain the complete path in the
	// index and use a collision suffix if a future fixture ever needs one.
	var hash uint64 = 1469598103934665603
	for _, value := range idBytes {
		hash ^= uint64(value)
		hash *= 1099511628211
	}
	id := fmt.Sprintf("s%016x", hash)
	for suffix := 1; ; suffix++ {
		collision := false
		for _, source := range b.index.Sources {
			if source.ID == id && source.Path != recorded {
				collision = true
				break
			}
		}
		if !collision {
			break
		}
		id = fmt.Sprintf("s%016x-%d", hash, suffix)
	}
	local, canonicalErr := filepath.EvalSymlinks(b.localSourcePath(recorded))
	available := false
	if canonicalErr == nil {
		local, canonicalErr = filepath.Abs(local)
		if info, err := os.Stat(local); canonicalErr == nil && err == nil && info.Mode().IsRegular() {
			for _, root := range b.sourceRoots {
				if _, ok := pathPrefix(local, root); ok {
					available = true
					break
				}
			}
		}
	}
	b.index.Sources = append(b.index.Sources, Source{
		ID: id, Path: recorded, URL: "/__llgo/source/" + id, Local: available,
	})
	b.sourceByPath[recorded] = id
	if available {
		b.sourceFiles[id] = local
	}
	return id
}

func (b *indexBuilder) localSourcePath(recorded string) string {
	for _, mapping := range b.mappings {
		if suffix, ok := pathPrefix(recorded, mapping.From); ok {
			return filepath.Join(mapping.To, suffix)
		}
	}
	return recorded
}

func normalizeMappings(mappings []PathMapping) []PathMapping {
	result := append([]PathMapping(nil), mappings...)
	for index := range result {
		result[index].From = filepath.Clean(result[index].From)
		result[index].To = filepath.Clean(result[index].To)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return len(result[i].From) > len(result[j].From)
	})
	return result
}

func pathPrefix(path, prefix string) (string, bool) {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	if path == prefix {
		return "", true
	}
	withSeparator := prefix + string(filepath.Separator)
	if strings.HasPrefix(path, withSeparator) {
		return strings.TrimPrefix(path, withSeparator), true
	}
	return "", false
}

func (b *indexBuilder) addVariable(entry *dwarf.Entry, state scopeState, depth int) {
	name, _ := b.attribute(entry, dwarf.AttrName).(string)
	if name == "" {
		return
	}
	if name == "$env" {
		b.addClosureEnv(entry, state.functionName)
		return
	}
	value := Variable{Name: name, Depth: depth}
	if entry.Tag == dwarf.TagFormalParameter {
		value.Scope = "PARAMETER"
	} else if state.function {
		value.Scope = "LOCAL"
	} else {
		value.Scope = "GLOBAL"
	}
	if value.Scope != "GLOBAL" && len(state.ranges) == 0 {
		// Empty ranges mean unrestricted visibility in the transport index.
		// Optimized-out functions must not leak locals into every paused frame.
		return
	}
	value.Ranges = append(value.Ranges, state.ranges...)
	value.FrameBase = append(value.FrameBase, state.frameBase...)
	if typeOffset, ok := b.attribute(entry, dwarf.AttrType).(dwarf.Offset); ok {
		if dwarfType, err := b.data.Type(typeOffset); err == nil {
			value.Type = b.addType(dwarfType)
		} else {
			b.index.Diagnostics = append(b.index.Diagnostics,
				fmt.Sprintf("DWARF type at %#x: %v", typeOffset, err))
		}
	}
	value.Constant = constantValue(entry.Val(dwarf.AttrConstValue))
	value.Locations = b.locations(entry.Val(dwarf.AttrLocation), name)
	// Storage-free declarations are intentionally absent from the scope view;
	// this matches the native adapters' optimized-out policy.
	if value.Constant == nil && len(value.Locations) == 0 {
		return
	}
	b.index.Variables = append(b.index.Variables, value)
}

func (b *indexBuilder) locations(raw any, name string) []Location {
	switch location := raw.(type) {
	case []byte:
		return []Location{{Expression: hex.EncodeToString(location)}}
	case int64:
		locations, err := parseDebugLoc(b.sections[".debug_loc"], uint64(location), b.addressSize)
		if err == nil {
			return locations
		}
		b.index.Diagnostics = append(b.index.Diagnostics, fmt.Sprintf("DWARF location for %s: %v", name, err))
	}
	return nil
}

func constantValue(raw any) *Constant {
	switch value := raw.(type) {
	case int64:
		return &Constant{Kind: "signed", Value: fmt.Sprint(value)}
	case uint64:
		return &Constant{Kind: "unsigned", Value: fmt.Sprint(value)}
	case string:
		return &Constant{Kind: "string", Value: value}
	case []byte:
		return &Constant{Kind: "bytes", Value: hex.EncodeToString(value)}
	default:
		return nil
	}
}

func parseDebugLoc(section []byte, offset uint64, addressSize int) ([]Location, error) {
	if addressSize != 4 && addressSize != 8 {
		return nil, fmt.Errorf("unsupported DWARF address size %d", addressSize)
	}
	if offset > uint64(len(section)) {
		return nil, errors.New("location-list offset is outside .debug_loc")
	}
	position := int(offset)
	var result []Location
	var base uint64
	maximum := ^uint64(0)
	if addressSize == 4 {
		maximum = uint64(^uint32(0))
	}
	readAddress := func() (uint64, error) {
		if position+addressSize > len(section) {
			return 0, io.ErrUnexpectedEOF
		}
		var value uint64
		for index := 0; index < addressSize; index++ {
			value |= uint64(section[position+index]) << (8 * index)
		}
		position += addressSize
		return value, nil
	}
	for {
		low, err := readAddress()
		if err != nil {
			return nil, err
		}
		high, err := readAddress()
		if err != nil {
			return nil, err
		}
		if low == 0 && high == 0 {
			return result, nil
		}
		if low == maximum {
			base = high
			continue
		}
		if position+2 > len(section) {
			return nil, io.ErrUnexpectedEOF
		}
		size := int(section[position]) | int(section[position+1])<<8
		position += 2
		if position+size > len(section) {
			return nil, io.ErrUnexpectedEOF
		}
		expression := section[position : position+size]
		position += size
		if high > low {
			result = append(result, Location{
				Start: base + low, End: base + high,
				Expression: hex.EncodeToString(expression),
			})
		}
	}
}

func (b *indexBuilder) addType(value dwarf.Type) string {
	if value == nil {
		return ""
	}
	key := typeKey(value)
	if id, ok := b.typeByKey[key]; ok {
		return id
	}
	id := fmt.Sprintf("t%d", len(b.index.Types)+1)
	b.typeByKey[key] = id
	info := Type{ID: id, Name: value.String(), Size: value.Size(), Complete: true}
	if name := value.Common().Name; name != "" {
		info.Name = name
	}
	b.index.Types = append(b.index.Types, info)
	index := len(b.index.Types) - 1
	switch current := value.(type) {
	case *dwarf.BoolType:
		info.Kind = "bool"
	case *dwarf.IntType, *dwarf.CharType:
		info.Kind, info.Signed = "integer", true
	case *dwarf.UintType, *dwarf.UcharType, *dwarf.AddrType:
		info.Kind = "integer"
	case *dwarf.FloatType:
		info.Kind = "float"
	case *dwarf.ComplexType:
		info.Kind = "complex"
	case *dwarf.PtrType:
		info.Kind = "pointer"
		info.Elem = b.addType(current.Type)
	case *dwarf.ArrayType:
		info.Kind = "array"
		info.Elem = b.addType(current.Type)
		info.Count = current.Count
	case *dwarf.StructType:
		info.Kind = current.Kind
		info.Complete = !current.Incomplete
		for _, field := range current.Field {
			info.Fields = append(info.Fields, TypeField{
				Name: field.Name, Type: b.addType(field.Type), Offset: field.ByteOffset,
			})
		}
	case *dwarf.TypedefType:
		info.Kind = "typedef"
		info.Elem = b.addType(current.Type)
	case *dwarf.QualType:
		info.Kind = "qualified"
		info.Elem = b.addType(current.Type)
	case *dwarf.EnumType:
		info.Kind = "enum"
		for _, item := range current.Val {
			info.Enum = append(info.Enum, EnumValue{Name: item.Name, Value: item.Val})
		}
	case *dwarf.FuncType:
		info.Kind = "function"
		info.Elem = b.addType(current.ReturnType)
	default:
		info.Kind = "unknown"
	}
	b.index.Types[index] = info
	return id
}

func typeKey(value dwarf.Type) uintptr {
	reflected := reflect.ValueOf(value)
	if reflected.Kind() == reflect.Pointer && !reflected.IsNil() {
		return reflected.Pointer()
	}
	// debug/dwarf currently returns pointer-backed concrete types. Keep a
	// deterministic non-zero fallback if that ever changes.
	return uintptr(len(value.String()) + 1)
}

func (b *indexBuilder) sort() {
	sort.SliceStable(b.index.Sources, func(i, j int) bool {
		return b.index.Sources[i].Path < b.index.Sources[j].Path
	})
	sort.SliceStable(b.index.Lines, func(i, j int) bool {
		left, right := b.index.Lines[i], b.index.Lines[j]
		if left.Start != right.Start {
			return left.Start < right.Start
		}
		if left.Source != right.Source {
			return left.Source < right.Source
		}
		return left.Line < right.Line
	})
	sort.SliceStable(b.index.Functions, func(i, j int) bool {
		left, right := b.index.Functions[i], b.index.Functions[j]
		if len(left.Ranges) != 0 && len(right.Ranges) != 0 && left.Ranges[0].Start != right.Ranges[0].Start {
			return left.Ranges[0].Start < right.Ranges[0].Start
		}
		return left.Name < right.Name
	})
}
