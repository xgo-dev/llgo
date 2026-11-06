// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package types

import (
	"strconv"
	"strings"
)

func llgoLayoutTag(tag string, attrs LLGoTypeAttrs, padding string) string {
	tag = LLGoFieldTag(tag)
	if !attrs.Native && attrs.Pack == 0 && padding == "" {
		return tag
	}
	native := attrs.CallConv
	if attrs.Native && native == "" {
		native = "C"
	}
	return tag + llgoLayoutTagPrefix + "n=" + native + ";p=" + strconv.FormatInt(attrs.Pack, 10) + ";f=" + padding + `"`
}

func llgoTagLayout(tag string) (attrs LLGoTypeAttrs, padding string, ok bool) {
	i := strings.LastIndex(tag, llgoLayoutTagPrefix)
	if i < 0 || !strings.HasSuffix(tag, `"`) {
		return
	}
	for _, item := range strings.Split(tag[i+len(llgoLayoutTagPrefix):len(tag)-1], ";") {
		key, value, _ := strings.Cut(item, "=")
		switch key {
		case "n":
			attrs.Native, attrs.CallConv = value != "", value
		case "p":
			attrs.Pack, _ = strconv.ParseInt(value, 10, 64)
		case "f":
			padding = value
		}
	}
	ok = true
	return
}
