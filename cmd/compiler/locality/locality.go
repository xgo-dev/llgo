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

// Package locality defines the source-level TLS and GLS declaration model.
// It intentionally has no dependency on LLGo's SSA or LLVM lowering layers.
package locality

import "fmt"

const (
	ThreadDirective    = "//llgointernal:tls"
	GoroutineDirective = "//llgointernal:gls"
	InitPrefix         = "__llgo_local_init_"
)

// Kind identifies the execution context that owns a package variable.
type Kind uint8

const (
	None Kind = iota
	Thread
	Goroutine
)

// Info is the locality-specific part of a declaration's compiler metadata.
type Info struct {
	Locality       Kind
	HasInitializer bool
	InitFunc       string
	InitOrder      int
}

func (kind Kind) String() string {
	switch kind {
	case None:
		return ""
	case Thread:
		return "tls"
	case Goroutine:
		return "gls"
	default:
		return fmt.Sprintf("invalid:%d", kind)
	}
}

// Parse converts the cache representation of a locality into a Kind.
func Parse(name string) (Kind, bool) {
	switch name {
	case "":
		return None, true
	case "tls":
		return Thread, true
	case "gls":
		return Goroutine, true
	default:
		return None, false
	}
}

// Directive returns the source directive for kind.
func Directive(kind Kind) string {
	if kind == Goroutine {
		return GoroutineDirective
	}
	return ThreadDirective
}

// Merge combines declaration- and spec-level locality directives.
func Merge(a, b Kind) (Kind, bool) {
	if a != None && b != None && a != b {
		return None, false
	}
	if b != None {
		return b, true
	}
	return a, true
}
