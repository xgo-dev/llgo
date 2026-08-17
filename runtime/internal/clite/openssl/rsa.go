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
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// -----------------------------------------------------------------------------

type RSA_METHOD struct {
	Unused [0]byte
}

// OSSL_DEPRECATEDIN_3_0 RSA_METHOD *RSA_meth_new(const char *name, int flags);
//
//go:linkname RSAMethNew C.RSA_meth_new
func RSAMethNew(name *byte, flags c.Int) *RSA_METHOD

// OSSL_DEPRECATEDIN_3_0 void RSA_meth_free(RSA_METHOD *meth);
//
// llgo:link (*RSA_METHOD).Free C.RSA_meth_free
func (*RSA_METHOD) Free() {}

// OSSL_DEPRECATEDIN_3_0
// int RSA_meth_set_init(RSA_METHOD *rsa, int (*init) (RSA *rsa));
//
// llgo:link (*RSA_METHOD).SetInit C.RSA_meth_set_init
func (*RSA_METHOD) SetInit(init func(rsa *RSA) c.Int) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0
// int RSA_meth_set_finish(RSA_METHOD *rsa, int (*finish) (RSA *rsa));
//
// llgo:link (*RSA_METHOD).SetFinish C.RSA_meth_set_finish
func (*RSA_METHOD) SetFinish(finish func(rsa *RSA) c.Int) c.Int { return 0 }

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_mod_exp(RSA_METHOD *rsa,
                         int (*mod_exp) (BIGNUM *r0, const BIGNUM *i, RSA *rsa,
                                         BN_CTX *ctx));
*/
// llgo:link (*RSA_METHOD).SetModExp C.RSA_meth_set_mod_exp
func (*RSA_METHOD) SetModExp(modExp func(
	r0 *BIGNUM, i *BIGNUM, rsa *RSA, ctx *BN_CTX) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_bn_mod_exp(RSA_METHOD *rsa,
                            int (*bn_mod_exp) (BIGNUM *r,
                                               const BIGNUM *a,
                                               const BIGNUM *p,
                                               const BIGNUM *m,
                                               BN_CTX *ctx,
                                               BN_MONT_CTX *m_ctx));
//-llgo:link (*RSA_METHOD).SetBnModExp C.RSA_meth_set_bn_mod_exp
func (*RSA_METHOD) SetBnModExp(bnModExp func(
	r *BIGNUM, a *BIGNUM, p *BIGNUM, m *BIGNUM, ctx *BN_CTX, mCtx *BN_MONT_CTX) c.Int) c.Int {
	return 0
}
*/

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_pub_enc(RSA_METHOD *rsa,
                         int (*pub_enc) (int flen, const unsigned char *from,
                                         unsigned char *to, RSA *rsa,
                                         int padding));
*/
// llgo:link (*RSA_METHOD).SetPubEnc C.RSA_meth_set_pub_enc
func (*RSA_METHOD) SetPubEnc(pubEnc func(
	flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_pub_dec(RSA_METHOD *rsa,
                         int (*pub_dec) (int flen, const unsigned char *from,
                                         unsigned char *to, RSA *rsa,
                                         int padding));
*/
// llgo:link (*RSA_METHOD).SetPubDec C.RSA_meth_set_pub_dec
func (*RSA_METHOD) SetPubDec(pubDec func(
	flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_priv_enc(RSA_METHOD *rsa,
                          int (*priv_enc) (int flen, const unsigned char *from,
                                           unsigned char *to, RSA *rsa,
                                           int padding));
*/
// llgo:link (*RSA_METHOD).SetPrivEnc C.RSA_meth_set_priv_enc
func (*RSA_METHOD) SetPrivEnc(privEnc func(
	flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_priv_dec(RSA_METHOD *rsa,
                          int (*priv_dec) (int flen, const unsigned char *from,
                                           unsigned char *to, RSA *rsa,
                                           int padding));
*/
// llgo:link (*RSA_METHOD).SetPrivDec C.RSA_meth_set_priv_dec
func (*RSA_METHOD) SetPrivDec(privDec func(
	flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_sign(RSA_METHOD *rsa,
                      int (*sign) (int type, const unsigned char *m,
                                   unsigned int m_length,
                                   unsigned char *sigret, unsigned int *siglen,
                                   const RSA *rsa));
*/
// llgo:link (*RSA_METHOD).SetSign C.RSA_meth_set_sign
func (*RSA_METHOD) SetSign(sign func(
	typ c.Int, msg *byte, mlen c.Uint,
	sigret *byte, siglen *c.Uint, rsa *RSA) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_verify(RSA_METHOD *rsa,
                        int (*verify) (int dtype, const unsigned char *m,
                                       unsigned int m_length,
                                       const unsigned char *sigbuf,
                                       unsigned int siglen, const RSA *rsa));
*/
// llgo:link (*RSA_METHOD).SetVerify C.RSA_meth_set_verify
func (*RSA_METHOD) SetVerify(verify func(
	dtype c.Int, msg *byte, mlen c.Uint,
	sigbuf *byte, siglen c.Uint, rsa *RSA) c.Int) c.Int {
	return 0
}

/*
OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_keygen(RSA_METHOD *rsa,
                        int (*keygen) (RSA *rsa, int bits, BIGNUM *e,
                                       BN_GENCB *cb));

OSSL_DEPRECATEDIN_3_0
int RSA_meth_set_multi_prime_keygen(RSA_METHOD *meth,
                                    int (*keygen) (RSA *rsa, int bits,
                                                   int primes, BIGNUM *e,
                                                   BN_GENCB *cb));
*/

// -----------------------------------------------------------------------------

type RSA struct {
	Unused [0]byte
}

// OSSL_DEPRECATEDIN_3_0 RSA *RSA_new(void);
//
//go:linkname RSANew C.RSA_new
func RSANew() *RSA

// OSSL_DEPRECATEDIN_3_0 RSA *RSA_new_method(ENGINE *engine);

// OSSL_DEPRECATEDIN_3_0 void RSA_free(RSA *r);
//
// llgo:link (*RSA).Free C.RSA_free
func (*RSA) Free() {}

// "up" the RSA object's reference count
// OSSL_DEPRECATEDIN_3_0 int RSA_up_ref(RSA *r);
//
// llgo:link (*RSA).UpRef C.RSA_up_ref
func (*RSA) UpRef() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_bits(const RSA *rsa);
//
// llgo:link (*RSA).Bits C.RSA_bits
func (*RSA) Bits() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_size(const RSA *rsa);
//
// llgo:link (*RSA).Size C.RSA_size
func (*RSA) Size() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_security_bits(const RSA *rsa);
//
// llgo:link (*RSA).SecurityBits C.RSA_security_bits
func (*RSA) SecurityBits() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_flags(const RSA *r);
//
// llgo:link (*RSA).Flags C.RSA_flags
func (*RSA) Flags() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 void RSA_set_flags(RSA *r, int flags);
//
// llgo:link (*RSA).SetFlags C.RSA_set_flags
func (*RSA) SetFlags(flags c.Int) {}

// OSSL_DEPRECATEDIN_3_0 void RSA_clear_flags(RSA *r, int flags);
//
// llgo:link (*RSA).ClearFlags C.RSA_clear_flags
func (*RSA) ClearFlags(flags c.Int) {}

// OSSL_DEPRECATEDIN_3_0 int RSA_test_flags(const RSA *r, int flags);
//
// llgo:link (*RSA).TestFlags C.RSA_test_flags
func (*RSA) TestFlags(flags c.Int) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_get_version(RSA *r);
//
// llgo:link (*RSA).GetVersion C.RSA_get_version
func (*RSA) GetVersion() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_set_ex_data(RSA *r, int idx, void *arg);
//
// llgo:link (*RSA).SetExData C.RSA_set_ex_data
func (*RSA) SetExData(idx c.Int, arg unsafe.Pointer) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 void *RSA_get_ex_data(const RSA *r, int idx);
//
// llgo:link (*RSA).GetExData C.RSA_get_ex_data
func (*RSA) GetExData(idx c.Int) unsafe.Pointer { return nil }

// OSSL_DEPRECATEDIN_3_0 int RSA_set_method(RSA *rsa, const RSA_METHOD *meth);
//
// llgo:link (*RSA).SetMethod C.RSA_set_method
func (*RSA) SetMethod(meth *RSA_METHOD) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_generate_key_ex(RSA *rsa, int bits, BIGNUM *e, BN_GENCB *cb);
//
// llgo:link (*RSA).GenerateKeyEx C.RSA_generate_key_ex
func (*RSA) GenerateKeyEx(bits c.Int, e *BIGNUM, cb *BN_GENCB) c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int RSA_generate_multi_prime_key(RSA *rsa, int bits, int primes, BIGNUM *e, BN_GENCB *cb);
//
// llgo:link (*RSA).GenerateMultiPrimeKey C.RSA_generate_multi_prime_key
func (*RSA) GenerateMultiPrimeKey(bits, primes c.Int, e *BIGNUM, cb *BN_GENCB) c.Int { return 0 }

/*
// next 4 return -1 on error

OSSL_DEPRECATEDIN_3_0
int RSA_public_encrypt(int flen, const unsigned char *from, unsigned char *to,
						RSA *rsa, int padding);
OSSL_DEPRECATEDIN_3_0
int RSA_private_encrypt(int flen, const unsigned char *from, unsigned char *to,
						RSA *rsa, int padding);
OSSL_DEPRECATEDIN_3_0
int RSA_public_decrypt(int flen, const unsigned char *from, unsigned char *to,
						RSA *rsa, int padding);
OSSL_DEPRECATEDIN_3_0
int RSA_private_decrypt(int flen, const unsigned char *from, unsigned char *to,
						RSA *rsa, int padding);
*/
//go:linkname RSAPublicEncrypt C.RSA_public_encrypt
func RSAPublicEncrypt(flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int

//go:linkname RSAPrivateEncrypt C.RSA_private_encrypt
func RSAPrivateEncrypt(flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int

//go:linkname RSAPublicDecrypt C.RSA_public_decrypt
func RSAPublicDecrypt(flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int

//go:linkname RSAPrivateDecrypt C.RSA_private_decrypt
func RSAPrivateDecrypt(flen c.Int, from *byte, to *byte, rsa *RSA, padding c.Int) c.Int

// OSSL_DEPRECATEDIN_3_0 int RSA_sign(
// int type, const unsigned char *m, unsigned int m_length,
// unsigned char *sigret, unsigned int *siglen, RSA *rsa);
//
//go:linkname RSASign C.RSA_sign
func RSASign(typ c.Int, msg *byte, mlen c.Uint, sigret *byte, siglen *c.Uint, rsa *RSA) c.Int

// OSSL_DEPRECATEDIN_3_0 int RSA_verify(int type, const unsigned char *m,
//	  unsigned int m_length,
//	  const unsigned char *sigbuf,
//	  unsigned int siglen, RSA *rsa);

// -----------------------------------------------------------------------------
