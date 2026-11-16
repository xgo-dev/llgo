package main

import (
	"bytes"
	"text/template"
)

const page = `{{range .}}{{.ID}}|{{.Name}}|{{.Amount}}|{{if .Active}}active{{else}}inactive{{end}}
{{end}}`

type record struct {
	ID     int
	Name   string
	Amount int
	Active bool
}

func makeRecords() []record {
	records := make([]record, 64)
	for i := range records {
		records[i] = record{ID: i + 1, Name: "customer", Amount: i * 100, Active: i%2 == 0}
	}
	return records
}

func parseTemplate() *template.Template {
	return template.Must(template.New("records").Parse(page))
}

func main() {
	var output bytes.Buffer
	if err := parseTemplate().Execute(&output, makeRecords()); err != nil {
		panic(err)
	}
	print(output.String())
}
