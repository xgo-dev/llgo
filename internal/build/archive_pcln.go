// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xgo-dev/llvm"
)

// This member is a native object even in LTO archives. It has no externally
// visible symbols or relocations to code: source mappings must not keep dead
// functions alive. LLGo reads it before linking; the system linker need not
// extract it. A raw JSON member is not accepted by every native linker.
const packagePCLNMember = "__llgo_pcln.o"

type packagePCLN struct {
	functions []funcInfoRecord
	lines     []pcLineRecord
}

type objectFuncInfo struct {
	Symbol, Name, File  string
	Line, Column, Flags uint32
}

type objectPCLine struct {
	ID           uint64
	Symbol, File string
	Line, Column uint32
}

// This is a compiler-private package artifact. Compiler identity already
// participates in archive fingerprints, so it needs no cache-version protocol.
type objectPCLN struct {
	Functions []objectFuncInfo
	Lines     []objectPCLine
}

func emitPackagePCLN(ctx *context, pkg Package) error {
	if ctx.buildConf.PCLNMode == PCLNNone {
		return nil
	}
	mod := pkg.LPkg.Module()
	data := objectPCLN{}
	for _, r := range readFuncInfo(mod) {
		data.Functions = append(data.Functions, objectFuncInfo{r.symbol, r.name, r.file, r.line, r.column, r.flags})
	}
	for _, r := range readPCLineInfo(mod) {
		data.Lines = append(data.Lines, objectPCLine{r.id, r.symbol, r.file, r.line, r.column})
	}
	// This fixed schema contains only slices, strings and integers, so JSON
	// encoding cannot fail (there are no cycles, floats or custom marshalers).
	raw, _ := json.Marshal(data)
	// Object sections may contain trailing alignment padding.
	payload := binary.LittleEndian.AppendUint64(nil, uint64(len(raw)))
	payload = append(payload, raw...)
	carrier := mod.Context().NewModule("llgo.pcln")
	defer carrier.Dispose()
	carrier.SetTarget(mod.Target())
	carrier.SetDataLayout(mod.DataLayout())
	triple := ctx.crossCompile.LLVMTarget
	if triple == "" {
		triple = mod.Target()
	}
	if strings.HasPrefix(triple, "wasm") {
		carrier.AddNamedMetadataOperand("wasm.custom_sections", mod.Context().MDNode([]llvm.Metadata{
			mod.Context().MDString("llgo.pcln"), mod.Context().MDString(string(payload)),
		}))
	} else {
		section := ".llgo_pcln"
		switch {
		case strings.Contains(triple, "-apple-"):
			section = "__DATA,__llgo_pcln"
		case strings.Contains(triple, "windows") || strings.Contains(triple, "mingw"):
			section = ".llgopcl"
		}
		init := mod.Context().ConstString(string(payload), false)
		value := llvm.AddGlobal(carrier, init.Type(), "__llgo_pcln_payload")
		value.SetInitializer(init)
		value.SetGlobalConstant(true)
		value.SetLinkage(llvm.PrivateLinkage)
		value.SetSection(section)
		value.SetAlignment(1)
		// Retain bytes through compiler optimization without imposing linker
		// liveness if a consumer force-loads every member of a C archive.
		usedInit := llvm.ConstArray(value.Type(), []llvm.Value{value})
		used := llvm.AddGlobal(carrier, usedInit.Type(), "llvm.compiler.used")
		used.SetInitializer(usedInit)
		used.SetLinkage(llvm.AppendingLinkage)
		used.SetSection("llvm.metadata")
	}
	buf, err := emitPCLNObject(ctx, pkg, carrier)
	if err != nil {
		return fmt.Errorf("emit PCLN object for %s: %w", pkg.PkgPath, err)
	}
	pkg.ObjBuffers = append(pkg.ObjBuffers, packageArchiveBuffer{packagePCLNMember, buf})
	return nil
}

func emitPCLNObject(ctx *context, pkg Package, carrier llvm.Module) (llvm.MemoryBuffer, error) {
	if useInMemoryNativeCodegen(ctx) {
		return pkg.LPkg.Prog.TargetMachine().EmitToMemoryBuffer(carrier, llvm.ObjectFile)
	}
	// Preserve the selected cross compiler and its object ABI flags, including
	// embedded targets whose physical backend is absent from the host LLVM.
	// Force native output after the ordinary flags, even for LTO builds.
	f, err := os.CreateTemp("", "llgo-pcln-*.ll")
	if err != nil {
		return llvm.MemoryBuffer{}, err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(carrier.String()); err != nil {
		f.Close()
		return llvm.MemoryBuffer{}, err
	}
	if err := f.Close(); err != nil {
		return llvm.MemoryBuffer{}, err
	}
	obj := f.Name() + ".o"
	defer os.Remove(obj)
	if err := ctx.irCompiler().Compile("-c", f.Name(), "-o", obj, "-fno-lto", "-Wno-override-module"); err != nil {
		return llvm.MemoryBuffer{}, err
	}
	return llvm.NewMemoryBufferFromFile(obj)
}

// loadPackagePCLN is identical for fresh and reused archives. Neither frontend
// reconstruction nor frontend-export snapshots can describe backend-generated
// physical functions reliably.
func loadPackagePCLN(pkg Package) error {
	pkg.pcln = nil
	if pkg.ArchiveFile == "" {
		return nil
	}
	raw, err := readPCLNArchiveMember(pkg.ArchiveFile)
	if err != nil {
		return fmt.Errorf("read PCLN for %s: %w", pkg.PkgPath, err)
	}
	section, err := readPCLNObject(raw)
	if err != nil {
		return fmt.Errorf("read PCLN object for %s: %w", pkg.PkgPath, err)
	}
	if len(section) < 8 || binary.LittleEndian.Uint64(section) > uint64(len(section)-8) {
		return fmt.Errorf("invalid PCLN payload length for %s", pkg.PkgPath)
	}
	var data objectPCLN
	if err := json.Unmarshal(section[8:8+binary.LittleEndian.Uint64(section)], &data); err != nil {
		return fmt.Errorf("decode PCLN for %s: %w", pkg.PkgPath, err)
	}
	result := &packagePCLN{}
	for _, r := range data.Functions {
		result.functions = append(result.functions, funcInfoRecord{r.Symbol, r.Name, r.File, r.Line, r.Column, r.Flags})
	}
	for _, r := range data.Lines {
		result.lines = append(result.lines, pcLineRecord{r.ID, r.Symbol, r.File, r.Line, r.Column})
	}
	pkg.pcln = result
	return nil
}

// The compiler always emits a regular archive and a short member name. Other
// members (including GNU name/symbol tables) can be skipped without loading
// machine code. BSD archives put even short filenames in the member payload.
func readPCLNArchiveMember(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var magic [8]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil || string(magic[:]) != "!<arch>\n" {
		return nil, fmt.Errorf("invalid package archive %s", path)
	}
	var result []byte
	for off := int64(8); off < stat.Size(); {
		var header [60]byte
		if _, err := f.ReadAt(header[:], off); err != nil {
			return nil, err
		}
		size, err := strconv.ParseInt(strings.TrimSpace(string(header[48:58])), 10, 64)
		start := off + 60
		if err != nil || size < 0 || size > stat.Size()-start || string(header[58:]) != "`\n" {
			return nil, fmt.Errorf("invalid archive member at %d", off)
		}
		off = start + size + size%2
		name := strings.TrimSuffix(strings.TrimSpace(string(header[:16])), "/")
		if strings.HasPrefix(name, "#1/") {
			n, err := strconv.ParseInt(name[3:], 10, 64)
			if err != nil || n < 0 || n > size || n > 4096 {
				return nil, fmt.Errorf("invalid BSD archive member name")
			}
			buf := make([]byte, n)
			if _, err := f.ReadAt(buf, start); err != nil {
				return nil, err
			}
			name = strings.TrimRight(string(buf), "\x00")
			start, size = start+n, size-n
		}
		if name != packagePCLNMember {
			continue
		}
		if result != nil {
			return nil, fmt.Errorf("duplicate %s", packagePCLNMember)
		}
		result, err = io.ReadAll(io.NewSectionReader(f, start, size))
		if err != nil {
			return nil, err
		}
	}
	if result == nil {
		return nil, fmt.Errorf("missing %s in %s", packagePCLNMember, path)
	}
	return result, nil
}

func readPCLNObject(raw []byte) ([]byte, error) {
	reader := bytes.NewReader(raw)
	switch {
	case bytes.HasPrefix(raw, []byte("\x7fELF")):
		f, err := elf.NewFile(reader)
		if err != nil {
			return nil, err
		}
		if s := f.Section(".llgo_pcln"); s != nil {
			return s.Data()
		}
	case bytes.HasPrefix(raw, []byte("\x00asm")):
		return readWasmPCLN(raw)
	default:
		if f, err := macho.NewFile(reader); err == nil {
			if s := f.Section("__llgo_pcln"); s != nil {
				return s.Data()
			}
		} else if f, err := pe.NewFile(reader); err == nil {
			if s := f.Section(".llgopcl"); s != nil {
				return s.Data()
			}
		} else {
			return nil, fmt.Errorf("invalid PCLN object: %w", err)
		}
	}
	return nil, fmt.Errorf("missing PCLN section")
}

func readWasmPCLN(raw []byte) ([]byte, error) {
	if len(raw) < 8 || !bytes.Equal(raw[4:8], []byte{1, 0, 0, 0}) {
		return nil, fmt.Errorf("invalid Wasm object header")
	}
	for raw = raw[8:]; len(raw) != 0; {
		id := raw[0]
		size, n := binary.Uvarint(raw[1:])
		if n <= 0 || n > 5 || size > uint64(len(raw)-1-n) {
			return nil, fmt.Errorf("invalid Wasm section length")
		}
		section := raw[1+n : 1+n+int(size)]
		raw = raw[1+n+int(size):]
		if id != 0 {
			continue
		}
		size, n = binary.Uvarint(section)
		if n <= 0 || n > 5 || size > uint64(len(section)-n) {
			return nil, fmt.Errorf("invalid Wasm custom section name")
		}
		if string(section[n:n+int(size)]) == "llgo.pcln" {
			return section[n+int(size):], nil
		}
	}
	return nil, fmt.Errorf("missing Wasm PCLN section")
}

func packageFuncInfo(pkg Package) []funcInfoRecord {
	if pkg.pcln != nil {
		return pkg.pcln.functions
	}
	// ModeGen and callers that consume an in-memory module have no archive.
	if pkg.ArchiveFile == "" && pkg.LPkg != nil {
		return readFuncInfo(pkg.LPkg.Module())
	}
	return nil
}

func packagePCLineInfo(pkg Package) []pcLineRecord {
	if pkg.pcln != nil {
		return pkg.pcln.lines
	}
	if pkg.ArchiveFile == "" && pkg.LPkg != nil {
		return readPCLineInfo(pkg.LPkg.Module())
	}
	return nil
}
