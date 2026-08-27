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

package pysig

import "testing"

func TestParse(t *testing.T) {
	type testCase struct {
		sig  string
		args []*Arg
	}
	cases := []testCase{
		{"(start=None, *, unit: 'str | None' = None) -> 'TimedeltaIndex'", []*Arg{
			{Name: "start", DefVal: "None"},
			{Name: "*"},
			{Name: "unit", Type: "'str | None'", DefVal: "None"},
		}},
		{"()", nil},
		{"(a =", []*Arg{{Name: "a"}}},
		{"(a) -> int", []*Arg{{Name: "a"}}},
		{"(a: int)", []*Arg{{Name: "a", Type: "int"}}},
		{"(a: int = 1, b: float)", []*Arg{{Name: "a", Type: "int", DefVal: "1"}, {Name: "b", Type: "float"}}},
		{"(a = <1>, b = 2.0)", []*Arg{{Name: "a", DefVal: "<1>"}, {Name: "b", DefVal: "2.0"}}},
		{"(a: 'Suffixes' = ('_x', '_y'))", []*Arg{{Name: "a", Type: "'Suffixes'", DefVal: "('_x', '_y')"}}},
	}
	for _, c := range cases {
		args := Parse(c.sig)
		if len(args) != len(c.args) {
			t.Fatalf("%s: len(args) = %v, want %v", c.sig, len(args), len(c.args))
		}
		for i, arg := range args {
			want := c.args[i]
			if arg.Name != want.Name || arg.Type != want.Type || arg.DefVal != want.DefVal {
				t.Fatalf("%s: args[%v] = %v, want %v", c.sig, i, arg, want)
			}
		}
	}
}
