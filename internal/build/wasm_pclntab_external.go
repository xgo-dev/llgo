package build

import (
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
	identity, err := wasmdebug.PCLNModuleIdentity(raw)
	if err != nil {
		return err
	}
	data, err := linkExternalWasmPCLNData(ctx, raw)
	if err != nil {
		return err
	}
	// Function-table indexes remain in the small linked entry-site array.
	// Shadow locations use stable IDs, independent of final code offsets.
	data.EntrySites, data.PCSites = nil, nil
	data.ImageBase, data.TextStart, data.TextEnd = 0, 1, 2
	data.Identity = identity
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

func linkExternalWasmPCLNData(ctx *context, raw []byte) (pclnmap.Data, error) {
	data := *ctx.pclnExternal
	if ctx.frontendOptions.CompactCallerLocations && len(data.Table.PCLines) != 0 {
		// An inlined frame can survive even when its original function is
		// absent from the Wasm name section.
		data.PCSites = make([]pclnmap.Site, 0, len(data.Table.PCLines))
		for _, line := range data.Table.PCLines {
			data.PCSites = append(data.PCSites, pclnmap.Site{ID: line.ID})
		}
		if canPruneWasmTypeMetadata(ctx) && ctx.linkRuntimeSites != nil && !shouldEmitRuntimeSites(ctx) {
			return pclnmap.Prune(data)
		}
	}
	analysis, err := analyzeWasmPCLNData(raw, data)
	if err != nil {
		return pclnmap.Data{}, err
	}
	return linkedPCLNData(data, analysis)
}
