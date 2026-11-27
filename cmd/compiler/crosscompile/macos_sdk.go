package crosscompile

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type macOSSDK struct {
	path    string
	version string
	parts   []int
}

// Keep native Darwin on the LLVM 22 toolchain. SDK 27 introduces text stubs
// that LLD 22 cannot read; newer SDKs require a separate compatibility update.
func readMacOSSDK(path string) (macOSSDK, error) {
	sdk := macOSSDK{}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return sdk, err
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return sdk, err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return sdk, err
	}
	if !info.IsDir() {
		return sdk, fmt.Errorf("not a directory")
	}
	var settings struct{ Version string }
	data, err := os.ReadFile(filepath.Join(canonical, "SDKSettings.json"))
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return sdk, fmt.Errorf("read SDK version: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return sdk, err
	}
	version := settings.Version
	if version == "" {
		name := filepath.Base(canonical)
		if strings.HasPrefix(name, "MacOSX") && strings.HasSuffix(name, ".sdk") {
			version = strings.TrimSuffix(strings.TrimPrefix(name, "MacOSX"), ".sdk")
		}
	}
	var parts []int
	for _, part := range strings.Split(version, ".") {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return sdk, fmt.Errorf("cannot determine numeric macOS SDK version from %q", version)
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return sdk, fmt.Errorf("invalid macOS SDK version %q", version)
		}
		parts = append(parts, n)
	}
	if parts[0] < 1 || parts[0] >= 27 {
		return sdk, fmt.Errorf("macOS SDK %s is unsupported; LLVM 22 requires an SDK older than 27", version)
	}
	return macOSSDK{path: canonical, version: version, parts: parts}, nil
}

// Honor an explicit SDKROOT without fallback. Otherwise keep the active SDK
// when supported, and search installed alternatives only when it is unusable.
func selectMacOSSDK(explicit, preferred string, discover func() []string) (macOSSDK, error) {
	if explicit != "" {
		sdk, err := readMacOSSDK(explicit)
		if err != nil {
			return sdk, fmt.Errorf("SDKROOT %q: %w", explicit, err)
		}
		return sdk, nil
	}
	if preferred != "" {
		if sdk, err := readMacOSSDK(preferred); err == nil {
			return sdk, nil
		}
	}
	var candidates []macOSSDK
	seen := make(map[string]bool)
	for _, path := range discover() {
		sdk, err := readMacOSSDK(path)
		if err == nil && !seen[sdk.path] {
			seen[sdk.path] = true
			candidates = append(candidates, sdk)
		}
	}
	slices.SortFunc(candidates, func(a, b macOSSDK) int {
		for i := 0; i < max(len(a.parts), len(b.parts)); i++ {
			var av, bv int
			if i < len(a.parts) {
				av = a.parts[i]
			}
			if i < len(b.parts) {
				bv = b.parts[i]
			}
			if av < bv {
				return 1
			}
			if av > bv {
				return -1
			}
		}
		return strings.Compare(a.path, b.path)
	})
	if len(candidates) != 0 {
		return candidates[0], nil
	}
	return macOSSDK{}, fmt.Errorf("no supported macOS SDK found (active SDK: %q); install an SDK older than 27 with Xcode or Command Line Tools, or set SDKROOT to an installed older SDK", preferred)
}

func installedMacOSSDKs(preferred, developer, home string) []string {
	var patterns []string
	if preferred != "" {
		patterns = append(patterns, filepath.Join(filepath.Dir(preferred), "MacOSX*.sdk"))
	}
	if strings.HasSuffix(developer, ".app") {
		developer = filepath.Join(developer, "Contents", "Developer")
	}
	if developer != "" {
		patterns = append(patterns,
			filepath.Join(developer, "SDKs", "MacOSX*.sdk"),
			filepath.Join(developer, "Platforms", "MacOSX.platform", "Developer", "SDKs", "MacOSX*.sdk"))
	}
	patterns = append(patterns,
		"/Library/Developer/CommandLineTools/SDKs/MacOSX*.sdk",
		"/Applications/*.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX*.sdk")
	if home != "" {
		patterns = append(patterns, filepath.Join(home, "Applications", "*.app", "Contents", "Developer", "Platforms", "MacOSX.platform", "Developer", "SDKs", "MacOSX*.sdk"))
	}
	var paths []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		paths = append(paths, matches...)
	}
	return paths
}

func getMacOSSDK() (macOSSDK, error) {
	if explicit := os.Getenv("SDKROOT"); explicit != "" {
		return selectMacOSSDK(explicit, "", nil)
	}
	output, _ := exec.Command("xcrun", "--sdk", "macosx", "--show-sdk-path").Output()
	preferred := strings.TrimSpace(string(output))
	return selectMacOSSDK("", preferred, func() []string {
		developer := os.Getenv("DEVELOPER_DIR")
		if developer == "" {
			out, _ := exec.Command("xcode-select", "-p").Output()
			developer = strings.TrimSpace(string(out))
		}
		return installedMacOSSDKs(preferred, developer, os.Getenv("HOME"))
	})
}
