package firmware

import (
	"fmt"
	"os"
	"strings"
)

// makeFirmwareImage creates a firmware image from the given input file.
func makeFirmwareImage(infile, outfile, format, fmtDetail string) error {
	fmt.Fprintf(os.Stderr, "Generating firmware image: %s -> %s (format: %s, detail: %s)\n", infile, outfile, format, fmtDetail)
	if strings.HasPrefix(format, "esp") {
		return makeESPFirmareImage(infile, outfile, format)
	} else if format == "uf2" {
		uf2Family := fmtDetail
		return convertELFFileToUF2File(infile, outfile, uf2Family)
	} else if format == "nrf-dfu" {
		return makeDFUFirmwareImage(infile, outfile)
	}
	return fmt.Errorf("unsupported firmware format: %s", format)
}

// ExtractFileFormatFromCommand extracts file format from command template
// Returns the format if found (e.g. "bin", "hex", "zip", "img"), empty string if not found
func ExtractFileFormatFromCommand(cmd string) string {
	formats := []string{"bin", "hex", "zip", "img", "uf2"}
	for _, format := range formats {
		if strings.Contains(cmd, "{"+format+"}") {
			return format
		}
	}
	return ""
}

// ConvertFormats processes format conversions for embedded targets only
func ConvertFormats(binFmt, fmtDetail string, envMap map[string]string) error {
	var err error
	// Convert to bin format first (needed for img)
	if envMap["bin"] != "" {
		if strings.HasPrefix(binFmt, "esp") {
			err = makeFirmwareImage(envMap["out"], envMap["bin"], binFmt, fmtDetail)
		} else {
			err = objcopy(envMap["out"], envMap["bin"], "bin")
		}
		if err != nil {
			return fmt.Errorf("failed to convert to bin format: %w", err)
		}
	}

	// Convert to hex format
	if envMap["hex"] != "" {
		err := objcopy(envMap["out"], envMap["hex"], "hex")
		if err != nil {
			return fmt.Errorf("failed to convert to hex format: %w", err)
		}
	}

	// Convert to img format
	if envMap["img"] != "" {
		err = makeFirmwareImage(envMap["out"], envMap["img"], binFmt+"-img", fmtDetail)
		if err != nil {
			return fmt.Errorf("failed to convert to img format: %w", err)
		}
	}

	// Convert to uf2 format
	if envMap["uf2"] != "" {
		err := makeFirmwareImage(envMap["out"], envMap["uf2"], binFmt, fmtDetail)
		if err != nil {
			return fmt.Errorf("failed to convert to uf2 format: %w", err)
		}
	}

	// Convert to zip format
	if envMap["zip"] != "" {
		err := makeFirmwareImage(envMap["out"], envMap["zip"], binFmt, fmtDetail)
		if err != nil {
			return fmt.Errorf("failed to convert to zip format: %w", err)
		}
	}

	return nil
}
