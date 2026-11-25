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
	"strconv"
	"strings"
)

func parseBuildParallelism(args []string) (parallelism int, present bool, err error) {
	for _, arg := range args {
		value, ok := strings.CutPrefix(arg, "-p=")
		if !ok {
			continue
		}

		n, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return 0, false, fmt.Errorf("invalid value %q for flag -p: parse error", value)
		}
		if n <= 0 {
			return 0, false, fmt.Errorf("go: -p must be a positive integer: %d", n)
		}
		parallelism, present = n, true
	}
	return parallelism, present, nil
}
