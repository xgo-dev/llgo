//go:build !windows

package gotest

import "testing"

func enterWindowsExceptionTest(t *testing.T) bool { return true }

func ensureWindowsExceptionStackHeadroom() {}
