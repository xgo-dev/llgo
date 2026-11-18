//go:build go1.23

// Package structs keeps the runtime's host layout marker usable with older Go toolchains.
package structs

import "structs"

type HostLayout = structs.HostLayout
