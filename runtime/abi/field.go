package abi

import "unsafe"

const (
	fieldExported = 1 << iota
	fieldEmbedded
	fieldHasTag

	fieldNameShift = 3
	fieldLongName  = 1<<(8-fieldNameShift) - 1
)

// EncodeStructFieldName encodes the field's name and tag as one immutable
// string. The first byte carries three flags and the length of a short name.
// A longer name has a varint length after that byte. If a tag is present, its
// varint length and bytes follow the name. Both the compiler and StructOf
// produce the same layout.
func EncodeStructFieldName(name, tag string, embedded bool) string {
	var header [1 + 10]byte
	size := len(name)
	if size > fieldLongName {
		size = fieldLongName
	}
	header[0] = byte(size << fieldNameShift)
	if IsExported(name) {
		header[0] |= fieldExported
	}
	if embedded {
		header[0] |= fieldEmbedded
	}
	if tag != "" {
		header[0] |= fieldHasTag
	}
	n := 1
	if size == fieldLongName {
		n += putFieldLength(header[1:], len(name))
	}
	if tag == "" {
		return string(header[:n]) + name
	}
	var tagHeader [10]byte
	m := putFieldLength(tagHeader[:], len(tag))
	return string(header[:n]) + name + string(tagHeader[:m]) + tag
}

func putFieldLength(dst []byte, size int) int {
	i := 0
	for size >= 128 {
		dst[i] = byte(size) | 0x80
		size >>= 7
		i++
	}
	dst[i] = byte(size)
	return i + 1
}

func readFieldLength(p *byte) (int, *byte) {
	var size int
	for shift := 0; ; shift += 7 {
		b := *p
		size |= int(b&0x7f) << shift
		p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1))
		if b < 128 {
			return size, p
		}
	}
}

// NewStructField constructs a field descriptor for a runtime-created type.
func NewStructField(name string, typ *Type, offset uintptr, tag string, embedded bool) StructField {
	return StructField{
		Name_:  unsafe.StringData(EncodeStructFieldName(name, tag, embedded)),
		Typ:    typ,
		Offset: offset,
	}
}

func (f *StructField) nameData() (int, *byte) {
	p := (*byte)(unsafe.Add(unsafe.Pointer(f.Name_), 1))
	size := int(*f.Name_) >> fieldNameShift
	if size == fieldLongName {
		return readFieldLength(p)
	}
	return size, p
}

func (f *StructField) Name() string {
	size, p := f.nameData()
	return unsafe.String(p, size)
}

func (f *StructField) Tag() string {
	if *f.Name_&fieldHasTag == 0 {
		return ""
	}
	size, p := f.nameData()
	tagSize, tag := readFieldLength((*byte)(unsafe.Add(unsafe.Pointer(p), size)))
	return unsafe.String(tag, tagSize)
}

func (f *StructField) Embedded() bool {
	return *f.Name_&fieldEmbedded != 0
}

func (f *StructField) Exported() bool {
	return *f.Name_&fieldExported != 0
}

func (f *StructField) Blank() bool {
	return *f.Name_>>fieldNameShift == 1 && *(*byte)(unsafe.Add(unsafe.Pointer(f.Name_), 1)) == '_'
}
