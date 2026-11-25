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

package goflags

import (
	"reflect"
	"testing"
)

func TestParseFlagFileFormats(t *testing.T) {
	input := `
# full-line comment
-trimpath -tags='integration,debug' # inline comment
-ldflags=-s -w=false
-gcflags="all=-N -l"
-toolexec=tool --mode check
--ldflags "-s -w=false"
`
	want := []string{
		"-trimpath",
		"-tags=integration,debug",
		"-ldflags=-s -w=false",
		"-gcflags=all=-N -l",
		"-toolexec=tool --mode check",
		"-ldflags=-s -w=false",
	}
	got, err := ParseFlagFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseFlagFile() = %#v, want %#v", got, want)
	}
}

func TestParseFlagFileArgumentListSpellingsAndComments(t *testing.T) {
	input := `
-ldflags="-w=false" # quoted equals form
-ldflags="-w=false" -trimpath
--gcflags 'all=-N -l'
--toolexec=tool --mode check # whole-line value
-ldflags=-X 'example.com/p.value=hello # world' -w=false # trailing comment
`
	want := []string{
		"-ldflags=-w=false",
		"-ldflags=-w=false", "-trimpath",
		"-gcflags=all=-N -l",
		"-toolexec=tool --mode check",
		"-ldflags=-X 'example.com/p.value=hello # world' -w=false",
	}
	got, err := ParseFlagFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseFlagFile() = %#v, want %#v", got, want)
	}
}

func TestParseFlagFilePreservesQuotedLinkArguments(t *testing.T) {
	input := `-ldflags=-X 'example.com/p.value=hello world' -w=false`
	got, err := ParseFlagFile(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{input}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseFlagFile() = %#v, want %#v", got, want)
	}
}

func TestParseFlagFileEscapesAndEmptyQuotedValues(t *testing.T) {
	input := "-tags=hello\\ world -modfile=\"a\\\"b\" -overlay='' -x#literal # trailing\n"
	got, err := ParseFlagFile(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-tags=hello world", `-modfile=a"b`, "-overlay=", "-x#literal"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseFlagFile() = %#v, want %#v", got, want)
	}
}

func TestParseFlagFileErrors(t *testing.T) {
	for _, input := range []string{
		`-tags='unterminated`,
		"-trimpath \\",
		`-ldflags='-s -w=false`,
	} {
		if _, err := ParseFlagFile(input); err == nil {
			t.Fatalf("ParseFlagFile(%q) succeeded, want error", input)
		}
	}
}
