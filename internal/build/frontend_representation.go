// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"fmt"
	"github.com/xgo-dev/llgo/internal/frontcache"
	llssa "github.com/xgo-dev/llgo/ssa"
	"go/types"
)

func exportFrontendRepresentation(prog llssa.Program, graph *frontcache.TypeGraph) {
	graph.ExportRepresentations(func(obj *types.TypeName) (int, types.Type) {
		representation := prog.RepresentationOf(obj.Type())
		if representation.Storage == llssa.NativeRepresentation {
			return int(representation.NativeABI), representation.Origin.Type()
		}
		return 0, nil
	})
}

func importFrontendRepresentation(prog llssa.Program, importer *frontcache.TypeImporter) error {
	representations, err := importer.ImportRepresentations()
	if err != nil {
		return err
	}
	for _, representation := range representations {
		if representation.ABI != 0 && llssa.CallConv(representation.ABI) != llssa.CDecl && llssa.CallConv(representation.ABI) != llssa.Stdcall {
			return fmt.Errorf("unsupported cached native ABI %d", representation.ABI)
		}
	}
	layouts, err := importer.ImportLayouts()
	if err != nil {
		return err
	}
	for typ := range layouts {
		prog.SetNativeAnonymousLayout(typ)
	}
	for obj, imported := range representations {
		r := llssa.TypeRepresentation{}
		if imported.ABI != 0 {
			r.Storage, r.NativeABI, r.Origin = llssa.NativeRepresentation, llssa.CallConv(imported.ABI), imported.Origin
		}
		prog.SetImportedRepresentation(obj, r)
	}
	return nil
}
