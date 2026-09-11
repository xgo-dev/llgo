//go:build llgo

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

// Package localitynilfailure exercises cached panic(nil) from a GLS package
// initializer independently of other deliberately failing initializers.
package localitynilfailure

var attempts int

func initialize() int {
	attempts++
	if attempts > 1 {
		panic(nil)
	}
	return attempts
}

//llgointernal:gls
var value = initialize()

//go:noinline
func Value() int { return value }

func Attempts() int { return attempts }
