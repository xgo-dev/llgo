package pclnpost

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func symbolCarrierFixture(t *testing.T) string {
	t.Helper()
	path := machoRewriteFixture(t, 65536)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	// Replace the two body records with a data-only reservation. Keep the
	// independent catalog and its meta record, as in the synthetic root module.
	carrier := info.entrySec[48:80]
	clear(carrier)
	binary.LittleEndian.PutUint64(carrier, SymbolEntryCarrierMagic)
	binary.LittleEndian.PutUint64(carrier[8:], info.entryVMAddr+48)
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
	// Foreign text is not in the Go catalog. Mach-O's leading underscores are
	// accepted only when the complete remaining name hashes to a catalog ID.
	info.syms[0].name = "___example.com/p.A"
	info.syms = append(info.syms, textSym{addr: info.textStart + 0x20, name: "_foreign"})
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
