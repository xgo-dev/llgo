package build

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestHeapProfileSourcePolicy(t *testing.T) {
	for _, tt := range []struct {
		source string
		want   bool
	}{
		{`package p; import "runtime"; func F(){runtime.MemProfile(nil,true)}`, true},
		{`package p; import r "runtime"; var F=r.MemProfile`, true},
		{`package p; import . "runtime"; func F(){MemProfile(nil,true)}`, true},
		{"package p\nimport _ \"unsafe\"\n//go:linkname sample runtime.MemProfile\nfunc sample()\n", true},
		{`package p; import "runtime"; func F(){runtime.GC()}`, false},
		{`package p; import fake "example.com/runtime"; func F(){fake.MemProfile()}`, false},
		{`package p; var x="runtime.MemProfile"`, false},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), "p.go", tt.source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if got := fileUsesHeapProfiles(file); got != tt.want {
			t.Errorf("policy=%v want=%v: %s", got, tt.want, tt.source)
		}
	}
}

func TestHeapProfileArchivePolicyIdentity(t *testing.T) {
	c := dispatchInlineContext()
	original := newManifestBuilder()
	c.collectCommonInputs(original)
	c.heapProfileFrames = true
	profile := newManifestBuilder()
	c.collectCommonInputs(profile)
	if !profile.common.HeapProfileFrames || profile.Fingerprint() == original.Fingerprint() {
		t.Fatal("profile readers reused uninstrumented archives")
	}
	c.heapProfileFrames = false
	restored := newManifestBuilder()
	c.collectCommonInputs(restored)
	if original.Fingerprint() != restored.Fingerprint() {
		t.Fatal("unchanged frame policy did not retain archive identity")
	}
}
