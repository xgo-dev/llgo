package wasmdebug

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

// ReplaceData changes one compiler-reserved initialized linear-memory slot.
// It never rewrites code, offsets, or section sizes, preserving final DWARF.
func ReplaceData(module []byte, symbol string, expected, replacement []byte) ([]byte, error) {
	if len(expected) == 0 || len(expected) != len(replacement) {
		return nil, fmt.Errorf("invalid Wasm data replacement size")
	}
	sections, err := parse(module)
	if err != nil {
		return nil, err
	}
	match := -1
	sectionOffset := len(wasmHeader)
	for _, sec := range sections {
		contentOffset := sectionOffset + len(sec.raw) - len(sec.content)
		sectionOffset += len(sec.raw)
		if sec.id != 11 {
			continue
		}
		off := 0
		count, err := readULEB32(sec.content, &off)
		if err != nil {
			return nil, err
		}
		for range count {
			flags, err := readULEB32(sec.content, &off)
			if err != nil || flags > 2 {
				return nil, fmt.Errorf("invalid Wasm data segment")
			}
			if flags == 2 {
				if _, err = readULEB32(sec.content, &off); err != nil {
					return nil, err
				}
			}
			if flags != 1 {
				if _, err = readConstAddress(sec.content, &off); err != nil {
					return nil, err
				}
			}
			size, err := readULEB32(sec.content, &off)
			if err != nil || uint64(size) > uint64(len(sec.content)-off) {
				return nil, fmt.Errorf("truncated Wasm data")
			}
			payload := sec.content[off : off+int(size)]
			if n := bytes.Count(payload, expected); n != 0 {
				if n != 1 || match >= 0 {
					return nil, fmt.Errorf("ambiguous Wasm data slot %s", symbol)
				}
				match = contentOffset + off + bytes.Index(payload, expected)
			}
			off += int(size)
		}
	}
	if match < 0 {
		return nil, fmt.Errorf("missing Wasm data slot %s", symbol)
	}
	// Both passive/shared and active segments retain their exact layout. A
	// compiler-reserved slot is recognized by all 256 bits, not an export name
	// that Emscripten can minify. Ambiguity is an error, never a guessed rewrite.
	out := bytes.Clone(module)
	copy(out[match:], replacement)
	return out, nil
}

func PCLNIdentityPlaceholder() [32]byte { return sha256.Sum256([]byte("LLGo.PCLN.Identity.Slot.v1")) }

// PCLNModuleIdentity hashes the linked module with its identity slot still
// reserved. Custom sections can change during debugger packaging or stripping.
func PCLNModuleIdentity(module []byte) ([32]byte, error) {
	sections, err := parse(module)
	if err != nil {
		return [32]byte{}, err
	}
	hash := sha256.New()
	hash.Write(wasmHeader)
	for _, section := range sections {
		if section.id != 0 {
			hash.Write(section.raw)
		}
	}
	var identity [32]byte
	copy(identity[:], hash.Sum(nil))
	return identity, nil
}

// MatchPCLNIdentity verifies the sidecar against the linked module's contents.
func MatchPCLNIdentity(module []byte, identity [32]byte) error {
	slot := PCLNIdentityPlaceholder()
	unpatched, err := ReplaceData(module, "__llgo_pclntab_identity", identity[:], slot[:])
	if err != nil {
		return err
	}
	got, err := PCLNModuleIdentity(unpatched)
	if err != nil {
		return err
	}
	if got != identity {
		return fmt.Errorf("pclntab map does not match the Wasm module")
	}
	return nil
}

func readConstAddress(raw []byte, off *int) (uint64, error) {
	if *off >= len(raw) || (raw[*off] != 0x41 && raw[*off] != 0x42) {
		return 0, fmt.Errorf("unsupported Wasm address expression")
	}
	*off++
	var value uint64
	for shift := uint(0); shift < 70; shift += 7 {
		if *off >= len(raw) {
			break
		}
		b := raw[*off]
		*off++
		value |= uint64(b&127) << shift
		if b&128 == 0 {
			if *off >= len(raw) || raw[*off] != 0x0b {
				break
			}
			*off++
			return value, nil
		}
	}
	return 0, fmt.Errorf("invalid Wasm address expression")
}
