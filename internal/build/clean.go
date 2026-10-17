/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package build

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/xgo-dev/llgo/internal/packages"
)

/*
var (
	// TODO(xsw): complete clean flags
	cleanFlags = map[string]bool{
		"-v": false, // -v: print the paths of packages as they are clean
	}
)
*/

func Clean(patterns []string, conf *Config) error {
	if conf.Goos == "" {
		conf.Goos = runtime.GOOS
	}
	if conf.Goarch == "" {
		conf.Goarch = runtime.GOARCH
	}
	tags := "llgo"
	if conf.Tags != "" {
		tags += "," + conf.Tags
	}
	cfg := &packages.Config{
		Mode:       loadSyntax | packages.NeedExportFile,
		BuildFlags: []string{"-tags=" + tags},
	}
	if conf.GoFlagsParsed {
		cfg.Env = withEnv(os.Environ(), "GOFLAGS= ")
	}

	if patterns == nil {
		patterns = []string{"."}
	}
	initial, err := packages.LoadEx(nil, nil, cfg, patterns...)
	if err != nil {
		return err
	}
	if err := initialPackageListErrors(initial); err != nil {
		return err
	}

	err = cleanPkgs(initial, conf.Verbose)

	for _, pkg := range initial {
		if pkg.Name == "main" {
			err = errors.Join(err, cleanMainPkg(pkg, conf, conf.Verbose))
		}
	}
	return err
}

func cleanMainPkg(pkg *packages.Package, conf *Config, verbose bool) error {
	pkgPath := pkg.PkgPath
	name := path.Base(pkgPath)
	fname := name + conf.AppExt
	app := filepath.Join(conf.BinPath, fname)
	err := removeExecutableArtifacts(app, verbose)
	if len(pkg.CompiledGoFiles) > 0 {
		dir := filepath.Dir(pkg.CompiledGoFiles[0])
		buildApp := filepath.Join(dir, fname)
		err = errors.Join(err, removeExecutableArtifacts(buildApp, verbose))
	}
	return err
}

func removeExecutableArtifacts(executable string, verbose bool) error {
	return errors.Join(removeFile(executable, verbose), removeFile(pclnSidecarPath(executable), verbose), removeFile(dwarfSidecarPath(executable), verbose), removeArtifactDirectory(darwinLTOObjectDir(executable), verbose))
}

func removeArtifactDirectory(dir string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, "Remove", dir)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clean %s: %w", dir, err)
	}
	return nil
}

func cleanPkgs(initial []*packages.Package, verbose bool) error {
	var err error
	packages.Visit(initial, nil, func(p *packages.Package) {
		if p.ExportFile == "" {
			return
		}
		file := p.ExportFile + ".ll"
		err = errors.Join(err, removeFile(file, verbose))
	})
	return err
}

func removeFile(file string, verbose bool) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "Remove", file)
	}
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clean %s: %w", file, err)
	}
	return nil
}
