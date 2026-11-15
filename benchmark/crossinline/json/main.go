package main

import "encoding/json"

type record struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Active bool     `json:"active"`
	Amount int      `json:"amount"`
	Tags   []string `json:"tags"`
}

func makeInput() []byte {
	records := make([]record, 64)
	for i := range records {
		records[i] = record{
			ID: i + 1, Name: "customer", Active: i%2 == 0, Amount: i * 100,
			Tags: []string{"standard", "delivery"},
		}
	}
	data, err := json.Marshal(records)
	if err != nil {
		panic(err)
	}
	return data
}

func roundTrip(input []byte) ([]byte, error) {
	var records []record
	if err := json.Unmarshal(input, &records); err != nil {
		return nil, err
	}
	for i := range records {
		records[i].Amount += records[i].ID
		records[i].Active = !records[i].Active
	}
	return json.Marshal(records)
}

func main() {
	output, err := roundTrip(makeInput())
	if err != nil {
		panic(err)
	}
	// Emit the complete result so compiler revisions can be compared exactly.
	println(string(output))
}
