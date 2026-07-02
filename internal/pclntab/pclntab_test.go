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
