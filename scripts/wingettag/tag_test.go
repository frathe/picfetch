package wingettag

import (
	"os/exec"
	"testing"
)

var validCases = []struct {
	tag  string
	want bool
}{
	{tag: "v0.2.11", want: true},
	{tag: "v1.0.0", want: true},
	{tag: "v10.20.30", want: true},
	{tag: "v0.2.11-rc.1", want: false},
	{tag: "v1.0", want: false},
	{tag: "v1", want: false},
	{tag: "1.0.0", want: false},
	{tag: "main", want: false},
	{tag: "", want: false},
	{tag: "v0.2.11\n", want: false},
	{tag: "v1.0.0\r", want: false},
	{tag: "v1'2'3", want: false},
	{tag: "v1a2b3", want: false},
	{tag: "v0.2.11;whoami", want: false},
	{tag: "v1.0.0'; Invoke-Expression 'calc'; '", want: false},
	{tag: "v1.0.0$(whoami)", want: false},
	{tag: "v1.0.0`whoami`", want: false},
	{tag: "v1.0.0|calc", want: false},
	{tag: "v1.0.0&calc", want: false},
	{tag: "v1.0.0;calc", want: false},
	{tag: "v';calc", want: false},
	{tag: "../v1.0.0", want: false},
	{tag: "v1.0.0/extra", want: false},
}

func TestValid(t *testing.T) {
	for _, tc := range validCases {
		if got := Valid(tc.tag); got != tc.want {
			t.Errorf("Valid(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

func TestPatternMatchesBashERE(t *testing.T) {
	script := `[[ "$1" =~ ` + Pattern + ` ]]`
	for _, tc := range validCases {
		cmd := exec.Command("bash", "-c", script, "--", tc.tag)
		err := cmd.Run()
		got := err == nil
		if !got {
			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatalf("bash %q: %v", tc.tag, err)
			}
		}
		if got != tc.want {
			t.Errorf("bash [[ =~ Pattern ]] on %q = %v, want %v", tc.tag, got, tc.want)
		}
	}
}
