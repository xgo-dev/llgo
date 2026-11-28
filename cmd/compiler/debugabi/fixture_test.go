//go:build !llgo

package debugabi

import (
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type fixtureManifest struct {
	SchemaVersion uint8    `json:"schema_version"`
	Source        string   `json:"source"`
	Categories    []string `json:"categories"`
	Breakpoints   []struct {
		Name   string                     `json:"name"`
		Marker string                     `json:"marker"`
		Values map[string]json.RawMessage `json:"values"`
	} `json:"breakpoints"`
}

//go:embed testdata/fixture/manifest_v1.json
var fixtureManifestV1 []byte

func TestSharedFixtureManifest(t *testing.T) {
	var manifest fixtureManifest
	if err := json.Unmarshal(fixtureManifestV1, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != SchemaVersion || manifest.Source != "main.go" {
		t.Fatalf("fixture manifest identity = v%d %q", manifest.SchemaVersion, manifest.Source)
	}
	wantCategories := []string{
		"primitives", "aliases", "named_types", "recursive_types", "aggregates",
		"strings", "slices", "maps", "channels", "interfaces",
		"functions", "closures", "bound_methods", "goroutines",
	}
	if strings.Join(manifest.Categories, ",") != strings.Join(wantCategories, ",") {
		t.Fatalf("fixture categories = %v", manifest.Categories)
	}
	source, err := os.ReadFile("testdata/fixture/" + manifest.Source)
	if err != nil {
		t.Fatal(err)
	}
	for _, breakpoint := range manifest.Breakpoints {
		if breakpoint.Name == "" || len(breakpoint.Values) == 0 {
			t.Fatalf("incomplete fixture breakpoint: %+v", breakpoint)
		}
		if count := strings.Count(string(source), breakpoint.Marker); count != 1 {
			t.Errorf("fixture marker %q occurs %d times", breakpoint.Marker, count)
		}
	}
}
