package build

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/pclnmap"
)

func TestPCLNEmbeddedPruningIntegration(t *testing.T) {
	requireNativePCLNSidecars(t)
	setPCLNIntegrationEnv(t)
	t.Setenv("LLGO_PCLNPOST", "1")
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			for _, pureC := range []bool{true, false} {
				name, source := "runtime", pclntabModesFixture
				if pureC {
					name, source = "pure-c", pclntabPureCLibraryFixture
				}
				t.Run(name, func(t *testing.T) {
					// This compiled declaration has no live body after section GC.
					// Inspect the actual image, not the in-memory prune result.
					source += "\n//go:noinline\nfunc deadMetadataOwner2420() uintptr { return 2420 }\n"
					bin := filepath.Join(t.TempDir(), "embedded-pcln")
					conf := &Config{
						Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: bin, LTO: mode,
						PCLNMode: PCLNEmbedded, PCLNModeSet: true,
						LinkOptions: LinkOptions{DWARF: DWARFOmit},
					}
					if _, err := Do([]string{writePCLNIntegrationSource(t, source)}, conf); err != nil {
						t.Fatal(err)
					}
					verifyPCLNIntegrationSignature(t, bin)
					raw, err := os.ReadFile(bin)
					if err != nil {
						t.Fatal(err)
					}
					if bytes.Contains(raw, []byte("deadMetadataOwner2420")) {
						t.Fatal("linked image retains dead function metadata")
					}
					if pureC {
						if got := runPCLNIntegrationBinary(t, bin); got != "pure-c-pclntab\n" {
							t.Fatalf("pure C output = %q", got)
						}
						if pclnIntegrationHasLLGoRuntime(t, bin) {
							t.Fatal("pruning retained an unused LLGo runtime")
						}
					} else {
						if got := runPCLNIntegrationBinary(t, bin, "apis"); got != "caller=FULL frames=FULL func=FULL\n" {
							t.Fatalf("runtime API states = %q", got)
						}
						if got := runPCLNIntegrationBinary(t, bin, "closure"); got != "target=FULL funcval=FULL same-entry=true\n" {
							t.Fatalf("function-value metadata = %q", got)
						}
					}
				})
			}
		})
	}
}

func TestPCLNExternalPruningIntegration(t *testing.T) {
	requireNativePCLNSidecars(t)
	setPCLNIntegrationEnv(t)
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			for _, pureC := range []bool{true, false} {
				name, source := "runtime", pclntabModesFixture
				if pureC {
					name, source = "pure-c", pclntabPureCLibraryFixture
				}
				t.Run(name, func(t *testing.T) {
					bin := filepath.Join(t.TempDir(), "pruned-pcln")
					conf := &Config{
						Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: bin, LTO: mode,
						PCLNMode: PCLNExternal, PCLNModeSet: true,
						LinkOptions: LinkOptions{DWARF: DWARFOmit},
					}
					if _, err := Do([]string{writePCLNIntegrationSource(t, source)}, conf); err != nil {
						t.Fatal(err)
					}
					verifyPCLNIntegrationSignature(t, bin)
					if pureC {
						if got := runPCLNIntegrationBinary(t, bin); got != "pure-c-pclntab\n" {
							t.Fatalf("pure C output = %q", got)
						}
						if pclnIntegrationHasLLGoRuntime(t, bin) {
							t.Fatal("pruning linked an otherwise unused LLGo runtime")
						}
					} else {
						if got := runPCLNIntegrationBinary(t, bin, "apis"); got != "caller=FULL frames=FULL func=FULL\n" {
							t.Fatalf("runtime API states after pruning = %q", got)
						}
						if got := runPCLNIntegrationBinary(t, bin, "closure"); got != "target=FULL funcval=FULL same-entry=true\n" {
							t.Fatalf("function-value metadata after pruning = %q", got)
						}
					}
					assertExternalPCLNHasNoDeadRecords(t, bin)
				})
			}
		})
	}
}

func assertExternalPCLNHasNoDeadRecords(t *testing.T, bin string) {
	t.Helper()
	raw, err := os.ReadFile(bin + ".pclntab")
	if err != nil {
		t.Fatal(err)
	}
	view, err := pclnmap.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := pclnIntegrationBinaryIdentity(t, bin); got != view.Identity {
		t.Fatalf("binary/sidecar identities differ: %x != %x", got, view.Identity)
	}
	// v4 descriptors: functions, PC lines, strings, offsets, hash, symbol
	// index, entries, PC sites. Inspect the actual published wire payload.
	readRows := func(section int, size uint64, visit func([]byte)) {
		desc := view.Sections[section]
		for i := uint64(0); i < desc.Count; i++ {
			off := desc.Offset + i*size
			visit(raw[off : off+size])
		}
	}
	symbols := make(map[uint64]uint32)
	readRows(5, 16, func(row []byte) {
		symbols[binary.LittleEndian.Uint64(row)] = binary.LittleEndian.Uint32(row[8:])
	})
	liveFunctions := make(map[uint32]bool)
	readRows(6, 16, func(row []byte) {
		liveFunctions[symbols[binary.LittleEndian.Uint64(row[8:])]] = true
	})
	livePC := make(map[uint64]bool)
	readRows(7, 16, func(row []byte) {
		livePC[binary.LittleEndian.Uint64(row[8:])] = true
	})
	readRows(1, 24, func(row []byte) {
		id := binary.LittleEndian.Uint64(row)
		if !livePC[id] {
			t.Fatalf("PC-line %#x has no surviving site", id)
		}
		liveFunctions[binary.LittleEndian.Uint32(row[8:])] = true
	})
	for i := uint32(1); uint64(i) <= view.Sections[0].Count; i++ {
		if !liveFunctions[i] {
			t.Fatalf("function record %d has no surviving entry or PC-line site (%d records)", i, view.Sections[0].Count)
		}
	}
	t.Logf("sidecar: %d bytes, %d functions, %d PC lines, %d entry sites",
		len(raw), view.Sections[0].Count, view.Sections[1].Count, view.Sections[6].Count)
}
