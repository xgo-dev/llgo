package crosscompile

import (
	_ "embed"
	"encoding/json"
)

// Share all executable payload pins with release packaging and tests.
// The x86 development SDK has separate repository, version and checksum pins.
//
//go:embed espressif.json
var espJSON []byte

type espToolchainPins struct {
	Repository  string                     `json:"repository"`
	Release     string                     `json:"release"`
	LLVMVersion string                     `json:"llvm_version"`
	Payloads    map[string]espClangPayload `json:"payloads"`
}

type espClangPayload struct {
	Asset  string `json:"asset"`
	SHA256 string `json:"sha256"`
}

var espPins = mustParseESPPins(espJSON)

func mustParseESPPins(data []byte) (pins espToolchainPins) {
	if err := json.Unmarshal(data, &pins); err != nil {
		panic(err)
	}
	return
}

func espClangChecksums() map[string]string {
	checksums := make(map[string]string, len(espPins.Payloads))
	for platform, pin := range espPins.Payloads {
		checksums[platform] = pin.SHA256
	}
	return checksums
}
