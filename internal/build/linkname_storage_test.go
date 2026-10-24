package build

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLinknameDuplicateStaticVariables(t *testing.T) {
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "duplicate")
	output := captureStderr(t)
	_, err := Do([]string{"../../test/go/testdata/linknamevars/duplicate"}, conf)
	if err == nil {
		t.Fatal("two nonzero definitions of one linkname unexpectedly linked")
	}
	message := strings.ToLower(output())
	if !strings.Contains(message, "llgo_test_duplicate_data") ||
		!(strings.Contains(message, "duplicate") || strings.Contains(message, "multiply defined")) {
		t.Fatalf("expected duplicate-definition diagnostic for the shared symbol, got: %v\n%s", err, message)
	}
}
