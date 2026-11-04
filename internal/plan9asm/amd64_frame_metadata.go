package plan9asm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xgo-dev/plan9asm/gotoolprofile"
)

// Prepare normal Go checksum-verifier metadata before the separate original
// source capture becomes offline. The consumer's go.sum need not have populated
// an independent verifier. This supplies no frame, source or memory permission.
func prepareDownloadedFrameMetadata(ctx context.Context, binary, temporary string, environment []string, module, version string) error {
	directory, err := os.MkdirTemp(temporary, "module-metadata-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.invalid/llgo-frame-metadata\ngo 1.20\n"), 0o600); err != nil {
		return err
	}
	commandContext, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	data, stderr, err := gotoolprofile.RunBounded(commandContext, directory, environment, binary, "mod", "download", "-json", module+"@"+version)
	if err != nil {
		return fmt.Errorf("downloaded AMD64 frame metadata: %w", err)
	}
	var receipt struct {
		Path, Version, Sum, GoModSum, Error string
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	err = decoder.Decode(&receipt)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = fmt.Errorf("expected exactly one JSON receipt")
		}
	}
	if err == nil && (receipt.Path != module || receipt.Version != version || receipt.Error != "" ||
		!frameMetadataChecksum(receipt.Sum) || !frameMetadataChecksum(receipt.GoModSum)) {
		err = fmt.Errorf("download receipt identity, checksum or Error differs from the requested module")
	}
	if err != nil {
		return fmt.Errorf("downloaded AMD64 frame metadata: %w\nstdout:\n%s\nstderr:\n%s", err, data, stderr)
	}
	return nil
}

func frameMetadataChecksum(value string) bool {
	encoded, ok := strings.CutPrefix(value, "h1:")
	if !ok {
		return false
	}
	digest, err := base64.StdEncoding.DecodeString(encoded)
	return err == nil && len(digest) == 32
}
