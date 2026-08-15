package lto

import (
	"fmt"
	"strings"
)

type Mode uint8

const (
	Off Mode = iota
	Full
	Thin
)

func Parse(value string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "thin":
		return Thin, nil
	case "full":
		return Full, nil
	default:
		return Off, fmt.Errorf("invalid LTO mode %q, want thin or full", value)
	}
}

func (m Mode) String() string {
	switch m {
	case Off:
		return "off"
	case Full:
		return "full"
	case Thin:
		return "thin"
	default:
		return fmt.Sprintf("Mode(%d)", m)
	}
}

func (m Mode) Enabled() bool {
	return m == Full || m == Thin
}

func (m Mode) ClangFlag() string {
	switch m {
	case Full:
		return "-flto=full"
	case Thin:
		return "-flto=thin"
	default:
		return ""
	}
}

type PassPlugin struct {
	Path string
}

func (p PassPlugin) Enabled() bool {
	return p.Path != ""
}

func (p PassPlugin) LinkerFlags(goos string) ([]string, error) {
	if !p.Enabled() {
		return nil, nil
	}
	if goos == "darwin" {
		return nil, fmt.Errorf("LTO pass plugins are not supported on darwin by the bundled ld64.lld or Apple ld64")
	}
	return []string{"-Wl,--load-pass-plugin=" + p.Path}, nil
}
