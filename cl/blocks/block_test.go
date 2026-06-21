//go:build !llgo
// +build !llgo

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

package blocks

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	llssa "github.com/goplus/llgo/ssa"
)

func TestTestdefer(t *testing.T) {
	fromDir(t, "", "../_testdefer", func(name string) string {
		if strings.HasPrefix(name, "firstloop") {
			return "Loop"
		}
		if strings.HasPrefix(name, "gobuild") {
			return "Import"
		}
		return "main"
	})
}

func TestFirstLoop(t *testing.T) {
	blk := &ssa.BasicBlock{}
	blk.Index = 0
	blk.Preds = []*ssa.BasicBlock{blk}
	blk.Succs = []*ssa.BasicBlock{blk}
	infos := Infos([]*ssa.BasicBlock{blk})
	if infos[0].Kind != llssa.DeferInLoop {
		t.Fatal("TestFirstLoop")
	}
}

func TestFindLoopSkipsSearchedSharedSubgraphs(t *testing.T) {
	const n = 64
	states := make([]*blockState, n)
	for i := range states {
		succs := make([]int, 0, 2)
		if i+1 < n {
			succs = append(succs, i+1)
		}
		if i+2 < n {
			succs = append(succs, i+2)
		}
		states[i] = &blockState{succs: succs}
	}
	if loop := findLoop(states, make([]int, 0, n), make([]bool, n), 0); len(loop) != 0 {
		t.Fatalf("findLoop returned non-loop path: %v", loop)
	}
}

func fromDir(t *testing.T, sel, relDir string, fn func(string) string) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = path.Join(dir, relDir)
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testFrom(t, dir+"/"+name, sel, fn(name))
		})
	}
}

func testFrom(t *testing.T, pkgDir, sel, fn string) {
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}
	log.Println("Parsing", pkgDir)
	in := pkgDir + "/in.go"
	out := pkgDir + "/out.txt"
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal("ReadFile failed:", err)
	}
	expected := string(b)
	testBlockInfo(t, nil, in, expected, fn)
}

func testBlockInfo(t *testing.T, src any, fname, expected, fn string) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, src, parser.ParseComments)
	if err != nil {
		t.Fatal("ParseFile failed:", err)
	}
	files := []*ast.File{f}
	name := f.Name.Name
	pkg := types.NewPackage(name, name)
	imp := packages.NewImporter(fset)
	foo, _, err := ssautil.BuildPackage(
		&types.Config{Importer: imp}, fset, pkg, files, ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug)
	if err != nil {
		t.Fatal("BuildPackage failed:", err)
	}
	foo.WriteTo(os.Stderr)

	for _, member := range foo.Members {
		switch f := member.(type) {
		case *ssa.Function:
			if f.Name() == fn {
				f.WriteTo(os.Stderr)
				infos := Infos(f.Blocks)
				if v := resultOf(infos); v != expected {
					t.Fatalf("\n==> got:\n%s\n==> expected:\n%s\n", v, expected)
				}
				return
			}
		}
	}
}

func resultOf(infos []Info) string {
	var b bytes.Buffer
	i := 0
	for {
		fmt.Fprintf(&b, "%2d: %s\n", i, kinds[infos[i].Kind])
		if i = infos[i].Next; i < 0 {
			break
		}
	}
	return b.String()
}

var kinds = [...]string{
	llssa.DeferAlways: "always",
	llssa.DeferInCond: "cond",
	llssa.DeferInLoop: "loop",
}
