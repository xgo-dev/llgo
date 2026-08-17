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

// typedef int (*pem_password_cb)(char *buf, int size, int rwflag, void *userdata);
//
// llgo:type C
type PemPasswordCb func(buf *c.Char, size, rwflag c.Int, userdata unsafe.Pointer) c.Int

// RSA *PEM_read_bio_RSAPrivateKey(BIO *bp, RSA **x, pem_password_cb *cb, void *u);
//
//go:linkname PEMReadBioRSAPrivateKey C.PEM_read_bio_RSAPrivateKey
func PEMReadBioRSAPrivateKey(bp *BIO, x **RSA, cb PemPasswordCb, u unsafe.Pointer) *RSA

// -----------------------------------------------------------------------------
