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

// Portions of this file are derived from github.com/blakesmith/ar.
// Copyright (c) 2013 Blake Smith <blakesmith0@gmail.com>.
// See ../../LICENSES/BlakeSmith-AR-MIT.txt for license terms.

package ar

import (
	"errors"
	"time"
)

var (
	errInvalidHeader = errors.New("ar: invalid header")
	errWriteTooLong  = errors.New("ar: write too long")
)

const (
	globalHeader    = "!<arch>\n"
	globalHeaderLen = len(globalHeader)
	headerByteSize  = 60
)

type recHeader struct {
	name    [16]byte
	modTime [12]byte
	uid     [6]byte
	gid     [6]byte
	mode    [8]byte
	size    [10]byte
	eol     [2]byte
}

type Header struct {
	Name    string
	ModTime time.Time
	Uid     int
	Gid     int
	Mode    int64
	Size    int64
}

type slicer []byte

func (sp *slicer) next(n int) (b []byte) {
	s := *sp
	b, *sp = s[0:n], s[n:]
	return
}
