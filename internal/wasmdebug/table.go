package wasmdebug

import "fmt"

// FunctionTable reads the initial function indices in the module's ordinary
// indirect-call table. Unsupported element encodings disable this optional
// debugger view rather than guessing a mapping. LLGo's static browser links
// use active function-index segments with constant offsets in table zero.
// The result maps table indices to function indices.
func FunctionTable(module []byte) (map[uint32]uint32, error) {
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	var table map[uint32]uint32
	for _, section := range sections {
		if section.id != 9 {
			continue
		}
		if table != nil {
			return nil, fmt.Errorf("duplicate WebAssembly element section")
		}
		table = make(map[uint32]uint32)
		raw, pos := section.content, 0
		count, err := readULEB32(raw, &pos)
		if err != nil {
			return nil, err
		}
		for range count {
			flags, err := readULEB32(raw, &pos)
			if err != nil {
				return nil, err
			}
			// Flags 0 imply table zero and funcref; flags 2 encode the table
			// index and element kind explicitly.
			if flags != 0 && flags != 2 {
				return nil, fmt.Errorf("unsupported WebAssembly element encoding %d", flags)
			}
			if flags == 2 {
				index, err := readULEB32(raw, &pos)
				if err != nil || index != 0 {
					return nil, fmt.Errorf("unsupported WebAssembly element table")
				}
			}
			offset, err := tableOffset(raw, &pos)
			if err != nil {
				return nil, err
			}
			if flags == 2 {
				if pos == len(raw) || raw[pos] != 0 {
					return nil, fmt.Errorf("unsupported WebAssembly element kind")
				}
				pos++
			}
			n, err := readULEB32(raw, &pos)
			if err != nil || uint64(n) > uint64(len(raw)-pos) || uint64(offset)+uint64(n) > 1<<32 {
				return nil, fmt.Errorf("invalid WebAssembly element count")
			}
			for i := uint32(0); i < n; i++ {
				fn, err := readULEB32(raw, &pos)
				if err != nil {
					return nil, err
				}
				// Later segments overwrite earlier ones, as at instantiation.
				table[offset+i] = fn
			}
		}
		if pos != len(raw) {
			return nil, fmt.Errorf("trailing WebAssembly element bytes")
		}
	}
	return table, nil
}

func tableOffset(raw []byte, pos *int) (uint32, error) {
	if *pos < len(raw) && (raw[*pos] == 0x41 || raw[*pos] == 0x42) { // i32/i64.const
		bits := uint(32)
		if raw[*pos] == 0x42 {
			bits = 64
		}
		*pos++
		var value uint64
		for shift := uint(0); shift < bits && *pos < len(raw); shift += 7 {
			b := raw[*pos]
			*pos++
			if shift+7 >= bits && b>>(bits-shift-1) != 0 {
				break
			}
			value |= uint64(b&0x7f) << shift
			if b&0x80 == 0 {
				if b&0x40 == 0 && value < 1<<32 && *pos < len(raw) && raw[*pos] == 0x0b {
					*pos++
					return uint32(value), nil
				}
				break
			}
		}
	}
	return 0, fmt.Errorf("unsupported WebAssembly table offset")
}
