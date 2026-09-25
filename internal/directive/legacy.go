package directive

import (
	"os"
	"strings"
)

const (
	NoDirective = iota
	HasLinkname
	UnknownDirective = -1
)

type LegacyLink struct {
	Link
	Status int
	Valid  bool
	Raw    string
}

// ParseLegacyLink preserves the standalone compiler/importer's historical
// space-sensitive grammar. Package preloading uses Group.DeclarationLink.
func ParseLegacyLink(line string, allowExport bool) LegacyLink {
	r := LegacyLink{Raw: line}
	var text string
	switch {
	case strings.HasPrefix(line, "//go:linkname "):
		text = line[len("//go:linkname "):]
	case strings.HasPrefix(line, "// llgo:link "):
		text = line[len("// llgo:link "):]
	case strings.HasPrefix(line, "//llgo:link "):
		text = line[len("//llgo:link "):]
	case allowExport && strings.HasPrefix(line, "//export "):
		text = line[len("//export "):] + " " + strings.TrimSpace(line[len("//export "):])
		r.Export = true
	case strings.HasPrefix(line, "//go:"):
		r.Status = UnknownDirective
		return r
	default:
		return r
	}
	r.Status = HasLinkname
	text = strings.TrimSpace(text)
	if idx := strings.IndexByte(text, ' '); idx > 0 {
		r.Local = text[:idx]
		r.Target = strings.TrimLeft(text[idx+1:], " ")
		r.Valid = true
	}
	return r
}

// ReadLegacyLinks is the single source-discovery fallback for standalone
// compiler clients that supply imported type objects without dependency ASTs.
// The driver path supplies File records instead. Failed reads are cached too.
func (s *Index) ReadLegacyLinks(filename string) []LegacyLink {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.imports == nil {
		s.imports = make(map[string][]LegacyLink)
	}
	if r, ok := s.imports[filename]; ok {
		return r
	}
	if s.frozen {
		panic("import directives were not prepared before lowering")
	}
	var links []LegacyLink
	if data, err := os.ReadFile(filename); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(line, "//") {
				continue
			}
			r := ParseLegacyLink(line, true)
			if r.Status == HasLinkname {
				links = append(links, r)
			}
		}
	}
	s.imports[filename] = links
	return links
}
