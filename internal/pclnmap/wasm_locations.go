package pclnmap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
)

// WasmLocations resolves logical location IDs from a validated sidecar.
// The encoded map must remain immutable while the reader is in use.
type WasmLocations struct {
	raw  []byte
	view View
}

// WasmLocation is a source position recorded in the Wasm shadow stack.
type WasmLocation struct {
	Function  string
	File      string
	Line      uint32
	StartLine uint32
}

func ParseWasmLocations(raw []byte) (WasmLocations, error) {
	view, err := Parse(raw)
	if err != nil {
		return WasmLocations{}, err
	}
	if view.GOARCHCode != GOARCHWasm || (view.GOOSCode != GOOSWASI && view.GOOSCode != GOOSJS) {
		return WasmLocations{}, errors.New("pclntab map is not for hosted WebAssembly")
	}
	return WasmLocations{raw: raw, view: view}, nil
}

func (m WasmLocations) Identity() [32]byte {
	return m.view.Identity
}

func (m WasmLocations) Resolve(id uint64) (WasmLocation, bool) {
	lines := m.view.Sections[descPCLines]
	if id == 0 || lines.Count == 0 {
		return WasmLocation{}, false
	}
	lineAt := func(index int) []byte {
		off := lines.Offset + uint64(index)*pcLineSize
		return m.raw[off : off+pcLineSize]
	}
	index := sort.Search(int(lines.Count), func(i int) bool {
		return binary.LittleEndian.Uint64(lineAt(i)) >= id
	})
	if uint64(index) == lines.Count {
		return WasmLocation{}, false
	}
	rec := lineAt(index)
	funcIndex := uint64(binary.LittleEndian.Uint32(rec[8:]))
	funcs := m.view.Sections[descRecords]
	if binary.LittleEndian.Uint64(rec) != id || funcIndex == 0 || funcIndex > funcs.Count {
		return WasmLocation{}, false
	}
	funcOff := funcs.Offset + (funcIndex-1)*recordSize
	fn := m.raw[funcOff : funcOff+recordSize]
	name := m.join(fn[8:], ".", fn[12:])
	if name == "" {
		return WasmLocation{}, false
	}
	return WasmLocation{
		Function:  name,
		File:      m.join(rec[12:], "", rec[16:]),
		Line:      binary.LittleEndian.Uint32(rec[20:]),
		StartLine: binary.LittleEndian.Uint32(fn[24:]) & funcinfo.EncodedLineMask,
	}, true
}

func (m WasmLocations) stringAt(id uint32) string {
	index := m.view.Sections[descStringOffsets]
	if uint64(id) >= index.Count {
		return ""
	}
	off := uint64(binary.LittleEndian.Uint32(m.raw[index.Offset+uint64(id)*stringOffsetSize:]))
	pool := m.view.Sections[descStrings]
	if off >= pool.Count {
		return ""
	}
	s := m.raw[pool.Offset+off : pool.Offset+pool.Count]
	if end := bytes.IndexByte(s, 0); end >= 0 {
		return string(s[:end])
	}
	return ""
}

func (m WasmLocations) join(a []byte, sep string, b []byte) string {
	left := m.stringAt(binary.LittleEndian.Uint32(a))
	right := m.stringAt(binary.LittleEndian.Uint32(b))
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + sep + right
}
