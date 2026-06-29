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

package cl_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/goplus/llgo/cl/cltest"
	llssa "github.com/goplus/llgo/ssa"
)

type funcInfoRecord struct {
	symbol string
	name   string
	file   string
	line   int
	column int
}

func TestFuncInfoMetadataEmission(t *testing.T) {
	const src = `package foo

type T struct{}

func top() {
	_ = func() int { return leaf() }()
}

func leaf() int { return 1 }

func (T) method() {}
`
	ir := cltest.CompileIREx(t, src, "foo.go", false, func(prog llssa.Program) {
		prog.EnableFuncInfoMetadata(true)
	})

	for _, want := range []string{
		`!llgo.funcinfo = !{!`,
		`!"foo.top"`,
		`!"foo.top$1"`,
		`!"foo.T.method"`,
		`!"foo.go"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing funcinfo metadata %s:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, "llvm.compiler.used") {
		t.Fatalf("funcinfo metadata should not add llvm.compiler.used:\n%s", ir)
	}
	if strings.Contains(ir, `ptr @"foo.top"`) || strings.Contains(ir, `ptr @foo.top`) {
		t.Fatalf("funcinfo metadata should use symbol strings, not function pointers:\n%s", ir)
	}

	records := parseFuncInfoRecords(t, ir)
	stackSymbols := []string{"foo.leaf", "foo.top$1", "foo.top"}
	for _, symbol := range stackSymbols {
		record, ok := records[symbol]
		if !ok {
			t.Fatalf("stack symbol %q not found in funcinfo metadata: %#v", symbol, records)
		}
		if record.name == "" || record.file != "foo.go" || record.line <= 0 || record.column <= 0 {
			t.Fatalf("bad funcinfo for stack symbol %q: %#v", symbol, record)
		}
	}
	if got := records["foo.leaf"].name; got != "foo.leaf" {
		t.Fatalf("leaf stack frame name = %q, want foo.leaf", got)
	}
	if got := records["foo.top$1"].name; got != "foo.top$1" {
		t.Fatalf("closure stack frame name = %q, want foo.top$1", got)
	}
	if got := records["foo.top"].name; got != "foo.top" {
		t.Fatalf("caller stack frame name = %q, want foo.top", got)
	}
}

func TestNoInlineDirectiveDisablesTailCalls(t *testing.T) {
	const src = `package foo

func caller() { callee() }

//go:noinline
func callee() {}
`
	ir := cltest.CompileIREx(t, src, "foo.go", false, nil)
	if !strings.Contains(ir, `define void @foo.callee()`) {
		t.Fatalf("missing callee function:\n%s", ir)
	}
	if !strings.Contains(ir, `noinline`) || !strings.Contains(ir, `"disable-tail-calls"="true"`) {
		t.Fatalf("callee should disable inlining and tail calls:\n%s", ir)
	}
}

func parseFuncInfoRecords(t *testing.T, ir string) map[string]funcInfoRecord {
	t.Helper()

	listRE := regexp.MustCompile(`!llgo\.funcinfo = !\{([^}]*)\}`)
	listMatch := listRE.FindStringSubmatch(ir)
	if listMatch == nil {
		t.Fatalf("missing funcinfo metadata list:\n%s", ir)
	}
	refRE := regexp.MustCompile(`!(\d+)`)
	refs := refRE.FindAllStringSubmatch(listMatch[1], -1)
	if len(refs) == 0 {
		t.Fatalf("empty funcinfo metadata list:\n%s", ir)
	}
	wantRefs := make(map[string]bool, len(refs))
	for _, ref := range refs {
		wantRefs[ref[1]] = true
	}

	rowRE := regexp.MustCompile(`^!(\d+) = !\{i32 1, !"([^"]+)", !"([^"]+)", !"([^"]*)", i32 ([0-9]+), i32 ([0-9]+)\}$`)
	records := make(map[string]funcInfoRecord)
	for _, line := range strings.Split(ir, "\n") {
		row := rowRE.FindStringSubmatch(line)
		if row == nil || !wantRefs[row[1]] {
			continue
		}
		lineNo, err := strconv.Atoi(row[5])
		if err != nil {
			t.Fatalf("bad funcinfo line in %q: %v", line, err)
		}
		column, err := strconv.Atoi(row[6])
		if err != nil {
			t.Fatalf("bad funcinfo column in %q: %v", line, err)
		}
		records[row[2]] = funcInfoRecord{
			symbol: row[2],
			name:   row[3],
			file:   row[4],
			line:   lineNo,
			column: column,
		}
	}
	if len(records) != len(wantRefs) {
		t.Fatalf("parsed %d funcinfo records, want %d:\n%s", len(records), len(wantRefs), ir)
	}
	return records
}
