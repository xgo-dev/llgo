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

// Package pollbudget implements a fixed cooperative polling budget.
package pollbudget

// Budget reports every quantum-th call to Poll.
type Budget struct {
	remaining uint32
	quantum   uint32
}

// New returns a budget with the requested non-zero quantum.
func New(quantum uint32) Budget {
	if quantum == 0 {
		panic("pollbudget: zero quantum")
	}
	return Budget{remaining: quantum, quantum: quantum}
}

// Poll consumes one unit and reports whether the slow path should run.
func (b *Budget) Poll() bool {
	if b.remaining > 1 {
		b.remaining--
		return false
	}
	b.remaining = b.quantum
	return true
}
