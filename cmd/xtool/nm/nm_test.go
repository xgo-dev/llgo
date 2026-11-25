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

package nm

import "testing"

func TestListOutputCRLF(t *testing.T) {
	items, err := listOutput([]byte("\r\nfoo.o:\r\n00000000 T Foo\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].File != "foo.o" {
		t.Fatalf("items = %#v, want one foo.o object", items)
	}
	if symbols := items[0].Symbols; len(symbols) != 1 || symbols[0].Name != "Foo" || symbols[0].Type != Text {
		t.Fatalf("symbols = %#v, want text symbol Foo", symbols)
	}
}
