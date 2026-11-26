/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package build

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"sort"
)

const depfileMagic = "LLGODEP3"

type depfileStat struct {
	Path    string
	Size    int64
	ModTime int64
	Hash    string
}

type depfileRecord struct {
	Key              string
	PreprocessedHash string
	Files            []depfileStat
}

func statDepfiles(paths []string) ([]depfileStat, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool, len(paths))
	stats := make([]depfileStat, 0, len(paths))
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		fi, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		stats = append(stats, depfileStat{
			Path:    p,
			Size:    fi.Size(),
			ModTime: fi.ModTime().UnixNano(),
			Hash:    digestBytes(data),
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Path < stats[j].Path })
	return stats, nil
}

func validateDepfile(df *depfileRecord) bool {
	if df == nil || df.PreprocessedHash == "" {
		return false
	}
	for _, f := range df.Files {
		fi, err := os.Stat(f.Path)
		if err != nil {
			return false
		}
		if fi.Size() != f.Size || f.Hash == "" {
			return false
		}
		data, err := os.ReadFile(f.Path)
		if err != nil || digestBytes(data) != f.Hash {
			return false
		}
	}
	return true
}

func marshalDepfile(df *depfileRecord) ([]byte, error) {
	if df == nil {
		return nil, errors.New("nil depfile")
	}
	var buf bytes.Buffer
	buf.WriteString(depfileMagic)
	writeBinStr(&buf, df.Key)
	writeBinStr(&buf, df.PreprocessedHash)
	files := append([]depfileStat(nil), df.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	writeBinUvarint(&buf, uint64(len(files)))
	for _, f := range files {
		writeBinStr(&buf, f.Path)
		writeBinVarint(&buf, f.Size)
		writeBinVarint(&buf, f.ModTime)
		writeBinStr(&buf, f.Hash)
	}
	return buf.Bytes(), nil
}

func unmarshalDepfile(raw []byte) (*depfileRecord, error) {
	if len(raw) < len(depfileMagic) || string(raw[:len(depfileMagic)]) != depfileMagic {
		return nil, errors.New("invalid depfile magic")
	}
	r := &binReader{data: raw, pos: len(depfileMagic)}
	key, err := r.readStr()
	if err != nil {
		return nil, err
	}
	hash, err := r.readStr()
	if err != nil {
		return nil, err
	}
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(raw)) {
		return nil, errors.New("corrupt depfile")
	}
	df := &depfileRecord{
		Key:              key,
		PreprocessedHash: hash,
	}
	if n > 0 {
		df.Files = make([]depfileStat, n)
		for i := uint64(0); i < n; i++ {
			if df.Files[i].Path, err = r.readStr(); err != nil {
				return nil, err
			}
			if df.Files[i].Size, err = r.readVarint(); err != nil {
				return nil, err
			}
			if df.Files[i].ModTime, err = r.readVarint(); err != nil {
				return nil, err
			}
			if df.Files[i].Hash, err = r.readStr(); err != nil {
				return nil, err
			}
		}
	}
	return df, nil
}

func writeBinUvarint(buf *bytes.Buffer, v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], v)
	buf.Write(tmp[:n])
}

func writeBinVarint(buf *bytes.Buffer, v int64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutVarint(tmp[:], v)
	buf.Write(tmp[:n])
}

func writeBinStr(buf *bytes.Buffer, s string) {
	writeBinUvarint(buf, uint64(len(s)))
	buf.WriteString(s)
}

func writeBinBytes(buf *bytes.Buffer, p []byte) {
	writeBinUvarint(buf, uint64(len(p)))
	buf.Write(p)
}

func writeBinBool(buf *bytes.Buffer, v bool) {
	if v {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
}

type binReader struct {
	data []byte
	pos  int
}

func (r *binReader) readBool() (bool, error) {
	if r.pos >= len(r.data) {
		return false, errors.New("unexpected EOF")
	}
	b := r.data[r.pos]
	r.pos++
	return b != 0, nil
}

func (r *binReader) readUvarint() (uint64, error) {
	v, n := binary.Uvarint(r.data[r.pos:])
	if n <= 0 {
		return 0, errors.New("corrupt uvarint")
	}
	r.pos += n
	return v, nil
}

func (r *binReader) readVarint() (int64, error) {
	v, n := binary.Varint(r.data[r.pos:])
	if n <= 0 {
		return 0, errors.New("corrupt varint")
	}
	r.pos += n
	return v, nil
}

func (r *binReader) readStr() (string, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return "", errors.New("corrupt string")
	}
	s := string(r.data[r.pos : r.pos+int(n)])
	r.pos += int(n)
	return s, nil
}

func (r *binReader) readBytes() ([]byte, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return nil, errors.New("corrupt bytes")
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]byte, n)
	copy(out, r.data[r.pos:r.pos+int(n)])
	r.pos += int(n)
	return out, nil
}
