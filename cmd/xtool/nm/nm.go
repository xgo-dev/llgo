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

package nm

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	errInvalidOutput = errors.New("invalid nm output")
)

// -----------------------------------------------------------------------------

// Cmd represents a nm command.
type Cmd struct {
	app string
}

// New creates a new nm command.
func New(app string) *Cmd {
	if app == "" {
		app = "nm"
	}
	return &Cmd{app}
}

// -----------------------------------------------------------------------------

// SymbolType represents a symbol type.
type SymbolType uint8

const (
	Undefined = SymbolType('U') // Undefined
	Text      = SymbolType('T') // Text (code) section symbol
	Data      = SymbolType('D') // Data (global var) section symbol
	Rodata    = SymbolType('R') // Read-only data (rodata) section symbol
	BSS       = SymbolType('B') // BSS (uninitialized global var) section symbol

	LocalText = SymbolType('t') // Local text (code) section symbol
	LocalData = SymbolType('d') // Local data (local var) section symbol
	LocalBSS  = SymbolType('b') // Local BSS (uninitialized local var) section symbol
	LocalASym = SymbolType('s') // Local symbol in an assembler source file
)

// VersionType represents the version type of a symbol.
// This is specific to Linux systems.
// On macOS , this will always be VersionNone.
// https://sourceware.org/binutils/docs/binutils/nm.html
type VersionType int

const (
	VersionNone     VersionType = iota // No version information
	VersionSpecific                    // Specific version (@)
	VersionDefault                     // Default version (@@)
)

// Symbol represents a symbol in an object file.
type Symbol struct {
	Name        string      // symbol name
	Addr        uint64      // symbol address
	Type        SymbolType  // symbol type
	FAddr       bool        // address is valid
	VersionType VersionType // version type of the symbol
	Version     string      // version information of the symbol
}

// ObjectFile represents an object file.
type ObjectFile struct {
	File    string    // file name
	Symbols []*Symbol // symbols
}

// List lists symbols in an archive file
// accepts optional nm command flags.
// Note: The available flags may vary depending on the operating system.
// On Linux, the -D flag is used to display dynamic symbols from the dynamic symbol table.
// On macOS, there's no -D flag. The nm command displays all symbols (including dynamic ones) by default.
// This difference is due to the distinct ways Linux (using ELF format) and macOS (using Mach-O format)
// When working with dynamic libraries:
// On Linux: Use 'nm -D /path/to/library.so'
// On macOS: Simply use 'nm /path/to/library.dylib'
func (p *Cmd) List(arfile string, options ...string) (items []*ObjectFile, err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := append(options, arfile)
	cmd := exec.Command(p.app, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	e := cmd.Run()
	if stderr.Len() > 0 {
		listError(stderr.Bytes())
	}
	items, err = listOutput(stdout.Bytes())
	if err == nil {
		err = e
	}
	return
}

func listError(data []byte) {
	sep := []byte{'\n'}
	nosym := []byte(": no symbols")
	lines := bytes.Split(data, sep)
	for _, line := range lines {
		if len(line) == 0 || bytes.HasSuffix(line, nosym) {
			continue
		}
		os.Stderr.Write(line)
		os.Stderr.Write(sep)
	}
}

func listOutput(data []byte) (items []*ObjectFile, err error) {
	sep := []byte{'\n'}
	item := &ObjectFile{}
	lines := bytes.Split(data, sep)
	for _, line := range lines {
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if len(line) == 0 {
			if item.File == "" && len(item.Symbols) > 0 {
				items = append(items, item)
			}
			item = nil
			continue
		}
		if item == nil {
			s := string(line)
			if strings.HasSuffix(s, ":") {
				item = &ObjectFile{File: s[:len(s)-1]}
				items = append(items, item)
				continue
			}
			err = errInvalidOutput
			return
		}
		if len(line) < 10 {
			err = errInvalidOutput
			return
		}
		var sym *Symbol
		var fullSymName string
		if is64bits(line) {
			fullSymName = string(line[19:])
			sym = &Symbol{
				Type: SymbolType(line[17]),
			}
			if sym.FAddr = hasAddr(line); sym.FAddr {
				sym.Addr = hexUint64(line)
			}
		} else {
			fullSymName = string(line[11:])
			sym = &Symbol{
				Type: SymbolType(line[9]),
			}
			if sym.FAddr = hasAddr(line); sym.FAddr {
				sym.Addr = uint64(hexUint32(line))
			}
		}

		sym.Name, sym.VersionType, sym.Version = parseSymName(fullSymName)
		item.Symbols = append(item.Symbols, sym)
	}
	return
}

func parseSymName(symName string) (name string, versionType VersionType, version string) {
	if idx := strings.LastIndex(symName, "@"); idx != -1 {
		name = symName[:idx]
		version = symName[idx+1:]
		if idx > 0 && symName[idx-1] == '@' {
			versionType = VersionDefault
			name = symName[:idx-1]
		} else {
			versionType = VersionSpecific
		}
	} else {
		name = symName
		versionType = VersionNone
	}
	return
}

func hasAddr(line []byte) bool {
	c := line[0]
	return c != ' ' && c != '-'
}

func is64bits(line []byte) bool {
	if line[0] != ' ' {
		return line[8] != ' '
	}
	return line[9] == ' '
}

func hexUint64(b []byte) uint64 {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, "-->", string(b))
			panic(e)
		}
	}()
	_ = b[15] // bounds check hint to compiler; see golang.org/issue/14808
	return hex(b[15]) | hex(b[14])<<4 | hex(b[13])<<8 | hex(b[12])<<12 |
		hex(b[11])<<16 | hex(b[10])<<20 | hex(b[9])<<24 | hex(b[8])<<28 |
		hex(b[7])<<32 | hex(b[6])<<36 | hex(b[5])<<40 | hex(b[4])<<44 |
		hex(b[3])<<48 | hex(b[2])<<52 | hex(b[1])<<56 | hex(b[0])<<60
}

func hexUint32(b []byte) uint64 {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, "-->", string(b))
			panic(e)
		}
	}()
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return hex(b[7]) | hex(b[6])<<4 | hex(b[5])<<8 | hex(b[4])<<12 |
		hex(b[3])<<16 | hex(b[2])<<20 | hex(b[1])<<24 | hex(b[0])<<28
}

func hex(b byte) uint64 {
	return hexTable[b]
}

var hexTable = []uint64{
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4,
	'5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	'a': 10, 'b': 11, 'c': 12, 'd': 13, 'e': 14, 'f': 15,
	'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14, 'F': 15,
}

// -----------------------------------------------------------------------------
