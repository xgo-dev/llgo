package main

import (
	"encoding/json"
	"testing"
)

var outputSink []byte

func BenchmarkJSONRoundTrip(b *testing.B) {
	input := makeInput()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for range b.N {
		output, err := roundTrip(input)
		if err != nil {
			b.Fatal(err)
		}
		outputSink = output
	}
}

func TestRoundTrip(t *testing.T) {
	output, err := roundTrip(makeInput())
	if err != nil {
		t.Fatal(err)
	}
	var records []record
	if err := json.Unmarshal(output, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 64 {
		t.Fatalf("record count = %d, want 64", len(records))
	}
	for i, r := range records {
		if r.ID != i+1 || r.Amount != i*100+i+1 || r.Active != (i%2 != 0) ||
			r.Name != "customer" || len(r.Tags) != 2 || r.Tags[0] != "standard" || r.Tags[1] != "delivery" {
			t.Fatalf("record %d changed unexpectedly: %+v", i, r)
		}
	}
}
