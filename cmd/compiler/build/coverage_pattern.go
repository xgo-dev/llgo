// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

// matchLoadedCoveragePackage applies cmd/go's package-pattern rules to a
// package that is already in the build graph. go list deliberately omits
// testdata directories while expanding ..., but -coverpkg matches loaded
// dependencies in those directories too.
func matchLoadedCoveragePackage(pattern, cwd string, p *packages.Package) bool {
	if strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "../") || pattern == "." || pattern == ".." {
		var dir string
		if i := strings.Index(pattern, "..."); i < 0 {
			dir, pattern = pattern, ""
		} else {
			j := strings.LastIndex(pattern[:i], "/")
			dir, pattern = pattern[:j], pattern[j+1:]
		}
		dir = filepath.Join(cwd, dir)
		if pattern == "" {
			return p.Dir == dir
		}
		rel, err := filepath.Rel(dir, p.Dir)
		if err != nil {
			return false
		}
		rel = filepath.ToSlash(rel)
		return rel != ".." && !strings.HasPrefix(rel, "../") && matchCoverageImportPattern(pattern, rel)
	}

	// These names have meanings beyond import-path matching. The go list
	// result used by the caller remains authoritative for them.
	switch pattern {
	case "all", "std", "cmd", "tool", "work":
		return pattern == "all"
	}
	return matchCoverageImportPattern(pattern, p.PkgPath)
}

// matchCoverageImportPattern implements the limited ... glob accepted by Go
// package patterns, including the trailing /... and vendor exceptions.
func matchCoverageImportPattern(pattern, name string) bool {
	const vendorMarker = "\x00"
	if strings.Contains(pattern, vendorMarker) || !utf8.ValidString(pattern) || strings.Contains(name, vendorMarker) {
		return false
	}

	re := regexp.QuoteMeta(pattern)
	re = markCoverageVendorElements(re, vendorMarker)
	switch {
	case strings.HasSuffix(re, `/`+vendorMarker+`/\.\.\.`):
		re = strings.TrimSuffix(re, `/`+vendorMarker+`/\.\.\.`) + `(/vendor|/` + vendorMarker + `/\.\.\.)`
	case re == vendorMarker+`/\.\.\.`:
		re = `(/vendor|/` + vendorMarker + `/\.\.\.)`
	}
	if strings.HasSuffix(re, `/\.\.\.`) {
		re = strings.TrimSuffix(re, `/\.\.\.`) + `(/\.\.\.)?`
	}
	re = strings.ReplaceAll(re, `\.\.\.`, `[^`+vendorMarker+`]*`)
	name = markCoverageVendorElements(name, vendorMarker)
	return regexp.MustCompile(`^` + re + `$`).MatchString(name)
}

func markCoverageVendorElements(path, marker string) string {
	if !strings.Contains(path, "vendor") {
		return path
	}
	elem := strings.Split(path, "/")
	for i := 0; i < len(elem)-1; i++ {
		if elem[i] == "vendor" {
			elem[i] = marker
		}
	}
	return strings.Join(elem, "/")
}
