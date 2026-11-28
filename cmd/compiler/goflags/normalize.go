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
	"fmt"
	"strings"
)

// normalizeBuildFlags canonicalizes the argument-list flags shared by the Go
// command and LLGo. It accepts both -flag=value and -flag value, as well as
// the double-dash spellings accepted by Go's flag parser.
func normalizeBuildFlags(args []string) ([]string, error) {
	ret := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, hasValue, ok := buildParallelFlag(arg); ok {
			if !hasValue {
				if i+1 == len(args) {
					return nil, fmt.Errorf("-p requires a value")
				}
				i++
				value = args[i]
			}
			ret = append(ret, "-p="+value)
			continue
		}
		name, value, hasValue, ok := argumentListBuildFlag(arg)
		if !ok {
			name, value, hasValue = strings.Cut(strings.TrimLeft(arg, "-"), "=")
			ok = strings.HasPrefix(arg, "-") && (name == "overlay" || name == "mod" || name == "modfile")
		}
		if !ok {
			ret = append(ret, arg)
			continue
		}
		if !hasValue {
			if i+1 == len(args) {
				return nil, fmt.Errorf("-%s requires a value", name)
			}
			i++
			value = args[i]
		}
		ret = append(ret, "-"+name+"="+value)
	}
	return ret, nil
}

func buildParallelFlag(arg string) (value string, hasValue, ok bool) {
	trimmed := strings.TrimPrefix(arg, "-")
	trimmed = strings.TrimPrefix(trimmed, "-")
	if trimmed == arg || trimmed == "" {
		return "", false, false
	}
	name, value, hasValue := strings.Cut(trimmed, "=")
	if name != "p" {
		return "", false, false
	}
	return value, hasValue, true
}

func argumentListBuildFlag(arg string) (name, value string, hasValue, ok bool) {
	trimmed := strings.TrimPrefix(arg, "-")
	trimmed = strings.TrimPrefix(trimmed, "-")
	if trimmed == arg || trimmed == "" {
		return "", "", false, false
	}
	name, value, hasValue = strings.Cut(trimmed, "=")
	for _, candidate := range argumentListFlagNames {
		if name == candidate {
			return name, value, hasValue, true
		}
	}
	return "", "", false, false
}
