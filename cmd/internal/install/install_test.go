package install

import "testing"

func TestBuildFlagsWiring(t *testing.T) {
	if goBuildFlags.Flag != &Cmd.Flag || Cmd.Flag.Lookup("ldflags") == nil {
		t.Fatal("install build flags are not bound to the install command")
	}
}
