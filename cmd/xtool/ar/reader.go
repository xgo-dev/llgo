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
	"io"
	"strconv"
	"strings"
	"unsafe"
)

// Provides read access to an ar archive.
// Call next to skip files
//
// Example:
//	reader := NewReader(f)
//	var buf bytes.Buffer
//	for {
//		_, err := reader.Next()
//		if err == io.EOF {
//			break
//		}
//		if err != nil {
//			t.Errorf(err.Error())
//		}
//		io.Copy(&buf, reader)
//	}

type Reader struct {
	r   io.Reader
	nb  int64
	pad int64
}

// Copies read data to r. Strips the global ar header.
func NewReader(r io.Reader) (*Reader, error) {
	buf := make([]byte, globalHeaderLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if string(buf) != globalHeader {
		return nil, errInvalidHeader
	}

	return &Reader{r: r}, nil
}

func stringVal(b []byte) string {
	return strings.TrimRight(string(b), " ")
}

func intVal(b []byte) (int64, error) {
	return strconv.ParseInt(stringVal(b), 10, 64)
}

func (rd *Reader) skipUnread() error {
	skip := rd.nb + rd.pad
	rd.nb, rd.pad = 0, 0
	if seeker, ok := rd.r.(io.Seeker); ok {
		_, err := seeker.Seek(skip, io.SeekCurrent)
		return err
	}

	_, err := io.CopyN(io.Discard, rd.r, skip)
	return err
}

func (rd *Reader) readHeader() (header *Header, err error) {
	var rec recHeader
	var buf = (*[headerByteSize]byte)(unsafe.Pointer(&rec))[:]
	if _, err = io.ReadFull(rd.r, buf); err != nil {
		return
	}

	header = new(Header)
	header.Name = stringVal(rec.name[:])
	if header.Size, err = intVal(rec.size[:]); err != nil {
		return
	}

	if header.Size%2 == 1 {
		rd.pad = 1
	} else {
		rd.pad = 0
	}

	if rec.name[0] == '#' {
		if n, e := strconv.ParseInt(strings.TrimPrefix(header.Name[3:], "#1/"), 10, 64); e == nil {
			name := make([]byte, n)
			if _, err = io.ReadFull(rd.r, name); err != nil {
				return
			}
			header.Name = string(name)
			header.Size -= n
		}
	}

	rd.nb = int64(header.Size)
	return
}

// Call Next() to skip to the next file in the archive file.
// Returns a Header which contains the metadata about the
// file in the archive.
func (rd *Reader) Next() (*Header, error) {
	err := rd.skipUnread()
	if err != nil {
		return nil, err
	}

	return rd.readHeader()
}

// Read data from the current entry in the archive.
func (rd *Reader) Read(b []byte) (n int, err error) {
	if rd.nb == 0 {
		return 0, io.EOF
	}
	if int64(len(b)) > rd.nb {
		b = b[0:rd.nb]
	}
	n, err = rd.r.Read(b)
	rd.nb -= int64(n)

	return
}
