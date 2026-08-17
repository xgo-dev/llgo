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

package nmindex

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/xtool/nm"
)

type IndexBuilder struct {
	nm *nm.Cmd
}

func NewIndexBuilder(nm *nm.Cmd) *IndexBuilder {
	return &IndexBuilder{nm}
}

func (p *IndexBuilder) Index(fromDir []string, toDir string, progress func(path string)) error {
	for _, dir := range fromDir {
		if dir == "" {
			continue
		}
		if e := p.IndexDir(dir, toDir, progress); e != nil {
			if !os.IsNotExist(e) {
				log.Println(e)
			}
		}
	}
	return nil
}

func (p *IndexBuilder) IndexDir(fromDir, toDir string, progress func(path string)) error {
	if abs, e := filepath.Abs(fromDir); e == nil {
		fromDir = abs
	}
	return filepath.WalkDir(fromDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fname := d.Name()
		switch filepath.Ext(fname) {
		case ".a", ".dylib", ".tbd", ".so", ".dll", ".lib":
			progress(path)
			hash := md5.Sum([]byte(path))
			hashStr := base64.RawURLEncoding.EncodeToString(hash[:])
			outFile := filepath.Join(toDir, strings.TrimPrefix(fname, "lib")+hashStr+".pub")
			e := p.IndexFile(path, outFile)
			if e != nil {
				log.Println(e)
			}
		}
		return nil
	})
}

func (p *IndexBuilder) IndexFile(arFile, outFile string) (err error) {
	items, err := p.nm.List(arFile)
	if err != nil {
		if len(items) == 0 {
			return
		}
	}
	var b bytes.Buffer
	b.WriteString("nm ")
	b.WriteString(arFile)
	b.WriteByte('\n')
	nbase := b.Len()
	for _, item := range items {
		if item.File != "" {
			b.WriteString("file ")
			b.WriteString(item.File)
			b.WriteByte('\n')
		}
		for _, sym := range item.Symbols {
			switch sym.Type {
			case nm.Text, nm.Data, nm.BSS, nm.Rodata, 'S', 'C', 'W', 'A':
				b.WriteByte(byte(sym.Type))
				b.WriteByte(' ')
				b.WriteString(sym.Name)
				b.WriteByte('\n')
			case nm.Undefined, nm.LocalText, nm.LocalData, nm.LocalBSS, nm.LocalASym, 'I', 'i', 'a', 'w':
				/*
					if sym.Type != Undefined && strings.Contains(sym.Name, "fprintf") {
						log.Printf("skip symbol type %c: %s\n", sym.Type, sym.Name)
					}
				*/
			default:
				log.Printf("unknown symbol type %c: %s\n", sym.Type, sym.Name)
			}
		}
	}
	buf := b.Bytes()
	if len(buf) <= nbase {
		return
	}
	return os.WriteFile(outFile, buf, 0666)
}
