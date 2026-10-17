package pclnmap

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
)

func TestPruneFinalSites(t *testing.T) {
	records := []funcinfo.Record{
		{Symbol: "dead.first", Name: "dead.First", File: "/dead/removed.go", Line: 10},
		{Symbol: "main.main", Name: "main.main", File: "/src/main.go", Line: 3},
		{Symbol: "main.main$wrapper", Name: "main.main", File: "/src/main.go", Line: 3, Flags: funcinfo.RecordFlagWrapper},
		{Symbol: "dead.last", Name: "dead.Last", File: "/dead/removed.go", Line: 20},
		{Symbol: "main.pcOnly", Name: "main.PCOnly", File: "/src/pc.go", Line: 30},
	}
	pcLines := []funcinfo.PCLineRecord{
		{ID: 101, Symbol: "main.main", File: "/src/main.go", Line: 4},
		{ID: 202, Symbol: "main.main$wrapper", File: "/src/inlined.go", Line: 37},
		{ID: 303, Symbol: "main.main", File: "/dead/unused-site.go", Line: 99},
		{ID: 404, Symbol: "main.pcOnly", File: "/src/pc.go", Line: 31},
		{ID: 505, Symbol: "dead.last", File: "/dead/removed.go", Line: 22},
	}
	table, err := funcinfo.EncodeWithPCLines(records, pcLines)
	if err != nil {
		t.Fatal(err)
	}
	data := sampleData(t)
	data.Table = table
	data.SymbolIndex = []SymbolIndexEntry{{11, 1}, {22, 2}, {33, 3}, {44, 4}, {55, 5}}
	data.EntrySites = []Site{{0x1010, 22}, {0x2010, 33}}
	data.PCSites = []Site{{0x1020, 101}, {0x1030, 101}, {0x2020, 202}, {0x2030, 404}}
	before, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	pruned, err := Prune(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned.Table.Records) != 3 || len(pruned.Table.PCLines) != 3 {
		t.Fatalf("retained %d functions and %d PC lines, want 3 of each", len(pruned.Table.Records), len(pruned.Table.PCLines))
	}
	for newIndex, oldIndex := range []int{1, 2, 4} {
		rec := pruned.Table.Records[newIndex]
		want := records[oldIndex]
		if pruned.Table.Symbol(rec) != want.Symbol || pruned.Table.Name(rec) != want.Name ||
			pruned.Table.File(rec) != want.File || rec.Line&funcinfo.EncodedLineMask != want.Line {
			t.Fatalf("function %d lost source metadata: %+v", newIndex, rec)
		}
		if wrapper := rec.Line&funcinfo.EncodedLineWrapper != 0; wrapper != (want.Flags != 0) {
			t.Fatalf("function %s wrapper flag = %v", want.Symbol, wrapper)
		}
		if idx, ok := pruned.Table.LookupSymbol(want.Symbol); !ok || idx != newIndex {
			t.Fatalf("symbol hash lookup for %s = %d, %v; want %d, true", want.Symbol, idx, ok, newIndex)
		}
	}
	for i, want := range []struct {
		id   uint64
		fn   uint32
		file string
		line uint32
	}{{101, 1, "/src/main.go", 4}, {202, 2, "/src/inlined.go", 37}, {404, 3, "/src/pc.go", 31}} {
		rec := pruned.Table.PCLines[i]
		if rec.ID != want.id || rec.Func != want.fn || pruned.Table.PCLineFile(rec) != want.file || rec.Line != want.line {
			t.Fatalf("PC-line %d lost its function/source join: %+v", i, rec)
		}
	}
	if want := []SymbolIndexEntry{{22, 1}, {33, 2}, {55, 3}}; !reflect.DeepEqual(pruned.SymbolIndex, want) {
		t.Fatalf("symbol index = %+v, want %+v", pruned.SymbolIndex, want)
	}
	if bytes.Contains(pruned.Table.Strings, []byte("dead")) {
		t.Fatalf("dead function/PC-line strings survived: %q", pruned.Table.Strings)
	}
	if !reflect.DeepEqual(pruned.EntrySites, data.EntrySites) || !reflect.DeepEqual(pruned.PCSites, data.PCSites) {
		t.Fatal("pruning changed final PC sites")
	}
	after, err := Encode(pruned)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) >= len(before) {
		t.Fatalf("map did not shrink: %d -> %d", len(before), len(after))
	}
	view, err := Parse(after)
	if err != nil {
		t.Fatal(err)
	}
	if view.Identity != data.Identity || view.ImageBase != data.ImageBase ||
		view.TextStartOffset != data.TextStart-data.ImageBase || view.TextEndOffset != data.TextEnd-data.ImageBase {
		t.Fatal("pruning changed executable identity or text range")
	}
	original, err := Encode(data)
	if err != nil || !bytes.Equal(original, before) {
		t.Fatalf("pruning mutated the input: %v", err)
	}
	again, err := Prune(pruned)
	if err != nil {
		t.Fatal(err)
	}
	rawAgain, err := Encode(again)
	if err != nil || !bytes.Equal(after, rawAgain) {
		t.Fatalf("pruning is not idempotent: %v", err)
	}
}

func TestPruneNoSites(t *testing.T) {
	data := sampleData(t)
	data.EntrySites, data.PCSites = nil, nil
	pruned, err := Prune(data)
	if err != nil {
		t.Fatal(err)
	}
	if pruned.Table.SizeBytes() != 0 || len(pruned.SymbolIndex) != 0 {
		t.Fatalf("map without live sites retained metadata: %+v", pruned)
	}
}

func TestPruneRejectsBrokenJoins(t *testing.T) {
	for name, mutate := range map[string]func(*Data){
		"zero symbol index":  func(d *Data) { d.SymbolIndex[0].FuncIndex = 0 },
		"large symbol index": func(d *Data) { d.SymbolIndex[0].FuncIndex = 2 },
		"missing entry":      func(d *Data) { d.EntrySites[0].ID = 99 },
		"missing PC line":    func(d *Data) { d.PCSites[0].ID = 99 },
		"zero PC function":   func(d *Data) { d.Table.PCLines[0].Func = 0 },
		"large PC function":  func(d *Data) { d.Table.PCLines[0].Func = 2 },
		"duplicate PC line":  func(d *Data) { d.Table.PCLines = append(d.Table.PCLines, d.Table.PCLines[0]) },
		"empty symbol": func(d *Data) {
			d.Table.Records[0].SymbolPkg, d.Table.Records[0].SymbolName = 0, 0
		},
		"duplicate symbol": func(d *Data) {
			d.Table.Records = append(d.Table.Records, d.Table.Records[0])
			d.SymbolIndex = append(d.SymbolIndex, SymbolIndexEntry{SymbolID: 12, FuncIndex: 2})
			d.EntrySites = append(d.EntrySites, Site{PCOffset: 0x2010, ID: 12})
		},
	} {
		t.Run(name, func(t *testing.T) {
			data := sampleData(t)
			mutate(&data)
			if _, err := Prune(data); err == nil {
				t.Fatal("pruning accepted an invalid function/PC-line join")
			}
		})
	}
}
