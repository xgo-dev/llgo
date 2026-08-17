package build

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

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

type moduleSize struct {
	Name   string
	Code   uint64
	ROData uint64
	Data   uint64
	BSS    uint64
}

func (m *moduleSize) Flash() uint64 {
	return m.Code + m.ROData + m.Data
}

func (m *moduleSize) RAM() uint64 {
	return m.Data + m.BSS
}

type sizeReport struct {
	Binary  string
	Modules map[string]*moduleSize
	Total   moduleSize
}

func (r *sizeReport) module(name string) *moduleSize {
	if name == "" {
		name = "(anonymous)"
	}
	if r.Modules == nil {
		r.Modules = make(map[string]*moduleSize)
	}
	m, ok := r.Modules[name]
	if !ok {
		m = &moduleSize{Name: name}
		r.Modules[name] = m
	}
	return m
}

func (r *sizeReport) add(name string, kind sectionKind, size uint64) {
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

func reportBinarySize(path, format, level string, pkgs []Package) error {
	report, err := collectBinarySize(path, pkgs, level)
	if err != nil {
		return err
	}
	switch format {
	case "", "text":
		printTextReport(os.Stdout, report)
	case "json":
		return emitJSONReport(os.Stdout, report)
	default:
		return fmt.Errorf("unknown size format %q (valid: text,json)", format)
	}
	return nil
}

func collectBinarySize(path string, pkgs []Package, level string) (*sizeReport, error) {
	cmd, err := llvm.New("").Readelf("--elf-output-style=LLVM", "--all", path)
	if err != nil {
		return nil, fmt.Errorf("llvm-readelf: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("llvm-readelf stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to execute llvm-readelf: %w", err)
	}
	parsed, parseErr := parseReadelfOutput(stdout)
	closeErr := stdout.Close()
	waitErr := cmd.Wait()
	if waitErr != nil {
		return nil, fmt.Errorf("llvm-readelf failed: %w\n%s", waitErr, stderr.String())
	}
	if parseErr != nil {
		return nil, fmt.Errorf("parsing llvm-readelf output failed: %w", parseErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing llvm-readelf stdout pipe failed: %w", closeErr)
	}
	report := buildSizeReport(path, parsed, pkgs, level)
	if report == nil || len(report.Modules) == 0 {
		return nil, fmt.Errorf("size report: no allocatable sections found in %s", path)
	}
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

func buildSizeReport(path string, data *readelfData, pkgs []Package, level string) *sizeReport {
	report := &sizeReport{Binary: path, Modules: make(map[string]*moduleSize)}
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

func emitJSONReport(w io.Writer, report *sizeReport) error {
	type moduleJSON struct {
		Name   string `json:"name"`
		Code   uint64 `json:"code"`
		ROData uint64 `json:"rodata"`
		Data   uint64 `json:"data"`
		BSS    uint64 `json:"bss"`
		Flash  uint64 `json:"flash"`
		RAM    uint64 `json:"ram"`
	}
	mods := report.sortedModules()
	jsonMods := make([]moduleJSON, 0, len(mods))
	for _, m := range mods {
		jsonMods = append(jsonMods, moduleJSON{
			Name:   m.Name,
			Code:   m.Code,
			ROData: m.ROData,
			Data:   m.Data,
			BSS:    m.BSS,
			Flash:  m.Flash(),
			RAM:    m.RAM(),
		})
	}
	payload := struct {
		Binary  string       `json:"binary"`
		Modules []moduleJSON `json:"modules"`
		Total   moduleJSON   `json:"total"`
	}{
		Binary:  filepath.Clean(report.Binary),
		Modules: jsonMods,
		Total: moduleJSON{
			Name:   "total",
			Code:   report.Total.Code,
			ROData: report.Total.ROData,
			Data:   report.Total.Data,
			BSS:    report.Total.BSS,
			Flash:  report.Total.Flash(),
			RAM:    report.Total.RAM(),
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func printTextReport(w io.Writer, report *sizeReport) {
	fmt.Fprintf(w, "\nSize report for %s\n", filepath.Clean(report.Binary))
	fmt.Fprintln(w, "   code  rodata    data     bss |   flash     ram | module")
	fmt.Fprintln(w, "------------------------------- | --------------- | ----------------")
	for _, m := range report.sortedModules() {
		fmt.Fprintf(w, "%7d %7d %7d %7d | %7d %7d | %s\n", m.Code, m.ROData, m.Data, m.BSS, m.Flash(), m.RAM(), m.Name)
	}
	fmt.Fprintln(w, "------------------------------- | --------------- | ----------------")
	fmt.Fprintf(w, "%7d %7d %7d %7d | %7d %7d | total\n", report.Total.Code, report.Total.ROData, report.Total.Data, report.Total.BSS, report.Total.Flash(), report.Total.RAM())
}

func (r *sizeReport) sortedModules() []*moduleSize {
	mods := make([]*moduleSize, 0, len(r.Modules))
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
	if idx := strings.Index(val, "("); idx >= 0 {
		val = strings.TrimSpace(val[:idx])
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

func ensureSizeReporting(conf *Config) error {
	if !conf.SizeReport {
		return nil
	}
	switch strings.ToLower(conf.SizeLevel) {
	case "", "module":
		conf.SizeLevel = "module"
	case "package", "full":
		conf.SizeLevel = strings.ToLower(conf.SizeLevel)
	default:
		return fmt.Errorf("invalid size level %q (valid: full,module,package)", conf.SizeLevel)
	}
	cmd, err := llvm.New("").Readelf("--version")
	if err != nil {
		return fmt.Errorf("llvm-readelf not available: %w", err)
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("llvm-readelf not available: %w", err)
	}
	return nil
}
