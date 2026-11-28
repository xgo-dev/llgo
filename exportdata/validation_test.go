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

package exportdata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/types"
)

func TestDecoderRejectsUnsupportedVersions(t *testing.T) {
	for _, ver := range []uint32{1, 2, 3, formatVersion + 1} {
		var raw bytes.Buffer
		raw.WriteString(magicHeader)
		raw.WriteByte(kindPackage)
		var v [4]byte
		binary.LittleEndian.PutUint32(v[:], ver)
		raw.Write(v[:])
		raw.WriteByte(0) // empty string table

		_, err := unmarshalPackage(raw.Bytes(), map[string]*types.Package{})
		if !errors.Is(err, errUnsupportedVer) {
			t.Fatalf("version %d: err = %v, want errUnsupportedVer", ver, err)
		}
	}
}

func TestDecoderRejectsInvalidEnumsAndFlags(t *testing.T) {
	t.Run("PackageKind", func(t *testing.T) {
		e := newEncoder()
		e.writeStr("example.com/badkind")
		e.writeStr("badkind")
		e.writeStr("")
		e.writeBool(true)
		e.writeByte(byte(directive.PkgLinkExtern) + 1)
		e.writeStr("")
		e.writeBool(false)
		e.writeStringSlice(nil)
		_, err := unmarshalPackage(e.finish(kindPackage), map[string]*types.Package{})
		if !errors.Is(err, errCorrupt) {
			t.Fatalf("invalid PackageKind: err = %v, want errCorrupt", err)
		}
	})

	t.Run("NamedCallConv", func(t *testing.T) {
		e := newEncoder()
		// Header
		e.writeStr("example.com/badcc")
		e.writeStr("badcc")
		e.writeStr("")
		e.writeBool(true)
		e.writeByte(byte(directive.PkgNormal))
		e.writeStr("")
		e.writeBool(false)
		e.writeStringSlice(nil)
		// Package table: 1 (self)
		e.writeUvarint(1)
		e.writeStr("example.com/badcc")
		e.writeStr("badcc")
		// Imports: 0
		e.writeUvarint(0)
		// Named shells: 1 with invalid CallConv
		e.writeUvarint(1)
		e.writeUvarint(1) // pkgID=1 (self)
		e.writeStr("BadNamed")
		e.writeByte(byte(types.Wasm) + 1)

		_, err := unmarshalPackage(e.finish(kindPackage), map[string]*types.Package{})
		if !errors.Is(err, errCorrupt) {
			t.Fatalf("invalid Named CallConv: err = %v, want errCorrupt", err)
		}
	})

	t.Run("AliasCallConv", func(t *testing.T) {
		e := newEncoder()
		// Header
		e.writeStr("example.com/badaliascc")
		e.writeStr("badaliascc")
		e.writeStr("")
		e.writeBool(true)
		e.writeByte(byte(directive.PkgNormal))
		e.writeStr("")
		e.writeBool(false)
		e.writeStringSlice(nil)
		// Package table: 1 (self)
		e.writeUvarint(1)
		e.writeStr("example.com/badaliascc")
		e.writeStr("badaliascc")
		// Imports: 0
		e.writeUvarint(0)
		// Named shells: 0
		e.writeUvarint(0)
		// Alias shells: 1 with invalid CallConv
		e.writeUvarint(1)
		e.writeUvarint(1)
		e.writeStr("BadAlias")
		e.writeByte(byte(types.Wasm) + 1)

		_, err := unmarshalPackage(e.finish(kindPackage), map[string]*types.Package{})
		if !errors.Is(err, errCorrupt) {
			t.Fatalf("invalid Alias CallConv: err = %v, want errCorrupt", err)
		}
	})

	t.Run("SignatureCallConv", func(t *testing.T) {
		e := newEncoder()
		e.writeByte(typeTagSignature)
		e.writeByte(byte(types.Wasm) + 1) // invalid CallConv
		e.writeBool(false)                // variadic
		e.writeBool(false)                // no recv
		e.writeUvarint(0)                 // rtps
		e.writeUvarint(0)                 // tps
		e.writeByte(typeTagNil)           // params
		e.writeByte(typeTagNil)           // results
		dec, err := newDecoder(e.finish(kindPackage), kindPackage)
		if err != nil {
			t.Fatal(err)
		}
		pr := &pkgReader{dec: dec}
		if _, err := pr.readType(); !errors.Is(err, errCorrupt) {
			t.Fatalf("invalid Signature CallConv: err = %v, want errCorrupt", err)
		}
	})

	t.Run("UnknownFuncMetaFlags", func(t *testing.T) {
		e := newEncoder()
		e.writeUvarint(funcMetaNoSplit << 1)
		dec, err := newDecoder(e.finish(kindPackage), kindPackage)
		if err != nil {
			t.Fatal(err)
		}
		pr := &pkgReader{dec: dec}
		if _, err := pr.readFuncMeta(); !errors.Is(err, errCorrupt) {
			t.Fatalf("unknown funcMeta flag: err = %v, want errCorrupt", err)
		}
	})

	t.Run("UnknownVarMetaFlags", func(t *testing.T) {
		e := newEncoder()
		e.writeByte(varMetaLocality << 1)
		dec, err := newDecoder(e.finish(kindPackage), kindPackage)
		if err != nil {
			t.Fatal(err)
		}
		pr := &pkgReader{dec: dec}
		if _, err := pr.readVarMeta(); !errors.Is(err, errCorrupt) {
			t.Fatalf("unknown varMeta flag: err = %v, want errCorrupt", err)
		}
	})

	t.Run("InvalidVarLocality", func(t *testing.T) {
		e := newEncoder()
		e.writeByte(varMetaLocality)
		e.writeByte(byte(directive.LocalityGoroutine) + 1)
		dec, err := newDecoder(e.finish(kindPackage), kindPackage)
		if err != nil {
			t.Fatal(err)
		}
		pr := &pkgReader{dec: dec}
		if _, err := pr.readVarMeta(); !errors.Is(err, errCorrupt) {
			t.Fatalf("invalid Locality: err = %v, want errCorrupt", err)
		}
	})
}

func TestExportDataWriteReadEmptyPackage(t *testing.T) {
	pkg := types.NewPackage("example.com/empty", "empty")
	pkg.MarkComplete()
	var buf bytes.Buffer
	if err := Write(&buf, token.NewFileSet(), pkg); err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(buf.Bytes()), token.NewFileSet(), map[string]*types.Package{}, pkg.Path())
	if err != nil || got.Path() != pkg.Path() {
		t.Fatalf("Read empty package = (%v, %v)", got, err)
	}
}
