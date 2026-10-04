package build

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xgo-dev/llgo/internal/sizereport"
	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

func reportFinalSize(conf *Config, out *OutFmtDetails, pkgs []Package, artifacts []Artifact, w io.Writer) error {
	packages := sizeReportPackages(pkgs)
	path := debugWasmModulePath(conf, out.Out)
	report, err := sizereport.Collect(path, packages, conf.SizeLevel)
	if errors.Is(err, sizereport.ErrUnsupportedFormat) {
		report, err = collectReadelfSize(path, packages, conf.SizeLevel)
	}
	if err != nil {
		return fmt.Errorf("size report: %w", err)
	}
	for _, a := range artifacts {
		report.Artifacts = append(report.Artifacts, sizereport.Artifact{
			Path: a.Path, Format: a.Format, Role: string(a.Role), Size: a.Size,
		})
	}
	return sizereport.Write(w, report, conf.SizeFormat)
}

func sizeReportPackages(pkgs []Package) []sizereport.Package {
	result := make([]sizereport.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Package == nil {
			continue
		}
		info := sizereport.Package{Path: pkg.PkgPath}
		if pkg.Module != nil {
			info.Module = pkg.Module.Path
		}
		result = append(result, info)
	}
	return result
}

func collectReadelfSize(path string, pkgs []sizereport.Package, level string) (*sizereport.Report, error) {
	cmd, err := llvm.New("").Readelf("--elf-output-style=LLVM", "--all", path)
	if err != nil {
		return nil, fmt.Errorf("llvm-readelf: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("llvm-readelf stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to execute llvm-readelf: %w", err)
	}
	report, parseErr := sizereport.ReadReadelf(path, stdout, pkgs, level)
	closeErr := stdout.Close()
	waitErr := cmd.Wait()
	if waitErr != nil {
		return nil, fmt.Errorf("llvm-readelf failed: %w\n%s", waitErr, stderr.String())
	}
	if parseErr != nil {
		return nil, fmt.Errorf("parsing llvm-readelf output failed: %w", parseErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing llvm-readelf stdout pipe failed: %w", closeErr)
	}
	return report, nil
}

func ensureSizeReporting(conf *Config) error {
	if !conf.SizeReport {
		return nil
	}
	switch strings.ToLower(conf.SizeLevel) {
	case "", "module":
		conf.SizeLevel = "module"
	case "package", "full":
		conf.SizeLevel = strings.ToLower(conf.SizeLevel)
	default:
		return fmt.Errorf("invalid size level %q (valid: full,module,package)", conf.SizeLevel)
	}
	switch strings.ToLower(conf.SizeFormat) {
	case "", "text":
		conf.SizeFormat = "text"
	case "json":
		conf.SizeFormat = "json"
	default:
		return fmt.Errorf("invalid size format %q (valid: text,json)", conf.SizeFormat)
	}
	return nil
}
