package pclntab

import "testing"

func TestBuildFindFuncBucketsLookup(t *testing.T) {
	ftab := []FuncTabEntry{
		{EntryOff: 0, FuncOff: 11},
		{EntryOff: 16, FuncOff: 22},
		{EntryOff: 64, FuncOff: 33},
		{EntryOff: 4096, FuncOff: 44},
		{EntryOff: 4352, FuncOff: 55},
		{EntryOff: 8192, FuncOff: 0}, // sentinel
	}
	buckets, err := BuildFindFuncBuckets(ftab, 8192)
	if err != nil {
		t.Fatalf("BuildFindFuncBuckets: %v", err)
	}
	if got, want := len(buckets), 2; got != want {
		t.Fatalf("bucket count = %d, want %d", got, want)
	}
	for _, tt := range []struct {
		pc   uint32
		want int
	}{
		{pc: 0, want: 0},
		{pc: 15, want: 0},
		{pc: 16, want: 1},
		{pc: 63, want: 1},
		{pc: 64, want: 2},
		{pc: 4095, want: 2},
		{pc: 4096, want: 3},
		{pc: 4351, want: 3},
		{pc: 4352, want: 4},
		{pc: 8191, want: 4},
	} {
		if got := LookupFuncIndex(ftab, buckets, tt.pc); got != tt.want {
			t.Fatalf("lookup(%d) = %d, want %d", tt.pc, got, tt.want)
		}
	}
}

func TestBuildFindFuncBucketsRejectsOverflow(t *testing.T) {
	ftab := make([]FuncTabEntry, 0, 302)
	for i := 0; i < 301; i++ {
		ftab = append(ftab, FuncTabEntry{EntryOff: uint32(i), FuncOff: uint32(i + 1)})
	}
	ftab = append(ftab, FuncTabEntry{EntryOff: FuncTabBucketSize, FuncOff: 0})
	if _, err := BuildFindFuncBuckets(ftab, FuncTabBucketSize); err == nil {
		t.Fatal("expected subbucket overflow error")
	}
}

func TestBuildFindFuncBucketsValidation(t *testing.T) {
	if bs, err := BuildFindFuncBuckets(nil, 0); err != nil || bs != nil {
		t.Fatalf("textSize=0: got %v, %v", bs, err)
	}
	cases := []struct {
		name string
		ftab []FuncTabEntry
	}{
		{"too short", []FuncTabEntry{{EntryOff: 0}}},
		{"not increasing", []FuncTabEntry{{EntryOff: 0}, {EntryOff: 8}, {EntryOff: 8}}},
		{"first not zero", []FuncTabEntry{{EntryOff: 4}, {EntryOff: 64}}},
		{"sentinel below text", []FuncTabEntry{{EntryOff: 0}, {EntryOff: 16}}},
	}
	for _, c := range cases {
		if _, err := BuildFindFuncBuckets(c.ftab, 64); err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestFuncIndexForPCEdges(t *testing.T) {
	ftab := []FuncTabEntry{{EntryOff: 0}, {EntryOff: 16}, {EntryOff: 64}}
	if got := FuncIndexForPC(ftab, 0); got != 0 {
		t.Fatalf("pc=0: %d", got)
	}
	if got := FuncIndexForPC(ftab, 63); got != 1 {
		t.Fatalf("pc=63: %d", got)
	}
	// At or past the sentinel, clamp to the last real function.
	if got := FuncIndexForPC(ftab, 64); got != len(ftab)-2 {
		t.Fatalf("pc=sentinel: %d", got)
	}
	if got := FuncIndexForPC(ftab, 1<<20); got != len(ftab)-2 {
		t.Fatalf("pc=big: %d", got)
	}
}

func TestLookupFuncIndexEdges(t *testing.T) {
	ftab := []FuncTabEntry{{EntryOff: 0}, {EntryOff: 16}, {EntryOff: 8192}}
	buckets, err := BuildFindFuncBuckets(ftab, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if got := LookupFuncIndex(nil, buckets, 0); got != -1 {
		t.Fatalf("short ftab: %d", got)
	}
	if got := LookupFuncIndex(ftab, nil, 0); got != -1 {
		t.Fatalf("no buckets: %d", got)
	}
	if got := LookupFuncIndex(ftab, buckets, 1<<30); got != -1 {
		t.Fatalf("pc out of buckets: %d", got)
	}
	if got := LookupFuncIndex(ftab, buckets, 17); got != 1 {
		t.Fatalf("pc=17: %d", got)
	}
	if got := LookupFuncIndex(ftab, buckets, 8191); got != 1 {
		t.Fatalf("pc=8191: %d", got)
	}
}
