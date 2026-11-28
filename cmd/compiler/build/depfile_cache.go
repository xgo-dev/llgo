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
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const depfileMagic = "LLGODEP5"

type depfileStat struct {
	Path    string
	Size    int64
	ModTime int64
	Hash    string
}

type depfileDirStat struct {
	Path   string
	Exists bool
	Hash   string
}

type depfileRecord struct {
	Key              string
	PreprocessedHash string
	Files            []depfileStat
	Dirs             []depfileDirStat
	Missing          []string
}

func statDepfiles(paths []string) ([]depfileStat, error) {
	stats, _, err := statDepfilesWithContent(paths)
	return stats, err
}

func statDepfilesWithContent(paths []string) ([]depfileStat, map[string][]byte, error) {
	if len(paths) == 0 {
		return nil, nil, nil
	}
	seen := make(map[string]bool, len(paths))
	stats := make([]depfileStat, 0, len(paths))
	contents := make(map[string][]byte, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if seen[p] {
			continue
		}
		seen[p] = true
		fi, err := os.Stat(p)
		if err != nil {
			return nil, nil, err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, nil, err
		}
		contents[p] = data
		stats = append(stats, depfileStat{
			Path:    p,
			Size:    fi.Size(),
			ModTime: fi.ModTime().UnixNano(),
			Hash:    digestBytes(data),
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Path < stats[j].Path })
	return stats, contents, nil
}

func hashDirEntries(entries []os.DirEntry) string {
	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.Name()))
		if e.IsDir() {
			h.Write([]byte{0, 'd', '\n'})
		} else {
			h.Write([]byte{0, 'f', '\n'})
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func statDepfileDirs(dirs []string) ([]depfileDirStat, error) {
	if len(dirs) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool, len(dirs))
	stats := make([]depfileDirStat, 0, len(dirs))
	for _, d := range dirs {
		if d == "" {
			continue
		}
		d = filepath.Clean(d)
		if seen[d] {
			continue
		}
		seen[d] = true
		entries, err := os.ReadDir(d)
		if err != nil {
			if os.IsNotExist(err) {
				stats = append(stats, depfileDirStat{Path: d, Exists: false})
				continue
			}
			return nil, err
		}
		stats = append(stats, depfileDirStat{
			Path:   d,
			Exists: true,
			Hash:   hashDirEntries(entries),
		})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Path < stats[j].Path })
	return stats, nil
}

type depfileFileCheckResult struct {
	size int64
	hash string
	ok   bool
}

type depfileDirCheckResult struct {
	exists bool
	hash   string
	ok     bool
}

// depfileCheckCache memoizes header file/dir/missing checks within a single
// build invocation so multiple C files sharing system SDK headers do not
// re-read and re-hash the same headers repeatedly.
type depfileCheckCache struct {
	mu      sync.RWMutex
	files   map[string]depfileFileCheckResult
	dirs    map[string]depfileDirCheckResult
	missing map[string]bool
}

func newDepfileCheckCache() *depfileCheckCache {
	return &depfileCheckCache{
		files:   make(map[string]depfileFileCheckResult),
		dirs:    make(map[string]depfileDirCheckResult),
		missing: make(map[string]bool),
	}
}

func (c *depfileCheckCache) checkFile(path string) depfileFileCheckResult {
	if c != nil {
		c.mu.RLock()
		res, ok := c.files[path]
		c.mu.RUnlock()
		if ok {
			return res
		}
	}
	var res depfileFileCheckResult
	if fi, err := os.Stat(path); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			res = depfileFileCheckResult{size: fi.Size(), hash: digestBytes(data), ok: true}
		}
	}
	if c != nil {
		c.mu.Lock()
		c.files[path] = res
		c.mu.Unlock()
	}
	return res
}

func (c *depfileCheckCache) checkDir(path string) depfileDirCheckResult {
	if c != nil {
		c.mu.RLock()
		res, ok := c.dirs[path]
		c.mu.RUnlock()
		if ok {
			return res
		}
	}
	var res depfileDirCheckResult
	entries, err := os.ReadDir(path)
	if err == nil {
		res = depfileDirCheckResult{exists: true, hash: hashDirEntries(entries), ok: true}
	} else if os.IsNotExist(err) {
		res = depfileDirCheckResult{exists: false, ok: true}
	}
	if c != nil {
		c.mu.Lock()
		c.dirs[path] = res
		c.mu.Unlock()
	}
	return res
}

func (c *depfileCheckCache) checkMissing(path string) bool {
	if c != nil {
		c.mu.RLock()
		res, ok := c.missing[path]
		c.mu.RUnlock()
		if ok {
			return res
		}
	}
	_, err := os.Stat(path)
	notExist := os.IsNotExist(err)
	if c != nil {
		c.mu.Lock()
		c.missing[path] = notExist
		c.mu.Unlock()
	}
	return notExist
}

func validateDepfile(df *depfileRecord) bool {
	return validateDepfileCached(df, nil)
}

func validateDepfileCached(df *depfileRecord, cache *depfileCheckCache) bool {
	if df == nil || df.PreprocessedHash == "" {
		return false
	}
	for _, f := range df.Files {
		if f.Hash == "" {
			return false
		}
		res := cache.checkFile(f.Path)
		if !res.ok || res.size != f.Size || res.hash != f.Hash {
			return false
		}
	}
	for _, d := range df.Dirs {
		res := cache.checkDir(d.Path)
		if !res.ok || res.exists != d.Exists {
			return false
		}
		if d.Exists && res.hash != d.Hash {
			return false
		}
	}
	for _, p := range df.Missing {
		if !cache.checkMissing(p) {
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
	dirs := append([]depfileDirStat(nil), df.Dirs...)
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Path < dirs[j].Path })
	writeBinUvarint(&buf, uint64(len(dirs)))
	for _, d := range dirs {
		writeBinStr(&buf, d.Path)
		writeBinBool(&buf, d.Exists)
		writeBinStr(&buf, d.Hash)
	}
	missing := append([]string(nil), df.Missing...)
	sort.Strings(missing)
	writeBinUvarint(&buf, uint64(len(missing)))
	for _, m := range missing {
		writeBinStr(&buf, m)
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
	nd, err := r.readUvarint()
	if err != nil || nd > uint64(len(raw)) {
		return nil, errors.New("corrupt depfile")
	}
	if nd > 0 {
		df.Dirs = make([]depfileDirStat, nd)
		for i := uint64(0); i < nd; i++ {
			if df.Dirs[i].Path, err = r.readStr(); err != nil {
				return nil, err
			}
			if df.Dirs[i].Exists, err = r.readBool(); err != nil {
				return nil, err
			}
			if df.Dirs[i].Hash, err = r.readStr(); err != nil {
				return nil, err
			}
		}
	}
	nm, err := r.readUvarint()
	if err != nil || nm > uint64(len(raw)) {
		return nil, errors.New("corrupt depfile")
	}
	if nm > 0 {
		df.Missing = make([]string, nm)
		for i := uint64(0); i < nm; i++ {
			if df.Missing[i], err = r.readStr(); err != nil {
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
