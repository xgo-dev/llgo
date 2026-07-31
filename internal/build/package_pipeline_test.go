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

package build

import (
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestNewPackagePipelineSSANodesDeduplicates(t *testing.T) {
	first := new(ssa.Package)
	second := new(ssa.Package)
	nodes, byPackage := newPackagePipelineSSANodes([]ssaBuildEntry{
		{pkg: first},
		{pkg: nil, fixOrder: true},
		{pkg: second},
		{pkg: first, fixOrder: true},
	})

	if len(nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(nodes))
	}
	if nodes[0] != byPackage[first] || nodes[1] != byPackage[second] {
		t.Fatal("package lookup does not preserve first-seen order")
	}
	if !nodes[0].entry.fixOrder {
		t.Fatal("duplicate package did not retain fixOrder")
	}
	if nodes[0].order != 0 || nodes[1].order != 1 {
		t.Fatalf("node order = (%d, %d), want (0, 1)", nodes[0].order, nodes[1].order)
	}
}

func TestOrderPackagePipelineSSADependenciesFirst(t *testing.T) {
	first := &packagePipelineSSA{order: 0}
	second := &packagePipelineSSA{order: 1}
	third := &packagePipelineSSA{order: 2}
	tasks := []*packagePipelineTask{
		{ownSSA: first, ssaDeps: []*packagePipelineSSA{second}},
		{ownSSA: second, ssaDeps: []*packagePipelineSSA{third}},
	}

	got := orderPackagePipelineSSA([]*packagePipelineSSA{first, second, third}, tasks)
	want := []*packagePipelineSSA{third, second, first}
	if len(got) != len(want) {
		t.Fatalf("ordered node count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordered node %d = %p, want %p", i, got[i], want[i])
		}
	}
}

func TestOrderPackagePipelineSSAToleratesCyclesAndEmptyTasks(t *testing.T) {
	first := &packagePipelineSSA{order: 0}
	second := &packagePipelineSSA{order: 1}
	tasks := []*packagePipelineTask{
		{},
		{ownSSA: first, ssaDeps: []*packagePipelineSSA{second}},
		{ownSSA: second, ssaDeps: []*packagePipelineSSA{first}},
	}

	got := orderPackagePipelineSSA([]*packagePipelineSSA{nil, first, second, first}, tasks)
	if len(got) != 2 {
		t.Fatalf("ordered node count = %d, want 2", len(got))
	}
	seen := map[*packagePipelineSSA]bool{}
	for _, node := range got {
		if seen[node] {
			t.Fatalf("SSA node %p was returned more than once", node)
		}
		seen[node] = true
	}
	if !seen[first] || !seen[second] {
		t.Fatalf("ordered nodes = %v, want both cycle members", got)
	}
}

func TestBuildPackagePipelineSSARejectsNilPackage(t *testing.T) {
	if err := buildPackagePipelineSSA(ssaBuildEntry{}); err == nil {
		t.Fatal("nil SSA package was accepted")
	}
}

func TestBuildPackagePipelineSSAConvertsPanicsToErrors(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	pkg := prog.CreatePackage(types.NewPackage("example.com/p", "p"), nil, &types.Info{}, true)
	pkg.Prog = nil
	if err := buildPackagePipelineSSA(ssaBuildEntry{pkg: pkg}); err == nil {
		t.Fatal("SSA build panic was not converted to an error")
	}
	if _, err := finishPackagePipelineSSA(ssaBuildEntry{}); err == nil {
		t.Fatal("SSA finalization panic was not converted to an error")
	}
}
