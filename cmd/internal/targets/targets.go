/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package targets implements the "llgo targets" command.
package targets

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/internal/mockable"
	targetcfg "github.com/xgo-dev/llgo/internal/targets"
)

var Cmd = &base.Command{
	UsageLine: "llgo targets [-json] [name ...]",
	Short:     "List and inspect LLGo target configurations",
	Run:       runCmd,
}

func runCmd(_ *base.Command, args []string) {
	if err := run(args, os.Stdout, os.Stderr, targetcfg.NewDefaultResolver()); err != nil {
		fmt.Fprintln(os.Stderr, "llgo targets:", err)
		mockable.Exit(1)
	}
}

type targetInfo struct {
	// Config.Name is intentionally omitted from JSON; expose the stable target
	// name once at the top level of each resolved object.
	Name string `json:"name"`
	*targetcfg.Config
}

func run(args []string, stdout, stderr io.Writer, resolver *targetcfg.Resolver) error {
	fs := flag.NewFlagSet("llgo targets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonMode := fs.Bool("json", false, "print resolved target configurations as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	names := fs.Args()
	explicitNames := len(names) != 0
	if len(names) == 0 {
		var err error
		names, err = resolver.ListAvailableTargets()
		if err != nil {
			return err
		}
	}
	slices.Sort(names)
	if !*jsonMode {
		for _, name := range names {
			if explicitNames {
				if _, err := resolver.Resolve(name); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(stdout, name); err != nil {
				return err
			}
		}
		return nil
	}

	configs := make([]targetInfo, 0, len(names))
	for _, name := range names {
		config, err := resolver.Resolve(name)
		if err != nil {
			return err
		}
		configs = append(configs, targetInfo{Name: name, Config: config})
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "\t")
	return encoder.Encode(configs)
}
