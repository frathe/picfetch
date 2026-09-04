package sitegen

import (
	"strings"
	"testing"
)

func TestTranslationEndpointRejectsQueryAndFragment(t *testing.T) {
	tests := []struct {
		name string
		base string
	}{
		{name: "query", base: "https://proxy.test/api?tenant=x"},
		{name: "empty query", base: "https://proxy.test/api?"},
		{name: "fragment", base: "https://proxy.test/api#tenant"},
		{name: "empty fragment", base: "https://proxy.test/api#"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := translationEndpoint(testCase.base, "key")
			if err == nil {
				t.Fatalf("translationEndpoint accepted %q", testCase.base)
			}
			if !strings.Contains(err.Error(), "must not include a query or fragment") {
				t.Fatalf("query-or-fragment diagnostic is not actionable: %v", err)
			}
		})
	}
}

func TestTranslationEndpointRejectsMalformedAuthority(t *testing.T) {
	tests := []string{
		"https://user@:443/api",
		"https://proxy.test:/api",
		"https://proxy.test:0/api",
		"https://proxy.test:99999/api",
	}
	for _, base := range tests {
		t.Run(base, func(t *testing.T) {
			_, err := translationEndpoint(base, "key")
			if err == nil {
				t.Fatalf("translationEndpoint accepted malformed authority %q", base)
			}
		})
	}
}

func TestTranslationEndpointAppendsRouteToPath(t *testing.T) {
	endpoint, err := translationEndpoint("https://proxy.test/api/", "key")
	if err != nil {
		t.Fatalf("translationEndpoint rejected a path prefix: %v", err)
	}
	if endpoint != "https://proxy.test/api/v2/translate" {
		t.Fatalf("unexpected translation endpoint: %q", endpoint)
	}
}
