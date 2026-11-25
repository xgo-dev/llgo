package crc16

import "hash"

// This file contains the CRC16 implementation of the
// go standard library hash.Hash interface

type Hash16 interface {
	hash.Hash
	Sum16() uint16
}

type digest struct {
	sum uint16
	t   *Table
}

// Write adds more data to the running digest.
// It never returns an error.
func (h *digest) Write(data []byte) (int, error) {
	h.sum = Update(h.sum, data, h.t)
	return len(data), nil
}

// Sum appends the current digest (leftmost byte first, big-endian)
// to b and returns the resulting slice.
// It does not change the underlying digest state.
func (h digest) Sum(b []byte) []byte {
	s := h.Sum16()
	return append(b, byte(s>>8), byte(s))
}

// Reset resets the Hash to its initial state.
func (h *digest) Reset() {
	h.sum = h.t.params.Init
}

// Size returns the number of bytes Sum will return.
func (h digest) Size() int {
	return 2
}

// BlockSize returns the undelying block size.
// See digest.Hash.BlockSize
func (h digest) BlockSize() int {
	return 1
}

// Sum16 returns the CRC16 checksum.
func (h digest) Sum16() uint16 {
	return Complete(h.sum, h.t)
}

// New creates a new CRC16 digest for the given table.
func New(t *Table) Hash16 {
	h := digest{t: t}
	h.Reset()
	return &h
}
