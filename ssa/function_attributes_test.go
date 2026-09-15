package ssa

import "testing"

func TestFunctionAttributeSymbolResolution(t *testing.T) {
	for _, order := range []string{"links first", "attributes first"} {
		t.Run(order, func(t *testing.T) {
			linksFirst := order == "links first"
			prog := NewProgram(nil)
			defer prog.Dispose()
			setLinks := func() {
				prog.SetLinkname("p.Rare", "C.shared")
				prog.SetLinkname("p.Stop", "stdcall.shared")
				prog.SetLinkname("p.Indirect", "p.Rare")
			}
			if linksFirst {
				setLinks()
			}
			prog.SetFunctionAttributes("p.Rare", FunctionCold)
			prog.SetFunctionAttributes("p.Stop", FunctionNoReturn)
			prog.SetFunctionAttributes("p.Indirect", FunctionNoReturn)
			prog.SetFunctionAttributeOrigin("p.Rare[int]", "p.Rare")
			if !linksFirst {
				setLinks()
			}
			for _, test := range []struct {
				name string
				want FunctionAttributes
			}{
				{"p.Rare", FunctionCold | FunctionNoReturn},
				{"p.Stop", FunctionCold | FunctionNoReturn},
				{"shared", FunctionCold | FunctionNoReturn},
				{"p.Rare[int]", FunctionCold | FunctionNoReturn},
				// One linkname mapping does not imply a transitive alias chain.
				{"p.Indirect", FunctionNoReturn},
				{"p.Unknown", 0},
			} {
				if got := prog.functionAttributes(test.name); got != test.want {
					t.Errorf("linksFirst=%v, %s: got %v, want %v", linksFirst, test.name, got, test.want)
				}
			}
		})
	}
}
