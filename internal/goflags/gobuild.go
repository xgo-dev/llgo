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

import "github.com/xgo-dev/llgo/internal/build"

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

	parallelism, parallelismPresent, err := parseBuildParallelism(all)
	if err != nil {
		return err
	}
	linkFlags, err := ParseLinkFlags(all)
	if err != nil {
		return err
	}
	next := *conf
	next.GoBuildFlags = all
	applyFrontendGCFlags(&next)
	if parallelismPresent {
		next.BuildParallelism = parallelism
	}
	if linkFlags.Present {
		next.LinkOptions = linkFlags.Options
	}
	*conf = next
	return nil
}
