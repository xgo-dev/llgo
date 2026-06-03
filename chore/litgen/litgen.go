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

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goplus/llgo/internal/littest"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <file-or-dir> [<file-or-dir>...]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	for _, arg := range flag.Args() {
		fatal(processPath(arg))
	}
}

func processPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return processTree(abs)
	}
	if filepath.Ext(abs) != ".go" {
		return fmt.Errorf("%s: expected .go file or directory", abs)
	}
	ok, err := littest.HasMarker(abs)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s: missing // LITTEST marker", abs)
	}
	target, err := resolveTarget(abs, filepath.Dir(abs))
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "litgen", target.sourceFile)
	return generateFile(target)
}

func processTree(root string) error {
	var targets []resolvedTarget
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && len(d.Name()) > 0 && d.Name()[0] == '_' {
			return filepath.SkipDir
		}
		marked, found, err := littest.FindMarkedSourceFile(path)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		target, err := resolveTarget(marked, path)
		if err != nil {
			return err
		}
		targets = append(targets, target)
		return nil
	})
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("%s: no // LITTEST sources found", root)
	}
	for _, target := range targets {
		fmt.Fprintln(os.Stderr, "litgen", target.sourceFile)
		if err := generateFile(target); err != nil {
			return err
		}
	}
	return nil
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "litgen: %v\n", err)
		os.Exit(1)
	}
}
