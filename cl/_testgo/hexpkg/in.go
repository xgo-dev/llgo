// Behavior regression fixture: golden output must stay identical on hosted
// and on the RV32 baremetal XIP flash target. The image only fits the flash
// (XIP) layout; SRAM and the 256 KiB ESP32-C3 IRAM cannot hold the pulled-in
// fmt dependency.
package main

import "encoding/hex"

func main() {
	s := hex.EncodeToString([]byte{0xde, 0xad, 0xbe, 0xef})
	println("enc", s)
	b, err := hex.DecodeString("48656c6c6f")
	println("dec", string(b), err == nil)
	println("lens", hex.EncodedLen(3), hex.DecodedLen(8))
	bad, err := hex.DecodeString("zz")
	println("bad", len(bad), err != nil)
	println("dumplen", len(hex.Dump([]byte("abc"))))
}
