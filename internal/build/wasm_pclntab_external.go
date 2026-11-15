package build

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func writeExternalWasmPCLN(ctx *context, out *OutFmtDetails, verbose bool) error {
	module := debugWasmModulePath(ctx.buildConf, out.Out)
	raw, err := os.ReadFile(module)
	if err != nil {
		return err
	}
	analysis, err := analyzeWasmPCLNData(raw, *ctx.pclnExternal)
	if err != nil {
		return err
	}
	data, err := linkedPCLNData(*ctx.pclnExternal, analysis)
	if err != nil {
		return err
	}
	// Function-table indexes remain in the small linked entry-site array.
	// Shadow locations use stable IDs, independent of final code offsets.
	data.EntrySites, data.PCSites = nil, nil
	data.ImageBase, data.TextStart, data.TextEnd = 0, 1, 2
	data.Identity = sha256.Sum256(raw)
	placeholder := wasmdebug.PCLNIdentityPlaceholder()
	patched, err := wasmdebug.ReplaceData(raw, pclnIdentitySymbol, placeholder[:], data.Identity[:])
	if err != nil {
		return err
	}
	sidecar, err := pclnmap.Encode(data)
	if err != nil {
		return err
	}
	if err := writeDebugArtifactFile(out.PCLN, sidecar, 0o644); err != nil {
		return err
	}
	if err := writeDebugArtifactFile(module, patched, 0o644); err != nil {
		_ = os.Remove(out.PCLN)
		return err
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "llgo: external Wasm pclntab: %d records, %d bytes -> %s\n", len(data.Table.Records), len(sidecar), out.PCLN)
	}
	return nil
}
