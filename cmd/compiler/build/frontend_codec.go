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
	"errors"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
)

const frontendExportMagic = "LLGOFE01"

const frontendExportHeaderSize = len(frontendExportMagic) + sha256.Size

func marshalFrontendExport(rec *frontendExport) ([]byte, error) {
	if rec == nil {
		return nil, errors.New("nil frontend export")
	}
	var payload bytes.Buffer
	writeBinVarint(&payload, int64(rec.Version))
	writeBinStr(&payload, rec.Input)
	writeBinStr(&payload, rec.Archive)

	writeBinUvarint(&payload, uint64(len(rec.Functions)))
	for i := range rec.Functions {
		fn := &rec.Functions[i]
		writeBinStr(&payload, fn.ID)
		writeBinStr(&payload, fn.CacheKey)
		var factBits byte
		if fn.Facts.CallerBase {
			factBits |= 1 << 0
		}
		if fn.Facts.CallerFrame {
			factBits |= 1 << 1
		}
		if fn.Facts.RecoverPanicSite {
			factBits |= 1 << 2
		}
		if fn.Facts.UsesRecover {
			factBits |= 1 << 3
		}
		if fn.Facts.RecoverScope {
			factBits |= 1 << 4
		}
		if fn.Escape != nil {
			factBits |= 1 << 5
		}
		payload.WriteByte(factBits)
		writeBinVarint(&payload, int64(fn.Facts.InlineCost))
		if fn.Escape != nil {
			writeBinUvarint(&payload, uint64(len(fn.Escape.Parameters)))
			for j := range fn.Escape.Parameters {
				p := &fn.Escape.Parameters[j]
				writeBinUvarint(&payload, uint64(p.Heap))
				writeBinUvarint(&payload, uint64(p.Mutator))
				writeBinUvarint(&payload, uint64(p.Callee))
				writeBinUvarint(&payload, uint64(len(p.Results)))
				for _, eff := range p.Results {
					writeBinUvarint(&payload, uint64(eff))
				}
			}
		}
	}

	writeBinTypeGraph(&payload, &rec.Types)
	writeBinInts(&payload, rec.RuntimeTypes)

	writeBinBool(&payload, rec.Link != nil)
	if rec.Link != nil {
		l := rec.Link
		writeBinVarint(&payload, int64(l.NeedABI))
		writeBinBool(&payload, l.LocalExports)
		writeBinInts(&payload, l.MethodIndexes)
		writeBinStrs(&payload, l.MethodNames)
		writeBinStrs(&payload, l.ABISymbols)
		writeBinTypeGraph(&payload, &l.Types)
		writeBinUvarint(&payload, uint64(len(l.ABI)))
		for i := range l.ABI {
			writeBinStr(&payload, l.ABI[i].Name)
			writeBinVarint(&payload, int64(l.ABI[i].Type))
		}
		writeBinUvarint(&payload, uint64(len(l.CExports)))
		for i := range l.CExports {
			writeBinStr(&payload, l.CExports[i].GoName)
			writeBinStr(&payload, l.CExports[i].CName)
			writeBinVarint(&payload, int64(l.CExports[i].Signature))
		}
	}

	body := payload.Bytes()
	out := make([]byte, frontendExportHeaderSize+len(body))
	copy(out, frontendExportMagic)
	sum := sha256.Sum256(body)
	copy(out[len(frontendExportMagic):frontendExportHeaderSize], sum[:])
	copy(out[frontendExportHeaderSize:], body)
	return out, nil
}

func unmarshalFrontendExport(raw []byte) (*frontendExport, error) {
	if len(raw) < frontendExportHeaderSize || string(raw[:len(frontendExportMagic)]) != frontendExportMagic {
		return nil, errors.New("invalid frontend export magic")
	}
	body := raw[frontendExportHeaderSize:]
	sum := sha256.Sum256(body)
	if !bytes.Equal(raw[len(frontendExportMagic):frontendExportHeaderSize], sum[:]) {
		return nil, errors.New("corrupt frontend export digest")
	}
	r := &binReader{data: body}
	ver, err := r.readVarint()
	if err != nil {
		return nil, err
	}
	input, err := r.readStr()
	if err != nil {
		return nil, err
	}
	archive, err := r.readStr()
	if err != nil {
		return nil, err
	}
	nFuncs, err := r.readUvarint()
	if err != nil || nFuncs > uint64(len(body)-r.pos) {
		return nil, errors.New("corrupt frontend export functions")
	}
	rec := &frontendExport{
		Version: int(ver),
		Input:   input,
		Archive: archive,
	}
	if nFuncs > 0 {
		rec.Functions = make([]frontendFunction, nFuncs)
		for i := uint64(0); i < nFuncs; i++ {
			fn := &rec.Functions[i]
			if fn.ID, err = r.readStr(); err != nil {
				return nil, err
			}
			if fn.CacheKey, err = r.readStr(); err != nil {
				return nil, err
			}
			bits, err := r.readByte()
			if err != nil {
				return nil, err
			}
			cost, err := r.readVarint()
			if err != nil {
				return nil, err
			}
			fn.Facts = cl.FrontendFacts{
				CallerBase:       bits&(1<<0) != 0,
				CallerFrame:      bits&(1<<1) != 0,
				RecoverPanicSite: bits&(1<<2) != 0,
				UsesRecover:      bits&(1<<3) != 0,
				RecoverScope:     bits&(1<<4) != 0,
				InlineCost:       int(cost),
			}
			if bits&(1<<5) != 0 {
				nParams, err := r.readUvarint()
				if err != nil || nParams > uint64(len(body)-r.pos) {
					return nil, errors.New("corrupt escape summary")
				}
				params := make([]goescape.Parameter, nParams)
				for j := uint64(0); j < nParams; j++ {
					heap, err := r.readUvarint()
					if err != nil {
						return nil, err
					}
					mutator, err := r.readUvarint()
					if err != nil {
						return nil, err
					}
					callee, err := r.readUvarint()
					if err != nil {
						return nil, err
					}
					nRes, err := r.readUvarint()
					if err != nil || nRes > uint64(len(body)-r.pos) {
						return nil, errors.New("corrupt escape results")
					}
					var results []goescape.Effect
					if nRes > 0 {
						results = make([]goescape.Effect, nRes)
						for k := uint64(0); k < nRes; k++ {
							eff, err := r.readUvarint()
							if err != nil {
								return nil, err
							}
							results[k] = goescape.Effect(eff)
						}
					}
					params[j] = goescape.Parameter{
						Heap:    goescape.Effect(heap),
						Mutator: goescape.Effect(mutator),
						Callee:  goescape.Effect(callee),
						Results: results,
					}
				}
				fn.Escape = &goescape.Summary{Parameters: params}
			}
		}
	}

	if err := readBinTypeGraph(r, &rec.Types); err != nil {
		return nil, err
	}
	if rec.RuntimeTypes, err = readBinInts(r); err != nil {
		return nil, err
	}

	hasLink, err := r.readBool()
	if err != nil {
		return nil, err
	}
	if hasLink {
		needABI, err := r.readVarint()
		if err != nil {
			return nil, err
		}
		localExports, err := r.readBool()
		if err != nil {
			return nil, err
		}
		methodIndexes, err := readBinInts(r)
		if err != nil {
			return nil, err
		}
		methodNames, err := readBinStrs(r)
		if err != nil {
			return nil, err
		}
		abiSymbols, err := readBinStrs(r)
		if err != nil {
			return nil, err
		}
		l := &frontendLinkExport{
			NeedABI:       int(needABI),
			LocalExports:  localExports,
			MethodIndexes: methodIndexes,
			MethodNames:   methodNames,
			ABISymbols:    abiSymbols,
		}
		if err := readBinTypeGraph(r, &l.Types); err != nil {
			return nil, err
		}
		nABI, err := r.readUvarint()
		if err != nil || nABI > uint64(len(body)-r.pos) {
			return nil, errors.New("corrupt link ABI types")
		}
		if nABI > 0 {
			l.ABI = make([]frontendABIType, nABI)
			for i := uint64(0); i < nABI; i++ {
				if l.ABI[i].Name, err = r.readStr(); err != nil {
					return nil, err
				}
				typ, err := r.readVarint()
				if err != nil {
					return nil, err
				}
				l.ABI[i].Type = int(typ)
			}
		}
		nCExp, err := r.readUvarint()
		if err != nil || nCExp > uint64(len(body)-r.pos) {
			return nil, errors.New("corrupt link C exports")
		}
		if nCExp > 0 {
			l.CExports = make([]frontendCExport, nCExp)
			for i := uint64(0); i < nCExp; i++ {
				if l.CExports[i].GoName, err = r.readStr(); err != nil {
					return nil, err
				}
				if l.CExports[i].CName, err = r.readStr(); err != nil {
					return nil, err
				}
				sig, err := r.readVarint()
				if err != nil {
					return nil, err
				}
				l.CExports[i].Signature = int(sig)
			}
		}
		rec.Link = l
	}
	if r.pos != len(body) {
		return nil, errors.New("trailing bytes in frontend export")
	}
	return rec, nil
}

func writeBinInts(buf *bytes.Buffer, vals []int) {
	writeBinUvarint(buf, uint64(len(vals)))
	for _, v := range vals {
		writeBinVarint(buf, int64(v))
	}
}

func readBinInts(r *binReader) ([]int, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return nil, errors.New("corrupt int slice")
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]int, n)
	for i := uint64(0); i < n; i++ {
		v, err := r.readVarint()
		if err != nil {
			return nil, err
		}
		out[i] = int(v)
	}
	return out, nil
}

func writeBinStrs(buf *bytes.Buffer, vals []string) {
	writeBinUvarint(buf, uint64(len(vals)))
	for _, s := range vals {
		writeBinStr(buf, s)
	}
}

func readBinStrs(r *binReader) ([]string, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return nil, errors.New("corrupt string slice")
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]string, n)
	for i := uint64(0); i < n; i++ {
		if out[i], err = r.readStr(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func writeBinObject(buf *bytes.Buffer, obj *frontcache.Object) {
	writeBinStr(buf, obj.Name)
	writeBinStr(buf, obj.Package)
	writeBinStr(buf, obj.Position)
	writeBinVarint(buf, int64(obj.Type))
	var flags byte
	if obj.Embedded {
		flags |= 1 << 0
	}
	if len(obj.Tag) > 0 {
		flags |= 1 << 1
	}
	buf.WriteByte(flags)
	if len(obj.Tag) > 0 {
		writeBinBytes(buf, obj.Tag)
	}
}

func readBinObject(r *binReader) (frontcache.Object, error) {
	var obj frontcache.Object
	var err error
	if obj.Name, err = r.readStr(); err != nil {
		return obj, err
	}
	if obj.Package, err = r.readStr(); err != nil {
		return obj, err
	}
	if obj.Position, err = r.readStr(); err != nil {
		return obj, err
	}
	typ, err := r.readVarint()
	if err != nil {
		return obj, err
	}
	obj.Type = int(typ)
	flags, err := r.readByte()
	if err != nil {
		return obj, err
	}
	obj.Embedded = flags&(1<<0) != 0
	if flags&(1<<1) != 0 {
		if obj.Tag, err = r.readBytes(); err != nil {
			return obj, err
		}
	}
	return obj, nil
}

func writeBinObjects(buf *bytes.Buffer, objs []frontcache.Object) {
	writeBinUvarint(buf, uint64(len(objs)))
	for i := range objs {
		writeBinObject(buf, &objs[i])
	}
}

func readBinObjects(r *binReader) ([]frontcache.Object, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return nil, errors.New("corrupt object slice")
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]frontcache.Object, n)
	for i := uint64(0); i < n; i++ {
		if out[i], err = readBinObject(r); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func writeBinTypeGraph(buf *bytes.Buffer, g *frontcache.TypeGraph) {
	writeBinUvarint(buf, uint64(len(g.Nodes)))
	for i := range g.Nodes {
		n := &g.Nodes[i]
		writeBinStr(buf, n.Kind)
		writeBinStr(buf, n.Key)
		var mask uint32
		if n.Object.Name != "" || n.Object.Package != "" || n.Object.Position != "" || n.Object.Type != 0 || n.Object.Embedded || len(n.Object.Tag) > 0 {
			mask |= 1 << 0
		}
		if n.Elem != -1 {
			mask |= 1 << 1
		}
		if n.Base != -1 {
			mask |= 1 << 2
		}
		if n.Basic != 0 {
			mask |= 1 << 3
		}
		if n.Direction != 0 {
			mask |= 1 << 4
		}
		if n.CallConv != 0 {
			mask |= 1 << 5
		}
		if n.PtrKind != 0 {
			mask |= 1 << 6
		}
		if n.Length != 0 {
			mask |= 1 << 7
		}
		if n.Variadic {
			mask |= 1 << 8
		}
		if n.Implicit {
			mask |= 1 << 9
		}
		if n.NativeLayout {
			mask |= 1 << 10
		}
		if n.IsMask {
			mask |= 1 << 11
		}
		if n.Universe {
			mask |= 1 << 12
		}
		if len(n.Types) > 0 {
			mask |= 1 << 13
		}
		if len(n.Params) > 0 {
			mask |= 1 << 14
		}
		if len(n.Results) > 0 {
			mask |= 1 << 15
		}
		if len(n.TypeParams) > 0 {
			mask |= 1 << 16
		}
		if len(n.RecvParams) > 0 {
			mask |= 1 << 17
		}
		if n.Receiver != nil {
			mask |= 1 << 18
		}
		if len(n.Parameters) > 0 {
			mask |= 1 << 19
		}
		if len(n.Returns) > 0 {
			mask |= 1 << 20
		}
		if len(n.Elements) > 0 {
			mask |= 1 << 21
		}
		if len(n.Fields) > 0 {
			mask |= 1 << 22
		}
		if len(n.Methods) > 0 {
			mask |= 1 << 23
		}
		if len(n.Tilde) > 0 {
			mask |= 1 << 24
		}
		writeBinUvarint(buf, uint64(mask))
		if mask&(1<<0) != 0 {
			writeBinObject(buf, &n.Object)
		}
		if mask&(1<<1) != 0 {
			writeBinVarint(buf, int64(n.Elem))
		}
		if mask&(1<<2) != 0 {
			writeBinVarint(buf, int64(n.Base))
		}
		if mask&(1<<3) != 0 {
			writeBinVarint(buf, int64(n.Basic))
		}
		if mask&(1<<4) != 0 {
			writeBinVarint(buf, int64(n.Direction))
		}
		if mask&(1<<5) != 0 {
			buf.WriteByte(n.CallConv)
		}
		if mask&(1<<6) != 0 {
			buf.WriteByte(n.PtrKind)
		}
		if mask&(1<<7) != 0 {
			writeBinVarint(buf, n.Length)
		}
		if mask&(1<<13) != 0 {
			writeBinInts(buf, n.Types)
		}
		if mask&(1<<14) != 0 {
			writeBinInts(buf, n.Params)
		}
		if mask&(1<<15) != 0 {
			writeBinInts(buf, n.Results)
		}
		if mask&(1<<16) != 0 {
			writeBinInts(buf, n.TypeParams)
		}
		if mask&(1<<17) != 0 {
			writeBinInts(buf, n.RecvParams)
		}
		if mask&(1<<18) != 0 {
			writeBinObject(buf, n.Receiver)
		}
		if mask&(1<<19) != 0 {
			writeBinObjects(buf, n.Parameters)
		}
		if mask&(1<<20) != 0 {
			writeBinObjects(buf, n.Returns)
		}
		if mask&(1<<21) != 0 {
			writeBinObjects(buf, n.Elements)
		}
		if mask&(1<<22) != 0 {
			writeBinObjects(buf, n.Fields)
		}
		if mask&(1<<23) != 0 {
			writeBinObjects(buf, n.Methods)
		}
		if mask&(1<<24) != 0 {
			writeBinUvarint(buf, uint64(len(n.Tilde)))
			for _, t := range n.Tilde {
				writeBinBool(buf, t)
			}
		}
	}
}

func readBinTypeGraph(r *binReader, g *frontcache.TypeGraph) error {
	nNodes, err := r.readUvarint()
	if err != nil || nNodes > uint64(len(r.data)-r.pos) {
		return errors.New("corrupt type graph")
	}
	if nNodes == 0 {
		g.Nodes = nil
		return nil
	}
	nodes := make([]frontcache.TypeNode, nNodes)
	for i := uint64(0); i < nNodes; i++ {
		n := &nodes[i]
		n.Elem = -1
		n.Base = -1
		if n.Kind, err = r.readStr(); err != nil {
			return err
		}
		if n.Key, err = r.readStr(); err != nil {
			return err
		}
		mask, err := r.readUvarint()
		if err != nil {
			return err
		}
		if mask&(1<<0) != 0 {
			if n.Object, err = readBinObject(r); err != nil {
				return err
			}
		}
		if mask&(1<<1) != 0 {
			v, err := r.readVarint()
			if err != nil {
				return err
			}
			n.Elem = int(v)
		}
		if mask&(1<<2) != 0 {
			v, err := r.readVarint()
			if err != nil {
				return err
			}
			n.Base = int(v)
		}
		if mask&(1<<3) != 0 {
			v, err := r.readVarint()
			if err != nil {
				return err
			}
			n.Basic = int(v)
		}
		if mask&(1<<4) != 0 {
			v, err := r.readVarint()
			if err != nil {
				return err
			}
			n.Direction = int(v)
		}
		if mask&(1<<5) != 0 {
			if n.CallConv, err = r.readByte(); err != nil {
				return err
			}
		}
		if mask&(1<<6) != 0 {
			if n.PtrKind, err = r.readByte(); err != nil {
				return err
			}
		}
		if mask&(1<<7) != 0 {
			if n.Length, err = r.readVarint(); err != nil {
				return err
			}
		}
		n.Variadic = mask&(1<<8) != 0
		n.Implicit = mask&(1<<9) != 0
		n.NativeLayout = mask&(1<<10) != 0
		n.IsMask = mask&(1<<11) != 0
		n.Universe = mask&(1<<12) != 0
		if mask&(1<<13) != 0 {
			if n.Types, err = readBinInts(r); err != nil {
				return err
			}
		}
		if mask&(1<<14) != 0 {
			if n.Params, err = readBinInts(r); err != nil {
				return err
			}
		}
		if mask&(1<<15) != 0 {
			if n.Results, err = readBinInts(r); err != nil {
				return err
			}
		}
		if mask&(1<<16) != 0 {
			if n.TypeParams, err = readBinInts(r); err != nil {
				return err
			}
		}
		if mask&(1<<17) != 0 {
			if n.RecvParams, err = readBinInts(r); err != nil {
				return err
			}
		}
		if mask&(1<<18) != 0 {
			obj, err := readBinObject(r)
			if err != nil {
				return err
			}
			n.Receiver = &obj
		}
		if mask&(1<<19) != 0 {
			if n.Parameters, err = readBinObjects(r); err != nil {
				return err
			}
		}
		if mask&(1<<20) != 0 {
			if n.Returns, err = readBinObjects(r); err != nil {
				return err
			}
		}
		if mask&(1<<21) != 0 {
			if n.Elements, err = readBinObjects(r); err != nil {
				return err
			}
		}
		if mask&(1<<22) != 0 {
			if n.Fields, err = readBinObjects(r); err != nil {
				return err
			}
		}
		if mask&(1<<23) != 0 {
			if n.Methods, err = readBinObjects(r); err != nil {
				return err
			}
		}
		if mask&(1<<24) != 0 {
			nt, err := r.readUvarint()
			if err != nil || nt > uint64(len(r.data)-r.pos) {
				return errors.New("corrupt tilde slice")
			}
			n.Tilde = make([]bool, nt)
			for j := uint64(0); j < nt; j++ {
				if n.Tilde[j], err = r.readBool(); err != nil {
					return err
				}
			}
		}
	}
	g.Nodes = nodes
	return nil
}

func (r *binReader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, errors.New("unexpected EOF")
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}
