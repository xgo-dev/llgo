/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package directive

import (
	"fmt"
	"go/ast"
	"strings"
)

func scanGroup(doc *ast.CommentGroup) *Group {
	g := &Group{Items: ParseGroup(doc)}
	for _, d := range g.Items {
		switch d.Name {
		case "llgo:env":
			g.Function.ClosureEnv = true
		case "go:nosplit":
			g.Function.NoSplit = true
		case "go:noinline":
			g.Function.NoInline = true
		case "go:uintptrescapes":
			g.Function.UintptrEscapes = true
		case "go:wasmimport":
			// A malformed last occurrence suppresses an earlier valid import.
			g.Function.WasmImport = nil
			if fields := strings.Fields(d.Args); len(fields) == 2 {
				g.Function.WasmImport = &WasmImport{fields[0], fields[1]}
			}
		case "go:linkname", "llgo:link":
			if fields := strings.Fields(d.Args); len(fields) >= 2 {
				g.Links = append(g.Links, Link{Local: fields[0], Target: strings.Join(fields[1:], " "), Pos: d.Pos})
			}
		case "export":
			if d.Args != "" {
				g.Links = append(g.Links, Link{Local: d.Args, Target: d.Args, Export: true, Pos: d.Pos})
			}
		}
	}
	for i := len(doc.List) - 1; i >= 0; i-- {
		if doc.List[i] == nil {
			continue
		}
		line := doc.List[i].Text
		if line == "//go:nointerface" {
			g.Function.NoInterface = true
			break
		}
		if !strings.HasPrefix(line, "//go:") {
			break
		}
	}
	for i := len(doc.List) - 1; i >= 0; i-- {
		if doc.List[i] == nil {
			continue
		}
		all, names, ok := LegacySkip(doc.List[i].Text)
		if !ok {
			break
		}
		g.Skip.All = g.Skip.All || all
		g.Skip.Names = append(g.Skip.Names, names...)
	}
	if len(doc.List) > 0 && doc.List[len(doc.List)-1] != nil {
		line := doc.List[len(doc.List)-1].Text
		g.LastLine = line
		_, _, g.LastSkip = LegacySkip(line)
		for _, prefix := range []string{"//llgo:type ", "// llgo:type "} {
			if strings.HasPrefix(line, prefix) {
				g.TypeBackground = strings.TrimSpace(line[len(prefix):])
				break
			}
		}
	}
	g.Embed = scanEmbed(doc)
	return g
}

// DeclarationLink follows the preloader's reverse-order precedence. Export
// name validation depends on the declaration and is deferred until association.
func (g *Group) DeclarationLink(name string, exportRename bool) (Link, bool, error) {
	for i := len(g.Links) - 1; i >= 0; i-- {
		l := g.Links[i]
		if l.Export {
			if l.Local != name && !exportRename {
				return Link{}, false, fmt.Errorf("export comment has wrong name %q", l.Local)
			}
			return l, true, nil
		}
		if l.Local == name {
			return l, true, nil
		}
	}
	return Link{}, false, nil
}
func packageLink(c *ast.Comment) (Link, bool) {
	const prefix = "//go:linkname "
	if !strings.HasPrefix(c.Text, prefix) {
		return Link{}, false
	}
	fields := strings.Fields(c.Text[len(prefix):])
	if len(fields) < 2 {
		return Link{}, false
	}
	return Link{Local: fields[0], Target: strings.Join(fields[1:], " "), Pos: c.Pos()}, true
}
func isInternal(d Directive) bool { return strings.HasPrefix(d.Name, "llgointernal:") }

// LegacySkip retains the trailing contiguous annotation rule used by package
// patches. Other go:/llgo: lines count as annotations without adding skips.
func LegacySkip(line string) (all bool, names []string, annotation bool) {
	if strings.HasPrefix(line, "//go:") {
		return false, nil, true
	}
	var tail string
	switch {
	case strings.HasPrefix(line, "//llgo:"):
		tail = line[len("//llgo:"):]
	case strings.HasPrefix(line, "// llgo:"):
		tail = line[len("// llgo:"):]
	default:
		return
	}
	if tail == "skipall" {
		return true, nil, true
	}
	if strings.HasPrefix(tail, "skip ") {
		for _, name := range strings.Split(tail[len("skip "):], " ") {
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return false, names, true
}

// SourcePatch recognizes load-time patch commands. Unlike LegacySkip, these
// apply anywhere in a patch file and accept whitespace-separated names.
func SourcePatch(line string) (all bool, names []string, ok bool) {
	line = strings.TrimSpace(line)
	var tail string
	switch {
	case strings.HasPrefix(line, "//llgo:"):
		tail = line[len("//llgo:"):]
	case strings.HasPrefix(line, "// llgo:"):
		tail = line[len("// llgo:"):]
	default:
		return
	}
	if tail == "skipall" {
		return true, nil, true
	}
	if strings.HasPrefix(tail, "skip ") {
		return false, strings.Fields(tail[len("skip "):]), true
	}
	return
}
