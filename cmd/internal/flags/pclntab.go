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

	"github.com/xgo-dev/llgo/internal/build"
)

type pclnFlag struct {
	Specified bool
	Mode      build.PCLNMode
}

func (f *pclnFlag) String() string {
	return f.Mode.String()
}

func (f *pclnFlag) Set(value string) error {
	var mode build.PCLNMode
	switch value {
	case "embedded":
		mode = build.PCLNEmbedded
	case "external":
		mode = build.PCLNExternal
	case "none":
		mode = build.PCLNNone
	default:
		return fmt.Errorf("invalid pclntab mode %q (valid: embedded, external, none)", value)
	}
	f.Specified = true
	f.Mode = mode
	return nil
}

var PCLN pclnFlag

func addPCLNFlag(fs *flag.FlagSet) {
	PCLN = pclnFlag{Mode: build.PCLNEmbedded}
	fs.Var(&PCLN, "pclntab", "PCLN metadata mode: embedded, external, or none (default: embedded)")
}
