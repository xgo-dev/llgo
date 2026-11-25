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

package build

import (
	"debug/pe"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	llvmenv "github.com/xgo-dev/llgo/cmd/xtool/env/llvm"
)

func TestWindowsPDBLinkPath(t *testing.T) {
	if runtime.GOOS != "windows" || os.Getenv("LLGO_WINDOWS_ABI") != "msvc" {
		t.Skip("native MSVC linker integration test")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "pdb-standard.exe")
	pdb := filepath.Join(dir, "pdb-standard.pdb")
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = bin
	conf.OptLevel = optlevel.O0
	conf.LinkOptions.DWARF = DWARFPreserve
	// Keep the default Go-compatible DWARF while asking lld-link to publish a
	// companion PDB. The PDB supplies native Windows public symbols and source
	// lines; LLDB and runtime tracebacks continue to use embedded DWARF/PCLN.
	conf.LinkOptions.ExternalLinkerFlags = fmt.Sprintf(
		`-Xlinker /debug:full -Xlinker "/pdb:%s"`, filepath.ToSlash(pdb))
	if _, err := Do([]string{"./testdata/dwarf"}, conf); err != nil {
		t.Fatalf("build Windows PDB fixture: %v", err)
	}
	if got := runBinary(t, bin); got != "dwarf-ok\n" {
		t.Fatalf("PDB fixture output = %q", got)
	}

	info, err := os.Stat(pdb)
	if err != nil {
		t.Fatalf("stat PDB: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("PDB is empty")
	}

	file, err := pe.Open(bin)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.DWARF(); err != nil {
		t.Fatalf("PDB link stripped embedded DWARF: %v", err)
	}

	binDir := llvmenv.New("").BinDir()
	pdbutil := filepath.Join(binDir, "llvm-pdbutil")
	out, err := exec.Command(pdbutil, "dump", "-summary", "-publics", "-files", "-l", pdb).CombinedOutput()
	if err != nil {
		t.Fatalf("llvm-pdbutil: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{
		"Has Publics: true",
		"main.main",
		"main.go",
		"helper.go",
		"line/addr entries",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("PDB output does not contain %q:\n%s", want, out)
		}
	}

	readobj := filepath.Join(binDir, "llvm-readobj")
	out, err = exec.Command(readobj, "--coff-debug-directory", bin).CombinedOutput()
	if err != nil {
		t.Fatalf("llvm-readobj: %v\n%s", err, out)
	}
	if !strings.Contains(strings.ToLower(string(out)), strings.ToLower(filepath.Base(pdb))) {
		t.Fatalf("PE debug directory does not reference %s:\n%s", filepath.Base(pdb), out)
	}
}
