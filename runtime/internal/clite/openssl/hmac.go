package openssl

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	EVP_MAX_MD_SIZE = 64 /* longest known is SHA512 */
)

// -----------------------------------------------------------------------------

type EVP_MD struct {
	Unused [0]byte
}

// const EVP_MD *EVP_sha1(void)
//
//go:linkname EVP_sha1 C.EVP_sha1
func EVP_sha1() *EVP_MD

// const EVP_MD *EVP_sha224(void)
//
//go:linkname EVP_sha224 C.EVP_sha224
func EVP_sha224() *EVP_MD

// func EVP_sha256() *EVP_MD
//
//go:linkname EVP_sha256 C.EVP_sha256
func EVP_sha256() *EVP_MD

// const EVP_MD *EVP_sha512_224(void)
//
//go:linkname EVP_sha512_224 C.EVP_sha512_224
func EVP_sha512_224() *EVP_MD

// const EVP_MD *EVP_sha512_256(void)
//
//go:linkname EVP_sha512_256 C.EVP_sha512_256
func EVP_sha512_256() *EVP_MD

// const EVP_MD *EVP_sha384(void)
//
//go:linkname EVP_sha384 C.EVP_sha384
func EVP_sha384() *EVP_MD

// const EVP_MD *EVP_sha512(void)
//
//go:linkname EVP_sha512 C.EVP_sha512
func EVP_sha512() *EVP_MD

// -----------------------------------------------------------------------------

type HMAC_CTX struct {
	Unused [0]byte
}

// OSSL_DEPRECATEDIN_3_0 HMAC_CTX *HMAC_CTX_new(void);
//
//go:linkname NewHMAC_CTX C.HMAC_CTX_new
func NewHMAC_CTX() *HMAC_CTX

// OSSL_DEPRECATEDIN_3_0 void HMAC_CTX_free(HMAC_CTX *ctx);
//
// llgo:link (*HMAC_CTX).Free C.HMAC_CTX_free
func (ctx *HMAC_CTX) Free() {}

// OSSL_DEPRECATEDIN_3_0 size_t HMAC_size(const HMAC_CTX *e);
//
// llgo:link (*HMAC_CTX).Size C.HMAC_size
func (ctx *HMAC_CTX) Size() uintptr { return 0 }

// OSSL_DEPRECATEDIN_3_0 int HMAC_CTX_reset(HMAC_CTX *ctx);
//
// llgo:link (*HMAC_CTX).Reset C.HMAC_CTX_reset
func (ctx *HMAC_CTX) Reset() c.Int { return 0 }

// OSSL_DEPRECATEDIN_1_1_0 __owur int HMAC_Init(HMAC_CTX *ctx,
// const void *key, int len,
// const EVP_MD *md);
//
// llgo:link (*HMAC_CTX).Init C.HMAC_Init
func (ctx *HMAC_CTX) Init(key unsafe.Pointer, len c.Int, md *EVP_MD) c.Int { return 0 }

func (ctx *HMAC_CTX) InitBytes(key []byte, md *EVP_MD) c.Int {
	return ctx.Init(unsafe.Pointer(unsafe.SliceData(key)), c.Int(len(key)), md)
}

func (ctx *HMAC_CTX) InitString(key string, md *EVP_MD) c.Int {
	return ctx.Init(unsafe.Pointer(unsafe.StringData(key)), c.Int(len(key)), md)
}

// OSSL_DEPRECATEDIN_3_0 int HMAC_Init_ex(HMAC_CTX *ctx, const void *key, int len,
// const EVP_MD *md, ENGINE *impl);
//
// llgo:link (*HMAC_CTX).InitEx C.HMAC_Init_ex
func (ctx *HMAC_CTX) InitEx(key unsafe.Pointer, len c.Int, md *EVP_MD, impl unsafe.Pointer) c.Int {
	return 0
}

// OSSL_DEPRECATEDIN_3_0 int HMAC_Update(HMAC_CTX *ctx, const unsigned char *data,
// size_t len);
//
// llgo:link (*HMAC_CTX).Update C.HMAC_Update
func (ctx *HMAC_CTX) Update(data unsafe.Pointer, len uintptr) c.Int { return 0 }

func (ctx *HMAC_CTX) UpdateBytes(data []byte) c.Int {
	return ctx.Update(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)))
}

func (ctx *HMAC_CTX) UpdateString(data string) c.Int {
	return ctx.Update(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)))
}

// OSSL_DEPRECATEDIN_3_0 int HMAC_Final(HMAC_CTX *ctx, unsigned char *md,
// unsigned int *len);
//
// llgo:link (*HMAC_CTX).Final C.HMAC_Final
func (ctx *HMAC_CTX) Final(md *byte, len *c.Uint) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 __owur int HMAC_CTX_copy(HMAC_CTX *dctx, HMAC_CTX *sctx);
//
// llgo:link (*HMAC_CTX).Copy C.HMAC_CTX_copy
func (ctx *HMAC_CTX) Copy(sctx *HMAC_CTX) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 void HMAC_CTX_set_flags(HMAC_CTX *ctx, unsigned long flags);
//
// llgo:link (*HMAC_CTX).SetFlags C.HMAC_CTX_set_flags
func (ctx *HMAC_CTX) SetFlags(flags c.Ulong) {}

// -----------------------------------------------------------------------------
