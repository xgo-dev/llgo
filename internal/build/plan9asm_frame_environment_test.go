package build

import (
	"path/filepath"
	"testing"
)

func TestPlan9AsmFrameSourceEnvironment(t *testing.T) {
	for _, experiment := range []string{"", "nosimd"} {
		t.Run("original="+experiment, func(t *testing.T) {
			commands := experimentCommands(t, experiment)
			cfg := mustResolveSourceGo(t, commands, "")
			if cfg.originalExperiment != experiment {
				t.Fatalf("lost original SDK experiment setting: %q", cfg.originalExperiment)
			}
			commands.environ = cfg.apply(commands.environ)
			ctx := &context{
				commands: commands,
				buildConf: &Config{GOEXPERIMENT: cfg.GOEXPERIMENT, sourceGoVersion: cfg.GOVERSION,
					sourceGoExperiment: cfg.originalExperiment},
			}
			opt := plan9asmTranslateOptions(ctx)
			observed := commandEnv{environ: opt.Environment}
			if observed.lookup("GOEXPERIMENT") != experiment || commands.lookup("GOEXPERIMENT") != cfg.GOEXPERIMENT {
				t.Fatal("frame capture must retain the original source profile without mutating the canonical cache profile")
			}
			for _, key := range []string{"GOPATH", "GOMODCACHE", "GOCACHE"} {
				if !filepath.IsAbs(observed.lookup(key)) || observed.lookup(key) != commands.lookup(key) {
					t.Fatalf("%s was not resolved once by the actual source SDK", key)
				}
			}
		})
	}
}
