package main

import "testing"

func TestValidateQodanaSyntax(t *testing.T) {
	for _, tt := range []struct {
		name, data string
		valid      bool
	}{
		{"ordinary configuration", "version: \"1.0\"\nexclude:\n  - name: DuplicatedCode\n    paths:\n      - sample_test.go\n", true},
		{"unfinished sequence", "version: \"1.0\"\nexclude: [\n", false},
		{"duplicate root key", "exclude: []\nexclude: []\n", false},
		{"commented parent mapping", "version: \"1.0\"\n#exclude:\n  - name: DuplicatedCode\n", false},
		{"sequence root", "- DuplicatedCode\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate([]byte(tt.data)); (err == nil) != tt.valid {
				t.Fatalf("valid = %v, error = %v", tt.valid, err)
			}
		})
	}
}
