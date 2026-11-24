// In-package SHA-1 and base64 for the WebSocket handshake.
//
// History and current rationale: when this package was written, the
// standard library was unlinkable on the esp32s31-idf target - every one of
// crypto/sha1, encoding/base64, bufio, io, bytes and strings transitively
// reaches internal/reflectlite, which llgo maps to its own reflect layer,
// which dragged in libffi C glue that did not preprocess for this target
// (fixed since: the glue is freestanding-header-only and the FIPS entropy
// scratch buffer is patched to an embedded-friendly size). The package
// keeps its own primitives anyway, for image-size and dependency reasons:
// pulling crypto/sha1 through the stdlib reflect machinery costs megabytes
// of image for a 20-byte digest, while these implementations are
// allocation-light (SHA1Sum allocates nothing, the base64 encoder one
// string) and table-free, so they sit comfortably in the few-KiB budget a
// pkg/esp program has for the whole WebSocket stack. The WS handshake
// cannot exist without SHA-1 and base64 (RFC 6455 section 4.2.1 defines
// the accept key as base64(SHA-1(key + GUID))), so this file carries the
// two primitives it needs, implementing exactly the required profile and
// nothing more:
//
//   - SHA1Sum: the full RFC 3174 digest over one buffer. Exported so that
//     a pkg/esp program on a lean target has a dependency-free SHA-1.
//   - Base64StdEncode / Base64StdDecode: RFC 4648 standard alphabet
//     (+/), with padding on encode and strict padding checks on decode.
package ws

// SHA1Sum returns the RFC 3174 SHA-1 digest of data. Allocation-free.
//
// The WebSocket accept key is defined in terms of SHA-1, so the protocol
// layer needs it regardless; carrying it here keeps the package
// dependency-free (see the file comment for the image-size rationale).
func SHA1Sum(data []byte) [20]byte {
	h0, h1, h2, h3, h4 := uint32(0x67452301), uint32(0xEFCDAB89),
		uint32(0x98BADCFE), uint32(0x10325476), uint32(0xC3D2E1F0)
	bits := uint64(len(data)) * 8

	var block [64]byte
	for len(data) >= 64 {
		h0, h1, h2, h3, h4 = sha1Block(h0, h1, h2, h3, h4, data[:64])
		data = data[64:]
	}
	rem := copy(block[:], data)
	block[rem] = 0x80
	if rem >= 56 {
		// The 0x80 and the padding fill this block; the length needs a
		// second, all-zero-except-length block.
		h0, h1, h2, h3, h4 = sha1Block(h0, h1, h2, h3, h4, block[:])
		for i := range block {
			block[i] = 0
		}
	}
	putUint64be(block[56:64], bits)
	h0, h1, h2, h3, h4 = sha1Block(h0, h1, h2, h3, h4, block[:])

	var digest [20]byte
	putUint32be(digest[0:4], h0)
	putUint32be(digest[4:8], h1)
	putUint32be(digest[8:12], h2)
	putUint32be(digest[12:16], h3)
	putUint32be(digest[16:20], h4)
	return digest
}

// sha1Block compresses one 64-byte block into the five state words and
// returns the new state.
func sha1Block(h0, h1, h2, h3, h4 uint32, block []byte) (uint32, uint32, uint32, uint32, uint32) {
	var w [80]uint32
	for i := 0; i < 16; i++ {
		w[i] = uint32(block[i*4])<<24 | uint32(block[i*4+1])<<16 |
			uint32(block[i*4+2])<<8 | uint32(block[i*4+3])
	}
	for i := 16; i < 80; i++ {
		x := w[i-3] ^ w[i-8] ^ w[i-14] ^ w[i-16]
		w[i] = x<<1 | x>>31
	}
	a, b, c, d, e := h0, h1, h2, h3, h4
	for i := 0; i < 80; i++ {
		var f, k uint32
		switch {
		case i < 20:
			f = (b & c) | (^b & d)
			k = 0x5A827999
		case i < 40:
			f = b ^ c ^ d
			k = 0x6ED9EBA1
		case i < 60:
			f = (b & c) | (b & d) | (c & d)
			k = 0x8F1BBCDC
		default:
			f = b ^ c ^ d
			k = 0xCA62C1D6
		}
		tmp := a<<5 | a>>27
		tmp += f + e + k + w[i]
		a, b, c, d, e = tmp, a, b<<30|b>>2, c, d
	}
	return h0 + a, h1 + b, h2 + c, h3 + d, h4 + e
}

func putUint32be(p []byte, v uint32) {
	p[0] = byte(v >> 24)
	p[1] = byte(v >> 16)
	p[2] = byte(v >> 8)
	p[3] = byte(v)
}

func putUint64be(p []byte, v uint64) {
	for i := 0; i < 8; i++ {
		p[i] = byte(v >> (8 * (7 - i)))
	}
}

// b64alpha is the RFC 4648 standard ("base64") alphabet.
const b64alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// Base64StdEncode encodes src with the RFC 4648 standard alphabet (the form
// the WebSocket handshake uses for the client key and the accept value):
// 4 characters per 3 bytes, '=' padding for a 1- or 2-byte tail.
func Base64StdEncode(src []byte) string {
	n := (len(src) + 2) / 3 * 4
	dst := make([]byte, n)
	di, si := 0, 0
	for ; si+3 <= len(src); si, di = si+3, di+4 {
		v := uint32(src[si])<<16 | uint32(src[si+1])<<8 | uint32(src[si+2])
		dst[di+0] = b64alpha[v>>18&0x3f]
		dst[di+1] = b64alpha[v>>12&0x3f]
		dst[di+2] = b64alpha[v>>6&0x3f]
		dst[di+3] = b64alpha[v&0x3f]
	}
	rem := len(src) - si
	if rem == 1 {
		v := uint32(src[si]) << 16
		dst[di+0] = b64alpha[v>>18&0x3f]
		dst[di+1] = b64alpha[v>>12&0x3f]
		dst[di+2] = '='
		dst[di+3] = '='
	} else if rem == 2 {
		v := uint32(src[si])<<16 | uint32(src[si+1])<<8
		dst[di+0] = b64alpha[v>>18&0x3f]
		dst[di+1] = b64alpha[v>>12&0x3f]
		dst[di+2] = b64alpha[v>>6&0x3f]
		dst[di+3] = '='
	}
	return string(dst)
}

// Base64StdDecode decodes a standard-alphabet base64 string (padding
// required: the length must be a multiple of 4 and '=' may only close the
// final group). It returns the decoded bytes, or nil with an EINVAL error
// when the input is not well-formed: wrong length, a character outside the
// alphabet, or padding in the wrong place.
func Base64StdDecode(s string) ([]byte, error) {
	if len(s)%4 != 0 {
		return nil, errOf(einval)
	}
	pad := 0
	if len(s) > 0 {
		if s[len(s)-1] == '=' {
			pad++
		}
		if s[len(s)-2] == '=' {
			pad++
		}
		if pad > 0 && s[len(s)-pad-1] == '=' { // "x===" or "=="
			return nil, errOf(einval)
		}
	}
	dst := make([]byte, len(s)/4*3-pad)
	di := 0
	for si := 0; si < len(s); si += 4 {
		lim := si + 4
		if pad > 0 && si+4 == len(s) {
			lim -= pad
		}
		var v uint32
		n := 0
		for i := si; i < lim; i++ {
			b := b64Value(s[i])
			if b < 0 {
				return nil, errOf(einval)
			}
			v = v<<6 | uint32(b)
			n++
		}
		v <<= 6 * (4 - n) // realign to the full 24-bit group
		switch n {
		case 4:
			dst[di+0] = byte(v >> 16)
			dst[di+1] = byte(v >> 8)
			dst[di+2] = byte(v)
			di += 3
		case 3:
			dst[di+0] = byte(v >> 16)
			dst[di+1] = byte(v >> 8)
			di += 2
		case 2:
			dst[di+0] = byte(v >> 16)
			di++
		default: // n < 2: a "==" group carries no bytes
			return nil, errOf(einval)
		}
	}
	return dst, nil
}

// b64Value maps one base64 character to its 6-bit value, or -1.
func b64Value(c byte) int32 {
	switch {
	case c >= 'A' && c <= 'Z':
		return int32(c - 'A')
	case c >= 'a' && c <= 'z':
		return int32(c-'a') + 26
	case c >= '0' && c <= '9':
		return int32(c-'0') + 52
	case c == '+':
		return 62
	case c == '/':
		return 63
	}
	return -1
}
