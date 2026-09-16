// Command qodanaconfig validates YAML before the exclusion inventory check.
package main

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

func main() {
	data, err := os.ReadFile("qodana.yaml")
	if err == nil {
		err = validate(data)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "invalid qodana.yaml: %v\n", err)
		os.Exit(1)
	}
}

func validate(data []byte) error {
	var document map[string]any
	return yaml.Unmarshal(data, &document)
}
