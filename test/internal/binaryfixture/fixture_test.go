package binaryfixture

import (
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"testing"
)

func TestParserFixturesContainCodeAndDWARF(t *testing.T) {
	for _, kind := range []string{"ELF", "MachO", "PE"} {
		t.Run(kind, func(t *testing.T) {
			var data *dwarf.Data
			var code []byte
			var err error
			switch kind {
			case "ELF":
				f, e := elf.Open(ELF(t))
				if e != nil {
					t.Fatal(e)
				}
				defer f.Close()
				code, err = f.Section(".text").Data()
				if err != nil {
					t.Fatal(err)
				}
				data, err = f.DWARF()
			case "MachO":
				f, e := macho.Open(MachO(t))
				if e != nil {
					t.Fatal(e)
				}
				defer f.Close()
				code, err = f.Section("__text").Data()
				if err != nil {
					t.Fatal(err)
				}
				data, err = f.DWARF()
			case "PE":
				f, e := pe.Open(PE(t))
				if e != nil {
					t.Fatal(e)
				}
				defer f.Close()
				if f.OptionalHeader == nil {
					t.Fatal("missing PE optional header")
				}
				code, err = f.Section(".text").Data()
				if err != nil {
					t.Fatal(err)
				}
				data, err = f.DWARF()
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(code) == 0 {
				t.Fatal("missing fixture code")
			}
			entry, err := data.Reader().Next()
			if err != nil {
				t.Fatal(err)
			}
			if entry == nil || entry.Tag != dwarf.TagCompileUnit || entry.Val(dwarf.AttrName) == nil {
				t.Fatalf("invalid compile unit: %#v", entry)
			}
			lines, err := data.LineReader(entry)
			if err != nil {
				t.Fatal(err)
			}
			if lines == nil || len(lines.Files()) == 0 {
				t.Fatal("missing source line table")
			}
			var line dwarf.LineEntry
			if err := lines.Next(&line); err != nil {
				t.Fatal(err)
			}
		})
	}
}
