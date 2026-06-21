package main

import "encoding/binary"

func main() {
	var order binary.ByteOrder = binary.LittleEndian
	_ = order.Uint16([]byte{1, 2})
}
