package abi

import "github.com/xgo-dev/llgo/internal/funcattrs"

func indirectAttributeMapping(params int) funcattrs.ABIMapping {
	m := funcattrs.ABIMapping{Result: funcattrs.ABIValue{Kind: funcattrs.Indirect, Indices: []int{1}}, Params: make([]funcattrs.ABIValue, params)}
	for i := range m.Params {
		m.Params[i] = funcattrs.DirectValue(i + 2)
	}
	return m
}
