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
	"fmt"
	"strings"
	"unicode"
)

// Parse parses a shell command string into command and arguments,
// properly handling quoted arguments with spaces
func Parse(cmd string) ([]string, error) {
	args := make([]string, 0)
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var hasContent bool // Track if we've seen content (including empty quotes)

	runes := []rune(cmd)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case !inQuotes && (r == '"' || r == '\''):
			// Start of quoted string
			inQuotes = true
			quoteChar = r
			hasContent = true // Empty quotes still count as content
		case inQuotes && r == quoteChar:
			// End of quoted string
			inQuotes = false
			quoteChar = 0
		case !inQuotes && unicode.IsSpace(r):
			// Space outside quotes - end current argument
			if hasContent {
				args = append(args, current.String())
				current.Reset()
				hasContent = false
			}
		case inQuotes && r == '\\' && i+1 < len(runes):
			// Handle escape sequences in quotes
			if quoteChar == '"' {
				next := runes[i+1]
				if next == quoteChar || next == '\\' {
					current.WriteRune(next)
					i++ // Skip the next rune
				} else {
					current.WriteRune(r)
				}
			} else {
				// In single quotes, backslash is a literal character.
				current.WriteRune(r)
			}
		default:
			// Regular character
			current.WriteRune(r)
			hasContent = true
		}
	}

	// Handle unterminated quotes
	if inQuotes {
		return nil, fmt.Errorf("unterminated quote in command: %s", cmd)
	}

	// Add final argument if any
	if hasContent {
		args = append(args, current.String())
	}

	return args, nil
}
