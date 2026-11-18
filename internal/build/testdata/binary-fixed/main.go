package main

import (
	"encoding/binary"
	"math"
	"slices"
)

type number int16
type truth bool
type piece struct {
	OK truth
	_  uint8
	N  number
}
type packet struct {
	Kind   uint16
	Pieces [2]piece
	Float  float32
	Pair   complex64
	Long   complex128
}

func check(ok bool) {
	if !ok {
		panic("incorrect fixed binary encoding")
	}
}

func main() {
	in := packet{0x1245, [2]piece{{OK: true, N: -2}, {N: 3}}, 1.5, 2 - 4i, -16 + 32i}
	const size = 38
	check(binary.Size(in) == size && binary.Size(&in) == size)
	var nilPacket *packet
	check(binary.Size(nilPacket) == -1)
	buf := make([]byte, size+1)
	for i := range buf {
		buf[i] = 0xa5
	}
	n, err := binary.Encode(buf, binary.BigEndian, in)
	check(n == size && err == nil)
	check(slices.Equal(buf[:18], []byte{0x12, 0x45, 1, 0, 0xff, 0xfe, 0, 0, 0, 3, 0x3f, 0xc0, 0, 0, 0x40, 0, 0, 0}))
	check(buf[size] == 0xa5)
	// Blank fields consume input and retain no value.
	buf[3], buf[7] = 0xa5, 0xff
	var out packet
	n, err = binary.Decode(buf, binary.BigEndian, &out)
	check(n == size && err == nil && out == in)
	check(binary.Size([]packet{in, out}) == 2*size)
	list := []packet{in, out}
	copyBuffer := make([]byte, 2*size)
	n, err = binary.Encode(copyBuffer, binary.LittleEndian, &list)
	check(n == 2*size && err == nil && copyBuffer[0] == 0x45 && copyBuffer[1] == 0x12)
	decoded := make([]packet, 2)
	n, err = binary.Decode(copyBuffer, binary.LittleEndian, decoded)
	check(n == 2*size && err == nil && slices.Equal(list, decoded))
	n, err = binary.Encode(buf[:1], binary.NativeEndian, in)
	m, other := binary.Decode(buf[:1], binary.NativeEndian, &out)
	check(n == 0 && m == 0 && err != nil && err == other && err.Error() == "buffer too small")
	check(out == in && buf[0] == 0x12)
	zero := [0]packet{}
	n, err = binary.Encode(nil, binary.LittleEndian, zero)
	m, other = binary.Decode(nil, binary.LittleEndian, &zero)
	check(n == 0 && m == 0 && err == nil && other == nil)
	testFloatingBits()
	println("fixed binary codec ok")
}

func testFloatingBits() {
	type floats struct {
		A float32
		B complex64
		C float64
	}
	in := floats{math.Float32frombits(0x7fa00001), complex(1, math.Float32frombits(0x7fa00001)), math.Float64frombits(0x7ff4000000000001)}
	var buf [20]byte
	n, err := binary.Encode(buf[:], binary.LittleEndian, in)
	check(n == len(buf) && err == nil)
	// The composite codec widens to float64, quieting signaling NaNs.
	check(binary.LittleEndian.Uint32(buf[0:4]) == 0x7fe00001)
	check(binary.LittleEndian.Uint32(buf[8:12]) == 0x7fe00001)
	check(binary.LittleEndian.Uint64(buf[12:]) == math.Float64bits(in.C))
	binary.LittleEndian.PutUint32(buf[0:4], 0x7fa00001)
	var out floats
	n, err = binary.Decode(buf[:], binary.LittleEndian, &out)
	check(n == len(buf) && err == nil)
	check(math.Float32bits(out.A) == 0x7fe00001)
	check(math.Float64bits(out.C) == math.Float64bits(in.C))
}
