package crosscompile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Keep this release and its checksums in sync with
// .github/workflows/install-esp-qemu.sh (also used by Windows CI).
const espQEMUVersion = "9.2.2-20250817"

var espQEMUBaseURL = "https://github.com/espressif/qemu/releases/download/esp-develop-" + espQEMUVersion

// Espressif publishes separate RISC-V and Xtensa payloads. Keep their cache
// directories separate: both archives contain a qemu/ tree and shared ROMs.
var espQEMUChecksums = map[string]map[string]string{
	"aarch64-apple-darwin": {
		"riscv32": "234690b6fa7c1d5dfe3dbb2bdd0c2810755e7c98999a9f21c389a6046b7eb76d",
		"xtensa":  "aa92e337461d482f5d9f31cd8efc0bd67b3de8fcfcfb567289cb43a59c184651",
	},
	"x86_64-apple-darwin": {
		"riscv32": "820028ee7cd2dd8fe8cd8ca5519ab6e792d15fea9367c4525cf63c0f707c0b1f",
		"xtensa":  "00b9dbc2124cf7633cb86f264fbc524226ad4001bce68bbdba43c9bdc4eb026e",
	},
	"aarch64-linux-gnu": {
		"riscv32": "f907a54313058f8a9681d2f48257d518950ff98bcd5a319194b4bee7c10cf223",
		"xtensa":  "317f6e0fd1dba0886d8110709823d909593ef29438822a14f81ebe19d72ce7cd",
	},
	"x86_64-linux-gnu": {
		"riscv32": "373b37a68bae3ef441ead24a7bfc950fcbfc274cbdd2b628fc6915f179eb1d8e",
		"xtensa":  "588bfaccd0f929650655d10a580f020c6ba9c131712d8fa519280081b8d126eb",
	},
}

// ResolveESPQEMU acquires the pinned emulator only when executing an ESP
// machine on a supported Unix host. Explicit executable paths and other QEMU
// machines retain their configured runner. Windows keeps its existing setup.
func ResolveESPQEMU(command string, args []string) (string, error) {
	return resolveESPQEMU(command, args, runtime.GOOS, runtime.GOARCH)
}

func resolveESPQEMU(command string, args []string, goos, goarch string) (string, error) {
	arch := espQEMUArchitecture(command, args)
	if arch == "" || goos == "windows" {
		return command, nil
	}
	platform := getESPClangPlatform(goos, goarch)
	checksum := espQEMUChecksums[platform][arch]
	if checksum == "" {
		return command, nil
	}
	// Preserve installed Espressif builds, including offline installations.
	// Generic QEMU packages often have the same executable names without ESP
	// machine support, so checking only LookPath would select the wrong binary.
	machine := "esp32"
	if arch == "riscv32" {
		machine = "esp32c3"
	}
	if executable, err := exec.LookPath(command); err == nil && espQEMUSupportsMachine(executable, machine) {
		return executable, nil
	}
	dir := filepath.Join(cacheDir(), "esp-qemu-"+espQEMUVersion, platform, arch)
	asset := "qemu-" + arch + "-softmmu-esp_develop_" + strings.ReplaceAll(espQEMUVersion, "-", "_") + "-" + platform + ".tar.xz"
	return installESPQEMU(dir, command, espQEMUBaseURL+"/"+asset, checksum)
}

func espQEMUSupportsMachine(executable, machine string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-machine", "help")
	cmd.WaitDelay = time.Second
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if fields := strings.Fields(line); len(fields) != 0 && fields[0] == machine {
			return true
		}
	}
	return false
}

func espQEMUArchitecture(command string, args []string) string {
	var arch, machine string
	switch command {
	case "qemu-system-riscv32":
		arch, machine = "riscv32", "esp32c3"
	case "qemu-system-xtensa":
		arch, machine = "xtensa", "esp32"
	default:
		return ""
	}
	var selected string
	for i, arg := range args {
		if (arg == "-machine" || arg == "-M") && i+1 < len(args) {
			selected = args[i+1]
		} else if strings.HasPrefix(arg, "-machine=") {
			selected = strings.TrimPrefix(arg, "-machine=")
		}
	}
	selected, _, _ = strings.Cut(selected, ",")
	if selected != machine {
		return ""
	}
	return arch
}

func espQEMUReady(dir, command string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, "bin", command))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0, nil
}

func installESPQEMU(dir, command, url, checksum string) (string, error) {
	executable := filepath.Join(dir, "bin", command)
	if ready, err := espQEMUReady(dir, command); ready || err != nil {
		return executable, err
	}
	lock, err := acquireLock(dir + ".lock")
	if err != nil {
		return "", err
	}
	defer releaseLock(lock)
	if ready, err := espQEMUReady(dir, command); ready || err != nil {
		return executable, err
	}
	extracted := dir + ".extract"
	if err := os.RemoveAll(extracted); err != nil {
		return "", err
	}
	defer os.RemoveAll(extracted)
	if err := downloadAndExtractArchiveWithChecksum(url, extracted, "ESP QEMU "+command, checksum); err != nil {
		return "", err
	}
	root := filepath.Join(extracted, "qemu")
	if ready, err := espQEMUReady(root, command); err != nil {
		return "", err
	} else if !ready {
		return "", fmt.Errorf("ESP QEMU archive does not contain executable bin/%s", command)
	}
	// Publish only a verified, fully extracted payload. Repair an incomplete
	// cache under the same lock used by concurrent installers.
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.Rename(root, dir); err != nil {
		return "", err
	}
	return executable, nil
}
