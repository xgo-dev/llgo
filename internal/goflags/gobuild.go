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

	"github.com/goplus/llgo/internal/build"
)

// ApplyBuildFlags validates and appends normalized Go build flags, and maps
// the supported compiler and linker semantics into typed build configuration.
// Configuration remains unchanged on error.
func ApplyBuildFlags(conf *build.Config, args []string) error {
	all := make([]string, 0, len(conf.GoBuildFlags)+len(args))
	all = append(all, conf.GoBuildFlags...)
	all = append(all, args...)
	all, err := normalizeBuildFlags(all)
	if err != nil {
		return err
	}

	linkFlags, err := ParseLinkFlags(all)
	if err != nil {
		return err
	}
	parallel, parallelSet, err := parseBuildParallel(all)
	if err != nil {
		return err
	}
	next := *conf
	next.GoBuildFlags = all
	applyFrontendGCFlags(&next)
	if parallelSet {
		next.Parallel = parallel
	}
	if linkFlags.Present {
		next.LinkOptions = linkFlags.Options
	}
	*conf = next
	return nil
}

// parseBuildParallel extracts Go's -p build concurrency flag after it has
// been normalized. Keeping it in GoBuildFlags still lets go/packages apply the
// same setting while Config.Parallel controls LLGo's own build stages.
func parseBuildParallel(flags []string) (parallel int, present bool, err error) {
	for _, flag := range flags {
		value, ok := strings.CutPrefix(flag, "-p=")
		if !ok {
			continue
		}
		parallel, err = strconv.Atoi(value)
		if err != nil || parallel <= 0 {
			return 0, false, fmt.Errorf("-p must be a positive integer, got %q", value)
		}
		present = true
	}
	return parallel, present, nil
}
