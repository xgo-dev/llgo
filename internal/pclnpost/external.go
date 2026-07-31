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

package pclnpost

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

const (
	// ExternalFormatELF and ExternalFormatMachO are the values returned in
	// ExternalAnalysis.Format.
	ExternalFormatELF   = "elf"
	ExternalFormatMachO = "macho"
)

// ExternalSite is one final linked site. PCOffset is relative to the image
// base, so it remains valid after ASLR. ID is a funcinfo symbol ID for an
// entry site and a pcline record ID for a PC-line site. PC-line sites also
// carry the final linked owner in OwnerSymbol; it is a post-link join key and
// is not serialized into the sidecar.
type ExternalSite struct {
	PCOffset    uint64
	ID          uint64
	OwnerSymbol string
}

// ExternalAnalysis is the immutable input for an external pclntab sidecar.
// TextStart and TextEnd are link-time virtual addresses; sites are expressed
// relative to ImageBase after LTO-copy deduplication against the final symbol
// table. Identity is SHA-256 of the complete, unmodified linked binary.
type ExternalAnalysis struct {
	Format      string
	PointerSize int
	ImageBase   uint64
	TextStart   uint64
	TextEnd     uint64
	Identity    [sha256.Size]byte

	EntrySites  []ExternalSite
	PCLineSites []ExternalSite

	EntryRecords  int
	PCLineRecords int
	InlineCopies  int
	NoSymbol      int
}

// AnalyzeExternal reads a linked ELF or Mach-O executable without modifying
// it. It resolves Mach-O chained pointer slots, deduplicates function
// records against final text symbols, and returns deterministic image-base-
// relative site lists suitable for an external sidecar.
func AnalyzeExternal(path string) (ExternalAnalysis, error) {
	var out ExternalAnalysis
	info, err := load(path)
	if err != nil {
		return out, err
	}
	if err := validateExternalLayout(info); err != nil {
		return out, err
	}
	if len(info.entrySec) >= 8 {
		if magic := binary.LittleEndian.Uint64(info.entrySec); magic == prebuiltMagic {
			return out, fmt.Errorf("entry sites have already been rewritten")
		}
	}

	entries := parseRecords(info, info.entrySec)
	pcLines := parseRecords(info, info.pcLineSec)
	out.EntryRecords = len(entries)
	out.PCLineRecords = len(pcLines)
	if len(entries) == 0 {
		return out, fmt.Errorf("no entry records")
	}
	kept, inline, noSymbol := dedupe(info, entries, false)
	if len(kept) == 0 {
		return out, fmt.Errorf("no records survived dedup")
	}
	out.Format = info.format
	out.PointerSize = info.pointerSize
	out.ImageBase = info.imageBase
	out.TextStart = info.textStart
	out.TextEnd = info.textEnd
	out.Identity = sha256.Sum256(info.raw)
	out.InlineCopies = inline
	out.NoSymbol = noSymbol
	out.EntrySites = externalSites(info, kept)
	out.PCLineSites = externalPCLineSites(info, pcLines)
	return out, nil
}

func validateExternalLayout(info *binaryInfo) error {
	if info.format != ExternalFormatELF && info.format != ExternalFormatMachO {
		return fmt.Errorf("unsupported binary format %q", info.format)
	}
	if info.pointerSize != 8 {
		return fmt.Errorf("external pclntab requires a 64-bit binary (pointer size %d)", info.pointerSize)
	}
	if !info.littleEndian {
		return fmt.Errorf("external pclntab requires a little-endian binary")
	}
	if info.textEnd <= info.textStart {
		return fmt.Errorf("missing or invalid text range [%#x,%#x)", info.textStart, info.textEnd)
	}
	if info.imageBase > info.textStart {
		return fmt.Errorf("image base %#x is above text start %#x", info.imageBase, info.textStart)
	}
	return nil
}

// externalSites validates, orders, and pair-deduplicates final sites. PC-line
// IDs cannot be deduplicated against their owning function here because that
// association lives in the separately encoded funcinfo table; distinct IDs
// at the same PC are therefore retained for that later join.
func externalSites(info *binaryInfo, records []siteRecord) []ExternalSite {
	sites := make([]ExternalSite, 0, len(records))
	for _, record := range records {
		if record.pc < info.textStart || record.pc >= info.textEnd || record.pc < info.imageBase {
			continue
		}
		sites = append(sites, ExternalSite{
			PCOffset: record.pc - info.imageBase,
			ID:       record.symbolID,
		})
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].PCOffset != sites[j].PCOffset {
			return sites[i].PCOffset < sites[j].PCOffset
		}
		return sites[i].ID < sites[j].ID
	})
	if len(sites) < 2 {
		return sites
	}
	out := sites[:1]
	for _, site := range sites[1:] {
		last := out[len(out)-1]
		if site.PCOffset == last.PCOffset && site.ID == last.ID {
			continue
		}
		out = append(out, site)
	}
	return out
}

func externalPCLineSites(info *binaryInfo, records []siteRecord) []ExternalSite {
	sites := make([]ExternalSite, 0, len(records))
	for _, record := range records {
		if record.pc < info.textStart || record.pc >= info.textEnd || record.pc < info.imageBase {
			continue
		}
		sym, ok := owner(info, record.pc)
		if !ok {
			continue
		}
		sites = append(sites, ExternalSite{
			PCOffset:    record.pc - info.imageBase,
			ID:          record.symbolID,
			OwnerSymbol: sym.name,
		})
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].PCOffset != sites[j].PCOffset {
			return sites[i].PCOffset < sites[j].PCOffset
		}
		if sites[i].ID != sites[j].ID {
			return sites[i].ID < sites[j].ID
		}
		return sites[i].OwnerSymbol < sites[j].OwnerSymbol
	})
	if len(sites) < 2 {
		return sites
	}
	out := sites[:1]
	for _, site := range sites[1:] {
		last := out[len(out)-1]
		if site.PCOffset == last.PCOffset && site.ID == last.ID && site.OwnerSymbol == last.OwnerSymbol {
			continue
		}
		out = append(out, site)
	}
	return out
}

// DetachExternal verifies that identity still names the unmodified binary,
// writes it into the dedicated 32-byte identity section, and clears all
// link-only entry and PC-line site sections. Mach-O chained fixups in
// those ranges are removed before the bytes are cleared. A signed Mach-O is
// ad-hoc re-signed once, after all mutations, before the replacement is
// published atomically.
func DetachExternal(path string, identity [sha256.Size]byte) error {
	info, err := load(path)
	if err != nil {
		return err
	}
	if err := validateExternalLayout(info); err != nil {
		return err
	}
	if actual := sha256.Sum256(info.raw); actual != identity {
		return fmt.Errorf("binary identity mismatch: executable changed after analysis")
	}
	if info.identityVMSize != sha256.Size {
		return fmt.Errorf("pclntab identity section has size %d, want %d", info.identityVMSize, sha256.Size)
	}
	if info.identityFileOff == 0 {
		return fmt.Errorf("pclntab identity section is not file-backed")
	}
	if info.entryVMSize == 0 {
		return fmt.Errorf("missing entry-site section")
	}

	type fileRange struct {
		name       string
		start, end uint64
	}
	var siteRanges []fileRange
	addRange := func(name string, off, size uint64) error {
		if size == 0 {
			return nil
		}
		if off == 0 {
			return fmt.Errorf("%s section is not file-backed", name)
		}
		siteRanges = append(siteRanges, fileRange{name: name, start: off, end: off + size})
		return nil
	}
	if err := addRange("entry-site", info.entryFileOff, info.entryVMSize); err != nil {
		return err
	}
	if err := addRange("pcline-site", info.pcLineFileOff, info.pcLineVMSize); err != nil {
		return err
	}
	idStart, idEnd := info.identityFileOff, info.identityFileOff+info.identityVMSize
	for _, section := range siteRanges {
		if section.start < idEnd && idStart < section.end {
			return fmt.Errorf("pclntab identity section overlaps %s section", section.name)
		}
	}

	raw := append([]byte(nil), info.raw...)
	if info.format == ExternalFormatMachO {
		ranges := make([][2]uint64, len(siteRanges))
		for i, section := range siteRanges {
			ranges[i] = [2]uint64{section.start, section.end}
		}
		if _, err := unchainRanges(raw, ranges, nil); err != nil {
			return fmt.Errorf("chained fixups: %w", err)
		}
	}
	for _, section := range siteRanges {
		clear(raw[section.start:section.end])
	}
	copy(raw[idStart:idEnd], identity[:])
	return replaceExternalBinary(path, raw, info.format == ExternalFormatMachO && info.hasCodeSignature)
}

func replaceExternalBinary(path string, raw []byte, sign bool) (err error) {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".pclnpost-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		if tmp != nil {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(st.Mode()); err != nil {
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
	tmp = nil
	if sign && runtime.GOOS == "darwin" {
		if output, err := exec.Command("codesign", "-f", "-s", "-", tmpPath).CombinedOutput(); err != nil {
			return fmt.Errorf("codesign: %v: %s", err, output)
		}
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}
