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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"os"
	"path/filepath"
	"sort"

	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
)

const linkSnapMagic = "LLGOLSN1"

type linkSnapTypePayload struct {
	Types    frontcache.TypeGraph `json:"types"`
	ABI      []frontendABIType    `json:"abi,omitempty"`
	CExports []frontendCExport    `json:"cexports,omitempty"`
}

func (c *context) writeLinkSnapshotSidecar(path string, pkg *aPackage) error {
	if pkg == nil || pkg.linkSnapshot == nil {
		return nil
	}
	snap := pkg.linkSnapshot
	methodIndexes := make([]int, 0, len(snap.methodByIndex))
	for idx := range snap.methodByIndex {
		methodIndexes = append(methodIndexes, idx)
	}
	sort.Ints(methodIndexes)

	methodByName := make([]string, 0, len(snap.methodByName))
	for name := range snap.methodByName {
		methodByName = append(methodByName, name)
	}
	sort.Strings(methodByName)

	abiSymbols := make([]string, 0, len(snap.abiSymbols))
	for sym := range snap.abiSymbols {
		abiSymbols = append(abiSymbols, sym)
	}
	sort.Strings(abiSymbols)

	var ownedABI []llvmgen.AbiTypeInfo
	var abiTypeNames []string
	for _, info := range snap.abiTypes {
		if _, owned := snap.abiSymbols[info.Name]; owned && info.Raw != nil {
			ownedABI = append(ownedABI, info)
			abiTypeNames = append(abiTypeNames, info.Name)
		}
	}
	var cExportNames []string
	for _, ce := range snap.cExports {
		if ce.cName != "" {
			cExportNames = append(cExportNames, ce.cName)
		}
	}

	var typeGraphBlob []byte
	if (len(ownedABI) > 0 || len(snap.cExports) > 0) && c.progSSA != nil {
		ids := make(map[*types.Package]string, len(c.pkgByID))
		for id, p := range c.pkgByID {
			if p != nil && p.Package != nil && p.Types != nil {
				ids[p.Types] = id
			}
		}
		index := frontcache.NewTypes(ids, c.progSSA.Fset)
		graph := frontcache.NewTypeGraph(index, c.progSSA.Fset)
		payload := linkSnapTypePayload{}
		for _, info := range ownedABI {
			payload.ABI = append(payload.ABI, frontendABIType{Name: info.Name, Type: graph.Add(info.Raw)})
		}
		for _, ce := range snap.cExports {
			ref := -1
			if ce.sig != nil {
				ref = graph.Add(ce.sig)
			}
			payload.CExports = append(payload.CExports, frontendCExport{
				GoName:    ce.goName,
				CName:     ce.cName,
				Signature: ref,
			})
		}
		payload.Types = *graph
		blob, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		typeGraphBlob = blob
	}

	var buf bytes.Buffer
	buf.WriteString(linkSnapMagic)
	writeBinVarint(&buf, int64(snap.needAbiInit))
	writeBinBool(&buf, snap.hasLocalExports)
	writeBinUvarint(&buf, uint64(len(methodIndexes)))
	for _, idx := range methodIndexes {
		writeBinVarint(&buf, int64(idx))
	}
	writeBinUvarint(&buf, uint64(len(methodByName)))
	for _, name := range methodByName {
		writeBinStr(&buf, name)
	}
	writeBinUvarint(&buf, uint64(len(abiSymbols)))
	for _, sym := range abiSymbols {
		writeBinStr(&buf, sym)
	}
	writeBinUvarint(&buf, uint64(len(abiTypeNames)))
	for _, name := range abiTypeNames {
		writeBinStr(&buf, name)
	}
	writeBinUvarint(&buf, uint64(len(cExportNames)))
	for _, name := range cExportNames {
		writeBinStr(&buf, name)
	}
	writeBinBytes(&buf, typeGraphBlob)
	return writeFileAtomicBytes(path, buf.Bytes())
}

func (c *context) readLinkSnapshotSidecar(path string) (*packageLinkSnapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < len(linkSnapMagic) || string(raw[:len(linkSnapMagic)]) != linkSnapMagic {
		return nil, errors.New("invalid link snapshot magic")
	}
	r := &binReader{data: raw, pos: len(linkSnapMagic)}
	needAbiInit, err := r.readVarint()
	if err != nil {
		return nil, err
	}
	hasLocalExports, err := r.readBool()
	if err != nil {
		return nil, err
	}
	nIdx, err := r.readUvarint()
	if err != nil || nIdx > uint64(len(raw)) {
		return nil, errors.New("corrupt link snapshot")
	}
	snap := &packageLinkSnapshot{
		needAbiInit:     int(needAbiInit),
		hasLocalExports: hasLocalExports,
		methodByIndex:   make(map[int]none, nIdx),
		methodByName:    make(map[string]none),
		abiSymbols:      make(map[string]none),
	}
	for i := uint64(0); i < nIdx; i++ {
		v, err := r.readVarint()
		if err != nil || v < 0 {
			return nil, fmt.Errorf("invalid cached method index %d", v)
		}
		snap.methodByIndex[int(v)] = none{}
	}
	nNames, err := r.readUvarint()
	if err != nil || nNames > uint64(len(raw)) {
		return nil, errors.New("corrupt link snapshot")
	}
	for i := uint64(0); i < nNames; i++ {
		name, err := r.readStr()
		if err != nil {
			return nil, err
		}
		snap.methodByName[name] = none{}
	}
	nSyms, err := r.readUvarint()
	if err != nil || nSyms > uint64(len(raw)) {
		return nil, errors.New("corrupt link snapshot")
	}
	for i := uint64(0); i < nSyms; i++ {
		sym, err := r.readStr()
		if err != nil {
			return nil, err
		}
		snap.abiSymbols[sym] = none{}
	}
	nAbiTypes, err := r.readUvarint()
	if err != nil || nAbiTypes > uint64(len(raw)) {
		return nil, errors.New("corrupt link snapshot")
	}
	for i := uint64(0); i < nAbiTypes; i++ {
		if _, err := r.readStr(); err != nil {
			return nil, err
		}
	}
	nCExports, err := r.readUvarint()
	if err != nil || nCExports > uint64(len(raw)) {
		return nil, errors.New("corrupt link snapshot")
	}
	for i := uint64(0); i < nCExports; i++ {
		if _, err := r.readStr(); err != nil {
			return nil, err
		}
	}
	typeGraphBlob, err := r.readBytes()
	if err != nil {
		return nil, err
	}

	if len(typeGraphBlob) > 0 {
		if c.progSSA == nil {
			return nil, fmt.Errorf("missing SSA program for type graph import")
		}
		var payload linkSnapTypePayload
		if err := json.Unmarshal(typeGraphBlob, &payload); err != nil {
			return nil, err
		}
		pkgs := make(map[string]*types.Package, len(c.pkgByID))
		ids := make(map[*types.Package]string, len(c.pkgByID))
		for id, p := range c.pkgByID {
			if p != nil && p.Package != nil && p.Types != nil {
				pkgs[id] = p.Types
				ids[p.Types] = id
			}
		}
		existing := frontcache.NewTypes(ids, c.progSSA.Fset)
		importer := &frontcache.TypeImporter{
			Graph:    &payload.Types,
			Packages: pkgs,
			Existing: existing,
			Position: frontendPosition(c.progSSA.Fset),
		}
		for _, info := range payload.ABI {
			typ, err := importer.Import(info.Type)
			if err != nil || info.Name == "" {
				return nil, fmt.Errorf("import cached ABI type %s: %v", info.Name, err)
			}
			snap.abiTypes = append(snap.abiTypes, llvmgen.AbiTypeInfo{Name: info.Name, Raw: typ})
		}
		for _, ce := range payload.CExports {
			if ce.Signature == -1 && ce.GoName != "" && ce.CName != "" {
				snap.cExports = append(snap.cExports, cExport{goName: ce.GoName, cName: ce.CName})
				continue
			}
			typ, err := importer.Import(ce.Signature)
			sig, ok := typ.(*types.Signature)
			if err != nil || !ok || ce.GoName == "" || ce.CName == "" {
				return nil, fmt.Errorf("import cached C export %s: %v", ce.GoName, err)
			}
			snap.cExports = append(snap.cExports, cExport{goName: ce.GoName, cName: ce.CName, sig: sig})
		}
	} else if nAbiTypes > 0 || nCExports > 0 {
		return nil, fmt.Errorf("incomplete link snapshot sidecar")
	}
	return snap, nil
}

func writeFileAtomicBytes(dst string, data []byte) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".llgo-sidecar-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := replaceFile(tmpName, dst); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
