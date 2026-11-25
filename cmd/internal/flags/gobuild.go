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
	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/goflags"
)

// CaptureGoBuildFlags registers the Go build flags that LLGo forwards to
// go/packages and returns their command-local collector. LLGo-native flags are
// registered separately by AddBuildFlags.
func CaptureGoBuildFlags(cmd *base.Command) *base.PassArgs {
	p := base.NewPassArgs(&cmd.Flag)
	p.Help = buildFlagHelp
	p.Bool("modcacherw")
	p.Var("p", "gcflags", "ldflags", "mod", "modfile", "overlay")
	for _, name := range []string{"n", "linkshared", "race", "msan", "asan", "trimpath", "work"} {
		rejectBool(&cmd.Flag, name, buildFlagHelp(name))
	}
	for _, name := range []string{"asmflags", "compiler", "gccgoflags", "installsuffix", "pkgdir", "toolexec", "buildvcs", "pgo"} {
		rejectValue(&cmd.Flag, name, buildFlagHelp(name))
	}
	return p
}

// ApplyGoBuildFlags validates and appends normalized Go build flags captured
// by the command layer, and stores supported compiler and linker semantics in
// typed backend configuration. Configuration remains unchanged on error.
func ApplyGoBuildFlags(conf *build.Config, args []string) error {
	if err := goflags.ApplyBuildFlags(conf, args); err != nil {
		return err
	}
	conf.GoFlagsParsed = true
	return nil
}
