package compile

import (
	"reflect"
	"testing"
)

func TestGoCompilerFlagNamesAndTypes(t *testing.T) {
	opts := new(options)
	fs := newFlagSet(opts)
	err := fs.Parse([]string{
		"-c=2",
		"-C",
		"-e",
		"-l=4",
		"-lang=go1.17",
		"-d=panic,ssa/check/on",
		"-p=p",
		"-importcfg=importcfg",
		"case.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.concurrency != 2 || opts.noColumns.value != 1 || opts.allErrors.value != 1 || opts.noInline.value != 4 {
		t.Fatalf("parsed flags: %+v", opts)
	}
	if opts.lang != "go1.17" || opts.pkgPath != "p" || opts.importCfg != "importcfg" {
		t.Fatalf("parsed flags: %+v", opts)
	}
	if !reflect.DeepEqual(fs.Args(), []string{"case.go"}) {
		t.Fatalf("files=%v", fs.Args())
	}
	if unsupported := opts.unsupported(); len(unsupported) != 0 {
		t.Fatalf("unsupported=%v, want none", unsupported)
	}
}

func TestGoCompilerSpecificFlagIsExplicitlyUnsupported(t *testing.T) {
	opts := new(options)
	fs := newFlagSet(opts)
	if err := fs.Parse([]string{"-d=libfuzzer", "case.go"}); err != nil {
		t.Fatal(err)
	}
	if got := opts.unsupported(); !reflect.DeepEqual(got, []string{"-d=libfuzzer"}) {
		t.Fatalf("unsupported=%v", got)
	}
}
