package exportdata

import "testing"

func TestPackageValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		data  *Package
		valid bool
	}{
		{"missing", nil, false},
		{"empty", &Package{Version: Version}, true},
		{"version", &Package{Version: Version + 1}, false},
		{"empty-name", &Package{Version: Version, Functions: []Function{{Cold: true}}}, false},
		{"duplicate", &Package{Version: Version, Functions: []Function{{Name: "F"}, {Name: "F"}}}, false},
		{"unordered", &Package{Version: Version, Functions: []Function{{Name: "Z"}, {Name: "A"}}}, false},
		{"ordered", &Package{Version: Version, Functions: []Function{{Name: "(*T).F", NoReturn: true}, {Name: "F", Cold: true}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.data.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate = %v, valid = %v", err, tc.valid)
			}
			if tc.valid && len(tc.data.Functions) != 0 {
				if got := tc.data.Function("F"); !got.Cold || got.NoReturn {
					t.Fatalf("lookup F: %+v", got)
				}
			}
			if got := tc.data.Function("absent"); got != (Function{}) {
				t.Fatalf("absent function: %+v", got)
			}
		})
	}
}
