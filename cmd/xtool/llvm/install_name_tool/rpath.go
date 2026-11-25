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

package install_name_tool

import (
	"io"
	"os/exec"
	"path/filepath"
)

// Cmd represents a install_name_tool command.
type Cmd struct {
	app string

	Stdout io.Writer
	Stderr io.Writer
}

// New creates a new install_name_tool command.
func New(app string) *Cmd {
	if app == "" {
		app = "install_name_tool"
	}
	return &Cmd{app: app}
}

// Exec executes a install_name_tool command.
func (p *Cmd) Exec(args ...string) error {
	cmd := exec.Command(p.app, args...)
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	return cmd.Run()
}

// Change represents a dependent shared library install name change.
type Change struct {
	Old string
	New string
}

// Change changes dependent shared library install name.
func (p *Cmd) Change(target string, chgs ...Change) error {
	args := make([]string, 0, len(chgs)*3+1)
	for _, chg := range chgs {
		args = append(args, "-change", chg.Old, chg.New)
	}
	args = append(args, target)
	return p.Exec(args...)
}

// ChangeToRpath changes dependent shared library install name to @rpath.
func (p *Cmd) ChangeToRpath(target string, dylibDeps ...string) error {
	args := make([]string, 0, len(dylibDeps)*3+1)
	for _, dep := range dylibDeps {
		args = append(args, "-change", dep, "@rpath/"+filepath.Base(dep))
	}
	args = append(args, target)
	return p.Exec(args...)
}
