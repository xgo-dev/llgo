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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xgo-dev/llgo/cmd/compiler/llgen"
	"github.com/xgo-dev/llgo/cmd/xtool/env/llvm"
)

var (
	phase = flag.String("phase", string(llgen.PhasePreABI), "compiler phase to capture (pre-abi or post-abi)")
	abi   = flag.Int("abi", 0, "deprecated; use -phase instead (0 = pre-abi, 2 = post-abi)")
)

func main() {
	llvm.SetupPath()
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: llgen [flags] <pkg>")
		return
	}

	phaseSet := false
	abiSet := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "phase":
			phaseSet = true
		case "abi":
			abiSet = true
		}
	})
	selected, err := selectPhase(*phase, phaseSet, *abi, abiSet)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	llgen.SmartDoFileAtPhase(flag.Args()[0], selected)
}

func selectPhase(phaseValue string, phaseSet bool, abiValue int, abiSet bool) (llgen.Phase, error) {
	if phaseSet && abiSet {
		return "", fmt.Errorf("-phase and deprecated -abi cannot be used together")
	}
	if abiSet {
		switch abiValue {
		case 0:
			return llgen.PhasePreABI, nil
		case 2:
			return llgen.PhasePostABI, nil
		default:
			return "", fmt.Errorf("invalid -abi=%d: use 0 (pre-abi) or 2 (post-abi)", abiValue)
		}
	}
	selected := llgen.Phase(phaseValue)
	if selected != llgen.PhasePreABI && selected != llgen.PhasePostABI {
		return "", fmt.Errorf("invalid -phase=%q: use pre-abi or post-abi", selected)
	}
	return selected, nil
}
