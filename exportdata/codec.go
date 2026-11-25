/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package exportdata implements compact, deterministic binary codecs for LLGo's
// pure-Go compiler artifacts: types.Package, analysis/SSA PackageMeta,
// LinkSnapshot (with binary PCLN records), and C header Depfiles.
//
// Analogous to golang.org/x/tools/go/gcexportdata, it requires only Go 1.20+
// stdlib with zero CGo, LLVM, or x/tools dependencies.
package exportdata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	magicHeader   = "LLGOEX"
	formatVersion = uint32(1)
)

const (
	kindPackage      byte = 1
	kindMeta         byte = 2
	kindLinkSnapshot byte = 3
	kindDepfile      byte = 4
)

var (
	ErrInvalidMagic   = errors.New("exportdata: invalid binary header magic")
	ErrUnsupportedVer = errors.New("exportdata: unsupported format version")
	ErrKindMismatch   = errors.New("exportdata: artifact kind mismatch")
	ErrCorrupt        = errors.New("exportdata: corrupt or truncated binary payload")
)

type encoder struct {
	buf     bytes.Buffer
	strings []string
	strIdx  map[string]uint64
}

func newEncoder() *encoder {
	return &encoder{strIdx: make(map[string]uint64)}
}

func (e *encoder) intern(s string) uint64 {
	if idx, ok := e.strIdx[s]; ok {
		return idx
	}
	idx := uint64(len(e.strings))
	e.strings = append(e.strings, s)
	e.strIdx[s] = idx
	return idx
}

func (e *encoder) writeByte(b byte) { e.buf.WriteByte(b) }

func (e *encoder) writeBool(v bool) {
	if v {
		e.buf.WriteByte(1)
	} else {
		e.buf.WriteByte(0)
	}
}

func (e *encoder) writeUvarint(v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], v)
	e.buf.Write(tmp[:n])
}

func (e *encoder) writeVarint(v int64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutVarint(tmp[:], v)
	e.buf.Write(tmp[:n])
}

func (e *encoder) writeFloat64(v float64) {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], math.Float64bits(v))
	e.buf.Write(tmp[:])
}

func (e *encoder) writeStr(s string) {
	e.writeUvarint(e.intern(s))
}

func (e *encoder) writeStringSlice(list []string) {
	e.writeUvarint(uint64(len(list)))
	for _, s := range list {
		e.writeStr(s)
	}
}

func (e *encoder) writeBytes(p []byte) {
	e.writeUvarint(uint64(len(p)))
	e.buf.Write(p)
}

func (e *encoder) finish(kind byte) []byte {
	var out bytes.Buffer
	out.WriteString(magicHeader)
	out.WriteByte(kind)
	var ver [4]byte
	binary.LittleEndian.PutUint32(ver[:], formatVersion)
	out.Write(ver[:])

	// String table first so decoders can resolve indices in a single forward pass.
	var hdr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(hdr[:], uint64(len(e.strings)))
	out.Write(hdr[:n])
	for _, s := range e.strings {
		n = binary.PutUvarint(hdr[:], uint64(len(s)))
		out.Write(hdr[:n])
		out.WriteString(s)
	}
	out.Write(e.buf.Bytes())
	return out.Bytes()
}

type decoder struct {
	data    []byte
	pos     int
	strings []string
}

func newDecoder(raw []byte, expectedKind byte) (*decoder, error) {
	if len(raw) < len(magicHeader)+1+4 {
		return nil, ErrCorrupt
	}
	if string(raw[:len(magicHeader)]) != magicHeader {
		return nil, ErrInvalidMagic
	}
	if raw[len(magicHeader)] != expectedKind {
		return nil, fmt.Errorf("%w: got %d want %d", ErrKindMismatch, raw[len(magicHeader)], expectedKind)
	}
	ver := binary.LittleEndian.Uint32(raw[len(magicHeader)+1 : len(magicHeader)+5])
	if ver != formatVersion {
		return nil, fmt.Errorf("%w: version %d", ErrUnsupportedVer, ver)
	}
	d := &decoder{data: raw, pos: len(magicHeader) + 5}
	strCount, err := d.readUvarint()
	if err != nil || strCount > uint64(len(raw)) {
		return nil, ErrCorrupt
	}
	d.strings = make([]string, strCount)
	for i := uint64(0); i < strCount; i++ {
		slen, err := d.readUvarint()
		if err != nil || slen > uint64(len(d.data)-d.pos) {
			return nil, ErrCorrupt
		}
		d.strings[i] = string(d.data[d.pos : d.pos+int(slen)])
		d.pos += int(slen)
	}
	return d, nil
}

func (d *decoder) readByte() (byte, error) {
	if d.pos >= len(d.data) {
		return 0, ErrCorrupt
	}
	b := d.data[d.pos]
	d.pos++
	return b, nil
}

func (d *decoder) readBool() (bool, error) {
	b, err := d.readByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (d *decoder) readUvarint() (uint64, error) {
	v, n := binary.Uvarint(d.data[d.pos:])
	if n <= 0 {
		return 0, ErrCorrupt
	}
	d.pos += n
	return v, nil
}

func (d *decoder) readVarint() (int64, error) {
	v, n := binary.Varint(d.data[d.pos:])
	if n <= 0 {
		return 0, ErrCorrupt
	}
	d.pos += n
	return v, nil
}

func (d *decoder) readFloat64() (float64, error) {
	if len(d.data)-d.pos < 8 {
		return 0, ErrCorrupt
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(d.data[d.pos : d.pos+8]))
	d.pos += 8
	return v, nil
}

func (d *decoder) readStr() (string, error) {
	idx, err := d.readUvarint()
	if err != nil || idx >= uint64(len(d.strings)) {
		return "", ErrCorrupt
	}
	return d.strings[idx], nil
}

func (d *decoder) readStringSlice() ([]string, error) {
	n, err := d.readUvarint()
	if err != nil || n > uint64(len(d.data)-d.pos) {
		return nil, ErrCorrupt
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]string, n)
	for i := uint64(0); i < n; i++ {
		s, err := d.readStr()
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}

func (d *decoder) readBytes() ([]byte, error) {
	n, err := d.readUvarint()
	if err != nil || n > uint64(len(d.data)-d.pos) {
		return nil, ErrCorrupt
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]byte, n)
	copy(out, d.data[d.pos:d.pos+int(n)])
	d.pos += int(n)
	return out, nil
}

func readAll(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, ErrCorrupt
	}
	return io.ReadAll(r)
}
