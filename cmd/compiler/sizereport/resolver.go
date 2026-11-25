package sizereport

import "strings"

// nameResolver maps symbol names to aggregation buckets based on the requested level.
type nameResolver struct {
	level       string
	pkgs        []Package
	pkgPrefixes []string // Pre-computed id+"." for each package
	moduleMap   map[string]string
	packageMap  map[string]string
}

func newNameResolver(level string, pkgs []Package) *nameResolver {
	lvl := strings.ToLower(strings.TrimSpace(level))
	if lvl == "" {
		lvl = "module"
	}
	pkgPrefixes := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		if pkg.Path != "" {
			pkgPrefixes[i] = pkg.Path + "."
		}
	}
	return &nameResolver{
		level:       lvl,
		pkgs:        pkgs,
		pkgPrefixes: pkgPrefixes,
		moduleMap:   make(map[string]string),
		packageMap:  make(map[string]string),
	}
}

func (r *nameResolver) resolve(sym string) string {
	base := moduleNameFromSymbol(sym)
	symbol := trimSymbolForMatch(sym)
	switch r.level {
	case "full":
		if sym != "" {
			return sym
		}
		return base
	case "package":
		if pkg := r.matchPackage(symbol); pkg != "" {
			return pkg
		}
	case "module":
		if mod := r.matchModule(symbol); mod != "" {
			return mod
		}
	}
	return base
}

func (r *nameResolver) matchPackage(symbol string) string {
	if symbol == "" {
		return ""
	}
	if cached, ok := r.packageMap[symbol]; ok {
		return cached
	}
	for i, pkg := range r.pkgs {
		id := pkg.Path
		if id == "" {
			continue
		}
		if strings.HasPrefix(symbol, r.pkgPrefixes[i]) {
			r.packageMap[symbol] = id
			return id
		}
	}
	r.packageMap[symbol] = ""
	return ""
}

func (r *nameResolver) matchModule(symbol string) string {
	if symbol == "" {
		return ""
	}
	if cached, ok := r.moduleMap[symbol]; ok {
		return cached
	}
	for _, pkg := range r.pkgs {
		path := pkg.Path
		if path == "" {
			continue
		}
		if strings.HasPrefix(symbol, path+".") {
			mod := path
			if pkg.Module != "" {
				mod = pkg.Module
			}
			r.moduleMap[symbol] = mod
			return mod
		}
	}
	r.moduleMap[symbol] = ""
	return ""
}

func trimSymbolForMatch(sym string) string {
	name := strings.TrimSpace(sym)
	for len(name) > 0 && (name[0] == '_' || name[0] == '.') {
		name = name[1:]
	}
	if idx := strings.Index(name, " "); idx >= 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "@"); idx >= 0 {
		name = name[:idx]
	}
	return name
}
