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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/pclnpost"
)

func finalizeRuntimePCLN(ctx *context, out *OutFmtDetails, verbose bool) error {
	if ctx == nil || ctx.buildConf == nil || out == nil || out.Out == "" {
		return nil
	}
	switch ctx.buildConf.PCLNMode {
	case PCLNEmbedded:
		rewritePrebuiltFuncTab(ctx, out.Out, verbose)
		// A successful non-external rebuild owns this conventional sibling
		// path and must not leave a stale optional artifact behind.
		_ = os.Remove(pclnSidecarPath(out.Out))
		return nil
	case PCLNNone:
		_ = os.Remove(pclnSidecarPath(out.Out))
		return nil
	case PCLNExternal:
		return writeExternalPCLN(ctx, out, verbose)
	default:
		return fmt.Errorf("invalid PCLN mode %d", ctx.buildConf.PCLNMode)
	}
}

func writeExternalPCLN(ctx *context, out *OutFmtDetails, verbose bool) (err error) {
	if out.PCLN == "" {
		return fmt.Errorf("external pclntab output path is empty")
	}
	if ctx.pclnExternal == nil {
		return fmt.Errorf("external pclntab metadata was not generated")
	}
	analysis, err := pclnpost.AnalyzeExternal(out.Out)
	if err != nil {
		return fmt.Errorf("analyze external pclntab: %w", err)
	}
	data := *ctx.pclnExternal
	if analysis.PointerSize != data.PointerSize {
		return fmt.Errorf("external pclntab pointer size changed from %d to %d", data.PointerSize, analysis.PointerSize)
	}
	data.ImageBase = analysis.ImageBase
	data.TextStart = analysis.TextStart
	data.TextEnd = analysis.TextEnd
	data.Identity = analysis.Identity
	data.EntrySites = make([]pclnmap.Site, len(analysis.EntrySites))
	for i, site := range analysis.EntrySites {
		data.EntrySites[i] = pclnmap.Site{PCOffset: site.PCOffset, ID: site.ID}
	}
	data.PCSites = make([]pclnmap.Site, len(analysis.PCLineSites))
	pcSiteOwners := make([]string, len(analysis.PCLineSites))
	for i, site := range analysis.PCLineSites {
		data.PCSites[i] = pclnmap.Site{PCOffset: site.PCOffset, ID: site.ID}
		pcSiteOwners[i] = site.OwnerSymbol
	}
	if err := filterExternalPCLNJoins(&data, pcSiteOwners); err != nil {
		return err
	}
	raw, err := pclnmap.Encode(data)
	if err != nil {
		return fmt.Errorf("encode external pclntab: %w", err)
	}

	// Stage and fsync the sidecar first. Detaching mutates/signs the binary;
	// only after that succeeds is the complete sidecar atomically published.
	_ = os.Remove(out.PCLN)
	tmp, err := os.CreateTemp(filepath.Dir(out.PCLN), "."+filepath.Base(out.PCLN)+"-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(0o644); err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := pclnpost.DetachExternal(out.Out, analysis.Identity); err != nil {
		return fmt.Errorf("detach external pclntab: %w", err)
	}
	if err := os.Rename(tmpPath, out.PCLN); err != nil {
		// Do not leave a binary which advertises external metadata without
		// its declared build output when publication itself failed.
		_ = os.Remove(out.Out)
		return err
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "llgo: external pclntab: %d entries, %d pcline sites (%d bytes) -> %s\n",
			len(data.EntrySites), len(data.PCSites), len(raw), out.PCLN)
	}
	return nil
}

func filterExternalPCLNJoins(data *pclnmap.Data, pcSiteOwners []string) error {
	symbols := make(map[uint64]uint32, len(data.SymbolIndex))
	for _, rec := range data.SymbolIndex {
		symbols[rec.SymbolID] = rec.FuncIndex
	}
	filterSymbolSites := func(sites []pclnmap.Site) []pclnmap.Site {
		filtered := sites[:0]
		var previousPC uint64
		for _, site := range sites {
			if symbols[site.ID] != 0 && (len(filtered) == 0 || site.PCOffset != previousPC) {
				filtered = append(filtered, site)
				previousPC = site.PCOffset
			}
		}
		return filtered
	}
	data.EntrySites = filterSymbolSites(data.EntrySites)
	if len(data.EntrySites) == 0 {
		return fmt.Errorf("external pclntab has no entry sites joined to funcinfo")
	}
	if len(pcSiteOwners) != len(data.PCSites) {
		return fmt.Errorf("external pclntab has %d pcline sites but %d owner joins", len(data.PCSites), len(pcSiteOwners))
	}
	pcLines := make(map[uint64]uint32, len(data.Table.PCLines))
	for _, rec := range data.Table.PCLines {
		pcLines[rec.ID] = rec.Func
	}
	pcSites := data.PCSites[:0]
	for i, site := range data.PCSites {
		funcIndex := pcLines[site.ID]
		if funcIndex != 0 && externalOwnerFuncIndex(symbols, pcSiteOwners[i], data.GOOS) == funcIndex {
			pcSites = append(pcSites, site)
		}
	}
	data.PCSites = pcSites
	return nil
}

func externalOwnerFuncIndex(symbols map[uint64]uint32, owner, goos string) uint32 {
	for {
		if index := symbols[funcInfoSymbolID(owner)]; index != 0 {
			return index
		}
		// Mach-O has one C-mangling underscore, while debug/macho's
		// suffix-shared string table can expose one more or less. Try the same
		// candidate sequence as pclnpost.canonicalOwner, stopping as soon as a
		// real funcinfo symbol joins.
		if goos != "darwin" || !strings.HasPrefix(owner, "_") {
			return 0
		}
		owner = owner[1:]
	}
}
