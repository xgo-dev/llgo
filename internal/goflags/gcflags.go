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
	"strconv"
	"strings"

	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/optlevel"
)

// applyFrontendGCFlags maps the supported Go compiler flag subset to typed
// frontend configuration. Raw flags remain in GoBuildFlags for go/packages.
func applyFrontendGCFlags(conf *build.Config) {
	type frontendFlags struct {
		goVersion               string
		nNoOpt                  bool
		lNoOpt                  bool
		saturatingFloatToUint32 bool
	}
	var applicable *frontendFlags
	for _, buildFlag := range conf.GoBuildFlags {
		value, ok := strings.CutPrefix(buildFlag, "-gcflags=")
		if !ok {
			continue
		}
		pattern, fields, err := splitPerPackageArgumentList(value)
		if err != nil || pattern != "" && pattern != "all" {
			// Raw package-specific values remain available to go/packages. LLGo
			// cannot map them to one global frontend configuration safely.
			continue
		}
		current := new(frontendFlags)
		for _, compilerFlag := range fields {
			switch {
			case strings.HasPrefix(compilerFlag, "-lang="):
				current.goVersion = strings.TrimPrefix(compilerFlag, "-lang=")
			case compilerFlag == "-N":
				current.nNoOpt = true
			case strings.HasPrefix(compilerFlag, "-N="):
				current.nNoOpt = countFlagEnabled(strings.TrimPrefix(compilerFlag, "-N="))
			case compilerFlag == "-l":
				current.lNoOpt = true
			case strings.HasPrefix(compilerFlag, "-l="):
				current.lNoOpt = countFlagIsOne(strings.TrimPrefix(compilerFlag, "-l="))
			case strings.HasPrefix(compilerFlag, "-d="):
				current.saturatingFloatToUint32 = converthashAlwaysEnabled(strings.TrimPrefix(compilerFlag, "-d="))
			}
		}
		applicable = current
	}
	if applicable == nil {
		return
	}
	if applicable.goVersion != "" {
		conf.GoVersion = applicable.goVersion
	}
	if applicable.nNoOpt || applicable.lNoOpt {
		conf.OptLevel = optlevel.O0
	}
	conf.SaturatingFloatToUint32 = applicable.saturatingFloatToUint32
}

func converthashAlwaysEnabled(debugFlags string) bool {
	enabled := false
	for _, flag := range strings.Split(debugFlags, ",") {
		value, ok := strings.CutPrefix(flag, "converthash=")
		if !ok {
			continue
		}
		switch strings.ToLower(value) {
		case "y", "qy":
			enabled = true
		default:
			enabled = false
		}
	}
	return enabled
}

func countFlagEnabled(value string) bool {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	v, err := strconv.Atoi(value)
	return err == nil && v != 0
}

func countFlagIsOne(value string) bool {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	v, err := strconv.Atoi(value)
	return err == nil && v == 1
}
