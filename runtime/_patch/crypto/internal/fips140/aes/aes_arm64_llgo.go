//go:build llgo && arm64 && !baremetal && go1.27 && !go1.28

package aes

import (
	"crypto/internal/impl"
	"internal/cpu"
	_ "unsafe"
)

var supportsAES = cpu.ARM64.HasAES

// Use a variable initializer: a patch init declaration would replace every
// upstream init, including cast.go's AES-CBC self-test.
var llgoAESRegistration = llgoRegisterAES()

func llgoRegisterAES() bool {
	impl.Register("aes", "Armv8.0", &supportsAES)
	return true
}

// Keep the generic big-endian uint32 key schedule so implementation selection
// can switch between hardware and portable code with the same expanded key.
func newBlock(c *Block, key []byte) *Block {
	if !supportsAES {
		newBlockExpanded(&c.blockExpanded, key)
		return c
	}
	c.rounds = len(key)/4 + 6
	llgoExpandKey(&key[0], uint32(len(key)), &c.enc[0], &c.dec[0])
	return c
}

func encryptBlock(c *Block, dst, src []byte) {
	if !supportsAES {
		encryptBlockGeneric(&c.blockExpanded, dst, src)
		return
	}
	llgoEncrypt(&c.enc[0], uint32(c.rounds), &dst[0], &src[0])
}

func decryptBlock(c *Block, dst, src []byte) {
	if !supportsAES {
		decryptBlockGeneric(&c.blockExpanded, dst, src)
		return
	}
	llgoDecrypt(&c.dec[0], uint32(c.rounds), &dst[0], &src[0])
}

//go:noescape
//go:linkname llgoExpandKey C.llgo_aes_expand_arm64
func llgoExpandKey(key *byte, size uint32, enc, dec *uint32)

//go:noescape
//go:linkname llgoEncrypt C.llgo_aes_encrypt_arm64
func llgoEncrypt(key *uint32, rounds uint32, dst, src *byte)

//go:noescape
//go:linkname llgoDecrypt C.llgo_aes_decrypt_arm64
func llgoDecrypt(key *uint32, rounds uint32, dst, src *byte)

//go:noescape
//go:linkname llgoCTR C.llgo_aes_ctr_arm64
func llgoCTR(key *uint32, rounds uint32, dst, src *byte, lo, hi uint64, blocks uint32)

func ctrBlocks1(b *Block, dst, src *[BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], uint32(b.rounds), &dst[0], &src[0], lo, hi, 1)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}

func ctrBlocks2(b *Block, dst, src *[2 * BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], uint32(b.rounds), &dst[0], &src[0], lo, hi, 2)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}

func ctrBlocks4(b *Block, dst, src *[4 * BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], uint32(b.rounds), &dst[0], &src[0], lo, hi, 4)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}

func ctrBlocks8(b *Block, dst, src *[8 * BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], uint32(b.rounds), &dst[0], &src[0], lo, hi, 8)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}
