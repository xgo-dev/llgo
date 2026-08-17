package llvm

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var dottedVersion = regexp.MustCompile(`(?:^|[^0-9])([0-9]+)\.[0-9]+(?:\.[0-9]+)?`)

// ValidateToolchainMajor checks that every named LLVM tool has the same major
// version as the LLVM library linked into the current LLGo process.
func ValidateToolchainMajor(linkedVersion string, tools ...string) error {
	linkedMajor, err := parseMajorVersion(linkedVersion)
	if err != nil {
		return fmt.Errorf("parse linked LLVM version %q: %w", linkedVersion, err)
	}
	for _, tool := range tools {
		output, err := exec.Command(tool, "--version").CombinedOutput()
		if err != nil {
			return fmt.Errorf("query LLVM tool %q: %w", tool, err)
		}
		toolVersion := strings.TrimSpace(string(output))
		toolMajor, err := parseMajorVersion(toolVersion)
		if err != nil {
			return fmt.Errorf("parse LLVM tool %q version %q: %w", tool, toolVersion, err)
		}
		if toolMajor != linkedMajor {
			return fmt.Errorf("LLVM major version mismatch: linked LLVM %s, %s reports %s", linkedVersion, tool, firstLine(toolVersion))
		}
	}
	return nil
}

func parseMajorVersion(version string) (int, error) {
	match := dottedVersion.FindStringSubmatch(version)
	if len(match) != 2 {
		return 0, fmt.Errorf("no dotted version found")
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	return major, nil
}

func firstLine(value string) string {
	if i := strings.IndexByte(value, '\n'); i >= 0 {
		return value[:i]
	}
	return value
}
