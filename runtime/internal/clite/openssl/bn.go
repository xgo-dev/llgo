/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package openssl

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

type BN_ULONG = uint64

// -----------------------------------------------------------------------------

type BN_CTX struct {
	Unused [0]byte
}

// BN_CTX *BN_CTX_new(void);
//
//go:linkname BN_CTXNew C.BN_CTX_new
func BN_CTXNew() *BN_CTX

// BN_CTX *BN_CTX_secure_new(void);
//
//go:linkname BN_CTXSecureNew C.BN_CTX_secure_new
func BN_CTXSecureNew() *BN_CTX

// BN_CTX *BN_CTX_new_ex(OSSL_LIB_CTX *ctx);
// BN_CTX *BN_CTX_secure_new_ex(OSSL_LIB_CTX *ctx);

// void BN_CTX_free(BN_CTX *c);
//
// llgo:link (*BN_CTX).Free C.BN_CTX_free
func (*BN_CTX) Free() {}

// void BN_CTX_start(BN_CTX *ctx);
//
// llgo:link (*BN_CTX).Start C.BN_CTX_start
func (*BN_CTX) Start() {}

// BIGNUM *BN_CTX_get(BN_CTX *ctx);
//
// llgo:link (*BN_CTX).Get C.BN_CTX_get
func (*BN_CTX) Get() *BIGNUM { return nil }

// void BN_CTX_end(BN_CTX *ctx);
//
// llgo:link (*BN_CTX).End C.BN_CTX_end
func (*BN_CTX) End() {}

// -----------------------------------------------------------------------------

type BIGNUM struct {
	Unused [0]byte
}

// BIGNUM *BN_new(void);
//
//go:linkname BNNew C.BN_new
func BNNew() *BIGNUM

// BIGNUM *BN_secure_new(void);
//
//go:linkname BNSecureNew C.BN_secure_new
func BNSecureNew() *BIGNUM

// void BN_free(BIGNUM *a);
//
// llgo:link (*BIGNUM).Free C.BN_free
func (*BIGNUM) Free() {}

// void BN_clear_free(BIGNUM *a);
//
// llgo:link (*BIGNUM).ClearFree C.BN_clear_free
func (*BIGNUM) ClearFree() {}

// void BN_clear(BIGNUM *a);
//
// llgo:link (*BIGNUM).Clear C.BN_clear
func (*BIGNUM) Clear() {}

// BIGNUM *BN_dup(const BIGNUM *a);
//
// llgo:link (*BIGNUM).Dup C.BN_dup
func (*BIGNUM) Dup() *BIGNUM { return nil }

// BIGNUM *BN_copy(BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Copy C.BN_copy
func (*BIGNUM) Copy(b *BIGNUM) *BIGNUM { return nil }

// void BN_swap(BIGNUM *a, BIGNUM *b);
//
// llgo:link (*BIGNUM).Swap C.BN_swap
func (*BIGNUM) Swap(b *BIGNUM) {}

// int BN_is_zero(const BIGNUM *a);
//
// llgo:link (*BIGNUM).IsZero C.BN_is_zero
func (*BIGNUM) IsZero() c.Int { return 0 }

// void BN_zero_ex(BIGNUM *a);
//
// llgo:link (*BIGNUM).SetZero C.BN_zero_ex
func (*BIGNUM) SetZero() {}

// int BN_is_one(const BIGNUM *a);
//
// llgo:link (*BIGNUM).IsOne C.BN_is_one
func (*BIGNUM) IsOne() c.Int { return 0 }

// int BN_is_odd(const BIGNUM *a);
//
// llgo:link (*BIGNUM).IsOdd C.BN_is_odd
func (*BIGNUM) IsOdd() c.Int { return 0 }

// int BN_abs_is_word(const BIGNUM *a, const BN_ULONG w);
//
// llgo:link (*BIGNUM).AbsIsWord C.BN_abs_is_word
func (*BIGNUM) AbsIsWord(w BN_ULONG) c.Int { return 0 }

// int BN_is_word(const BIGNUM *a, const BN_ULONG w);
//
// llgo:link (*BIGNUM).IsWord C.BN_is_word
func (*BIGNUM) IsWord(w BN_ULONG) c.Int { return 0 }

// int BN_set_word(BIGNUM *a, BN_ULONG w);
//
// llgo:link (*BIGNUM).SetWord C.BN_set_word
func (*BIGNUM) SetWord(w BN_ULONG) c.Int { return 0 }

// BN_ULONG BN_get_word(const BIGNUM *a);
//
// llgo:link (*BIGNUM).GetWord C.BN_get_word
func (*BIGNUM) GetWord() BN_ULONG { return 0 }

// BN_ULONG BN_mod_word(const BIGNUM *a, BN_ULONG w);
//
// llgo:link (*BIGNUM).ModWord C.BN_mod_word
func (*BIGNUM) ModWord(w BN_ULONG) BN_ULONG { return 0 }

// BN_ULONG BN_div_word(BIGNUM *a, BN_ULONG w);
//
// llgo:link (*BIGNUM).DivWord C.BN_div_word
func (*BIGNUM) DivWord(w BN_ULONG) BN_ULONG { return 0 }

// int BN_mul_word(BIGNUM *a, BN_ULONG w);
//
// llgo:link (*BIGNUM).MulWord C.BN_mul_word
func (*BIGNUM) MulWord(w BN_ULONG) c.Int { return 0 }

// int BN_add_word(BIGNUM *a, BN_ULONG w);
//
// llgo:link (*BIGNUM).AddWord C.BN_add_word
func (*BIGNUM) AddWord(w BN_ULONG) c.Int { return 0 }

// int BN_sub_word(BIGNUM *a, BN_ULONG w);
//
// llgo:link (*BIGNUM).SubWord C.BN_sub_word
func (*BIGNUM) SubWord(w BN_ULONG) c.Int { return 0 }

// char *BN_bn2hex(const BIGNUM *a);
//
// llgo:link (*BIGNUM).Bn2hex C.BN_bn2hex
func (*BIGNUM) Bn2hex() *c.Char { return nil }

// char *BN_bn2dec(const BIGNUM *a);
//
// llgo:link (*BIGNUM).Bn2dec C.BN_bn2dec
func (*BIGNUM) Bn2dec() *c.Char { return nil }

// llgo:link (*BIGNUM).CStr C.BN_bn2dec
func (*BIGNUM) CStr() *c.Char { return nil }

// int BN_hex2bn(BIGNUM **a, const char *str);
//
//go:linkname BNHex2bn C.BN_hex2bn
func BNHex2bn(a **BIGNUM, str *c.Char) c.Int

// int BN_dec2bn(BIGNUM **a, const char *str);
//
//go:linkname BNDec2bn C.BN_dec2bn
func BNDec2bn(a **BIGNUM, str *c.Char) c.Int

// int BN_asc2bn(BIGNUM **a, const char *str);
//
//go:linkname BNAsc2bn C.BN_asc2bn
func BNAsc2bn(a **BIGNUM, str *c.Char) c.Int

// BIGNUM *BN_bin2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNBin2bn C.BN_bin2bn
func BNBin2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// BIGNUM *BN_signed_bin2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNSignedBin2bn C.BN_signed_bin2bn
func BNSignedBin2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// int BN_bn2bin(const BIGNUM *a, unsigned char *to);
//
// llgo:link (*BIGNUM).Bn2bin C.BN_bn2bin
func (bn *BIGNUM) Bn2bin(to *byte) c.Int { return 0 }

// int BN_bn2binpad(const BIGNUM *a, unsigned char *to, int tolen);
//
// llgo:link (*BIGNUM).Bn2binpad C.BN_bn2binpad
func (bn *BIGNUM) Bn2binpad(to *byte, tolen c.Int) c.Int { return 0 }

// int BN_signed_bn2bin(const BIGNUM *a, unsigned char *to, int tolen);
//
// llgo:link (*BIGNUM).SignedBn2bin C.BN_signed_bn2bin
func (bn *BIGNUM) SignedBn2bin(to *byte, tolen c.Int) c.Int { return 0 }

// BIGNUM *BN_lebin2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNLebin2bn C.BN_lebin2bn
func BNLebin2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// BIGNUM *BN_signed_lebin2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNSignedLebin2bn C.BN_signed_lebin2bn
func BNSignedLebin2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// int BN_bn2lebinpad(const BIGNUM *a, unsigned char *to, int tolen);
//
// llgo:link (*BIGNUM).Bn2lebinpad C.BN_bn2lebinpad
func (bn *BIGNUM) Bn2lebinpad(to *byte, tolen c.Int) c.Int { return 0 }

// int BN_signed_bn2lebin(const BIGNUM *a, unsigned char *to, int tolen);
//
// llgo:link (*BIGNUM).SignedBn2lebin C.BN_signed_bn2lebin
func (bn *BIGNUM) SignedBn2lebin(to *byte, tolen c.Int) c.Int { return 0 }

// BIGNUM *BN_native2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNNative2bn C.BN_native2bn
func BNNative2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// BIGNUM *BN_signed_native2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNSignedNative2bn C.BN_signed_native2bn
func BNSignedNative2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// int BN_bn2nativepad(const BIGNUM *a, unsigned char *to, int tolen);
//
// llgo:link (*BIGNUM).Bn2nativepad C.BN_bn2nativepad
func (bn *BIGNUM) Bn2nativepad(to *byte, tolen c.Int) c.Int { return 0 }

// int BN_signed_bn2native(const BIGNUM *a, unsigned char *to, int tolen);
//
// llgo:link (*BIGNUM).SignedBn2native C.BN_signed_bn2native
func (bn *BIGNUM) SignedBn2native(to *byte, tolen c.Int) c.Int { return 0 }

// BIGNUM *BN_mpi2bn(const unsigned char *s, int len, BIGNUM *ret);
//
//go:linkname BNMpi2bn C.BN_mpi2bn
func BNMpi2bn(s *byte, len c.Int, ret *BIGNUM) *BIGNUM

// int BN_bn2mpi(const BIGNUM *a, unsigned char *to);
//
// llgo:link (*BIGNUM).Bn2mpi C.BN_bn2mpi
func (bn *BIGNUM) Bn2mpi(to *byte) c.Int { return 0 }

// int BN_sub(BIGNUM *r, const BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Sub C.BN_sub
func (*BIGNUM) Sub(a, b *BIGNUM) c.Int { return 0 }

// int BN_add(BIGNUM *r, const BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Add C.BN_add
func (*BIGNUM) Add(a, b *BIGNUM) c.Int { return 0 }

// int BN_usub(BIGNUM *r, const BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Usub C.BN_usub
func (*BIGNUM) Usub(a, b *BIGNUM) c.Int { return 0 }

// int BN_uadd(BIGNUM *r, const BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Uadd C.BN_uadd
func (*BIGNUM) Uadd(a, b *BIGNUM) c.Int { return 0 }

// int BN_mul(BIGNUM *r, const BIGNUM *a, const BIGNUM *b, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).Mul C.BN_mul
func (*BIGNUM) Mul(r, a, b *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// int BN_sqr(BIGNUM *r, const BIGNUM *a, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).Sqr C.BN_sqr
func (*BIGNUM) Sqr(r, a *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

/** BN_set_negative sets sign of a BIGNUM
 * \param  b  pointer to the BIGNUM object
 * \param  n  0 if the BIGNUM b should be positive and a value != 0 otherwise
 */
// void BN_set_negative(BIGNUM *b, int n);
//
// llgo:link (*BIGNUM).SetNegative C.BN_set_negative
func (*BIGNUM) SetNegative(n c.Int) {}

/** BN_is_negative returns 1 if the BIGNUM is negative
 * \param  b  pointer to the BIGNUM object
 * \return 1 if a < 0 and 0 otherwise
 */
// int BN_is_negative(const BIGNUM *b);
//
// llgo:link (*BIGNUM).IsNegative C.BN_is_negative
func (*BIGNUM) IsNegative() c.Int { return 0 }

// int BN_div(BIGNUM *dv, BIGNUM *rem, const BIGNUM *m, const BIGNUM *d, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).Div C.BN_div
func (*BIGNUM) Div(rem, m, d *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// int BN_nnmod(BIGNUM *r, const BIGNUM *m, const BIGNUM *d, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).Nnmod C.BN_nnmod
func (*BIGNUM) Nnmod(r, m, d *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// int BN_cmp(const BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Cmp C.BN_cmp
func (*BIGNUM) Cmp(b *BIGNUM) c.Int { return 0 }

// int BN_ucmp(const BIGNUM *a, const BIGNUM *b);
//
// llgo:link (*BIGNUM).Ucmp C.BN_ucmp
func (*BIGNUM) Ucmp(b *BIGNUM) c.Int { return 0 }

// int BN_is_bit_set(const BIGNUM *a, int n);
//
// llgo:link (*BIGNUM).IsBitSet C.BN_is_bit_set
func (*BIGNUM) IsBitSet(n c.Int) c.Int { return 0 }

// int BN_set_bit(BIGNUM *a, int n);
//
// llgo:link (*BIGNUM).SetBit C.BN_set_bit
func (*BIGNUM) SetBit(n c.Int) c.Int { return 0 }

// int BN_clear_bit(BIGNUM *a, int n);
//
// llgo:link (*BIGNUM).ClearBit C.BN_clear_bit
func (*BIGNUM) ClearBit(n c.Int) c.Int { return 0 }

// int BN_lshift(BIGNUM *r, const BIGNUM *a, int n);
//
// llgo:link (*BIGNUM).Lshift C.BN_lshift
func (*BIGNUM) Lshift(a *BIGNUM, n c.Int) c.Int { return 0 }

// int BN_lshift1(BIGNUM *r, const BIGNUM *a);
//
// llgo:link (*BIGNUM).Lshift1 C.BN_lshift1
func (*BIGNUM) Lshift1(a *BIGNUM) c.Int { return 0 }

// int BN_rshift(BIGNUM *r, const BIGNUM *a, int n);
//
// llgo:link (*BIGNUM).Rshift C.BN_rshift
func (*BIGNUM) Rshift(a *BIGNUM, n c.Int) c.Int { return 0 }

// int BN_rshift1(BIGNUM *r, const BIGNUM *a);
//
// llgo:link (*BIGNUM).Rshift1 C.BN_rshift1
func (*BIGNUM) Rshift1(a *BIGNUM) c.Int { return 0 }

// int BN_exp(BIGNUM *r, const BIGNUM *a, const BIGNUM *p, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).Exp C.BN_exp
func (*BIGNUM) Exp(a, p *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// int BN_mod_exp(BIGNUM *r, const BIGNUM *a, const BIGNUM *p, const BIGNUM *m, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).ModExp C.BN_mod_exp
func (*BIGNUM) ModExp(a, p, m *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// int BN_gcd(BIGNUM *r, const BIGNUM *a, const BIGNUM *b, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).Gcd C.BN_gcd
func (*BIGNUM) Gcd(a, b *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// int BN_are_coprime(const BIGNUM *a, const BIGNUM *b, BN_CTX *ctx);
//
// llgo:link (*BIGNUM).AreCoprime C.BN_are_coprime
func (*BIGNUM) AreCoprime(b *BIGNUM, ctx *BN_CTX) c.Int { return 0 }

// -----------------------------------------------------------------------------

type BN_GENCB struct {
	Unused [0]byte
}

// -----------------------------------------------------------------------------
