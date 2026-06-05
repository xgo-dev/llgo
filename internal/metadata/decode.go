package metadata

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
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

	if len(data) < len(magicV1)+1 {
		return nil, fmt.Errorf("data too short for header")
	}
	if magic := string(data[:len(magicV1)]); magic != magicV1 {
		return nil, fmt.Errorf("bad magic: %q", magic)
	}
	pos += len(magicV1)

	version, err := readUvarint(data, &pos)
	if err != nil {
		return nil, fmt.Errorf("decode version: %w", err)
	}
	if version != version1 {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	table, err := decodeStringTable(data, &pos)
	if err != nil {
		return nil, fmt.Errorf("stringTable: %w", err)
	}
	meta := NewPackageMeta(table)

	if err := decodeSymbolMap(data, &pos, meta.ordinaryEdges); err != nil {
		return nil, fmt.Errorf("ordinaryEdges: %w", err)
	}
	if err := decodeSymbolMap(data, &pos, meta.typeChildren); err != nil {
		return nil, fmt.Errorf("typeChildren: %w", err)
	}
	if err := decodeInterfaceInfo(data, &pos, meta.interfaceInfo); err != nil {
		return nil, fmt.Errorf("interfaceInfo: %w", err)
	}
	if err := decodeSymbolMap(data, &pos, meta.useIface); err != nil {
		return nil, fmt.Errorf("useIface: %w", err)
	}
	if err := decodeUseIfaceMethod(data, &pos, meta.useIfaceMethod); err != nil {
		return nil, fmt.Errorf("useIfaceMethod: %w", err)
	}
	if err := decodeMethodInfo(data, &pos, meta.methodInfo); err != nil {
		return nil, fmt.Errorf("methodInfo: %w", err)
	}
	if err := decodeUseNamedMethod(data, &pos, meta.useNamedMethod); err != nil {
		return nil, fmt.Errorf("useNamedMethod: %w", err)
	}
	if err := decodeReflectMethod(data, &pos, meta.reflectMethod); err != nil {
		return nil, fmt.Errorf("reflectMethod: %w", err)
	}
	if pos != len(data) {
		return nil, fmt.Errorf("trailing data at pos %d", pos)
	}
	return meta, nil
}

func decodeStringTable(data []byte, pos *int) ([]string, error) {
	count, err := readUvarint(data, pos)
	if err != nil {
		return nil, fmt.Errorf("decode count: %w", err)
	}
	if count > uint64(math.MaxInt) {
		return nil, fmt.Errorf("count overflows int: %d", count)
	}
	table := make([]string, int(count))
	for i := range table {
		size, err := readUvarint(data, pos)
		if err != nil {
			return nil, fmt.Errorf("decode string %d len: %w", i, err)
		}
		if size > uint64(len(data)-*pos) {
			return nil, fmt.Errorf("string %d out of range", i)
		}
		table[i] = string(data[*pos : *pos+int(size)])
		*pos += int(size)
	}
	return table, nil
}

func readUvarint(data []byte, pos *int) (uint64, error) {
	if *pos >= len(data) {
		return 0, io.ErrUnexpectedEOF
	}
	value, n := binary.Uvarint(data[*pos:])
	if n == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	if n < 0 {
		return 0, fmt.Errorf("uvarint overflow at pos %d", *pos)
	}
	*pos += n
	return value, nil
}

func readSymbol(data []byte, pos *int) (Symbol, error) {
	value, err := readUvarint(data, pos)
	if err != nil {
		return 0, err
	}
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("symbol overflows uint32: %d", value)
	}
	return Symbol(value), nil
}

func readName(data []byte, pos *int) (Name, error) {
	value, err := readUvarint(data, pos)
	if err != nil {
		return 0, err
	}
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("name overflows uint32: %d", value)
	}
	return Name(value), nil
}

func readCount(data []byte, pos *int, label string) (int, error) {
	count, err := readUvarint(data, pos)
	if err != nil {
		return 0, err
	}
	if count > uint64(math.MaxInt) {
		return 0, fmt.Errorf("%s count overflows int: %d", label, count)
	}
	return int(count), nil
}

func decodeSymbolMap(data []byte, pos *int, m map[Symbol][]Symbol) error {
	count, err := readCount(data, pos, "symbol map")
	if err != nil {
		return err
	}
	for range count {
		key, err := readSymbol(data, pos)
		if err != nil {
			return err
		}
		nvalues, err := readCount(data, pos, "symbol values")
		if err != nil {
			return err
		}
		values := make([]Symbol, nvalues)
		for i := range values {
			values[i], err = readSymbol(data, pos)
			if err != nil {
				return err
			}
		}
		m[key] = values
	}
	return nil
}

func decodeInterfaceInfo(data []byte, pos *int, m map[Symbol][]MethodSig) error {
	count, err := readCount(data, pos, "interface info")
	if err != nil {
		return err
	}
	for range count {
		iface, err := readSymbol(data, pos)
		if err != nil {
			return err
		}
		nmethods, err := readCount(data, pos, "interface methods")
		if err != nil {
			return err
		}
		methods := make([]MethodSig, nmethods)
		for i := range methods {
			methods[i], err = readMethodSig(data, pos)
			if err != nil {
				return err
			}
		}
		m[iface] = methods
	}
	return nil
}

func readMethodSig(data []byte, pos *int) (MethodSig, error) {
	name, err := readName(data, pos)
	if err != nil {
		return MethodSig{}, err
	}
	mtype, err := readSymbol(data, pos)
	if err != nil {
		return MethodSig{}, err
	}
	return MethodSig{Name: name, MType: mtype}, nil
}

func decodeUseIfaceMethod(data []byte, pos *int, m map[Symbol][]IfaceMethodDemand) error {
	count, err := readCount(data, pos, "use interface method")
	if err != nil {
		return err
	}
	for range count {
		owner, err := readSymbol(data, pos)
		if err != nil {
			return err
		}
		ndemands, err := readCount(data, pos, "interface method demands")
		if err != nil {
			return err
		}
		demands := make([]IfaceMethodDemand, ndemands)
		for i := range demands {
			target, err := readSymbol(data, pos)
			if err != nil {
				return err
			}
			sig, err := readMethodSig(data, pos)
			if err != nil {
				return err
			}
			demands[i] = IfaceMethodDemand{Target: target, Sig: sig}
		}
		m[owner] = demands
	}
	return nil
}

func decodeMethodInfo(data []byte, pos *int, m map[Symbol][]MethodSlot) error {
	count, err := readCount(data, pos, "method info")
	if err != nil {
		return err
	}
	for range count {
		typ, err := readSymbol(data, pos)
		if err != nil {
			return err
		}
		nslots, err := readCount(data, pos, "method slots")
		if err != nil {
			return err
		}
		slots := make([]MethodSlot, nslots)
		for i := range slots {
			sig, err := readMethodSig(data, pos)
			if err != nil {
				return err
			}
			ifn, err := readSymbol(data, pos)
			if err != nil {
				return err
			}
			tfn, err := readSymbol(data, pos)
			if err != nil {
				return err
			}
			slots[i] = MethodSlot{Sig: sig, IFn: ifn, TFn: tfn}
		}
		m[typ] = slots
	}
	return nil
}

func decodeUseNamedMethod(data []byte, pos *int, m map[Symbol][]Name) error {
	count, err := readCount(data, pos, "use named method")
	if err != nil {
		return err
	}
	for range count {
		owner, err := readSymbol(data, pos)
		if err != nil {
			return err
		}
		nnames, err := readCount(data, pos, "method names")
		if err != nil {
			return err
		}
		names := make([]Name, nnames)
		for i := range names {
			names[i], err = readName(data, pos)
			if err != nil {
				return err
			}
		}
		m[owner] = names
	}
	return nil
}

func decodeReflectMethod(data []byte, pos *int, m map[Symbol]struct{}) error {
	count, err := readCount(data, pos, "reflect method")
	if err != nil {
		return err
	}
	for range count {
		owner, err := readSymbol(data, pos)
		if err != nil {
			return err
		}
		m[owner] = struct{}{}
	}
	return nil
}
