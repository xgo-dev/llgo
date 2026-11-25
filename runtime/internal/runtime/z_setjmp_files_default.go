//go:build (!windows || (!amd64 && !arm64)) && (!darwin || !arm64)

package runtime

const platformSetjmpLLGoFiles = ""
