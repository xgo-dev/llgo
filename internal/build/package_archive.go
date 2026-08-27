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
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	gllvm "github.com/xgo-dev/llvm"
)

// packageArchiveBuffer owns an LLVM-produced archive member until package
// publication.
type packageArchiveBuffer struct {
	name   string
	buffer gllvm.MemoryBuffer
}

func (p *aPackage) disposeArchiveBuffers() {
	for i := range p.ObjBuffers {
		member := &p.ObjBuffers[i]
		if member.buffer.IsNil() {
			continue
		}
		member.buffer.Dispose()
		member.buffer = gllvm.MemoryBuffer{}
	}
	p.ObjBuffers = nil
}

func (c *context) closePackageArchiveBuffers() {
	for _, pkg := range c.pkgs {
		pkg.disposeArchiveBuffers()
	}
}

// createPackageArchiveFile writes path-backed auxiliary objects and
// LLVM-produced memory buffers into one archive. LLVM performs archive symbol
// indexing in-process; no temporary object is needed for memory members.
func (c *context) createPackageArchiveFile(archivePath string, pkg *aPackage, verbose bool) error {
	if len(pkg.ObjFiles) == 0 && len(pkg.ObjBuffers) == 0 {
		return fmt.Errorf("no object files provided for archive %s", archivePath)
	}
	if len(pkg.ObjBuffers) == 0 {
		return c.createArchiveFile(archivePath, pkg.ObjFiles, verbose)
	}

	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return err
	}
	// LLVM's in-process archive writer installs fatal-signal handlers while
	// creating its temporary archive. On Linux that replaces the SIGXCPU
	// handler BDWGC uses to resume stopped threads, so a self-hosted compiler
	// can terminate during its next collection. Keep in-memory code generation,
	// but let an external archiver publish those buffers when llgo is the host
	// compiler.
	if useExternalPackageArchiver {
		return c.createPackageArchiveFileWithExternalArchiver(archivePath, pkg, verbose)
	}
	tmp, err := os.CreateTemp(filepath.Dir(archivePath), filepath.Base(archivePath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Remove(tmpName); err != nil {
		return err
	}

	members := make([]gllvm.ArchiveMember, 0, len(pkg.ObjFiles)+len(pkg.ObjBuffers))
	for _, path := range pkg.ObjFiles {
		members = append(members, gllvm.NewArchiveMemberFromFile(path))
	}
	for _, member := range pkg.ObjBuffers {
		members = append(members, gllvm.NewArchiveMemberFromMemoryBuffer(member.name, member.buffer))
	}
	if c.shouldPrintCommands(verbose) {
		fmt.Fprintf(os.Stderr, "# llvm archive %s (%d file members, %d memory members)\n",
			tmpName, len(pkg.ObjFiles), len(pkg.ObjBuffers))
	}
	if err := gllvm.WriteArchive(tmpName, c.targetTriple(), members); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("create archive %s: %w", archivePath, err)
	}
	if err := os.Rename(tmpName, archivePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("publish archive %s: %w", archivePath, err)
	}
	return nil
}

func (c *context) createPackageArchiveFileWithExternalArchiver(archivePath string, pkg *aPackage, verbose bool) error {
	membersDir, err := os.MkdirTemp(filepath.Dir(archivePath), filepath.Base(archivePath)+".members-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(membersDir)

	objFiles := append([]string(nil), pkg.ObjFiles...)
	for i, member := range pkg.ObjBuffers {
		if member.name == "" {
			return fmt.Errorf("archive member %d has an empty name", i)
		}
		if member.buffer.IsNil() {
			return fmt.Errorf("archive member %d (%q) has a nil buffer", i, member.name)
		}
		memberDir := filepath.Join(membersDir, strconv.Itoa(i))
		if err := os.Mkdir(memberDir, 0o755); err != nil {
			return err
		}
		memberPath := filepath.Join(memberDir, filepath.Base(member.name))
		if err := os.WriteFile(memberPath, member.buffer.Bytes(), 0o644); err != nil {
			return err
		}
		objFiles = append(objFiles, memberPath)
	}
	return c.createArchiveFile(archivePath, objFiles, verbose)
}
