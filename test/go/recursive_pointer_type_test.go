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

package gotest

import "testing"

type recursiveLink *recursiveLink
type recursivePeano *recursivePeano

func boxRecursiveLink(v recursiveLink) *recursiveLink {
	p := new(recursiveLink)
	*p = v
	return p
}

func unboxRecursiveLink(p *recursiveLink) recursiveLink {
	return *p
}

func makeRecursivePeano(n int) *recursivePeano {
	if n == 0 {
		return nil
	}
	p := recursivePeano(makeRecursivePeano(n - 1))
	return &p
}

var recursivePeanoCountArg recursivePeano
var recursivePeanoCountResult int

func countRecursivePeano() {
	if recursivePeanoCountArg == nil {
		recursivePeanoCountResult = 0
		return
	}
	recursivePeanoCountArg = *recursivePeanoCountArg
	countRecursivePeano()
	recursivePeanoCountResult++
}

func TestRecursivePointerTypeBuilds(t *testing.T) {
	sentinel := recursiveLink(new(recursiveLink))
	p := boxRecursiveLink(sentinel)
	if unboxRecursiveLink(p) != sentinel {
		t.Fatal("recursive pointer type lost value")
	}

	recursivePeanoCountArg = recursivePeano(makeRecursivePeano(recursivePeanoDepth))
	countRecursivePeano()
	if recursivePeanoCountResult != recursivePeanoDepth {
		t.Fatalf("recursive Peano pointer count = %d, want %d", recursivePeanoCountResult, recursivePeanoDepth)
	}
}
