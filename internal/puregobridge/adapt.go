package puregobridge

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed backend/syscall_sysv.go.txt native/sysv_amd64.S
var assets embed.FS

// Changes contains paths relative to the module root. A caller applies it to a
// private staged module, never to GOMODCACHE. Files must pass through LLGo's
// normal LLGoFiles build path; merely copying the native source proves nothing.
type Changes struct {
	Files  map[string][]byte
	Remove []string
}

var sourceSHA256 = map[string]string{
	"cgo.go":            "3dcdbea296f8cb566b8332c16e1b038123000eba88d6cb43e14cd6dbac6dbfe2",
	"dlfcn.go":          "1654367b63c6b12329c211c8a357be3f478d47853078aa8725718863fbc0b832",
	"func.go":           "8de06dd18c5f3aca3c24c30777bb7072305733a08d382bad6b730d0741c18954",
	"go_runtime.go":     "d2c9cebb020a41414b706333612fcd5518b7cc327cfedf9cb2769e3b3ff085e3",
	"sys_amd64.s":       "d7e7e5487064ac0ef47db1ede7701ac0ff7818f42ee05577ca5769f47702c7ba",
	"syscall.go":        "45a0adf82f2b05371d057093ed9456a4797ed84a6ceb78d9fe6cb8c0692c9ac6",
	"syscall_sysv.go":   "30066e4370ecd9d5ddc2d8b0783d1c2265f01ddb137f0687b411a28b30e88e2d",
	"zcallback_amd64.s": "8c5a3c2e5a857505829536be6d10ecdcb59cda482caad13359979c2e23234087",
}

// Adapt selects a reviewed, exact-version backend. It does not infer ABI from
// instruction names, relax unsafe conversions, or adapt unknown versions.
// CGO_ENABLED=0 has a different fakecgo thread/TLS contract and is not enabled
// by this profile. Other platforms likewise need their own qualified profile.
func Adapt(modulePath, version, goos, goarch string, cgoEnabled bool, source fs.FS) (*Changes, error) {
	if modulePath != "github.com/ebitengine/purego" || version != "v0.9.1" {
		return nil, fmt.Errorf("purego adapter: unsupported module %s@%s", modulePath, version)
	}
	if goos != "linux" || goarch != "amd64" || !cgoEnabled {
		return nil, fmt.Errorf("purego adapter: v0.9.1 requires linux/amd64 with CGO_ENABLED=1")
	}
	if source == nil {
		return nil, fmt.Errorf("purego adapter: module source is required")
	}
	if err := verifySource(source, sourceSHA256); err != nil {
		return nil, err
	}
	backend, err := assets.ReadFile("backend/syscall_sysv.go.txt")
	if err != nil {
		return nil, err
	}
	native, err := assets.ReadFile("native/sysv_amd64.S")
	if err != nil {
		return nil, err
	}
	return &Changes{
		Files: map[string][]byte{
			"syscall_sysv.go":           backend,
			"_llgo_compat/sysv_amd64.S": native,
		},
		Remove: []string{"go_runtime.go", "sys_amd64.s", "zcallback_amd64.s"},
	}, nil
}

func verifySource(source fs.FS, expected map[string]string) error {
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content, err := fs.ReadFile(source, name)
		if err != nil {
			return fmt.Errorf("purego adapter: read %s: %w", name, err)
		}
		if actual := fmt.Sprintf("%x", sha256.Sum256(content)); actual != expected[name] {
			return fmt.Errorf("purego adapter: %s differs from the verified v0.9.1 source (sha256 %s)", name, actual)
		}
	}
	return nil
}
