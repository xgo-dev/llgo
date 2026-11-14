package pclnpost

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func symbolCarrierFixture(t *testing.T) string {
	t.Helper()
	return symbolCarrierFixtureNamed(t, "example.com/p.A", "example.com/p.B")
}

func symbolCarrierFixtureNamed(t *testing.T, nameA, nameB string) string {
	t.Helper()
	path := machoRewriteFixtureNamed(t, 65536, nameA, nameB)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	// Replace the two body records with a data-only reservation. Keep the
	// independent catalog and its meta record, as in the synthetic root module.
	carrier := info.entrySec[48:80]
	clear(carrier)
	binary.LittleEndian.PutUint64(carrier[16:], SymbolEntryCarrierMagic)
	binary.LittleEndian.PutUint64(carrier[24:], info.entryVMAddr+64)
	if err := os.WriteFile(path, info.raw, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSymbolEntryCarrierJoinsFinalSymbols(t *testing.T) {
	path := symbolCarrierFixture(t)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	// Foreign text is not in the Go catalog. debug/macho has already removed
	// the ABI prefix from dotted names, so another underscore is significant.
	info.syms = append(info.syms, textSym{addr: info.textStart + 0x20, name: "_foreign"})
	info.syms = append(info.syms, textSym{addr: info.textStart + 0x30, name: "_example.com/p.A"})
	entries, err := entryRecords(path, info)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].symbolID != fnv64("example.com/p.A") ||
		entries[1].symbolID != fnv64("example.com/p.B") || entries[0].pc != info.syms[0].addr {
		t.Fatalf("final symbol join = %+v", entries)
	}
	info.syms = nil
	if entries, err := entryRecords(path, info); err != nil || len(entries) != 0 {
		t.Fatalf("missing final symbols fabricate entries: %+v, %v", entries, err)
	}
}

func TestSymbolEntryCarrierPreservesUnderscoreLinknames(t *testing.T) {
	path := symbolCarrierFixtureNamed(t, "_probe", "__probe")
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	// A foreign C function named probe must not select Go's _probe either.
	info.syms = append(info.syms, textSym{addr: info.textStart + 0x20, name: "_probe"})
	entries, err := entryRecords(path, info)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].symbolID != fnv64("_probe") || entries[1].symbolID != fnv64("__probe") {
		t.Fatalf("distinct linknames were merged or foreign text matched the catalog: %+v", entries)
	}
	if _, err := Rewrite(path); err != nil {
		t.Fatal(err)
	}
}

func TestRewriteSymbolEntryCarrier(t *testing.T) {
	path := symbolCarrierFixture(t)
	st, err := Rewrite(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.EntryRecords != 2 || st.Kept != 2 || st.FtabEntries != 3 {
		t.Fatalf("symbol rewrite = %+v", st)
	}
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(info.entrySec) != prebuiltMagic {
		t.Fatal("symbol reservation was not replaced by the runtime table")
	}
}

func TestSymbolEntryCarrierRequiresCatalog(t *testing.T) {
	path := symbolCarrierFixture(t)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	// Remove the catalog's meta record without altering the carrier header.
	clear(info.entrySec[:48])
	if err := os.WriteFile(path, info.raw, 0755); err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(info.raw)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("symbol-only carrier accepted without a Go catalog")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed symbol rewrite mutated the executable")
	}
}
