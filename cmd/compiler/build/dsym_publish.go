// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Keep relative paths relative to the tool's working directory while preventing
// dsymutil from interpreting an output name beginning with '-' as an option.
func dsymutilInput(path string) string {
	if strings.HasPrefix(path, "-") {
		return "." + string(filepath.Separator) + path
	}
	return path
}

// dsymPublication shares bundle publication and rollback between compression
// and stripping. Callers must retain their staging directory when keepBackup
// is set, so a failed rollback never discards the previous bundle.
type dsymPublication struct {
	destination string
	backup      string
	keepBackup  bool
	files       symbolStripFileOps
}

func (p *dsymPublication) publish(staged, backup string) error {
	if st, err := p.files.lstat(p.destination); err == nil {
		if !st.IsDir() {
			return fmt.Errorf("dSYM output %q is occupied by a non-directory", p.destination)
		}
		if err := p.files.rename(p.destination, backup); err != nil {
			return err
		}
		p.backup = backup
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := p.files.rename(staged, p.destination); err != nil {
		return p.restore("publish dSYM", err)
	}
	return nil
}

// rollback removes the published bundle if publishing its executable failed.
func (p *dsymPublication) rollback(cause error) error {
	if err := p.files.removeAll(p.destination); err != nil {
		if p.backup != "" {
			p.keepBackup = true
			return errors.Join(cause, fmt.Errorf("publish stripped executable: previous dSYM retained at %s: %w", p.backup, err))
		}
		return errors.Join(cause, fmt.Errorf("cannot remove newly published dSYM %s: %w", p.destination, err))
	}
	return p.restore("publish stripped executable", cause)
}

func (p *dsymPublication) restore(action string, cause error) error {
	if p.backup != "" {
		if err := p.files.rename(p.backup, p.destination); err != nil {
			p.keepBackup = true
			return errors.Join(cause, fmt.Errorf("%s: previous dSYM retained at %s: %w", action, p.backup, err))
		}
	}
	return cause
}
