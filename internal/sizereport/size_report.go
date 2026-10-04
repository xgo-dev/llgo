// Package sizereport measures final binary artifacts independently of the compiler.
package sizereport

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Package describes the names needed for symbol aggregation without loading
// compiler packages or LLVM modules.
type Package struct {
	Path   string
	Module string
}

// Artifact is the final build-output metadata included in a report.
type Artifact struct {
	Path   string
	Format string
	Role   string
	Size   int64
}

// ErrUnsupportedFormat lets the build layer supply its native tool fallback.
var ErrUnsupportedFormat = errors.New("unsupported size report format")

type sectionKind int

const (
	sectionUnknown sectionKind = iota
	sectionText
	sectionROData
	sectionData
	sectionBSS
)

const (
	// readelfInitialBuffer is the initial buffer size for reading readelf output.
	// Most lines in readelf output are less than 1KB.
	readelfInitialBuffer = 64 * 1024

	// readelfMaxBuffer is the maximum buffer size to handle very long symbol names
	// or section dumps.
	readelfMaxBuffer = 4 * 1024 * 1024
)

// ELF special section indices (from ELF specification)
const (
	SHN_UNDEF     = 0x0000 // Undefined section
	SHN_LORESERVE = 0xFF00 // Start of reserved indices
	SHN_ABS       = 0xFFF1 // Absolute values
	SHN_COMMON    = 0xFFF2 // Common symbols
	SHN_XINDEX    = 0xFFFF // Escape value for extended section indices
)

type sectionInfo struct {
	Index   int
	Name    string
	Segment string
	Address uint64
	Size    uint64
	Kind    sectionKind
}

type symbolInfo struct {
	Name         string
	SectionIndex int
	Address      uint64
}

type readelfData struct {
	sections map[int]*sectionInfo
	symbols  map[int][]symbolInfo
}

type Module struct {
	Name   string
	Code   uint64
	ROData uint64
	Data   uint64
	BSS    uint64
}

func (m *Module) Flash() uint64 {
	return m.Code + m.ROData + m.Data
}

func (m *Module) RAM() uint64 {
	return m.Data + m.BSS
}

type Report struct {
	Binary    string
	Modules   map[string]*Module
	Total     Module
	Format    string
	FileSize  uint64
	Wasm      *wasmSizeDetails
	Artifacts []Artifact
	Warnings  []string
}

func (r *Report) module(name string) *Module {
	if name == "" {
		name = "(anonymous)"
	}
	if r.Modules == nil {
		r.Modules = make(map[string]*Module)
	}
	m, ok := r.Modules[name]
	if !ok {
		m = &Module{Name: name}
		r.Modules[name] = m
	}
	return m
}

func (r *Report) add(name string, kind sectionKind, size uint64) {
	if size == 0 {
		return
	}
	m := r.module(name)
	switch kind {
	case sectionText:
		m.Code += size
		r.Total.Code += size
	case sectionROData:
		m.ROData += size
		r.Total.ROData += size
	case sectionData:
		m.Data += size
		r.Total.Data += size
	case sectionBSS:
		m.BSS += size
		r.Total.BSS += size
	}
}

// Write emits a text or JSON report and propagates output failures.
func Write(w io.Writer, report *Report, format string) error {
	switch format {
	case "", "text":
		var buf bytes.Buffer
		printTextReport(&buf, report)
		_, err := w.Write(buf.Bytes())
		return err
	case "json":
		return emitJSONReport(w, report)
	default:
		return fmt.Errorf("unknown size format %q (valid: text,json)", format)
	}
}

// Collect measures a final Wasm, ELF or PE artifact.
func Collect(path string, pkgs []Package, level string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("size report input %q is not a regular file", path)
	}
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, fmt.Errorf("read binary header: %w", err)
	}
	var report *Report
	format := "native"
	switch string(magic[:]) {
	case "\x00asm":
		format = "wasm"
		if _, err = f.Seek(0, io.SeekStart); err == nil {
			var raw []byte
			raw, err = readWasmSizeBytes(f, info.Size())
			if err == nil {
				report, err = collectWasmSize(path, raw, pkgs, level)
			}
		}
	case "\x7fELF":
		format = "elf"
		report, err = collectELFSize(path, pkgs, level)
	default:
		if string(magic[:2]) != "MZ" {
			return nil, ErrUnsupportedFormat
		}
		format = "pe"
		report, err = collectPESize(path, pkgs, level)
	}
	if err != nil {
		return nil, err
	}
	report.Format = format
	report.FileSize = uint64(info.Size())
	return report, nil
}

// Read only the stat-sized artifact and allocate its backing buffer once.
// Growing ReadAll buffers can otherwise double peak memory for large modules.
func readWasmSizeBytes(r io.Reader, size int64) ([]byte, error) {
	if size < 0 || uint64(size) > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("WebAssembly file size %d cannot fit in memory on this host", size)
	}
	raw := make([]byte, int(size))
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, fmt.Errorf("reading WebAssembly artifact: %w", err)
	}
	// A changing file must not produce a report for a silently truncated prefix.
	var extra [1]byte
	if _, err := io.ReadFull(r, extra[:]); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("checking WebAssembly artifact size: %w", err)
		}
		return nil, fmt.Errorf("WebAssembly artifact grew while reading its size")
	}
	return raw, nil
}

// ReadReadelf measures a native artifact from the build layer's llvm-readelf
// output. Tool discovery and invocation stay outside this package.
func ReadReadelf(path string, output io.Reader, pkgs []Package, level string) (*Report, error) {
	parsed, err := parseReadelfOutput(output)
	if err != nil {
		return nil, err
	}
	report := buildSizeReport(path, parsed, pkgs, level)
	if report == nil || len(report.Modules) == 0 {
		return nil, fmt.Errorf("size report: no allocatable sections found in %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	report.Format, report.FileSize = "native", uint64(info.Size())
	return report, nil
}

func parseReadelfOutput(r io.Reader) (*readelfData, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, readelfInitialBuffer), readelfMaxBuffer)

	type ctxKind int
	const (
		ctxRoot ctxKind = iota
		ctxSections
		ctxSection
		ctxSymbols
		ctxSymbol
	)

	type ctx struct {
		kind   ctxKind
		indent int
	}

	stack := []ctx{{kind: ctxRoot, indent: -1}}
	push := func(kind ctxKind, indent int) {
		stack = append(stack, ctx{kind: kind, indent: indent})
	}
	pop := func(expected ctxKind, indent int) bool {
		top := stack[len(stack)-1]
		if top.kind != expected || top.indent != indent {
			return false
		}
		stack = stack[:len(stack)-1]
		return true
	}
	current := func() ctx {
		return stack[len(stack)-1]
	}

	data := &readelfData{
		sections: make(map[int]*sectionInfo),
		symbols:  make(map[int][]symbolInfo),
	}

	// readelf outputs section references differently:
	//   - Mach-O: section numbers are 1-based in symbol references
	//   - ELF: section numbers in symbol references match the Index directly
	secIndexBase := 1 // default to Mach-O behavior; switch to 0 for ELF once detected

	var currentSection *sectionInfo
	var currentSymbol *symbolInfo

	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		// Detect object format early to adjust section index base
		if strings.HasPrefix(trimmed, "Format:") {
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "mach-o") {
				secIndexBase = 1
			} else if strings.Contains(lower, "elf") {
				secIndexBase = 0
			}
		}
		indent := countLeadingSpaces(raw)
		top := current()

		switch {
		case strings.HasPrefix(trimmed, "Sections [") && top.kind == ctxRoot:
			push(ctxSections, indent)
			continue
		case strings.HasPrefix(trimmed, "Symbols [") && top.kind == ctxRoot:
			push(ctxSymbols, indent)
			continue
		case trimmed == "Section {" && top.kind == ctxSections && indent == top.indent+2:
			currentSection = &sectionInfo{Index: -1}
			push(ctxSection, indent)
			continue
		case trimmed == "Symbol {" && top.kind == ctxSymbols && indent == top.indent+2:
			currentSymbol = &symbolInfo{SectionIndex: -1}
			push(ctxSymbol, indent)
			continue
		case trimmed == "}" && pop(ctxSection, indent):
			if currentSection != nil && currentSection.Index >= 0 {
				currentSection.Kind = classifySection(currentSection.Name, currentSection.Segment)
				data.sections[currentSection.Index] = currentSection
			}
			currentSection = nil
			continue
		case trimmed == "}" && pop(ctxSymbol, indent):
			if currentSymbol != nil && currentSymbol.SectionIndex >= 0 {
				data.symbols[currentSymbol.SectionIndex] = append(data.symbols[currentSymbol.SectionIndex], *currentSymbol)
			}
			currentSymbol = nil
			continue
		case trimmed == "]" && (top.kind == ctxSections || top.kind == ctxSymbols) && indent == top.indent:
			stack = stack[:len(stack)-1]
			continue
		}

		switch top.kind {
		case ctxSection:
			if currentSection == nil {
				continue
			}
			switch {
			case strings.HasPrefix(trimmed, "Index: "):
				if idx, err := strconv.Atoi(strings.TrimSpace(trimmed[len("Index: "):])); err == nil {
					currentSection.Index = idx
				}
			case strings.HasPrefix(trimmed, "Name: "):
				currentSection.Name = parseNameField(trimmed[len("Name: "):])
			case strings.HasPrefix(trimmed, "Segment: "):
				currentSection.Segment = parseNameField(trimmed[len("Segment: "):])
			case strings.HasPrefix(trimmed, "Address: "):
				if val, err := parseUintField(trimmed[len("Address: "):]); err == nil {
					currentSection.Address = val
				}
			case strings.HasPrefix(trimmed, "Size: "):
				if val, err := parseUintField(trimmed[len("Size: "):]); err == nil {
					currentSection.Size = val
				}
			}
		case ctxSymbol:
			if currentSymbol == nil {
				continue
			}
			switch {
			case strings.HasPrefix(trimmed, "Name: "):
				currentSymbol.Name = parseNameField(trimmed[len("Name: "):])
			case strings.HasPrefix(trimmed, "Section: "):
				name, idx := parseSectionRef(trimmed[len("Section: "):], secIndexBase)
				currentSymbol.SectionIndex = idx
				if currentSymbol.Name == "" {
					currentSymbol.Name = name
				}
			case strings.HasPrefix(trimmed, "Value: "):
				if val, err := parseUintField(trimmed[len("Value: "):]); err == nil {
					currentSymbol.Address = val
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

func countLeadingSpaces(line string) int {
	count := 0
	for _, ch := range line {
		if ch != ' ' {
			break
		}
		count++
	}
	return count
}

func classifySection(name, segment string) sectionKind {
	ln := strings.ToLower(name)
	ls := strings.ToLower(segment)
	switch {
	case strings.Contains(ln, "text"), strings.Contains(ln, "code"), strings.Contains(ln, "plt"):
		return sectionText
	case strings.Contains(ln, "rodata"), strings.Contains(ln, "const"), strings.Contains(ln, "literal"), strings.Contains(ln, "cstring"):
		return sectionROData
	case strings.Contains(ln, "bss"), strings.Contains(ln, "tbss"), strings.Contains(ln, "sbss"), strings.Contains(ln, "common"), strings.Contains(ln, "zerofill"):
		return sectionBSS
	case strings.Contains(ln, "data"), strings.Contains(ln, "got"), strings.Contains(ln, "init_array"), strings.Contains(ln, "cfstring"), strings.Contains(ln, "tdata"):
		return sectionData
	}
	switch {
	case strings.Contains(ls, "__text"):
		return sectionText
	case strings.Contains(ls, "data_const"):
		return sectionROData
	case strings.Contains(ls, "__data"):
		return sectionData
	}
	return sectionUnknown
}

func buildSizeReport(path string, data *readelfData, pkgs []Package, level string) *Report {
	report := &Report{Binary: path, Modules: make(map[string]*Module)}
	if data == nil {
		return report
	}
	res := newNameResolver(level, pkgs)
	var recognized bool
	for idx, sec := range data.sections {
		if sec == nil || sec.Size == 0 {
			continue
		}
		if sec.Kind == sectionUnknown {
			continue
		}
		recognized = true
		end := sec.Address + sec.Size
		syms := data.symbols[idx]
		if len(syms) == 0 {
			report.add("(unknown "+sec.Name+")", sec.Kind, sec.Size)
			continue
		}
		// Sort symbols by address to calculate sizes based on address ranges
		sort.Slice(syms, func(i, j int) bool {
			if syms[i].Address == syms[j].Address {
				return syms[i].Name < syms[j].Name
			}
			return syms[i].Address < syms[j].Address
		})
		cursor := sec.Address
		for i := 0; i < len(syms); {
			addr := syms[i].Address
			if addr >= end {
				break
			}
			if addr < sec.Address {
				addr = sec.Address
			}
			// Group aliases that share the same address and pick the most
			// informative symbol. LLVM's dedup linker inserts pseudo symbols
			// like "$x" (text) / "$d" (data) that represent the shared blob,
			// so prefer any real symbol name over those placeholders.
			j := i + 1
			for j < len(syms) && syms[j].Address == syms[i].Address {
				j++
			}
			primary := syms[i]
			for k := i; k < j; k++ {
				name := syms[k].Name
				if !strings.HasPrefix(name, "$x") && !strings.HasPrefix(name, "$d") {
					primary = syms[k]
					break
				}
			}
			// Add padding bytes between cursor and current symbol
			if addr > cursor {
				report.add("(padding "+sec.Name+")", sec.Kind, addr-cursor)
				cursor = addr
			}
			next := end
			if j < len(syms) && syms[j].Address > addr {
				next = syms[j].Address
			}
			if next > end {
				next = end
			}
			if next > addr {
				mod := res.resolve(primary.Name)
				report.add(mod, sec.Kind, next-addr)
				cursor = next
			}
			i = j
		}
		// Add any remaining padding at the end of the section
		if cursor < end {
			report.add("(padding "+sec.Name+")", sec.Kind, end-cursor)
		}
	}
	if !recognized {
		return nil
	}
	return report
}

func emitJSONReport(w io.Writer, report *Report) error {
	type moduleJSON struct {
		Name   string  `json:"name"`
		Code   uint64  `json:"code"`
		ROData uint64  `json:"rodata"`
		Data   uint64  `json:"data"`
		BSS    uint64  `json:"bss"`
		Flash  *uint64 `json:"flash,omitempty"`
		RAM    *uint64 `json:"ram,omitempty"`
	}
	moduleValue := func(m *Module) moduleJSON {
		value := moduleJSON{Name: m.Name, Code: m.Code, ROData: m.ROData, Data: m.Data, BSS: m.BSS}
		if report.Wasm == nil {
			flash, ram := m.Flash(), m.RAM()
			value.Flash, value.RAM = &flash, &ram
		}
		return value
	}
	mods := report.sortedModules()
	jsonMods := make([]moduleJSON, 0, len(mods))
	for _, m := range mods {
		jsonMods = append(jsonMods, moduleValue(m))
	}
	total := moduleValue(&report.Total)
	total.Name = "total"
	type artifactJSON struct {
		Path   string `json:"path"`
		Format string `json:"format"`
		Role   string `json:"role"`
		Size   int64  `json:"size"`
	}
	artifacts := make([]artifactJSON, 0, len(report.Artifacts))
	for _, a := range report.Artifacts {
		artifacts = append(artifacts, artifactJSON{Path: a.Path, Format: a.Format, Role: a.Role, Size: a.Size})
	}
	payload := struct {
		Version   int              `json:"version"`
		Stage     string           `json:"stage"`
		Format    string           `json:"format"`
		FileSize  uint64           `json:"file_size"`
		Binary    string           `json:"binary"`
		Modules   []moduleJSON     `json:"modules"`
		Total     moduleJSON       `json:"total"`
		Wasm      *wasmSizeDetails `json:"wasm,omitempty"`
		Artifacts []artifactJSON   `json:"artifacts"`
		Warnings  []string         `json:"warnings,omitempty"`
	}{
		Version:   1,
		Stage:     "final",
		Format:    report.Format,
		FileSize:  report.FileSize,
		Binary:    filepath.Clean(report.Binary),
		Modules:   jsonMods,
		Total:     total,
		Wasm:      report.Wasm,
		Artifacts: artifacts,
		Warnings:  report.Warnings,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func printTextReport(w io.Writer, report *Report) {
	fmt.Fprintf(w, "\nSize report for %s\n", filepath.Clean(report.Binary))
	fmt.Fprintf(w, "Final file: %d bytes (%s)\n", report.FileSize, report.Format)
	for _, a := range report.Artifacts {
		fmt.Fprintf(w, "Artifact: %d bytes, %s, %s, %s\n", a.Size, a.Role, a.Format, a.Path)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(w, "Warning: %s\n", warning)
	}
	if report.Wasm != nil {
		fmt.Fprintln(w, "   code    data | module (encoded function bodies / stored data bytes)")
		for _, m := range report.sortedModules() {
			fmt.Fprintf(w, "%7d %7d | %s\n", m.Code, m.Data, m.Name)
		}
		fmt.Fprintf(w, "%7d %7d | total\n", report.Total.Code, report.Total.Data)
		fmt.Fprintf(w, "Custom sections: %d bytes; other encoding: %d bytes\n", report.Wasm.CustomBytes, report.Wasm.StructureBytes)
		for i, memory := range report.Wasm.Memories {
			fmt.Fprintf(w, "Memory %d: initial=%d", i, memory.InitialBytes)
			if memory.MaximumBytes != nil {
				fmt.Fprintf(w, " maximum=%d", *memory.MaximumBytes)
			}
			fmt.Fprintf(w, " imported=%t shared=%t memory64=%t\n", memory.Imported, memory.Shared, memory.Memory64)
		}
		fmt.Fprintln(w, "Memory limits include reservations; static BSS and peak RAM are not inferred.")
		return
	}
	fmt.Fprintln(w, "   code  rodata    data     bss |   flash     ram | module")
	fmt.Fprintln(w, "------------------------------- | --------------- | ----------------")
	for _, m := range report.sortedModules() {
		fmt.Fprintf(w, "%7d %7d %7d %7d | %7d %7d | %s\n", m.Code, m.ROData, m.Data, m.BSS, m.Flash(), m.RAM(), m.Name)
	}
	fmt.Fprintln(w, "------------------------------- | --------------- | ----------------")
	fmt.Fprintf(w, "%7d %7d %7d %7d | %7d %7d | total\n", report.Total.Code, report.Total.ROData, report.Total.Data, report.Total.BSS, report.Total.Flash(), report.Total.RAM())
}

func (r *Report) sortedModules() []*Module {
	mods := make([]*Module, 0, len(r.Modules))
	for _, m := range r.Modules {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool {
		if mods[i].Flash() == mods[j].Flash() {
			return mods[i].Name < mods[j].Name
		}
		return mods[i].Flash() > mods[j].Flash()
	})
	return mods
}

// moduleNameFromSymbol extracts the Go package/module bucket from a symbol
// name following the rules described in the size report.
func moduleNameFromSymbol(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "(anonymous)"
	}
	// Strip C / assembler prefixes when deriving pkgPath buckets.
	trimmed := strings.TrimPrefix(name, "_")
	trimmed = strings.TrimPrefix(trimmed, ".")
	if idx := strings.Index(trimmed, " "); idx > 0 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, "@"); idx > 0 {
		trimmed = trimmed[:idx]
	}
	if trimmed == "" {
		return name
	}
	pkgPath := trimmed
	firstDot := strings.Index(pkgPath, ".")
	if firstDot < 0 {
		firstDot = len(pkgPath)
	}
	if closeIdx := strings.LastIndex(pkgPath[:firstDot], "]"); closeIdx >= 0 {
		pkgPath = pkgPath[closeIdx+1:]
	}
	if bracket := strings.Index(pkgPath, "["); bracket >= 0 {
		pkgPath = pkgPath[:bracket]
	}
	lastDot := strings.LastIndex(pkgPath, ".")
	if lastDot > 0 {
		pkgPath = pkgPath[:lastDot]
	}
	if paren := strings.Index(pkgPath, "("); paren > 0 {
		pkgPath = pkgPath[:paren]
	}
	pkgPath = strings.TrimSpace(pkgPath)
	pkgPath = strings.TrimLeft(pkgPath, "._")
	if pkgPath != "" {
		return pkgPath
	}
	return trimmed
}

func parseNameField(field string) string {
	val := strings.TrimSpace(field)
	// Only remove readelf's trailing string-table index. Parentheses in a
	// Go method name, e.g. pkg.(*T).Method, are part of the symbol.
	if idx := strings.LastIndex(val, " ("); idx >= 0 && strings.HasSuffix(val, ")") {
		index := strings.TrimPrefix(val[idx+2:len(val)-1], "0x")
		if _, err := strconv.ParseUint(index, 16, 64); err == nil {
			val = val[:idx]
		}
	}
	return val
}

func parseSectionRef(field string, indexBase int) (string, int) {
	name := parseNameField(field)
	idx := strings.Index(field, "(")
	if idx < 0 {
		return name, -1
	}
	end := strings.Index(field[idx:], ")")
	if end < 0 {
		return name, -1
	}
	val := strings.TrimSpace(field[idx+1 : idx+end])
	val = strings.TrimPrefix(val, "0x")
	if val == "" {
		return name, -1
	}
	num, err := strconv.ParseUint(val, 16, 64)
	if err != nil {
		return name, -1
	}
	if num == 0 {
		return name, -1
	}
	if indexBase == 0 && num >= SHN_LORESERVE {
		// Special ELF section indices (SHN_ABS, SHN_COMMON, etc.)
		return name, -1
	}
	if num > math.MaxInt {
		return name, -1
	}
	res := int(num) - indexBase
	if res < 0 {
		return name, -1
	}
	return name, res
}

func parseUintField(field string) (uint64, error) {
	val := strings.TrimSpace(field)
	if strings.HasPrefix(val, "0x") || strings.HasPrefix(val, "0X") {
		return strconv.ParseUint(val[2:], 16, 64)
	}
	return strconv.ParseUint(val, 10, 64)
}
