package plan9asm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// NativeData describes a Go global whose storage is supplied by native assembly.
type NativeData struct {
	Name string
	Size uint32
}

// NativeARM64Object preserves the machine code of foreign-ABI assembly. Unlike
// Go-declared functions, raw callbacks have no signature from which LLVM can
// reconstruct incoming registers. The Go assembler encodes those registers;
// this bridge only converts its address and branch relocations to Mach-O asm.
//
// The input format is cmd/internal/goobj's go120ld format. Only explicitly
// selected functions, their data, and declared dynamic imports are accepted.
// Unknown formats, symbol references and relocations are errors.
func NativeARM64Object(obj []byte, funcs map[string]bool, imports map[string]string, pkgPath string) (string, []NativeData, error) {
	r, err := readNativeObject(obj)
	if err != nil {
		return "", nil, err
	}
	labels := make(map[int]string)
	var data []NativeData
	for i, s := range r.syms[:r.ndef] {
		if funcs[s.name] {
			if s.flags&16 == 0 {
				return "", nil, fmt.Errorf("native function %s is not NOSPLIT", s.name)
			}
			labels[i] = fmt.Sprintf("Lllgo_native_%d", i)
		} else if strings.HasPrefix(s.name, pkgPath+".") {
			labels[i] = "_" + s.name
			data = append(data, NativeData{s.name, s.size})
		}
	}
	for name := range funcs {
		found := false
		for i := range labels {
			if r.syms[i].name == name {
				found = true
				break
			}
		}
		if !found {
			return "", nil, fmt.Errorf("missing native function %s", name)
		}
	}
	var out strings.Builder
	for i, s := range r.syms[:r.ndef] {
		label, ok := labels[i]
		if !ok {
			continue
		}
		code := r.payload(i)
		if uint64(len(code)) > uint64(s.size) {
			return "", nil, fmt.Errorf("oversized native symbol %s", s.name)
		}
		code = append(append([]byte(nil), code...), make([]byte, int(s.size)-len(code))...)
		if funcs[s.name] {
			out.WriteString(".text\n.p2align 2\n")
		} else {
			out.WriteString(".data\n")
			fmt.Fprintf(&out, ".globl %s\n", strconv.Quote(label))
			align := s.align
			if align == 0 {
				align = 8
			}
			if align&(align-1) != 0 {
				return "", nil, fmt.Errorf("invalid native alignment %d", align)
			}
			fmt.Fprintf(&out, ".balign %d\n", align)
		}
		fmt.Fprintf(&out, "%s:\n", strconv.Quote(label))
		rels := r.relocs(i)
		sort.Slice(rels, func(i, j int) bool { return rels[i].off < rels[j].off })
		pos := uint32(0)
		for _, rel := range rels {
			if rel.off < pos || uint64(rel.off)+uint64(rel.size) > uint64(len(code)) {
				return "", nil, fmt.Errorf("invalid native relocation in %s", s.name)
			}
			target, err := r.target(rel.pkg, rel.sym)
			if err != nil {
				return "", nil, err
			}
			dst, ok := labels[target]
			if !ok {
				alias, ok := imports[r.syms[target].name]
				if !ok {
					return "", nil, fmt.Errorf("native assembly references undeclared foreign symbol %s", r.syms[target].name)
				}
				dst = "_" + alias
			}
			expr := strconv.Quote(dst)
			if rel.add != 0 {
				expr += fmt.Sprintf("%+d", rel.add)
			}
			nativeBytes(&out, code[pos:rel.off])
			switch {
			case rel.kind == 1 && rel.size == 8: // R_ADDR
				fmt.Fprintf(&out, ".quad %s\n", expr)
			case rel.kind == 9 && rel.size == 4 && funcs[s.name]: // R_CALLARM64: B or BL
				word := binary.LittleEndian.Uint32(code[rel.off:])
				op := "b"
				if word&0xfc000000 == 0x94000000 {
					op = "bl"
				} else if word&0xfc000000 != 0x14000000 {
					return "", nil, fmt.Errorf("invalid native branch in %s", s.name)
				}
				fmt.Fprintf(&out, "%s %s\n", op, expr)
			default:
				return "", nil, fmt.Errorf("unsupported native relocation %d/%d in %s", rel.kind, rel.size, s.name)
			}
			pos = rel.off + uint32(rel.size)
		}
		nativeBytes(&out, code[pos:])
	}
	return out.String(), data, nil
}

func nativeBytes(out *strings.Builder, b []byte) {
	for len(b) > 0 {
		n := min(16, len(b))
		out.WriteString(".byte ")
		for i, x := range b[:n] {
			if i != 0 {
				out.WriteByte(',')
			}
			fmt.Fprintf(out, "%d", x)
		}
		out.WriteByte('\n')
		b = b[n:]
	}
}

type nativeSym struct {
	name        string
	size, align uint32
	flags       byte
}
type nativeReloc struct {
	off      uint32
	size     byte
	kind     uint16
	add      int64
	pkg, sym uint32
}
type nativeObject struct {
	b      []byte
	blocks [19]uint32
	syms   []nativeSym
	ndef   int
	counts [5]int
}

func readNativeObject(obj []byte) (*nativeObject, error) {
	// cmd/asm prefixes the binary object with a target/version text header.
	h := bytes.Index(obj, []byte("\n!\n"))
	if h < 0 || !bytes.HasPrefix(obj, []byte("go object darwin arm64 ")) {
		return nil, fmt.Errorf("expected darwin/arm64 Go assembler object")
	}
	b := obj[h+3:]
	if len(b) < 96 || string(b[:8]) != "\x00go120ld" {
		return nil, fmt.Errorf("unsupported Go assembler object format")
	}
	r := &nativeObject{b: b}
	for i := range r.blocks {
		r.blocks[i] = binary.LittleEndian.Uint32(b[20+i*4:])
		if r.blocks[i] > uint32(len(b)) || (i > 0 && r.blocks[i] < r.blocks[i-1]) {
			return nil, fmt.Errorf("invalid native object block %d", i)
		}
	}
	if r.blocks[0] < 96 {
		return nil, fmt.Errorf("invalid native object header")
	}
	for block := 3; block <= 7; block++ {
		buf := r.block(block)
		if len(buf)%21 != 0 {
			return nil, fmt.Errorf("invalid native symbol table")
		}
		r.counts[block-3] = len(buf) / 21
		for len(buf) > 0 {
			s := buf[:21]
			n, off := binary.LittleEndian.Uint32(s), binary.LittleEndian.Uint32(s[4:])
			if uint64(off)+uint64(n) > uint64(len(b)) {
				return nil, fmt.Errorf("invalid native symbol name")
			}
			size := binary.LittleEndian.Uint32(s[13:])
			if size > 64<<20 {
				return nil, fmt.Errorf("native symbol too large")
			}
			r.syms = append(r.syms, nativeSym{string(b[off : off+n]), size, binary.LittleEndian.Uint32(s[17:]), s[11]})
			buf = buf[21:]
		}
	}
	r.ndef = len(r.syms) - r.counts[4]
	if len(r.block(11)) != (r.ndef+1)*4 || len(r.block(13)) != (r.ndef+1)*4 || len(r.block(14))%23 != 0 {
		return nil, fmt.Errorf("invalid native object indices")
	}
	for _, pair := range [][2]int{{11, len(r.block(14)) / 23}, {13, len(r.block(16))}} {
		prev := uint32(0)
		buf := r.block(pair[0])
		for len(buf) > 0 {
			v := binary.LittleEndian.Uint32(buf)
			if v < prev || uint64(v) > uint64(pair[1]) {
				return nil, fmt.Errorf("invalid native object index")
			}
			prev = v
			buf = buf[4:]
		}
	}
	return r, nil
}
func (r *nativeObject) block(i int) []byte { return r.b[r.blocks[i]:r.blocks[i+1]] }
func (r *nativeObject) payload(i int) []byte {
	idx := r.block(13)
	return r.block(16)[binary.LittleEndian.Uint32(idx[i*4:]):binary.LittleEndian.Uint32(idx[(i+1)*4:])]
}
func (r *nativeObject) relocs(i int) []nativeReloc {
	idx := r.block(11)
	a, z := binary.LittleEndian.Uint32(idx[i*4:]), binary.LittleEndian.Uint32(idx[(i+1)*4:])
	var out []nativeReloc
	for n := a; n < z; n++ {
		v := r.block(14)[int(n)*23:]
		out = append(out, nativeReloc{binary.LittleEndian.Uint32(v), v[4], binary.LittleEndian.Uint16(v[5:]), int64(binary.LittleEndian.Uint64(v[7:])), binary.LittleEndian.Uint32(v[15:]), binary.LittleEndian.Uint32(v[19:])})
	}
	return out
}
func (r *nativeObject) target(pkg, sym uint32) (int, error) {
	// Reserved package indices from cmd/internal/goobj. Package imports and
	// runtime builtins have Go ABI and cannot be called by these foreign stubs.
	var start, count int
	switch pkg {
	case 0x7ffffffb:
		count = r.counts[0]
	case 0x7ffffffe:
		start = r.counts[0]
		count = r.counts[1]
	case 0x7ffffffd:
		start = r.counts[0] + r.counts[1]
		count = r.counts[2]
	case 0x7fffffff:
		start = r.counts[0] + r.counts[1] + r.counts[2]
		count = r.counts[3] + r.counts[4]
	default:
		return 0, fmt.Errorf("native assembly references Go package index %#x", pkg)
	}
	if uint64(sym) >= uint64(count) {
		return 0, fmt.Errorf("invalid native symbol reference")
	}
	return start + int(sym), nil
}
