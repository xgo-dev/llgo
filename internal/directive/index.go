package directive

import (
	"go/ast"
	"go/token"
	"sync"
)

// Index prepares and indexes directive records for one compilation. AST identity
// is the key: overlays and reparsed patch files are distinct source views.
// Freeze ends preparation; published records are immutable. No records or LLVM
// values are shared between independent compilations.
type Index struct {
	frozen    bool
	imports   map[string][]LegacyLink
	mu        sync.Mutex
	files     map[*ast.File]*File
	groups    map[*ast.CommentGroup]*Group
	functions map[*ast.FuncDecl]Function
}

// File contains source-order file directives and declaration associations.
// All slices and maps are read-only after publication by Index.File.
type File struct {
	EmbedComments map[*ast.Comment]bool
	Syntax        *ast.File
	Groups        map[*ast.CommentGroup]*Group
	Functions     map[*ast.FuncDecl]Function
	GoLinks       []Link
	PatchSkip     Skip
	Internal      []Directive
}

// Group preserves the different attachment and spelling rules of the existing
// language. These facts are computed once, including negative/invalid results.
type Group struct {
	Items          []Directive
	Function       Function
	Links          []Link
	TypeBackground string
	Skip           Skip
	LastSkip       bool
	LastLine       string
	Embed          Embed
}

type Function struct {
	Cold           bool
	NoReturn       bool
	ClosureEnv     bool
	NoInline       bool
	NoSplit        bool
	UintptrEscapes bool
	NoInterface    bool
	WasmImport     *WasmImport
}
type WasmImport struct{ Module, Name string }
type Link struct {
	Local, Target string
	Export        bool
	Pos           token.Pos
}
type Skip struct {
	All   bool
	Names []string
}

func (s *Index) Group(doc *ast.CommentGroup) *Group {
	if doc == nil {
		return &Group{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.group(doc)
}
func (s *Index) group(doc *ast.CommentGroup) *Group {
	if doc == nil {
		return &Group{}
	}
	if s.groups == nil {
		s.groups = make(map[*ast.CommentGroup]*Group)
	}
	if g := s.groups[doc]; g != nil {
		return g
	}
	if s.frozen {
		panic("directive group was not prepared before lowering")
	}
	g := scanGroup(doc)
	s.groups[doc] = g
	return g
}
func (s *Index) File(file *ast.File) *File {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		s.files = make(map[*ast.File]*File)
	}
	if f := s.files[file]; f != nil {
		return f
	}
	if s.frozen {
		panic("directive file was not prepared before lowering")
	}
	f := &File{EmbedComments: make(map[*ast.Comment]bool), Syntax: file, Groups: make(map[*ast.CommentGroup]*Group), Functions: make(map[*ast.FuncDecl]Function)}
	add := func(g *ast.CommentGroup) {
		if g != nil {
			f.Groups[g] = s.group(g)
		}
	}
	for _, g := range file.Comments {
		if g == nil {
			continue
		}
		add(g)
		for _, c := range g.List {
			if c == nil {
				continue
			}
			if link, ok := packageLink(c); ok {
				f.GoLinks = append(f.GoLinks, link)
			}
			if IsEmbedComment(c) {
				f.EmbedComments[c] = true
			}
			if all, names, ok := SourcePatch(c.Text); ok {
				f.PatchSkip.All = f.PatchSkip.All || all
				f.PatchSkip.Names = append(f.PatchSkip.Names, names...)
			}
		}
		for _, d := range f.Groups[g].Items {
			if isInternal(d) {
				f.Internal = append(f.Internal, d)
			}
		}
	}
	// Include Doc groups in programmatically constructed ASTs as well. Nested
	// declarations are indexed for locality and embed placement diagnostics.
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			add(n.Doc)
			f.Functions[n] = s.group(n.Doc).Function
			if s.functions == nil {
				s.functions = make(map[*ast.FuncDecl]Function)
			}
			s.functions[n] = f.Functions[n]
		case *ast.GenDecl:
			add(n.Doc)
		case *ast.ValueSpec:
			add(n.Doc)
			add(n.Comment)
		case *ast.TypeSpec:
			add(n.Doc)
			add(n.Comment)
		case *ast.ImportSpec:
			add(n.Doc)
			add(n.Comment)
		}
		return true
	})
	s.files[file] = f
	return f
}
func (s *Index) Files(files []*ast.File) []*File {
	out := make([]*File, 0, len(files))
	for _, f := range files {
		if f != nil {
			out = append(out, s.File(f))
		}
	}
	return out
}
func (f *File) Group(g *ast.CommentGroup) *Group {
	if r := f.Groups[g]; r != nil {
		return r
	}
	return &Group{}
}
func (g *Group) Has(name string) bool {
	for _, d := range g.Items {
		if d.Name == name {
			return true
		}
	}
	return false
}

// Function prepares a declaration when a standalone SSA client supplies syntax
// without files. Normal build clients already registered it through File.
func (s *Index) Function(d *ast.FuncDecl) Function {
	if d == nil {
		return Function{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.functions[d]; ok {
		return p
	}
	if s.frozen {
		panic("function directives were not prepared before lowering")
	}
	if s.functions == nil {
		s.functions = make(map[*ast.FuncDecl]Function)
	}
	p := s.group(d.Doc).Function
	s.functions[d] = p
	return p
}

// LookupFunction reads a prepared standalone declaration without discovery.
func (s *Index) LookupFunction(d *ast.FuncDecl) (Function, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.functions[d]
	return p, ok
}

// Freeze closes discovery at the coordinator/worker boundary. Existing records
// remain readable; a missed preload is diagnosed instead of silently reparsed.
func (s *Index) Freeze() { s.mu.Lock(); s.frozen = true; s.mu.Unlock() }
