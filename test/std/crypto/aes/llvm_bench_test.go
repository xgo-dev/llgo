package aes_test

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func BenchmarkAES256CTR16K(b *testing.B) {
	block, err := aes.NewCipher(make([]byte, 32))
	if err != nil {
		b.Fatal(err)
	}
	stream := cipher.NewCTR(block, make([]byte, 16))
	buf := make([]byte, 16<<10)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.XORKeyStream(buf, buf)
	}
}
