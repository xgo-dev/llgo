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

package zlib

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = "link: $(pkg-config --libs zlib); -lz"
)

/* errno */
const (
	OK            = 0
	STREAM_END    = 1
	NEED_DICT     = 2
	ERRNO         = -1
	STREAM_ERROR  = -2
	DATA_ERROR    = -3
	MEM_ERROR     = -4
	BUF_ERROR     = -5
	VERSION_ERROR = -6
)

/* compression levels */
const (
	NO_COMPRESSION      = 0
	BEST_SPEED          = 1
	BEST_COMPRESSION    = 9
	DEFAULT_COMPRESSION = -1
)

const (
	NO_FLUSH      = 0
	PARTIAL_FLUSH = 1
	SYNC_FLUSH    = 2
	FULL_FLUSH    = 3
	FINISH        = 4
	BLOCK         = 5
	TREES         = 6
)

const (
	FILTERED         = 1
	HUFFMAN_ONLY     = 2
	RLE              = 3
	FIXED            = 4
	DEFAULT_STRATEGY = 0
)

const (
	BINARY  = 0
	TEXT    = 1
	ASCII   = TEXT
	UNKNOWN = 2
)

const (
	DEFLATED = 8
)

// -----------------------------------------------------------------------------

/*
ZEXTERN uLong ZEXPORT compressBound OF((uLong sourceLen));

compressBound() returns an upper bound on the compressed size after
compress() or compress2() on sourceLen bytes.  It would be used before a
compress() or compress2() call to allocate the destination buffer.
*/
//go:linkname CompressBound C.compressBound
func CompressBound(sourceLen c.Ulong) c.Ulong

/*
ZEXTERN int ZEXPORT compress OF((Bytef *dest,   uLongf *destLen,
	const Bytef *source, uLong sourceLen));

Compresses the source buffer into the destination buffer.  sourceLen is
the byte length of the source buffer.  Upon entry, destLen is the total size
of the destination buffer, which must be at least the value returned by
compressBound(sourceLen).  Upon exit, destLen is the actual size of the
compressed data.  compress() is equivalent to compress2() with a level
parameter of Z_DEFAULT_COMPRESSION.

compress returns Z_OK if success, Z_MEM_ERROR if there was not
enough memory, Z_BUF_ERROR if there was not enough room in the output
buffer.
*/
//go:linkname Compress C.compress
func Compress(dest *byte, destLen *c.Ulong, source *byte, sourceLen c.Ulong) c.Int

/*
ZEXTERN int ZEXPORT compress2 OF((Bytef *dest,   uLongf *destLen,
	const Bytef *source, uLong sourceLen, int level));

Compresses the source buffer into the destination buffer.  The level
parameter has the same meaning as in deflateInit.  sourceLen is the byte
length of the source buffer.  Upon entry, destLen is the total size of the
destination buffer, which must be at least the value returned by
compressBound(sourceLen).  Upon exit, destLen is the actual size of the
compressed data.

compress2 returns Z_OK if success, Z_MEM_ERROR if there was not enough
memory, Z_BUF_ERROR if there was not enough room in the output buffer,
Z_STREAM_ERROR if the level parameter is invalid.
*/
//go:linkname Compress2 C.compress2
func Compress2(dest *byte, destLen *c.Ulong, source *byte, sourceLen c.Ulong, level c.Int) c.Int

/*
ZEXTERN int ZEXPORT uncompress OF((Bytef *dest,   uLongf *destLen,
	const Bytef *source, uLong sourceLen));

Decompresses the source buffer into the destination buffer.  sourceLen is
the byte length of the source buffer.  Upon entry, destLen is the total size
of the destination buffer, which must be large enough to hold the entire
uncompressed data.  (The size of the uncompressed data must have been saved
previously by the compressor and transmitted to the decompressor by some
mechanism outside the scope of this compression library.) Upon exit, destLen
is the actual size of the uncompressed data.

uncompress returns Z_OK if success, Z_MEM_ERROR if there was not
enough memory, Z_BUF_ERROR if there was not enough room in the output
buffer, or Z_DATA_ERROR if the input data was corrupted or incomplete.  In
the case where there is not enough room, uncompress() will fill the output
buffer with the uncompressed data up to that point.
*/
//go:linkname Uncompress C.uncompress
func Uncompress(dest *byte, destLen *c.Ulong, source *byte, sourceLen c.Ulong) c.Int

/*
ZEXTERN int ZEXPORT uncompress2 OF((Bytef *dest,   uLongf *destLen,
	const Bytef *source, uLong *sourceLen));

Same as uncompress, except that sourceLen is a pointer, where the
length of the source is *sourceLen.  On return, *sourceLen is the number of
source bytes consumed.
*/
//go:linkname Uncompress2 C.uncompress2
func Uncompress2(dest *byte, destLen *c.Ulong, source *byte, sourceLen *c.Ulong) c.Int

// -----------------------------------------------------------------------------

/*
ZEXTERN uLong ZEXPORT crc32   OF((uLong crc, const Bytef *buf, uInt len));

Update a running CRC-32 with the bytes buf[0..len-1] and return the
updated CRC-32.  If buf is Z_NULL, this function returns the required
initial value for the crc.  Pre- and post-conditioning (one's complement) is
performed within this function so it shouldn't be done by the application.

Usage example:

	uLong crc = crc32(0L, Z_NULL, 0);

	while (read_buffer(buffer, length) != EOF) {
	crc = crc32(crc, buffer, length);
	}
	if (crc != original_crc) error();
*/
//go:linkname Crc32 C.crc32
func Crc32(crc c.Ulong, buf *byte, len c.Uint) c.Ulong

/*
ZEXTERN uLong ZEXPORT crc32_z OF((uLong adler, const Bytef *buf, z_size_t len));

Same as crc32(), but with a size_t length.
*/
//go:linkname Crc32Z C.crc32_z
func Crc32Z(crc c.Ulong, buf *byte, len uintptr) c.Ulong

func Crc32ZBytes(crc c.Ulong, buf []byte) c.Ulong {
	return Crc32Z(crc, unsafe.SliceData(buf), uintptr(len(buf)))
}

func Crc32ZString(crc c.Ulong, buf string) c.Ulong {
	return Crc32Z(crc, unsafe.StringData(buf), uintptr(len(buf)))
}

/*
ZEXTERN uLong ZEXPORT crc32_combine OF((uLong crc1, uLong crc2, z_off_t len2));

Combine two CRC-32 check values into one.  For two sequences of bytes,
seq1 and seq2 with lengths len1 and len2, CRC-32 check values were
calculated for each, crc1 and crc2.  crc32_combine() returns the CRC-32
check value of seq1 and seq2 concatenated, requiring only crc1, crc2, and
len2.
*/
//go:linkname Crc32Combine C.crc32_combine
func Crc32Combine(crc1 c.Ulong, crc2 c.Ulong, len2 int64) c.Ulong

/*
ZEXTERN uLong ZEXPORT adler32 OF((uLong adler, const Bytef *buf, uInt len));

Update a running Adler-32 checksum with the bytes buf[0..len-1] and
return the updated checksum.  If buf is Z_NULL, this function returns the
required initial value for the checksum.

An Adler-32 checksum is almost as reliable as a CRC-32 but can be computed
much faster.

Usage example:

	uLong adler = adler32(0L, Z_NULL, 0);

	while (read_buffer(buffer, length) != EOF) {
	adler = adler32(adler, buffer, length);
	}
	if (adler != original_adler) error();
*/
//go:linkname Adler32 C.adler32
func Adler32(adler c.Ulong, buf *byte, len c.Uint) c.Ulong

/*
ZEXTERN uLong ZEXPORT adler32_z OF((uLong adler, const Bytef *buf, z_size_t len));

Same as adler32(), but with a size_t length.
*/
//go:linkname Adler32Z C.adler32_z
func Adler32Z(adler c.Ulong, buf *byte, len uintptr) c.Ulong

func Adler32ZBytes(adler c.Ulong, buf []byte) c.Ulong {
	return Adler32Z(adler, unsafe.SliceData(buf), uintptr(len(buf)))
}

func Adler32ZString(adler c.Ulong, buf string) c.Ulong {
	return Adler32Z(adler, unsafe.StringData(buf), uintptr(len(buf)))
}

/*
ZEXTERN uLong ZEXPORT adler32_combine OF((uLong adler1, uLong adler2, z_off_t len2));

Combine two Adler-32 checksums into one.  For two sequences of bytes, seq1
and seq2 with lengths len1 and len2, Adler-32 checksums were calculated for
each, adler1 and adler2.  adler32_combine() returns the Adler-32 checksum of
seq1 and seq2 concatenated, requiring only adler1, adler2, and len2.  Note
that the z_off_t type (like off_t) is a signed integer.  If len2 is
negative, the result has no meaning or utility.
*/
//go:linkname Adler32Combine C.adler32_combine
func Adler32Combine(adler1 c.Ulong, adler2 c.Ulong, len2 int64) c.Ulong

// -----------------------------------------------------------------------------
