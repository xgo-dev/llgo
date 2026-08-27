// Package llvmpayload defines the revision-locked LLVM toolchain distributed
// with and downloaded by LLGo.
package llvmpayload

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

const releaseBaseURL = "https://github.com/goplus/espressif-llvm-project-prebuilt/releases/download"

// DefaultMajor is the LLVM payload bundled into LLGo release archives.
const DefaultMajor = 21

var llvmMajorPattern = regexp.MustCompile(`(?:^|[^0-9])([0-9]+)\.[0-9]+`)

type manifest struct {
	llvmMajor         int
	version           string
	compilerRTVersion string
	sha256            map[string]string
}

// Artifact identifies one host-specific LLVM payload archive.
type Artifact struct {
	Platform string
	Version  string
	URL      string
	SHA256   string
}

var manifests = map[int]manifest{
	21: {
		llvmMajor:         21,
		version:           "21.1.3_20260816",
		compilerRTVersion: "xtensa_release_21.1.3_20260408",
		sha256: map[string]string{
			"aarch64-apple-darwin": "a8c46104501c38a8a7359ec24bc4e9d646f9fec2bdb2b122cbbee78e060400d1",
			"aarch64-linux-gnu":    "77f49d832e5f309ecd6baaf169c62e3b064b27f9bee5aedddb6e66c981d56f44",
			"x86_64-apple-darwin":  "21159a4edb8948d83e1f73dfef394bca6941d0c4035da02f8c90ac59799893fa",
			"x86_64-linux-gnu":     "582b787057c9e36e7d4db20aaed7bbba74c7ad0481489f034f09476703befbd5",
		},
	},
}

// ForLLVMVersion returns the payload compatible with an in-process LLVM
// version such as "21.1.8".
func ForLLVMVersion(version string) (Manifest, error) {
	match := llvmMajorPattern.FindStringSubmatch(version)
	if len(match) != 2 {
		return Manifest{}, fmt.Errorf("parse LLVM version %q", version)
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return Manifest{}, fmt.Errorf("parse LLVM version %q: %w", version, err)
	}
	return ForMajor(major)
}

// ForMajor returns the published payload for one LLVM major version.
func ForMajor(major int) (Manifest, error) {
	payload, ok := manifests[major]
	if !ok {
		return Manifest{}, fmt.Errorf("no LLGo LLVM payload for major version %d", major)
	}
	return Manifest{payload: payload}, nil
}

func Default() (Manifest, error) { return ForMajor(DefaultMajor) }

// Manifest provides read-only access to one payload release.
type Manifest struct {
	payload manifest
}

func (m Manifest) LLVMMajor() int { return m.payload.llvmMajor }

func (m Manifest) Version() string { return m.payload.version }

func (m Manifest) CompilerRTVersion() string { return m.payload.compilerRTVersion }

func (m Manifest) BaseURL() string {
	return releaseBaseURL + "/" + m.payload.version
}

func (m Manifest) Platforms() []string {
	platforms := make([]string, 0, len(m.payload.sha256))
	for platform := range m.payload.sha256 {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)
	return platforms
}

func (m Manifest) Artifact(platform string) (Artifact, error) {
	checksum, ok := m.payload.sha256[platform]
	if !ok {
		return Artifact{}, fmt.Errorf("LLVM %d payload %s is unavailable for %s", m.LLVMMajor(), m.Version(), platform)
	}
	filename := fmt.Sprintf("clang-esp-%s-%s.tar.xz", m.Version(), platform)
	return Artifact{
		Platform: platform,
		Version:  m.Version(),
		URL:      m.BaseURL() + "/" + filename,
		SHA256:   checksum,
	}, nil
}

// PlatformSuffix maps the host platform to the suffix used by payload assets.
func PlatformSuffix(goos, goarch string) (string, bool) {
	switch goos + "/" + goarch {
	case "darwin/amd64":
		return "x86_64-apple-darwin", true
	case "darwin/arm64":
		return "aarch64-apple-darwin", true
	case "linux/amd64":
		return "x86_64-linux-gnu", true
	case "linux/arm64":
		return "aarch64-linux-gnu", true
	default:
		return "", false
	}
}
