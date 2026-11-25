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
	"encoding/json"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"sort"

	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/internal/frontcache"
	"github.com/xgo-dev/llgo/internal/llvmgen"
)

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
	ed := &exportdata.LinkSnapshot{
		PkgPath:          pkg.PkgPath,
		PkgName:          pkg.Name,
		NeedRuntime:      pkg.NeedRt,
		NeedPyInit:       pkg.NeedPyInit,
		NeedAbiInit:      snap.needAbiInit != 0,
		NeedAbiInitCount: snap.needAbiInit,
		HasLocalExports:  snap.hasLocalExports,
		MethodByIndex:    len(snap.methodByIndex) > 0,
	}
	for idx := range snap.methodByIndex {
		ed.MethodIndexes = append(ed.MethodIndexes, idx)
	}
	sort.Ints(ed.MethodIndexes)
	for name := range snap.methodByName {
		ed.MethodByName = append(ed.MethodByName, name)
	}
	for sym := range snap.abiSymbols {
		ed.AbiSymbols = append(ed.AbiSymbols, sym)
	}

	// Encode ABI type & CExport signature payloads only when needed.
	var ownedABI []llvmgen.AbiTypeInfo
	for _, info := range snap.abiTypes {
		if _, owned := snap.abiSymbols[info.Name]; owned && info.Raw != nil {
			ownedABI = append(ownedABI, info)
			ed.AbiTypes = append(ed.AbiTypes, info.Name)
		}
	}
	for _, ce := range snap.cExports {
		if ce.cName != "" {
			ed.CExports = append(ed.CExports, ce.cName)
		}
	}
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
		ed.TypeGraphBlob = blob
	}

	raw, err := exportdata.MarshalLinkSnapshot(ed)
	if err != nil {
		return err
	}
	return writeFileAtomicBytes(path, raw)
}

func (c *context) readLinkSnapshotSidecar(path string) (*packageLinkSnapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ed, err := exportdata.UnmarshalLinkSnapshot(raw)
	if err != nil {
		return nil, err
	}
	snap := &packageLinkSnapshot{
		needAbiInit:     ed.NeedAbiInitCount,
		hasLocalExports: ed.HasLocalExports,
		methodByIndex:   make(map[int]none, len(ed.MethodIndexes)),
		methodByName:    make(map[string]none, len(ed.MethodByName)),
		abiSymbols:      make(map[string]none, len(ed.AbiSymbols)),
	}
	for _, idx := range ed.MethodIndexes {
		if idx < 0 {
			return nil, fmt.Errorf("invalid cached method index %d", idx)
		}
		snap.methodByIndex[idx] = none{}
	}
	for _, name := range ed.MethodByName {
		snap.methodByName[name] = none{}
	}
	for _, sym := range ed.AbiSymbols {
		snap.abiSymbols[sym] = none{}
	}

	if len(ed.TypeGraphBlob) > 0 {
		if c.progSSA == nil {
			return nil, fmt.Errorf("missing SSA program for type graph import")
		}
		var payload linkSnapTypePayload
		if err := json.Unmarshal(ed.TypeGraphBlob, &payload); err != nil {
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
	} else if len(ed.AbiTypes) > 0 || len(ed.CExports) > 0 {
		// Sidecar declared ABI types or C exports without a type graph blob; fall back safely.
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
