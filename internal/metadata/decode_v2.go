package metadata

import (
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"
)

// ---- v2 in-memory views ----

// Range describes a contiguous slice within a values array.
type Range struct {
	Begin uint32
	Count uint32
}

// GroupView provides O(1) keyed lookup over groups+values arrays with zero-copy value access.
type GroupView[T any] struct {
	groups []rawGroup
	values []T
	index  map[Symbol]Range
}

// BuildIndex constructs the key→Range index from groups.
func (v *GroupView[T]) BuildIndex() error {
	if v.index != nil {
		return nil // already built
	}
	v.index = make(map[Symbol]Range, len(v.groups))
	for _, g := range v.groups {
		if uint64(g.Begin)+uint64(g.Count) > uint64(len(v.values)) {
			return fmt.Errorf("group out of bounds: key=%d begin=%d count=%d total=%d",
				g.Key, g.Begin, g.Count, len(v.values))
		}
		key := Symbol(g.Key)
		if _, exists := v.index[key]; exists {
			return fmt.Errorf("duplicate group key: %d", key)
		}
		v.index[key] = Range{Begin: g.Begin, Count: g.Count}
	}
	return nil
}

// Lookup returns the values slice for a key, or nil if not found.
// The returned slice references the backing data without copying.
func (v *GroupView[T]) Lookup(key Symbol) []T {
	r, ok := v.index[key]
	if !ok {
		return nil
	}
	return v.values[r.Begin : r.Begin+r.Count]
}

// Len returns the number of distinct keys.
func (v *GroupView[T]) Len() int { return len(v.index) }

// StringTableView provides O(1) string lookup using offsets+data.
type StringTableView struct {
	offsets []uint32
	data    []byte
	cached  []string // lazily filled
}

// Lookup returns the string for a given Symbol ID.
func (s *StringTableView) Lookup(id Symbol) string {
	if id < 0 || int(id)+1 >= len(s.offsets) {
		return ""
	}
	begin := s.offsets[id]
	end := s.offsets[id+1]
	if int(end) > len(s.data) {
		return ""
	}
	return string(s.data[begin:end])
}

// Len returns the number of strings.
func (s *StringTableView) Len() int {
	if len(s.offsets) == 0 {
		return 0
	}
	return len(s.offsets) - 1
}

// SetView provides O(1) membership testing over a sorted Symbol set.
type SetView struct {
	values []Symbol
	set    map[Symbol]struct{}
}

// BuildIndex populates the internal set.
func (s *SetView) BuildIndex() {
	if s.set != nil {
		return
	}
	s.set = make(map[Symbol]struct{}, len(s.values))
	for _, v := range s.values {
		s.set[v] = struct{}{}
	}
}

// Has returns true if the value is in the set.
func (s *SetView) Has(v Symbol) bool {
	_, ok := s.set[v]
	return ok
}

// PackageMetaView is the read-only, mmap-friendly view of a serialized PackageMeta.
// It uses fixed-width sections with an internal index for O(1) queries.
type PackageMetaView struct {
	data []byte // backing bytes (mmap or read)

	strings StringTableView

	ordinaryEdges  GroupView[Symbol]
	typeChildren   GroupView[Symbol]
	interfaceInfo  GroupView[rawMethodSig]
	useIface       GroupView[Symbol]
	useIfaceMethod GroupView[rawIfaceMethodDemand]
	methodInfo     GroupView[rawMethodSlot]
	useNamedMethod GroupView[Symbol]
	reflectMethod  SetView
}

// Strings returns the string table view (read-only).
func (v *PackageMetaView) Strings() *StringTableView { return &v.strings }

// ---- query methods ----

func (v *PackageMetaView) Lookup(id Symbol) string { return v.strings.Lookup(id) }

func (v *PackageMetaView) OrdinaryEdges(src Symbol) []Symbol {
	return v.ordinaryEdges.Lookup(src)
}
func (v *PackageMetaView) TypeChildren(parent Symbol) []Symbol {
	return v.typeChildren.Lookup(parent)
}
func (v *PackageMetaView) InterfaceMethods(iface Symbol) []rawMethodSig {
	return v.interfaceInfo.Lookup(iface)
}
func (v *PackageMetaView) UseIface(owner Symbol) []Symbol {
	return v.useIface.Lookup(owner)
}
func (v *PackageMetaView) UseIfaceMethod(owner Symbol) []rawIfaceMethodDemand {
	return v.useIfaceMethod.Lookup(owner)
}
func (v *PackageMetaView) MethodSlots(typ Symbol) []rawMethodSlot {
	return v.methodInfo.Lookup(typ)
}
func (v *PackageMetaView) UseNamedMethod(owner Symbol) []Symbol {
	return v.useNamedMethod.Lookup(owner)
}
func (v *PackageMetaView) HasReflectMethod(owner Symbol) bool {
	return v.reflectMethod.Has(owner)
}

// Close releases the backing data. After Close, all returned slices are invalid.
func (v *PackageMetaView) Close() {
	v.data = nil
}

// ---- section-as-typed-slice (safe decode version, no unsafe reinterpret) ----

func sectionAsSymbol(data []byte, desc rawSectionDesc) ([]Symbol, error) {
	u32s, err := sectionAsUint32(data, desc)
	if err != nil {
		return nil, err
	}
	out := make([]Symbol, len(u32s))
	for i, v := range u32s {
		out[i] = Symbol(v)
	}
	return out, nil
}

func sectionAsUint32(data []byte, desc rawSectionDesc) ([]uint32, error) {
	if desc.Count == 0 {
		return nil, nil
	}
	if desc.ElemSize != 4 {
		return nil, fmt.Errorf("elem size mismatch for uint32: got %d", desc.ElemSize)
	}
	if desc.ByteSize != desc.Count*uint64(desc.ElemSize) {
		return nil, fmt.Errorf("byte size mismatch")
	}
	off := desc.Offset
	end := off + desc.ByteSize
	if end > uint64(len(data)) {
		return nil, fmt.Errorf("section out of range")
	}
	out := make([]uint32, desc.Count)
	off32 := int(off)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(data[off32+i*4 : off32+(i+1)*4])
	}
	return out, nil
}

func sectionAsRawGroup(data []byte, desc rawSectionDesc) ([]rawGroup, error) {
	if desc.Count == 0 {
		return nil, nil
	}
	if desc.ElemSize != uint32(unsafe.Sizeof(rawGroup{})) {
		return nil, fmt.Errorf("elem size mismatch for rawGroup: got %d", desc.ElemSize)
	}
	return decodeSlice[rawGroup](data, desc, func(buf []byte) rawGroup {
		return rawGroup{
			Key:   binary.LittleEndian.Uint32(buf[0:4]),
			Begin: binary.LittleEndian.Uint32(buf[4:8]),
			Count: binary.LittleEndian.Uint32(buf[8:12]),
		}
	}), nil
}

func sectionAsRawMethodSig(data []byte, desc rawSectionDesc) ([]rawMethodSig, error) {
	return decodeSlice[rawMethodSig](data, desc, func(buf []byte) rawMethodSig {
		return rawMethodSig{
			Name:  binary.LittleEndian.Uint32(buf[0:4]),
			MType: binary.LittleEndian.Uint32(buf[4:8]),
		}
	}), nil
}

func sectionAsRawMethodSlot(data []byte, desc rawSectionDesc) ([]rawMethodSlot, error) {
	return decodeSlice[rawMethodSlot](data, desc, func(buf []byte) rawMethodSlot {
		return rawMethodSlot{
			Name:  binary.LittleEndian.Uint32(buf[0:4]),
			MType: binary.LittleEndian.Uint32(buf[4:8]),
			IFn:   binary.LittleEndian.Uint32(buf[8:12]),
			TFn:   binary.LittleEndian.Uint32(buf[12:16]),
		}
	}), nil
}

func sectionAsRawIfaceDemand(data []byte, desc rawSectionDesc) ([]rawIfaceMethodDemand, error) {
	return decodeSlice[rawIfaceMethodDemand](data, desc, func(buf []byte) rawIfaceMethodDemand {
		return rawIfaceMethodDemand{
			Target: binary.LittleEndian.Uint32(buf[0:4]),
			Name:   binary.LittleEndian.Uint32(buf[4:8]),
			MType:  binary.LittleEndian.Uint32(buf[8:12]),
		}
	}), nil
}

func decodeSlice[T any](data []byte, desc rawSectionDesc, fn func([]byte) T) []T {
	if desc.Count == 0 {
		return nil
	}
	elemSize := int(desc.ElemSize)
	off := int(desc.Offset)
	out := make([]T, desc.Count)
	for i := range out {
		out[i] = fn(data[off : off+elemSize])
		off += elemSize
	}
	return out
}

func sectionBytes(data []byte, desc rawSectionDesc) ([]byte, error) {
	if desc.Count == 0 {
		return nil, nil
	}
	off := desc.Offset
	end := off + desc.ByteSize
	if end > uint64(len(data)) {
		return nil, fmt.Errorf("section bytes out of range")
	}
	// Copy the bytes out
	out := make([]byte, desc.Count)
	copy(out, data[off:end])
	return out, nil
}

// ---- Open / Read v2 ----

// OpenMetaV2 reads a LLP2 file from disk and returns a PackageMetaView.
func OpenMetaV2(path string) (*PackageMetaView, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseMetaV2(data)
}

// ParseMetaV2 parses a LLP2 blob into a PackageMetaView.
func ParseMetaV2(data []byte) (*PackageMetaView, error) {
	// Parse header
	dataLen := uint64(len(data))
	if dataLen < uint64(unsafe.Sizeof(rawHeader{})) {
		return nil, fmt.Errorf("file too small for header")
	}
	header := parseHeader(data)
	if string(header.Magic[:]) != magicV2 {
		return nil, fmt.Errorf("bad magic: %q", string(header.Magic[:]))
	}
	if header.Version != version2 {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}
	if header.FileSize > dataLen {
		return nil, fmt.Errorf("truncated file: header says %d, got %d", header.FileSize, dataLen)
	}

	// Parse section directory
	descSize := uint32(unsafe.Sizeof(rawSectionDesc{}))
	dirOff := header.DirOffset
	dirEnd := dirOff + header.DirSize
	if dirEnd > dataLen {
		return nil, fmt.Errorf("directory out of range")
	}

	dirCount := header.SectionCount
	secMap := make(map[sectionKind]rawSectionDesc, dirCount)
	off := int(dirOff)
	for range dirCount {
		var d rawSectionDesc
		d.Kind = binary.LittleEndian.Uint32(data[off:])
		d.ElemSize = binary.LittleEndian.Uint32(data[off+4:])
		d.Offset = binary.LittleEndian.Uint64(data[off+8:])
		d.Count = binary.LittleEndian.Uint64(data[off+16:])
		d.ByteSize = binary.LittleEndian.Uint64(data[off+24:])
		d.Flags = binary.LittleEndian.Uint32(data[off+32:])
		d.Reserved = binary.LittleEndian.Uint32(data[off+36:])
		secMap[sectionKind(d.Kind)] = d
		off += int(descSize)
	}

	v := &PackageMetaView{data: data}

	// String table
	strOffs, err := sectionAsUint32(data, secMap[secStringOffsets])
	if err != nil {
		return nil, fmt.Errorf("stringOffsets: %w", err)
	}
	strData, err := sectionBytes(data, secMap[secStringData])
	if err != nil {
		return nil, fmt.Errorf("stringData: %w", err)
	}
	v.strings = StringTableView{offsets: strOffs, data: strData}

	// OrdinaryEdges
	if err := loadGroup(data, secMap, secOrdinaryGroups, secOrdinaryValues, &v.ordinaryEdges,
		func() ([]Symbol, error) { return sectionAsSymbol(data, secMap[secOrdinaryValues]) }); err != nil {
		return nil, fmt.Errorf("ordinaryEdges: %w", err)
	}

	// TypeChildren
	if err := loadGroup(data, secMap, secTypeChildGroups, secTypeChildValues, &v.typeChildren,
		func() ([]Symbol, error) { return sectionAsSymbol(data, secMap[secTypeChildValues]) }); err != nil {
		return nil, fmt.Errorf("typeChildren: %w", err)
	}

	// InterfaceInfo
	if err := loadGroup(data, secMap, secInterfaceGroups, secInterfaceValues, &v.interfaceInfo,
		func() ([]rawMethodSig, error) { return sectionAsRawMethodSig(data, secMap[secInterfaceValues]) }); err != nil {
		return nil, fmt.Errorf("interfaceInfo: %w", err)
	}

	// UseIface
	if err := loadGroup(data, secMap, secUseIfaceGroups, secUseIfaceValues, &v.useIface,
		func() ([]Symbol, error) { return sectionAsSymbol(data, secMap[secUseIfaceValues]) }); err != nil {
		return nil, fmt.Errorf("useIface: %w", err)
	}

	// UseIfaceMethod
	if err := loadGroup(data, secMap, secUseIfaceMGroups, secUseIfaceMValues, &v.useIfaceMethod,
		func() ([]rawIfaceMethodDemand, error) {
			return sectionAsRawIfaceDemand(data, secMap[secUseIfaceMValues])
		}); err != nil {
		return nil, fmt.Errorf("useIfaceMethod: %w", err)
	}

	// MethodInfo
	if err := loadGroup(data, secMap, secMethodInfoGroups, secMethodInfoValues, &v.methodInfo,
		func() ([]rawMethodSlot, error) {
			return sectionAsRawMethodSlot(data, secMap[secMethodInfoValues])
		}); err != nil {
		return nil, fmt.Errorf("methodInfo: %w", err)
	}

	// UseNamedMethod
	if err := loadGroup(data, secMap, secUseNamedMGroups, secUseNamedMValues, &v.useNamedMethod,
		func() ([]Symbol, error) { return sectionAsSymbol(data, secMap[secUseNamedMValues]) }); err != nil {
		return nil, fmt.Errorf("useNamedMethod: %w", err)
	}

	// ReflectMethod
	reflectVals, err := sectionAsUint32(data, secMap[secReflectMValues])
	if err != nil {
		return nil, fmt.Errorf("reflectMethod: %w", err)
	}
	rv := make([]Symbol, len(reflectVals))
	for i, v := range reflectVals {
		rv[i] = Symbol(v)
	}
	v.reflectMethod = SetView{values: rv}
	v.reflectMethod.BuildIndex()

	return v, nil
}

func loadGroup[T any](data []byte, secMap map[sectionKind]rawSectionDesc,
	gk, vk sectionKind, gv *GroupView[T], readVals func() ([]T, error)) error {

	groups, err := sectionAsRawGroup(data, secMap[gk])
	if err != nil {
		return fmt.Errorf("groups: %w", err)
	}
	if len(groups) == 0 {
		return nil
	}
	values, err := readVals()
	if err != nil {
		return fmt.Errorf("values: %w", err)
	}
	gv.groups = groups
	gv.values = values
	return gv.BuildIndex()
}

func parseHeader(data []byte) rawHeader {
	return rawHeader{
		Magic:        [4]byte{data[0], data[1], data[2], data[3]},
		Version:      binary.LittleEndian.Uint16(data[4:6]),
		HeaderSize:   binary.LittleEndian.Uint16(data[6:8]),
		EndianTag:    binary.LittleEndian.Uint32(data[8:12]),
		SectionCount: binary.LittleEndian.Uint32(data[12:16]),
		Flags:        binary.LittleEndian.Uint32(data[16:20]),
		FileSize:     binary.LittleEndian.Uint64(data[20:28]),
		DirOffset:    binary.LittleEndian.Uint64(data[28:36]),
		DirSize:      binary.LittleEndian.Uint64(data[36:44]),
		StringCount:  binary.LittleEndian.Uint32(data[44:48]),
		Reserved0:    binary.LittleEndian.Uint32(data[48:52]),
		Checksum:     binary.LittleEndian.Uint32(data[52:56]),
	}
}
