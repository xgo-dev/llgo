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

type SHA512_CTX struct {
	H          [8]SHA_LONG64
	N1, Nh     SHA_LONG64
	D          [SHA_LBLOCK]SHA_LONG64
	Num, MdLen c.Uint
}

type SHA384_CTX SHA512_CTX

// OSSL_DEPRECATEDIN_3_0 int SHA384_Init(SHA512_CTX *c);
//
// llgo:link (*SHA384_CTX).Init C.SHA384_Init
func (c *SHA384_CTX) Init() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int SHA384_Update(SHA512_CTX *c, const void *data, size_t len);
//
// llgo:link (*SHA384_CTX).Update C.SHA384_Update
func (c *SHA384_CTX) Update(data unsafe.Pointer, n uintptr) c.Int { return 0 }

func (c *SHA384_CTX) UpdateBytes(data []byte) c.Int {
	return c.Update(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)))
}

func (c *SHA384_CTX) UpdateString(data string) c.Int {
	return c.Update(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)))
}

// OSSL_DEPRECATEDIN_3_0 int SHA384_Final(unsigned char *md, SHA512_CTX *c);
//
//go:linkname sha384Final C.SHA384_Final
func sha384Final(md *byte, c *SHA384_CTX) c.Int

func (c *SHA384_CTX) Final(md *byte) c.Int {
	return sha384Final(md, c)
}

// OSSL_DEPRECATEDIN_3_0 int SHA512_Init(SHA512_CTX *c);
//
// llgo:link (*SHA512_CTX).Init C.SHA512_Init
func (c *SHA512_CTX) Init() c.Int { return 0 }

// OSSL_DEPRECATEDIN_3_0 int SHA512_Update(SHA512_CTX *c, const void *data, size_t len);
//
// llgo:link (*SHA512_CTX).Update C.SHA512_Update
func (c *SHA512_CTX) Update(data unsafe.Pointer, n uintptr) c.Int { return 0 }
func (c *SHA512_CTX) UpdateBytes(data []byte) c.Int {
	return c.Update(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)))
}
func (c *SHA512_CTX) UpdateString(data string) c.Int {
	return c.Update(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)))
}

// OSSL_DEPRECATEDIN_3_0 int SHA512_Final(unsigned char *md, SHA512_CTX *c);
//
//go:linkname sha512Final C.SHA512_Final
func sha512Final(md *byte, c *SHA512_CTX) c.Int

func (c *SHA512_CTX) Final(md *byte) c.Int {
	return sha512Final(md, c)
}

// OSSL_DEPRECATEDIN_3_0 void SHA512_Transform(SHA512_CTX *c, const unsigned char *data);
//
// llgo:link (*SHA512_CTX).Transform C.SHA512_Transform
func (c *SHA512_CTX) Transform(data *byte) {}

// unsigned char *SHA384(const unsigned char *d, size_t n, unsigned char *md);
//
//go:linkname SHA384 C.SHA384
func SHA384(data unsafe.Pointer, n uintptr, md *byte) *byte

func SHA384Bytes(data []byte, md *byte) *byte {
	return SHA384(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)), md)
}

func SHA384String(data string, md *byte) *byte {
	return SHA384(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)), md)
}

// unsigned char *SHA512(const unsigned char *d, size_t n, unsigned char *md);
//
//go:linkname SHA512 C.SHA512
func SHA512(data unsafe.Pointer, n uintptr, md *byte) *byte

func SHA512Bytes(data []byte, md *byte) *byte {
	return SHA512(unsafe.Pointer(unsafe.SliceData(data)), uintptr(len(data)), md)
}

func SHA512String(data string, md *byte) *byte {
	return SHA512(unsafe.Pointer(unsafe.StringData(data)), uintptr(len(data)), md)
}
