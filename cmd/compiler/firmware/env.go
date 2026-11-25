package firmware

import "strings"

// BinaryFormatToEnvName returns the environment variable name based on the binary format
// Returns the format name for template expansion (e.g., "bin", "uf2", "zip")
func BinaryFormatToEnvName(binaryFormat string) string {
	if strings.HasPrefix(binaryFormat, "esp") {
		return "bin"
	} else if strings.HasPrefix(binaryFormat, "uf2") {
		return "uf2"
	} else if strings.HasPrefix(binaryFormat, "nrf-dfu") {
		return "zip"
	}
	return ""
}
