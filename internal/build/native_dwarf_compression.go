// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package build

import (
	"bytes"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/dwarfcompress"
)

func nativeDSYMPath(output string) string { return output + ".dSYM" }

const generatedDSYMMarker = "llgo-zlib-dwarf-v1\n"

func dsymMarkerPath(bundle string) string {
	return filepath.Join(bundle, "Contents", "Resources", "llgo-zlib-dwarf")
}

func removeGeneratedDSYM(output string) error {
	bundle := nativeDSYMPath(output)
	marker, err := os.ReadFile(dsymMarkerPath(bundle))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if string(marker) == generatedDSYMMarker {
		return os.RemoveAll(bundle)
	}
	return nil
}

func finalizeNativeDWARFCompression(ctx *context, output string, verbose bool) error {
	conf := ctx.buildConf
	if conf.Target != "" || conf.BuildMode == BuildModeCArchive {
		return nil
	}
	if conf.LinkOptions.DWARFCompression != DWARFCompressionEnabled || !shouldEmitDebugInfo(conf, &ctx.crossCompile) {
		if ctx.crossCompile.Toolchain.ObjectFormat == crosscompile.ObjectFormatMachO {
			// A previous compressed bundle can share the rebuilt image's UUID
			// and mask its debug map in LLDB, even after opting out.
			return removeGeneratedDSYM(output)
		}
		return nil
	}
	switch ctx.crossCompile.Toolchain.ObjectFormat {
	case crosscompile.ObjectFormatMachO:
		if shouldStripNativeSymbols(conf, &ctx.crossCompile) {
			// Symbol stripping already generated and compressed a matching
			// bundle before discarding the executable's debug map.
			return nil
		}
		return compressDarwinDSYM(ctx, output, verbose)
	case crosscompile.ObjectFormatCOFF:
		return compressNativeArtifact(output)
	default:
		return nil // ELF is compressed by its linker; other backends are unchanged.
	}
}

// compressNativeArtifact stages all byte/header edits before replacing the
// published artifact. In particular an unsupported layout or I/O failure must
// not leave a partially rewritten executable.
func compressNativeArtifact(path string) (err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	compressed, err := dwarfcompress.Compress(raw)
	if err != nil {
		return fmt.Errorf("compress DWARF in %s: %w", path, err)
	}
	if bytes.Equal(raw, compressed) {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	return writeDebugArtifactFile(path, compressed, st.Mode())
}

func compressDarwinDSYM(ctx *context, output string, verbose bool) (err error) {
	root, err := os.MkdirTemp(filepath.Dir(output), "."+filepath.Base(output)+".dsym-*")
	if err != nil {
		return err
	}
	publication := dsymPublication{destination: nativeDSYMPath(output), files: symbolStripOSFileOps()}
	defer func() {
		if !publication.keepBackup {
			err = errors.Join(err, os.RemoveAll(root))
		}
	}()
	bundle := filepath.Join(root, filepath.Base(output)+".dSYM")
	cmd := ctx.commands.configure(exec.Command("dsymutil", dsymutilInput(output), "-o", bundle))
	if verbose {
		fmt.Fprintln(os.Stderr, cmd)
	}
	if data, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generate dSYM for zlib compression: %w: %s", err, data)
	}
	if err := compressDSYMBundle(bundle, filepath.Base(output)); err != nil {
		return err
	}
	return publication.publish(bundle, filepath.Join(root, "previous.dSYM"))
}

func compressDSYMBundle(bundle, imageName string) error {
	path := filepath.Join(bundle, "Contents", "Resources", "DWARF", imageName)
	if err := validateDSYMDWARF(path); err != nil {
		return err
	}
	if err := compressNativeArtifact(path); err != nil {
		return err
	}
	return os.WriteFile(dsymMarkerPath(bundle), []byte(generatedDSYMMarker), 0o644)
}

func validateDSYMDWARF(path string) error {
	f, err := macho.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := f.DWARF()
	if err != nil {
		return fmt.Errorf("generated dSYM has no usable DWARF: %w", err)
	}
	if entry, err := data.Reader().Next(); err != nil || entry == nil {
		return fmt.Errorf("generated dSYM has no DWARF entries (check the debug-map objects): %v", err)
	}
	return nil
}
