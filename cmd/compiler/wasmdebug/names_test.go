package wasmdebug

import "testing"

func TestFunctionNames(t *testing.T) {
	mapData := []byte{2, 0, 4, 'm', 'a', 'i', 'n', 7, 3, 'f', 'o', 'o'}
	nameData := append([]byte{1, byte(len(mapData))}, mapData...)
	module := appendCustomSection(append([]byte(nil), wasmHeader...), "name", nameData)
	names, err := FunctionNames(module)
	if err != nil || len(names) != 2 || names[0] != "main" || names[7] != "foo" {
		t.Fatalf("names = %v, %v", names, err)
	}
	if names, err := FunctionNames(wasmHeader); err != nil || names != nil {
		t.Fatalf("missing names = %v, %v", names, err)
	}
	for i := 0; i < len(module); i++ {
		if i == len(wasmHeader) {
			continue
		}
		if _, err := FunctionNames(module[:i]); err == nil {
			t.Fatalf("accepted truncation at %d", i)
		}
	}
	for _, payload := range [][]byte{
		{1},                             // missing subsection length
		{1, 2, 0},                       // truncated subsection
		{1, 0},                          // missing map count
		{1, 1, 1},                       // missing function index
		{1, 7, 2, 0, 1, 'a', 0, 1, 'b'}, // repeated index
		{1, 2, 0, 0},                    // trailing bytes
		{1, 1, 0, 1, 1, 0},              // repeated map
		{1, 4, 1, 0, 1, 0xff},           // invalid UTF-8
	} {
		if _, err := FunctionNames(appendCustomSection(append([]byte(nil), wasmHeader...), "name", payload)); err == nil {
			t.Fatalf("accepted invalid name map %x", payload)
		}
	}
}

func TestFunctionNamesOptionalSubsections(t *testing.T) {
	// Other custom sections and the module-name subsection can precede an
	// empty function map; neither supplies evidence that functions are dead.
	module := appendCustomSection(append([]byte(nil), wasmHeader...), "producers", nil)
	module = appendCustomSection(module, "name", []byte{0, 2, 1, 'm', 1, 1, 0})
	names, err := FunctionNames(module)
	if err != nil || names == nil || len(names) != 0 {
		t.Fatalf("empty function map = %v, %v", names, err)
	}
}

func TestFunctionNamesComplete(t *testing.T) {
	for _, tc := range []struct {
		name      string
		imports   []byte
		functions []byte
		names     map[uint32]string
		want      bool
	}{
		{"defined", nil, []byte{2, 0, 0}, map[uint32]string{0: "a", 1: "b"}, true},
		{"sparse", nil, []byte{2, 0, 0}, map[uint32]string{0: "a"}, false},
		{"wrong index", nil, []byte{1, 0}, map[uint32]string{7: "a"}, false},
		{"empty name", nil, []byte{1, 0}, map[uint32]string{0: ""}, false},
		{"imported", []byte{1, 1, 'm', 1, 'f', 0, 0}, []byte{1, 0}, map[uint32]string{0: "f", 1: "a"}, true},
		{"unnamed import", []byte{1, 1, 'm', 1, 'f', 0, 0}, []byte{1, 0}, map[uint32]string{1: "a"}, false},
		{"memory import", []byte{1, 1, 'm', 1, 'x', 2, 1, 1, 2}, []byte{1, 0}, map[uint32]string{0: "a"}, true},
		{"shared memory import", []byte{1, 1, 'm', 1, 'x', 2, 3, 1, 2}, []byte{1, 0}, map[uint32]string{0: "a"}, true},
		{"table import", []byte{1, 1, 'm', 1, 'x', 1, 0x70, 0, 1}, []byte{1, 0}, map[uint32]string{0: "a"}, true},
		{"global import", []byte{1, 1, 'm', 1, 'x', 3, 0x7f, 1}, []byte{1, 0}, map[uint32]string{0: "a"}, true},
		{"tag import", []byte{1, 1, 'm', 1, 'x', 4, 0, 0}, []byte{1, 0}, map[uint32]string{0: "a"}, true},
		{"unknown import", []byte{1, 1, 'm', 1, 'x', 9, 0}, []byte{1, 0}, map[uint32]string{0: "a"}, false},
		{"unknown limits", []byte{1, 1, 'm', 1, 'x', 2, 4, 1}, []byte{1, 0}, map[uint32]string{0: "a"}, false},
		{"truncated import", []byte{1, 1, 'm'}, []byte{1, 0}, map[uint32]string{0: "a"}, false},
		{"trailing functions", nil, []byte{1, 0, 0}, map[uint32]string{0: "a"}, false},
		{"no functions", nil, nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := append([]byte(nil), wasmHeader...)
			for _, section := range []struct {
				id   byte
				data []byte
			}{{2, tc.imports}, {3, tc.functions}} {
				if section.data != nil {
					module = append(module, section.id, byte(len(section.data)))
					module = append(module, section.data...)
				}
			}
			got, err := FunctionNamesComplete(module, tc.names)
			if err != nil || got != tc.want {
				t.Fatalf("complete=%v, %v; want %v", got, err, tc.want)
			}
		})
	}
}
