package metadata

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadMeta deserializes a PackageMeta from the LLPS v1 binary format.
func ReadMeta(r io.Reader) (*PackageMeta, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read meta: %w", err)
	}
	return decodeMeta(data)
}

func decodeMeta(data []byte) (*PackageMeta, error) {
	pos := 0

	// header
	if len(data) < 4+1 {
		return nil, fmt.Errorf("data too short for header")
	}
	magic := string(data[pos : pos+4])
	pos += 4
	if magic != magicV1 {
		return nil, fmt.Errorf("bad magic: %q", magic)
	}

	version, n := binary.Uvarint(data[pos:])
	if n <= 0 {
		return nil, fmt.Errorf("decode version: bad uvarint")
	}
	pos += n
	if version != version1 {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	// stringTable
	count, n := binary.Uvarint(data[pos:])
	if n <= 0 {
		return nil, fmt.Errorf("decode stringTable count")
	}
	pos += n

	stringTable := make([]string, count)
	for i := range stringTable {
		slen, n := binary.Uvarint(data[pos:])
		if n <= 0 {
			return nil, fmt.Errorf("decode string %d len", i)
		}
		pos += n
		if pos+int(slen) > len(data) {
			return nil, fmt.Errorf("string %d out of range", i)
		}
		stringTable[i] = string(data[pos : pos+int(slen)])
		pos += int(slen)
	}

	meta := NewPackageMeta(stringTable)

	// OrdinaryEdges
	if err := decodeOrdinaryEdges(data, &pos, meta.OrdinaryEdges); err != nil {
		return nil, fmt.Errorf("ordinaryEdges: %w", err)
	}

	// TypeChildren
	if err := decodeTypeChildren(data, &pos, meta.TypeChildren); err != nil {
		return nil, fmt.Errorf("typeChildren: %w", err)
	}

	// InterfaceInfo
	if err := decodeInterfaceInfo(data, &pos, meta.InterfaceInfo); err != nil {
		return nil, fmt.Errorf("interfaceInfo: %w", err)
	}

	// UseIface
	if err := decodeUseIface(data, &pos, meta.UseIface); err != nil {
		return nil, fmt.Errorf("useIface: %w", err)
	}

	// UseIfaceMethod
	if err := decodeUseIfaceMethod(data, &pos, meta.UseIfaceMethod); err != nil {
		return nil, fmt.Errorf("useIfaceMethod: %w", err)
	}

	// MethodInfo
	if err := decodeMethodInfo(data, &pos, meta.MethodInfo); err != nil {
		return nil, fmt.Errorf("methodInfo: %w", err)
	}

	// UseNamedMethod
	if err := decodeUseNamedMethod(data, &pos, meta.UseNamedMethod); err != nil {
		return nil, fmt.Errorf("useNamedMethod: %w", err)
	}

	// ReflectMethod
	if err := decodeReflectMethod(data, &pos, meta.ReflectMethod); err != nil {
		return nil, fmt.Errorf("reflectMethod: %w", err)
	}

	return meta, nil
}

func readUvarint(data []byte, pos *int) (uint64, error) {
	v, n := binary.Uvarint(data[*pos:])
	if n <= 0 {
		return 0, fmt.Errorf("bad uvarint at pos %d", *pos)
	}
	*pos += n
	return v, nil
}

func decodeOrdinaryEdges(data []byte, pos *int, m map[Symbol][]Symbol) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		src, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		ndst, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		dsts := make([]Symbol, ndst)
		for j := range dsts {
			v, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			dsts[j] = Symbol(v)
		}
		m[Symbol(src)] = dsts
	}
	return nil
}

func decodeTypeChildren(data []byte, pos *int, m map[Symbol][]Symbol) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		parent, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		nchild, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		children := make([]Symbol, nchild)
		for j := range children {
			v, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			children[j] = Symbol(v)
		}
		m[Symbol(parent)] = children
	}
	return nil
}

func decodeInterfaceInfo(data []byte, pos *int, m map[Symbol][]MethodSig) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		iface, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		nm, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		methods := make([]MethodSig, nm)
		for j := range methods {
			methods[j], err = readMethodSig(data, pos)
			if err != nil {
				return err
			}
		}
		m[Symbol(iface)] = methods
	}
	return nil
}

func readMethodSig(data []byte, pos *int) (MethodSig, error) {
	name, err := readUvarint(data, pos)
	if err != nil {
		return MethodSig{}, err
	}
	mtype, err := readUvarint(data, pos)
	if err != nil {
		return MethodSig{}, err
	}
	return MethodSig{Name: Symbol(name), MType: Symbol(mtype)}, nil
}

func decodeUseIface(data []byte, pos *int, m map[Symbol][]Symbol) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		owner, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		ntypes, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		types := make([]Symbol, ntypes)
		for j := range types {
			v, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			types[j] = Symbol(v)
		}
		m[Symbol(owner)] = types
	}
	return nil
}

func decodeUseIfaceMethod(data []byte, pos *int, m map[Symbol][]IfaceMethodDemand) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		owner, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		nd, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		demands := make([]IfaceMethodDemand, nd)
		for j := range demands {
			target, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			sig, err := readMethodSig(data, pos)
			if err != nil {
				return err
			}
			demands[j] = IfaceMethodDemand{Target: Symbol(target), Sig: sig}
		}
		m[Symbol(owner)] = demands
	}
	return nil
}

func decodeMethodInfo(data []byte, pos *int, m map[Symbol][]MethodSlot) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		typ, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		nslots, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		slots := make([]MethodSlot, nslots)
		for j := range slots {
			sig, err := readMethodSig(data, pos)
			if err != nil {
				return err
			}
			ifn, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			tfn, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			slots[j] = MethodSlot{Sig: sig, IFn: Symbol(ifn), TFn: Symbol(tfn)}
		}
		m[Symbol(typ)] = slots
	}
	return nil
}

func decodeUseNamedMethod(data []byte, pos *int, m map[Symbol][]Symbol) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		owner, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		nnames, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		names := make([]Symbol, nnames)
		for j := range names {
			v, err := readUvarint(data, pos)
			if err != nil {
				return err
			}
			names[j] = Symbol(v)
		}
		m[Symbol(owner)] = names
	}
	return nil
}

func decodeReflectMethod(data []byte, pos *int, m map[Symbol]struct{}) error {
	count, err := readUvarint(data, pos)
	if err != nil {
		return err
	}
	for range count {
		owner, err := readUvarint(data, pos)
		if err != nil {
			return err
		}
		m[Symbol(owner)] = struct{}{}
	}
	return nil
}
