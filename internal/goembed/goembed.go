package goembed

import (
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/mod/module"
)

type FileData struct {
	Name string
	Data []byte
}

type VarData struct {
	Files []FileData
}

type VarMap map[string]VarData

func LoadDirectives(fset *token.FileSet, files []*ast.File) (VarMap, error) {
	return LoadRecords(fset, new(directive.Store).Files(files))
}

// LoadRecords resolves files from previously parsed directive records.
func LoadRecords(fset *token.FileSet, files []*directive.File) (VarMap, error) {
	if len(files) == 0 {
		return nil, nil
	}
	byVar := make(VarMap)
	for _, source := range files {
		file := source.Syntax
		if file == nil {
			continue
		}
		fileName := positionFor(fset, file.Pos()).Filename
		if fileName == "" {
			continue
		}
		pkgDir := filepath.Dir(fileName)
		hasEmbedImport := FileImportsEmbed(file)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			if len(gen.Specs) > 1 {
				if _, hasDirective, _ := directive.EmbedPatterns(source.Group(gen.Doc)); hasDirective {
					pos := positionFor(fset, gen.Doc.Pos())
					return nil, fmt.Errorf("%s: misplaced go:embed directive", pos)
				}
			}
			for _, spec0 := range gen.Specs {
				spec, ok := spec0.(*ast.ValueSpec)
				if !ok {
					continue
				}
				docs := []*directive.Group{source.Group(spec.Doc)}
				if len(gen.Specs) == 1 {
					docs = append([]*directive.Group{source.Group(gen.Doc)}, docs...)
				}
				patterns, hasDirective, err := directive.EmbedPatterns(docs...)
				if err != nil {
					pos := positionFor(fset, spec.Pos())
					return nil, fmt.Errorf("%s: %v", pos, err)
				}
				if !hasDirective {
					continue
				}
				if len(spec.Names) != 1 {
					pos := positionFor(fset, spec.Pos())
					return nil, fmt.Errorf("%s: go:embed cannot apply to multiple vars", pos)
				}
				if !hasEmbedImport {
					pos := positionFor(fset, spec.Pos())
					return nil, fmt.Errorf("%s: go:embed only allowed in Go files that import \"embed\"", pos)
				}
				resolvedFiles, err := ResolvePatterns(pkgDir, patterns)
				if err != nil {
					pos := positionFor(fset, spec.Pos())
					return nil, fmt.Errorf("%s: %v", pos, err)
				}
				byVar[spec.Names[0].Name] = VarData{Files: resolvedFiles}
			}
		}
	}
	return byVar, nil
}

func FileImportsEmbed(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}
		path, err := strconv.Unquote(imp.Path.Value)
		if err == nil && path == "embed" {
			return true
		}
	}
	return false
}

// ParsePatterns is the standalone API; compilation clients use LoadRecords.
func ParsePatterns(docs ...*ast.CommentGroup) ([]string, bool, error) {
	store := new(directive.Store)
	groups := make([]*directive.Group, 0, len(docs))
	for _, doc := range docs {
		groups = append(groups, store.Group(doc))
	}
	return directive.EmbedPatterns(groups...)
}
func ParseDirective(line string) (string, bool) { return directive.EmbedDirective(line) }
func SplitArgs(s string) ([]string, error)      { return directive.SplitEmbedArgs(s) }

func ResolvePatterns(pkgDir string, patterns []string) ([]FileData, error) {
	var pat string

	wrapErr := func(err error) error {
		return fmt.Errorf("pattern %s: %w", pat, err)
	}

	seen := make(map[string][]byte)
	have := make(map[string]int)
	dirOK := make(map[string]bool)
	pid := 0

	addFile := func(abs, rel string) error {
		if _, ok := seen[rel]; ok {
			return nil
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		seen[rel] = data
		return nil
	}

	for _, pat = range patterns {
		pid++
		all := false
		if strings.HasPrefix(pat, "all:") {
			all = true
			pat = strings.TrimPrefix(pat, "all:")
		}
		if _, err := path.Match(pat, ""); err != nil || !ValidPattern(pat) {
			return nil, wrapErr(fmt.Errorf("invalid pattern syntax"))
		}

		absPattern := filepath.Join(pkgDir, filepath.FromSlash(pat))
		matches, _ := filepath.Glob(absPattern)

		listCount := 0
		for _, m := range matches {
			info, rel, err := CheckPath(pkgDir, m, dirOK)
			if err != nil {
				return nil, wrapErr(err)
			}
			what := "file"
			if info.IsDir() {
				what = "directory"
			}
			switch {
			case info.Mode().IsRegular():
				if have[rel] != pid {
					have[rel] = pid
					listCount++
				}
				if err := addFile(m, rel); err != nil {
					return nil, wrapErr(err)
				}
			case info.IsDir():
				count := 0
				err := filepath.WalkDir(m, func(cur string, d fs.DirEntry, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					rel, _ := RelPath(pkgDir, cur)
					name := d.Name()
					if cur != m && (IsBadName(name) || ((name[0] == '.' || name[0] == '_') && !all)) {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
					if d.IsDir() {
						if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
							return filepath.SkipDir
						}
						return nil
					}
					if !d.Type().IsRegular() {
						return nil
					}
					count++
					if have[rel] != pid {
						have[rel] = pid
						listCount++
					}
					return addFile(cur, rel)
				})
				if err != nil {
					return nil, wrapErr(err)
				}
				if count == 0 {
					return nil, wrapErr(fmt.Errorf("cannot embed directory %s: contains no embeddable files", rel))
				}
			default:
				return nil, wrapErr(fmt.Errorf("cannot embed irregular %s %s", what, rel))
			}
		}
		if listCount == 0 {
			return nil, wrapErr(fmt.Errorf("no matching files found"))
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]FileData, 0, len(names))
	for _, name := range names {
		out = append(out, FileData{Name: name, Data: seen[name]})
	}
	return out, nil
}

func ValidPattern(pattern string) bool {
	return pattern != "." && fs.ValidPath(pattern)
}

func RelPath(pkgDir, abs string) (string, error) {
	rel, _ := filepath.Rel(pkgDir, abs)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s is outside package directory", abs)
	}
	return filepath.ToSlash(rel), nil
}

func CheckPath(pkgDir, abs string, dirOK map[string]bool) (info fs.FileInfo, rel string, err error) {
	info, err = os.Lstat(abs)
	if err != nil {
		return nil, "", err
	}
	rel, err = RelPath(pkgDir, abs)
	if err != nil {
		return nil, "", err
	}

	what := "file"
	if info.IsDir() {
		what = "directory"
	}

	for dir := abs; ; dir = filepath.Dir(dir) {
		r, _ := RelPath(pkgDir, dir)
		if r == "." {
			break
		}
		if dirOK[dir] {
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return nil, "", fmt.Errorf("cannot embed %s %s: in different module", what, rel)
		}
		elem := filepath.Base(dir)
		if IsBadName(elem) {
			if dir == abs {
				return nil, "", fmt.Errorf("cannot embed %s %s: invalid name %s", what, rel, elem)
			}
			return nil, "", fmt.Errorf("cannot embed %s %s: in invalid directory %s", what, rel, elem)
		}
		dirOK[dir] = true
	}
	return info, rel, nil
}

func IsBadName(name string) bool {
	if err := module.CheckFilePath(name); err != nil {
		return true
	}
	switch name {
	case "", ".bzr", ".git", ".hg", ".svn":
		return true
	}
	return false
}

func BuildFSEntries(files []FileData) []FileData {
	entries := make(map[string][]byte)
	for _, f := range files {
		entries[f.Name] = append([]byte(nil), f.Data...)
		dir := path.Dir(f.Name)
		for dir != "." && dir != "/" {
			entries[dir+"/"] = nil
			dir = path.Dir(dir)
		}
	}
	out := make([]FileData, 0, len(entries))
	for name, data := range entries {
		out = append(out, FileData{Name: name, Data: data})
	}
	sort.Slice(out, func(i, j int) bool {
		di, ei := embedSplit(out[i].Name)
		dj, ej := embedSplit(out[j].Name)
		if di != dj {
			return di < dj
		}
		return ei < ej
	})
	return out
}

func embedSplit(name string) (dir, elem string) {
	name = strings.TrimSuffix(name, "/")
	if idx := strings.LastIndexByte(name, '/'); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	return ".", name
}

func positionFor(fset *token.FileSet, pos token.Pos) token.Position {
	if fset == nil {
		return token.Position{}
	}
	return fset.PositionFor(pos, false)
}
