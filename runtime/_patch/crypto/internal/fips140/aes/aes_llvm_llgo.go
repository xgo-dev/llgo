//go:build llgo && (amd64 || 386 || arm64 || arm || ppc64 || ppc64le || riscv64) && !baremetal && go1.27 && !go1.28

package aes

import (
	"crypto/internal/impl"
	"internal/goarch"
	"math/bits"
	"unsafe"
)

var supportsAES = llgoAESCPU()

//go:linkname llgoAESCPUEnvironment github.com/xgo-dev/llgo/runtime/internal/runtime.CPUEnvironment
func llgoAESCPUEnvironment() string

func llgoAESCPU() bool {
	return llgoAESFeaturesEnabled(llgoAESHardware(), llgoAESCPUEnvironment())
}

func llgoAESFeaturesEnabled(hardware bool, env string) bool {
	enabled := hardware
	powerAESDisabled := false
	for env != "" {
		i := 0
		for i < len(env) && env[i] != ',' {
			i++
		}
		item := env[:i]
		switch item {
		case "cpu.aes=off", "cpu.all=off":
			enabled = false
		case "cpu.aes=on", "cpu.all=on":
			enabled = hardware
		}
		// Match upstream's POWER-only knob independently of CPU options.
		// The last value wins, and only "off" disables the implementation.
		const powerAES = "ppc64aes="
		if goarch.IsPpc64|goarch.IsPpc64le == 1 && len(item) >= len(powerAES) && item[:len(powerAES)] == powerAES {
			powerAESDisabled = item[len(powerAES):] == "off"
		}
		if i == len(env) {
			break
		}
		env = env[i+1:]
	}
	return enabled && !powerAESDisabled
}

// A variable initializer preserves every upstream init, including CAST.
var llgoAESRegistration = llgoRegisterAES()

func llgoRegisterAES() bool {
	impl.Register("aes", llgoAESName, &supportsAES)
	return true
}

// These are compiler operations, not external functions. The compiler gives
// their containing functions the required CPU features and prevents inlining
// across the runtime feature check, including with full LTO.
//
//go:noescape
//go:linkname llgoAESRound llgo.aesRound
func llgoAESRound(op int, dst, src, key *byte)

//go:linkname llgoAESSubWord llgo.aesSubWord
func llgoAESSubWord(uint32) uint32

// Buffers contain no pointers and stay within the containing kernel's frame.
//
//go:linkname llgoAESAlloca llgo.aesAlloca
func llgoAESAlloca(uintptr) unsafe.Pointer

const (
	llgoEncryptRound = iota
	llgoEncryptLast
	llgoDecryptRound
	llgoDecryptLast
	llgoInverseMixKey
	llgoXORKey
	llgoXORBlock
)

func llgoKey(key *uint32, round int) *byte {
	return (*byte)(unsafe.Add(unsafe.Pointer(key), uintptr(round)*16))
}

// All implementations share the generic big-endian uint32 schedule, allowing
// the test dispatcher to switch back to the portable implementation safely.
func newBlock(c *Block, key []byte) *Block {
	if !supportsAES {
		newBlockExpanded(&c.blockExpanded, key)
		return c
	}
	c.rounds = len(key)/4 + 6
	llgoExpandKey(&key[0], len(key), &c.enc[0], &c.dec[0])
	return c
}

func llgoExpandKey(key *byte, size int, enc, dec *uint32) {
	k := unsafe.Slice(key, size)
	e, d := unsafe.Slice(enc, 60), unsafe.Slice(dec, 60)
	nk := size / 4
	rounds := nk + 6
	words := 4 * (rounds + 1)
	for i := 0; i < nk; i++ {
		e[i] = uint32(k[4*i])<<24 | uint32(k[4*i+1])<<16 | uint32(k[4*i+2])<<8 | uint32(k[4*i+3])
	}
	rc := uint32(1)
	for i := nk; i < words; i++ {
		t := e[i-1]
		if i%nk == 0 {
			t = llgoAESSubWord(t<<8|t>>24) ^ rc<<24
			rc = rc<<1 ^ (rc>>7)*0x11b
		} else if nk == 8 && i%nk == 4 {
			t = llgoAESSubWord(t)
		}
		e[i] = e[i-nk] ^ t
	}
	for r := 0; r <= rounds; r++ {
		for j := 0; j < 4; j++ {
			d[4*r+j] = e[4*(rounds-r)+j]
		}
		if r != 0 && r != rounds {
			p := llgoKey(dec, r)
			llgoAESRound(llgoInverseMixKey, p, p, nil)
		}
	}
}

func encryptBlock(c *Block, dst, src []byte) {
	if !supportsAES {
		encryptBlockGeneric(&c.blockExpanded, dst, src)
		return
	}
	llgoCryptBlock(&c.enc[0], c.rounds, &dst[0], &src[0], false)
}

func decryptBlock(c *Block, dst, src []byte) {
	if !supportsAES {
		decryptBlockGeneric(&c.blockExpanded, dst, src)
		return
	}
	llgoCryptBlock(&c.dec[0], c.rounds, &dst[0], &src[0], true)
}

func llgoCryptBlock(key *uint32, rounds int, dst, src *byte, decrypt bool) {
	state := (*[16]byte)(llgoAESAlloca(16))
	first := 0
	if !llgoAESPreKey {
		llgoAESRound(llgoXORKey, &state[0], src, llgoKey(key, 0))
	} else {
		if decrypt {
			llgoAESRound(llgoDecryptRound, &state[0], src, llgoKey(key, 0))
		} else {
			llgoAESRound(llgoEncryptRound, &state[0], src, llgoKey(key, 0))
		}
		first = 1
	}
	for r := first; r < rounds-1; r++ {
		index := r
		if !llgoAESPreKey {
			index++
		}
		if decrypt {
			llgoAESRound(llgoDecryptRound, &state[0], &state[0], llgoKey(key, index))
		} else {
			llgoAESRound(llgoEncryptRound, &state[0], &state[0], llgoKey(key, index))
		}
	}
	last := rounds
	lastDst := dst
	if llgoAESPreKey {
		last--
		lastDst = &state[0]
	}
	if decrypt {
		llgoAESRound(llgoDecryptLast, lastDst, &state[0], llgoKey(key, last))
	} else {
		llgoAESRound(llgoEncryptLast, lastDst, &state[0], llgoKey(key, last))
	}
	if llgoAESPreKey {
		llgoAESRound(llgoXORKey, dst, &state[0], llgoKey(key, rounds))
	}
}

// The public CTR implementation supplies exactly 1, 2, 4 or 8 complete blocks.
// Public CTR rejects inexact overlap, so reading each source block immediately
// before its store supports both disjoint buffers and exact in-place use.
func llgoCTR(key *uint32, rounds int, dst, src *byte, lo, hi uint64, blocks int) {
	states := (*[8][16]byte)(llgoAESAlloca(128))
	if blocks < 1 || blocks > len(states) {
		panic("invalid AES block count")
	}
	for i := 0; i < blocks; i++ {
		low, high := lo+uint64(i), hi
		if low < lo {
			high++
		}
		if !goarch.BigEndian {
			high, low = bits.ReverseBytes64(high), bits.ReverseBytes64(low)
		}
		counter := (*[2]uint64)(unsafe.Pointer(&states[i]))
		counter[0], counter[1] = high, low
		if !llgoAESPreKey {
			llgoAESRound(llgoXORKey, &states[i][0], &states[i][0], llgoKey(key, 0))
		}
	}
	for r := 0; r < rounds-1; r++ {
		index := r
		if !llgoAESPreKey {
			index++
		}
		for i := 0; i < blocks; i++ {
			llgoAESRound(llgoEncryptRound, &states[i][0], &states[i][0], llgoKey(key, index))
		}
	}
	last := rounds
	if llgoAESPreKey {
		last--
	}
	for i := 0; i < blocks; i++ {
		llgoAESRound(llgoEncryptLast, &states[i][0], &states[i][0], llgoKey(key, last))
		if llgoAESPreKey {
			llgoAESRound(llgoXORKey, &states[i][0], &states[i][0], llgoKey(key, rounds))
		}
		llgoAESRound(llgoXORBlock, &states[i][0], &states[i][0], (*byte)(unsafe.Add(unsafe.Pointer(src), i*16)))
		copy(unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(dst), i*16)), 16), states[i][:])
	}
}

func ctrBlocks1(b *Block, dst, src *[BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], b.rounds, &dst[0], &src[0], lo, hi, 1)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}

func ctrBlocks2(b *Block, dst, src *[2 * BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], b.rounds, &dst[0], &src[0], lo, hi, 2)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}

func ctrBlocks4(b *Block, dst, src *[4 * BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], b.rounds, &dst[0], &src[0], lo, hi, 4)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}

func ctrBlocks8(b *Block, dst, src *[8 * BlockSize]byte, lo, hi uint64) {
	if supportsAES {
		llgoCTR(&b.enc[0], b.rounds, &dst[0], &src[0], lo, hi, 8)
	} else {
		ctrBlocks(b, dst[:], src[:], lo, hi)
	}
}
