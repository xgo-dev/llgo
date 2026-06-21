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
