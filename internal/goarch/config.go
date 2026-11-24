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

// Package goarch delegates to the public pure-Go github.com/xgo-dev/llgo/target package.
package goarch

import "github.com/xgo-dev/llgo/target"

const (
	Default386   = target.Default386
	DefaultAMD64 = target.DefaultAMD64
	DefaultARM   = target.DefaultARM
	DefaultARM64 = target.DefaultARM64
)

type (
	ARM   = target.ARM
	ARM64 = target.ARM64
)

func Resolve386(value string) (string, error)   { return target.Resolve386(value) }
func ResolveAMD64(value string) (string, error) { return target.ResolveAMD64(value) }
func ParseARM(value string) (ARM, error)        { return target.ParseARM(value) }
func ParseARM64(value string) (ARM64, error)    { return target.ParseARM64(value) }
