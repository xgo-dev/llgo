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

package build

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goplus/llgo/internal/processenv"
)

func cmpTest(process processenv.Context, dir, pkgPath, llApp string, genExpect bool, runArgs []string) {
	var llgoOut, llgoErr bytes.Buffer
	var llgoRunErr = runApp(process, runArgs, dir, &llgoOut, &llgoErr, llApp)

	llgoExpect := formatExpect(llgoOut.Bytes(), llgoErr.Bytes(), llgoRunErr)
	llgoExpectFile := filepath.Join(dir, "llgo.expect")
	if genExpect {
		if _, err := os.Stat(llgoExpectFile); !errors.Is(err, os.ErrNotExist) {
			fatal(fmt.Errorf("llgo.expect file already exists: %s", llgoExpectFile))
		}
		if err := os.WriteFile(llgoExpectFile, llgoExpect, 0644); err != nil {
			fatal(err)
		}
		return
	}
	if b, err := os.ReadFile(llgoExpectFile); err == nil {
		checkEqual("llgo.expect", llgoExpect, b)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		fatal(err)
	}

	var goOut, goErr bytes.Buffer
	var goRunErr = runApp(process, runArgs, dir, &goOut, &goErr, "go", "run", pkgPath)

	checkEqual("output", llgoOut.Bytes(), goOut.Bytes())
	checkEqual("stderr", llgoErr.Bytes(), goErr.Bytes())
	checkEqualRunErr(llgoRunErr, goRunErr)
}

func formatExpect(stdout, stderr []byte, runErr error) []byte {
	var exitCode int
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else { // This should never happen, but just in case.
			exitCode = 255
		}
	}
	return []byte(fmt.Sprintf("#stdout\n%s\n#stderr\n%s\n#exit %d\n", stdout, stderr, exitCode))
}

func checkEqualRunErr(llgoRunErr, goRunErr error) {
	if llgoRunErr == goRunErr {
		return
	}
	fmt.Fprintln(os.Stderr, "=> Exit:", llgoRunErr)
	fmt.Fprintln(os.Stderr, "\n=> Expected Exit:", goRunErr)
}

func checkEqual(prompt string, a, expected []byte) {
	if bytes.Equal(a, expected) {
		return
	}

	fmt.Fprintln(os.Stderr, "=> Result of", prompt)
	os.Stderr.Write(a)

	fmt.Fprintln(os.Stderr, "\n=> Expected", prompt)
	os.Stderr.Write(expected)

	fatal(errors.New("checkEqual: unexpected " + prompt))
}

func runApp(process processenv.Context, runArgs []string, dir string, stdout, stderr io.Writer, app string, args ...string) error {
	if len(runArgs) > 0 {
		if len(args) > 0 {
			args = append(args, runArgs...)
		} else {
			args = runArgs
		}
	}
	cmd := process.Command(app, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func fatal(err error) {
	log.Panicln(err)
}
