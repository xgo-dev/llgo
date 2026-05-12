package metadata

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
	"unsafe"
)

// ---- v2 constants ----

const (
	magicV2  = "LLP2"
	version2 = 2

	endianTag uint32 = 0x01020304
)

type sectionKind uint32

const (
	secStringOffsets      sectionKind = 1
	secStringData         sectionKind = 2
	secOrdinaryGroups     sectionKind = 10
	secOrdinaryValues     sectionKind = 11
	secTypeChildGroups    sectionKind = 20
	secTypeChildValues    sectionKind = 21
	secInterfaceGroups    sectionKind = 30
	secInterfaceValues    sectionKind = 31
	secUseIfaceGroups     sectionKind = 40
	secUseIfaceValues     sectionKind = 41
	secUseIfaceMGroups    sectionKind = 50
	secUseIfaceMValues    sectionKind = 51
	secMethodInfoGroups   sectionKind = 60
	secMethodInfoValues   sectionKind = 61
	secUseNamedMGroups    sectionKind = 70
	secUseNamedMValues    sectionKind = 71
	secReflectMValues     sectionKind = 80
)

// ---- v2 on-disk types ----

type rawHeader struct {
	Magic        [4]byte  // "LLP2"
	Version      uint16   // 2
	HeaderSize   uint16   // sizeof(rawHeader)
	EndianTag    uint32   // 0x01020304
	SectionCount uint32
	Flags        uint32
	FileSize     uint64
	DirOffset    uint64
	DirSize      uint64
	StringCount  uint32
	Reserved0    uint32
	Checksum     uint32
	Reserved1    [28]byte
}

type rawSectionDesc struct {
	Kind     uint32
	ElemSize uint32
	Offset   uint64
	Count    uint64
	ByteSize uint64
	Flags    uint32
	Reserved uint32
}

type rawGroup struct {
	Key   uint32
	Begin uint32
	Count uint32
}

// ---- v2 types: value structs (same layout on disk and in memory) ----

type rawMethodSig struct {
	Name  uint32
	MType uint32
}

type rawMethodSlot struct {
	Name  uint32
	MType uint32
	IFn   uint32
	TFn   uint32
}

type rawIfaceMethodDemand struct {
	Target uint32
	Name   uint32
	MType uint32
}

// ---- v2 writer ----

type v2Writer struct {
	w       io.WriteSeeker
	base    int64
	offset  int64
	align8  bool
	sections []rawSectionDesc
}

func (w *v2Writer) Write(p []byte) (int, error) {
	if w.align8 {
		pad := (8 - (w.offset % 8)) % 8
		if pad > 0 {
			zeros := make([]byte, pad)
			n, err := w.w.Write(zeros)
			w.offset += int64(n)
			if err != nil {
				return 0, fmt.Errorf("align pad: %w", err)
			}
			if n != int(pad) {
				return 0, fmt.Errorf("align pad short write")
			}
		}
	}
	n, err := w.w.Write(p)
	w.offset += int64(n)
	return n, err
}

func (w *v2Writer) Seek(offset int64, whence int) (int64, error) {
	pos, err := w.w.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	w.offset = pos
	return pos, nil
}

func (w *v2Writer) align() {
	pad := (8 - (w.offset % 8)) % 8
	if pad > 0 {
		w.w.Write(make([]byte, pad))
		w.offset += pad
	}
}

// elemSizeOnDisk returns the serialized size for a type.
// This differs from unsafe.Sizeof for Symbol (int=8B) which is packed as uint32 (4B).
func elemSizeOnDisk[T any]() uint32 {
	var zero T
	switch any(zero).(type) {
	case Symbol:
		return 4
	case uint32:
		return 4
	default:
		return uint32(unsafe.Sizeof(zero))
	}
}

// writeTypedSection writes a slice of fixed-width elements as a section.
func writeTypedSection[T any](w *v2Writer, kind sectionKind, vals []T) error {
	if len(vals) == 0 {
		return nil
	}
	w.align()
	start := w.offset
	elemSize := elemSizeOnDisk[T]()
	byteSize := uint64(len(vals)) * uint64(elemSize)

	buf := make([]byte, byteSize)
	// Pack each element in little-endian
	for i, v := range vals {
		packLE(buf[i*int(elemSize):], v)
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}

	w.sections = append(w.sections, rawSectionDesc{
		Kind:     uint32(kind),
		ElemSize: elemSize,
		Offset:   uint64(start),
		Count:    uint64(len(vals)),
		ByteSize: byteSize,
	})
	return nil
}

func writeBytesSection(w *v2Writer, kind sectionKind, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	w.align()
	start := w.offset
	if _, err := w.Write(data); err != nil {
		return err
	}
	w.sections = append(w.sections, rawSectionDesc{
		Kind:     uint32(kind),
		ElemSize: 1,
		Offset:   uint64(start),
		Count:    uint64(len(data)),
		ByteSize: uint64(len(data)),
	})
	return nil
}

// packLE packs a fixed-width struct into bytes in little-endian.
func packLE(buf []byte, v any) {
	switch v := v.(type) {
	case rawGroup:
		binary.LittleEndian.PutUint32(buf[0:4], v.Key)
		binary.LittleEndian.PutUint32(buf[4:8], v.Begin)
		binary.LittleEndian.PutUint32(buf[8:12], v.Count)
	case rawMethodSig:
		binary.LittleEndian.PutUint32(buf[0:4], v.Name)
		binary.LittleEndian.PutUint32(buf[4:8], v.MType)
	case rawMethodSlot:
		binary.LittleEndian.PutUint32(buf[0:4], v.Name)
		binary.LittleEndian.PutUint32(buf[4:8], v.MType)
		binary.LittleEndian.PutUint32(buf[8:12], v.IFn)
		binary.LittleEndian.PutUint32(buf[12:16], v.TFn)
	case rawIfaceMethodDemand:
		binary.LittleEndian.PutUint32(buf[0:4], v.Target)
		binary.LittleEndian.PutUint32(buf[4:8], v.Name)
		binary.LittleEndian.PutUint32(buf[8:12], v.MType)
	case Symbol:
		binary.LittleEndian.PutUint32(buf[0:4], uint32(v))
	case uint32:
		binary.LittleEndian.PutUint32(buf[0:4], v)
	default:
		panic(fmt.Sprintf("unsupported type %T", v))
	}
}

// buildGroups converts a map[Symbol][]T into sorted groups + values.
// keepOrder: if true, values are appended in insertion order (for MethodInfo).
func buildGroups[T any](m map[Symbol][]T, keepOrder bool) ([]rawGroup, []T) {
	if len(m) == 0 {
		return nil, nil
	}

	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	var groups []rawGroup
	var values []T
	for _, ki := range keys {
		k := Symbol(ki)
		vals := m[k]
		begin := len(values)
		values = append(values, vals...)
		groups = append(groups, rawGroup{
			Key:   uint32(k),
			Begin: uint32(begin),
			Count: uint32(len(vals)),
		})
	}
	return groups, values
}

// buildStringSections builds the StringOffsets and StringData sections.
func buildStringSections(strings []string) ([]uint32, []byte, error) {
	if len(strings) == 0 {
		return nil, nil, nil
	}

	offsets := make([]uint32, 0, len(strings)+1)
	var data []byte

	offsets = append(offsets, 0)
	for _, s := range strings {
		if uint64(len(data))+uint64(len(s)) > math.MaxUint32 {
			return nil, nil, fmt.Errorf("string data too large")
		}
		data = append(data, s...)
		offsets = append(offsets, uint32(len(data)))
	}
	return offsets, data, nil
}

// setToSortedSlice converts a map[Symbol]struct{} to a sorted []Symbol.
func setToSortedSlice(m map[Symbol]struct{}) []Symbol {
	if len(m) == 0 {
		return nil
	}
	symbols := make([]Symbol, 0, len(m))
	for k := range m {
		symbols = append(symbols, k)
	}
	sort.Ints(symbols)
	return symbols
}

// convertMethodSig converts internal MethodSig to v2 rawMethodSig.
func convertMethodSig(sig MethodSig) rawMethodSig {
	return rawMethodSig{Name: uint32(sig.Name), MType: uint32(sig.MType)}
}

// convertMethodSlot converts internal MethodSlot to v2 rawMethodSlot.
func convertMethodSlot(slot MethodSlot) rawMethodSlot {
	return rawMethodSlot{
		Name:  uint32(slot.Sig.Name),
		MType: uint32(slot.Sig.MType),
		IFn:   uint32(slot.IFn),
		TFn:   uint32(slot.TFn),
	}
}

// convertIfaceDemand converts internal IfaceMethodDemand to v2 raw.
func convertIfaceDemand(d IfaceMethodDemand) rawIfaceMethodDemand {
	return rawIfaceMethodDemand{
		Target: uint32(d.Target),
		Name:   uint32(d.Sig.Name),
		MType:  uint32(d.Sig.MType),
	}
}

func convertSlice[T, U any](in []T, fn func(T) U) []U {
	if len(in) == 0 {
		return nil
	}
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}

func convertMapSlice[T, U any](m map[Symbol][]T, fn func(T) U) map[Symbol][]U {
	if len(m) == 0 {
		return nil
	}
	out := make(map[Symbol][]U, len(m))
	for k, v := range m {
		out[k] = convertSlice(v, fn)
	}
	return out
}

// WriteMetaV2 serializes PackageMeta to the LLP2 v2 binary format.
func (pm *PackageMeta) WriteMetaV2(w io.WriteSeeker) error {
	wr := &v2Writer{w: w}

	headerSize := uint16(unsafe.Sizeof(rawHeader{}))
	descSize := uint32(unsafe.Sizeof(rawSectionDesc{}))

	// Reserve space for header + directory: we'll patch at the end.
	// We know the section count: 2 string + 8*2 group/value + 1 reflect = 19 sections
	maxSections := uint32(32) // generous upper bound

	// Write placeholder header
	header := rawHeader{
		Magic:        [4]byte{'L', 'L', 'P', '2'},
		Version:      version2,
		HeaderSize:   headerSize,
		EndianTag:    endianTag,
		DirOffset:    uint64(headerSize),
		DirSize:      uint64(maxSections) * uint64(descSize),
	}

	buf := make([]byte, headerSize)
	packHeader(buf, header)
	if _, err := wr.Write(buf); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write placeholder directory entries
	placeholder := make([]byte, maxSections*descSize)
	if _, err := wr.Write(placeholder); err != nil {
		return fmt.Errorf("write dir placeholder: %w", err)
	}

	// Build string sections
	strOffs, strData, err := buildStringSections(pm.stringTable)
	if err != nil {
		return err
	}
	if err := writeTypedSection(wr, secStringOffsets, convertSlice(strOffs, func(v uint32) uint32 { return v })); err != nil {
		return fmt.Errorf("stringOffsets: %w", err)
	}
	if err := writeBytesSection(wr, secStringData, strData); err != nil {
		return fmt.Errorf("stringData: %w", err)
	}

	// OrdinaryEdges
	if err := writeGroupSection(wr, secOrdinaryGroups, secOrdinaryValues,
		pm.OrdinaryEdges, false); err != nil {
		return fmt.Errorf("ordinaryEdges: %w", err)
	}

	// TypeChildren
	if err := writeGroupSection(wr, secTypeChildGroups, secTypeChildValues,
		pm.TypeChildren, false); err != nil {
		return fmt.Errorf("typeChildren: %w", err)
	}

	// InterfaceInfo
	im := convertMapSlice(pm.InterfaceInfo, convertMethodSig)
	if err := writeGroupSection(wr, secInterfaceGroups, secInterfaceValues,
		im, true); err != nil {
		return fmt.Errorf("interfaceInfo: %w", err)
	}

	// UseIface
	if err := writeGroupSection(wr, secUseIfaceGroups, secUseIfaceValues,
		pm.UseIface, false); err != nil {
		return fmt.Errorf("useIface: %w", err)
	}

	// UseIfaceMethod
	uim := convertMapSlice(pm.UseIfaceMethod, convertIfaceDemand)
	if err := writeGroupSection(wr, secUseIfaceMGroups, secUseIfaceMValues,
		uim, false); err != nil {
		return fmt.Errorf("useIfaceMethod: %w", err)
	}

	// MethodInfo
	mi := convertMapSlice(pm.MethodInfo, convertMethodSlot)
	if err := writeGroupSection(wr, secMethodInfoGroups, secMethodInfoValues,
		mi, true); err != nil {
		return fmt.Errorf("methodInfo: %w", err)
	}

	// UseNamedMethod
	if err := writeGroupSection(wr, secUseNamedMGroups, secUseNamedMValues,
		pm.UseNamedMethod, false); err != nil {
		return fmt.Errorf("useNamedMethod: %w", err)
	}

	// ReflectMethod
	reflectVals := setToSortedSlice(pm.ReflectMethod)
	reflectU32 := convertSlice(reflectVals, func(v Symbol) uint32 { return uint32(v) })
	if err := writeTypedSection(wr, secReflectMValues, reflectU32); err != nil {
		return fmt.Errorf("reflectMethod: %w", err)
	}

	// Now patch header and directory
	fileSize := uint64(wr.offset)

	// Count actual sections
	sectionCount := uint32(len(wr.sections))

	// Go back and write header + directory
	if _, err := wr.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek start: %w", err)
	}

	header.FileSize = fileSize
	header.SectionCount = sectionCount
	header.DirSize = uint64(sectionCount) * uint64(descSize)
	header.StringCount = uint32(len(pm.stringTable))

	hdrBuf := make([]byte, headerSize)
	packHeader(hdrBuf, header)
	if _, err := wr.Write(hdrBuf); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write directory
	for _, s := range wr.sections {
		descBuf := make([]byte, descSize)
		packSectionDesc(descBuf, s)
		if _, err := wr.Write(descBuf); err != nil {
			return fmt.Errorf("write section desc: %w", err)
		}
	}

	return nil
}

func writeGroupSection[T any](w *v2Writer, gk, vk sectionKind, m map[Symbol][]T, keepOrder bool) error {
	groups, values := buildGroups(m, keepOrder)
	if err := writeTypedSection(w, gk, groups); err != nil {
		return err
	}
	if err := writeTypedSection(w, vk, values); err != nil {
		return err
	}
	return nil
}

func packHeader(buf []byte, h rawHeader) {
	copy(buf[0:4], h.Magic[:])
	binary.LittleEndian.PutUint16(buf[4:6], h.Version)
	binary.LittleEndian.PutUint16(buf[6:8], h.HeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], h.EndianTag)
	binary.LittleEndian.PutUint32(buf[12:16], h.SectionCount)
	binary.LittleEndian.PutUint32(buf[16:20], h.Flags)
	binary.LittleEndian.PutUint64(buf[20:28], h.FileSize)
	binary.LittleEndian.PutUint64(buf[28:36], h.DirOffset)
	binary.LittleEndian.PutUint64(buf[36:44], h.DirSize)
	binary.LittleEndian.PutUint32(buf[44:48], h.StringCount)
	binary.LittleEndian.PutUint32(buf[48:52], h.Reserved0)
	binary.LittleEndian.PutUint32(buf[52:56], h.Checksum)
	// Reserved1 [28]byte stays zero
}

func packSectionDesc(buf []byte, s rawSectionDesc) {
	binary.LittleEndian.PutUint32(buf[0:4], s.Kind)
	binary.LittleEndian.PutUint32(buf[4:8], s.ElemSize)
	binary.LittleEndian.PutUint64(buf[8:16], s.Offset)
	binary.LittleEndian.PutUint64(buf[16:24], s.Count)
	binary.LittleEndian.PutUint64(buf[24:32], s.ByteSize)
	binary.LittleEndian.PutUint32(buf[32:36], s.Flags)
	binary.LittleEndian.PutUint32(buf[36:40], s.Reserved)
}
