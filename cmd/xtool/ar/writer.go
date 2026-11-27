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
)

// Writer provides sequential writing of an ar archive.
// An ar archive is sequence of header file pairs
// Call WriteHeader to begin writing a new file, then call Write to supply the file's data
//
// Example:
// archive := ar.NewWriter(writer)
// archive.WriteGlobalHeader()
// header := new(ar.Header)
// header.Size = 15 // bytes
//
//	if err := archive.WriteHeader(header); err != nil {
//		return err
//	}
//
// io.Copy(archive, data)
type Writer struct {
	w  io.Writer
	nb int64 // number of unwritten bytes for the current file entry
}

// Create a new ar writer that writes to w
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

func (aw *Writer) numeric(b []byte, x int64) {
	s := strconv.FormatInt(x, 10)
	for len(s) < len(b) {
		s = s + " "
	}
	copy(b, []byte(s))
}

func (aw *Writer) octal(b []byte, x int64) {
	s := "100" + strconv.FormatInt(x, 8)
	for len(s) < len(b) {
		s = s + " "
	}
	copy(b, []byte(s))
}

func (aw *Writer) string(b []byte, str string) {
	s := str
	for len(s) < len(b) {
		s = s + " "
	}
	copy(b, []byte(s))
}

// Writes to the current entry in the ar archive
// Returns ErrWriteTooLong if more than header.Size
// bytes are written after a call to WriteHeader
func (aw *Writer) Write(b []byte) (n int, err error) {
	if int64(len(b)) > aw.nb {
		b = b[0:aw.nb]
		err = errWriteTooLong
	}
	n, werr := aw.w.Write(b)
	aw.nb -= int64(n)
	if werr != nil {
		return n, werr
	}

	if len(b)%2 == 1 { // data size must be aligned to an even byte
		n2, _ := aw.w.Write([]byte{'\n'})
		return n + n2, err
	}

	return
}

func (aw *Writer) WriteGlobalHeader() error {
	_, err := aw.w.Write([]byte(globalHeader))
	return err
}

// Writes the header to the underlying writer and prepares
// to receive the file payload
func (aw *Writer) WriteHeader(hdr *Header) error {
	aw.nb = int64(hdr.Size)
	header := make([]byte, headerByteSize)
	s := slicer(header)

	aw.string(s.next(16), hdr.Name)
	aw.numeric(s.next(12), hdr.ModTime.Unix())
	aw.numeric(s.next(6), int64(hdr.Uid))
	aw.numeric(s.next(6), int64(hdr.Gid))
	aw.octal(s.next(8), hdr.Mode)
	aw.numeric(s.next(10), hdr.Size)
	aw.string(s.next(2), "`\n")

	_, err := aw.w.Write(header)

	return err
}
