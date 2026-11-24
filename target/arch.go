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

// Package target provides pure-Go target specifications, architecture settings,
// LLVM target triple derivation, data-layout parameters, and JSON target
// configuration loading with zero CGo or LLVM library dependencies.
package target

import (
	"fmt"
	"strings"
)

const (
	Default386   = "sse2"
	DefaultAMD64 = "v1"
	DefaultARM   = "7"
	DefaultARM64 = "v8.0"
)

// Resolve386 validates and normalizes GO386. An empty value selects the Go
// toolchain default. Invalid input returns that default together with an error.
func Resolve386(value string) (string, error) {
	if value == "" {
		return Default386, nil
	}
	switch value {
	case "sse2", "softfloat":
		return value, nil
	case "387":
		return Default386, fmt.Errorf("unsupported setting GO386=387. Consider using GO386=softfloat instead.")
	default:
		return Default386, fmt.Errorf("unsupported setting GO386=%s", value)
	}
}

// ResolveAMD64 validates and normalizes GOAMD64. An empty value selects the Go
// toolchain default. Invalid input returns that default together with an error.
func ResolveAMD64(value string) (string, error) {
	if value == "" {
		return DefaultAMD64, nil
	}
	switch value {
	case "v1", "v2", "v3", "v4":
		return value, nil
	default:
		return DefaultAMD64, fmt.Errorf("invalid GOAMD64: must be v1, v2, v3, v4")
	}
}

// ARM describes a normalized GOARM value.
type ARM struct {
	Version   string
	SoftFloat bool
}

func (f ARM) String() string {
	if f.SoftFloat {
		return f.Version + ",softfloat"
	}
	return f.Version + ",hardfloat"
}

// ParseARM validates and normalizes GOARM. ARMv5 defaults to software floating
// point; ARMv6 and ARMv7 default to hardware floating point, matching Go.
func ParseARM(value string) (ARM, error) {
	if value == "" {
		value = DefaultARM
	}
	const (
		softFloat = ",softfloat"
		hardFloat = ",hardfloat"
	)
	var ret ARM
	floatSet := false
	if strings.HasSuffix(value, softFloat) {
		ret.SoftFloat = true
		floatSet = true
		value = strings.TrimSuffix(value, softFloat)
	} else if strings.HasSuffix(value, hardFloat) {
		floatSet = true
		value = strings.TrimSuffix(value, hardFloat)
	}
	switch value {
	case "5", "6", "7":
		ret.Version = value
	default:
		ret.Version = DefaultARM
		return ret, fmt.Errorf("invalid GOARM: must start with 5, 6, or 7, and may optionally end in either %q or %q", hardFloat, softFloat)
	}
	if !floatSet && ret.Version == "5" {
		ret.SoftFloat = true
	}
	return ret, nil
}

// ARM64 describes a normalized GOARM64 value.
type ARM64 struct {
	Version string
	LSE     bool
	Crypto  bool
}

func (f ARM64) String() string {
	value := f.Version
	if f.LSE {
		value += ",lse"
	}
	if f.Crypto {
		value += ",crypto"
	}
	return value
}

// ParseARM64 validates and normalizes GOARM64. Extension suffixes may occur in
// either order, matching the Go toolchain. LSE is mandatory from ARMv8.1.
func ParseARM64(value string) (ARM64, error) {
	if value == "" {
		value = DefaultARM64
	}
	var ret ARM64
	for {
		if strings.HasSuffix(value, ",lse") {
			ret.LSE = true
			value = strings.TrimSuffix(value, ",lse")
			continue
		}
		if strings.HasSuffix(value, ",crypto") {
			ret.Crypto = true
			value = strings.TrimSuffix(value, ",crypto")
			continue
		}
		break
	}
	switch value {
	case "v8.0":
		ret.Version = value
	case "v8.1", "v8.2", "v8.3", "v8.4", "v8.5", "v8.6", "v8.7", "v8.8", "v8.9",
		"v9.0", "v9.1", "v9.2", "v9.3", "v9.4", "v9.5":
		ret.Version = value
		ret.LSE = true
	default:
		ret.Version = DefaultARM64
		return ret, fmt.Errorf("invalid GOARM64: must start with v8.{0-9} or v9.{0-5} and may optionally end in %q and/or %q", ",lse", ",crypto")
	}
	return ret, nil
}
