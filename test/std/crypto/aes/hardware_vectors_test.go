package aes_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"
)

func TestFIPS197AllKeySizes(t *testing.T) {
	plain, _ := hex.DecodeString("00112233445566778899aabbccddeeff")
	for _, tc := range []struct{ key, ciphertext string }{
		{"000102030405060708090a0b0c0d0e0f", "69c4e0d86a7b0430d8cdb78070b4c55a"},
		{"000102030405060708090a0b0c0d0e0f1011121314151617", "dda97ca4864cdfe06eaf70a0ec0d7191"},
		{"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "8ea2b7ca516745bfeafc49904b496089"},
	} {
		key, _ := hex.DecodeString(tc.key)
		want, _ := hex.DecodeString(tc.ciphertext)
		block, err := aes.NewCipher(key)
		if err != nil {
			t.Fatal(err)
		}
		// Unaligned, in-place encryption and decryption.
		buf := make([]byte, 17)
		copy(buf[1:], plain)
		block.Encrypt(buf[1:], buf[1:])
		if !bytes.Equal(buf[1:], want) {
			t.Fatalf("AES-%d Encrypt = %x, want %x", len(key)*8, buf[1:], want)
		}
		block.Decrypt(buf[1:], buf[1:])
		if !bytes.Equal(buf[1:], plain) {
			t.Fatalf("AES-%d Decrypt = %x, want %x", len(key)*8, buf[1:], plain)
		}
	}
}

// Compare the batched CTR path with the definition: encrypt independent
// big-endian 128-bit counters and XOR. Exercise all batch/tail boundaries,
// counter carry/wrap, unaligned pointers, and partial-block stream state.
func TestCTRCounterAndChunkBoundaries(t *testing.T) {
	for _, keySize := range []int{16, 24, 32} {
		key := make([]byte, keySize)
		for i := range key {
			key[i] = byte(i*13 + 7)
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, carry := range []bool{false, true} {
			iv := make([]byte, aes.BlockSize)
			if carry {
				for i := range iv {
					iv[i] = 255
				}
				iv[15] = 248
			}
			for _, size := range []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 257, 16384} {
				src := make([]byte, size)
				for i := range src {
					src[i] = byte(i*29 + 11)
				}
				want := make([]byte, size)
				counter := bytes.Clone(iv)
				var stream [aes.BlockSize]byte
				for off := 0; off < size; off += aes.BlockSize {
					block.Encrypt(stream[:], counter)
					for j := 0; j < aes.BlockSize && off+j < size; j++ {
						want[off+j] = src[off+j] ^ stream[j]
					}
					for j := len(counter) - 1; j >= 0; j-- {
						counter[j]++
						if counter[j] != 0 {
							break
						}
					}
				}
				got := make([]byte, size+3)[3:]
				cipher.NewCTR(block, iv).XORKeyStream(got, src)
				if !bytes.Equal(got, want) {
					t.Fatalf("AES-%d CTR size=%d carry=%v", keySize*8, size, carry)
				}
				copy(got, src)
				ctr := cipher.NewCTR(block, iv)
				chunks := []int{1, 15, 17, 64, 129}
				for off, part := 0, 0; off < size; part++ {
					n := chunks[part%len(chunks)]
					if n > size-off {
						n = size - off
					}
					ctr.XORKeyStream(got[off:off+n], got[off:off+n])
					off += n
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("AES-%d chunked in-place CTR size=%d carry=%v", keySize*8, size, carry)
				}
			}
		}
	}
}
