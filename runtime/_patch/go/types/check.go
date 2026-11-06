package types

import (
	"go/ast"
	"runtime"
)

// Files registers declaration attributes before go/types folds unsafe
// constants. Explicit user-supplied Sizes implementations remain supported.
func (check *Checker) Files(files []*ast.File) (err error) {
	if check.pkg == Unsafe {
		return nil
	}
	if check.conf.Sizes == nil {
		conf := *check.conf
		conf.Sizes = SizesFor("llgo", runtime.GOARCH)
		check.conf = &conf
	}
	if s, ok := check.conf.Sizes.(*LLGoSizes); ok {
		if check.Types == nil {
			check.Types = make(map[ast.Expr]TypeAndValue)
			defer func() { check.Types = nil }()
		}
		if check.Instances == nil {
			check.Instances = make(map[*ast.Ident]Instance)
			defer func() { check.Instances = nil }()
		}
		copy := *s
		s = &copy
		if s.Layout == nil {
			s.Layout = &llgoSourceLayout
		}
		conf := *check.conf
		conf.Sizes = s
		if conf.Importer != nil {
			conf.Importer = llgoLayoutImporter{conf.Importer, s.Layout}
		}
		check.conf = &conf
		if err = s.Layout.ParseSyntax(check.fset, check.pkg, files, nil); err != nil {
			if check.conf.Error != nil {
				check.conf.Error(err)
			}
			return err
		}
	}
	defer check.handleBailout(&err)
	check.checkFiles(files)
	if check.firstErr == nil {
		if s, ok := check.conf.Sizes.(*LLGoSizes); ok {
			s.Layout.RecordPackage(check.pkg)
			err = s.ValidatePackage(check.pkg)
			if err == nil {
				err = s.ValidateInstances(check.fset, check.Info)
			}
			if err == nil {
				err = s.ValidateConversions(check.fset, files, check.Info)
			}
			if err != nil {
				check.firstErr = err
				if check.conf.Error != nil {
					check.conf.Error(err)
				}
			}
		}
	}
	return
}

type llgoLayoutImporter struct {
	Importer
	layout *LLGoLayout
}

func (i llgoLayoutImporter) Import(path string) (*Package, error) { return i.ImportFrom(path, "", 0) }
func (i llgoLayoutImporter) ImportFrom(path, dir string, mode ImportMode) (pkg *Package, err error) {
	if from, ok := i.Importer.(ImporterFrom); ok {
		pkg, err = from.ImportFrom(path, dir, mode)
	} else {
		pkg, err = i.Importer.Import(path)
	}
	if err == nil {
		i.layout.RecordPackage(pkg)
	}
	return
}
