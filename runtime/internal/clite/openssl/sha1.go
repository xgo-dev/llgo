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

const (
	SHA_DIGEST_LENGTH = 20
	SHA_LBLOCK        = 16
	SHA_CBLOCK        = (SHA_LBLOCK * 4)
	SHA_LAST_BLOCK    = (SHA_CBLOCK - 8)

	SHA256_CBLOCK            = (SHA_LBLOCK * 4)
	SHA256_192_DIGEST_LENGTH = 24
	SHA224_DIGEST_LENGTH     = 28
	SHA256_DIGEST_LENGTH     = 32
	SHA384_DIGEST_LENGTH     = 48
	SHA512_DIGEST_LENGTH     = 64
	SHA512_CBLOCK            = (SHA_LBLOCK * 8)
)

type SHA_LONG64 = c.UlongLong

type SHA_LONG = c.Uint

type SHA_CTX struct {
	H0, H1, H2, H3, H4 SHA_LONG
	Nl, Nh             SHA_LONG
	Data               [SHA_LBLOCK]SHA_LONG
	Num                c.Uint
}

// OSSL_DEPRECATEDIN_3_0 int SHA1_Init(SHA_CTX *c);
//
// llgo:link (*SHA_CTX).Init C.SHA1_Init
func (c *SHA_CTX) Init() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int SHA1_Update(SHA_CTX *c, const void *data, size_t len);
//
// llgo:link (*SHA_CTX).Update C.SHA1_Update
func (c *SHA_CTX) Update(data unsafe.Pointer, n uintptr) c.Int { return 0 }

func (c *SHA_CTX) UpdateBytes(data []byte) c.Int {
	return c.Update(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)))
}

func (c *SHA_CTX) UpdateString(data string) c.Int {
	return c.Update(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)))
}

// OSSL_DEPRECATEDIN_3_0 int SHA1_Final(unsigned char *md, SHA_CTX *c);
//
//go:linkname sha1Final C.SHA1_Final
func sha1Final(md *byte, c *SHA_CTX) c.Int

func (c *SHA_CTX) Final(md *byte) c.Int {
	return sha1Final(md, c)
}

// OSSL_DEPRECATEDIN_3_0 void SHA1_Transform(SHA_CTX *c, const unsigned char *data);
//
// llgo:link (*SHA_CTX).Transform C.SHA1_Transform
func (c *SHA_CTX) Transform(data *byte) {}

// unsigned char *SHA1(const unsigned char *d, size_t n, unsigned char *md);
//
//go:linkname SHA1 C.SHA1
func SHA1(data unsafe.Pointer, n uintptr, md *byte) *byte

func SHA1Bytes(data []byte, md *byte) *byte {
	return SHA1(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)), md)
}

func SHA1String(data string, md *byte) *byte {
	return SHA1(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)), md)
}
