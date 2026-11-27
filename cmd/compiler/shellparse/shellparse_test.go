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

package shellparse

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		want    []string
		wantErr bool
	}{
		{
			name: "simple command",
			cmd:  "echo hello",
			want: []string{"echo", "hello"},
		},
		{
			name: "command with quoted argument",
			cmd:  `echo "hello world"`,
			want: []string{"echo", "hello world"},
		},
		{
			name: "command with single quoted argument",
			cmd:  `echo 'hello world'`,
			want: []string{"echo", "hello world"},
		},
		{
			name: "command with multiple quoted arguments",
			cmd:  `cp "file with spaces.txt" "another file.txt"`,
			want: []string{"cp", "file with spaces.txt", "another file.txt"},
		},
		{
			name: "command with escaped quotes",
			cmd:  `echo "He said \"hello\""`,
			want: []string{"echo", `He said "hello"`},
		},
		{
			name: "command with escaped backslash",
			cmd:  `echo "path\\to\\file"`,
			want: []string{"echo", `path\to\file`},
		},
		{
			name: "mixed quotes",
			cmd:  `echo "double quote" 'single quote' normal`,
			want: []string{"echo", "double quote", "single quote", "normal"},
		},
		{
			name: "empty arguments",
			cmd:  `echo "" ''`,
			want: []string{"echo", "", ""},
		},
		{
			name: "multiple spaces",
			cmd:  "echo    hello    world",
			want: []string{"echo", "hello", "world"},
		},
		{
			name: "emulator command example",
			cmd:  `qemu-system-xtensa -machine esp32 -kernel "/path/with spaces/firmware.bin"`,
			want: []string{"qemu-system-xtensa", "-machine", "esp32", "-kernel", "/path/with spaces/firmware.bin"},
		},
		{
			name:    "unterminated double quote",
			cmd:     `echo "hello`,
			wantErr: true,
		},
		{
			name:    "unterminated single quote",
			cmd:     `echo 'hello`,
			wantErr: true,
		},
		{
			name: "empty command",
			cmd:  "",
			want: []string{},
		},
		{
			name: "only spaces",
			cmd:  "   ",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
