// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

// Package quoted implements the argument-list quoting rules used by the Go
// command for compiler environment variables and linker flag values.
package quoted

import "fmt"

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Split splits s into fields, allowing single or double quotes around a
// complete field. As in cmd/internal/quoted.Split, quoted text is not
// unescaped and quotes appearing after the beginning of a field are literal.
func Split(s string) ([]string, error) {
	var fields []string
	for len(s) > 0 {
		for len(s) > 0 && isSpaceByte(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		if s[0] == '\'' || s[0] == '"' {
			quote := s[0]
			s = s[1:]
			i := 0
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			fields = append(fields, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isSpaceByte(s[i]) {
			i++
		}
		fields = append(fields, s[:i])
		s = s[i:]
	}
	return fields, nil
}
