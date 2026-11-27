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

package env

import (
	"runtime/debug"
	"strings"
)

const (
	devel = "(devel)"
)

// buildVersion is the LLGo tree's version string at build time. It should be
// set by the linker.
var buildVersion string

// Version returns the version of the running LLGo binary.
func Version() string {
	if buildVersion != "" {
		return buildVersion
	}
	info, ok := readBuildInfo()
	if ok && info.Main.Version != "" && !strings.HasSuffix(info.Main.Version, "+dirty") {
		return info.Main.Version
	}
	return devel
}

func Devel() bool {
	return Version() == devel
}

var (
	readBuildInfo = debug.ReadBuildInfo
)
