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

package flags

import (
	"flag"
	"fmt"

	"github.com/xgo-dev/llgo/cmd/compiler/build"
)

type debugArtifactFlag struct {
	Specified bool
	Mode      build.DebugArtifactMode
}

func (f *debugArtifactFlag) String() string {
	return f.Mode.String()
}

func (f *debugArtifactFlag) Set(value string) error {
	var mode build.DebugArtifactMode
	switch value {
	case "embedded":
		mode = build.DebugArtifactEmbedded
	case "external":
		mode = build.DebugArtifactExternal
	case "host":
		mode = build.DebugArtifactHost
	case "none":
		mode = build.DebugArtifactNone
	default:
		return fmt.Errorf("invalid debug artifact mode %q (valid: embedded, external, host, none)", value)
	}
	f.Specified = true
	f.Mode = mode
	return nil
}

var DebugArtifact debugArtifactFlag

func addDebugArtifactFlag(fs *flag.FlagSet) {
	DebugArtifact = debugArtifactFlag{Mode: build.DebugArtifactDefault}
	fs.Var(&DebugArtifact, "debug-artifact", "DWARF artifact mode: embedded, external, host, or none")
}
