package exifwin

import "strings"

type cacheDirective struct {
	name, argument string
}

// parseCacheControl keeps quoted commas and quoted-pairs inside their directive.
// Invalid grammar is rejected so extensions cannot inject freshness directives.
func parseCacheControl(value string) ([]cacheDirective, bool) {
	var parts []string
	start := 0
	quoted, escaped := false, false
	for i := 0; i < len(value); i++ {
		if escaped {
			escaped = false
			continue
		}
		switch value[i] {
		case '\\':
			if !quoted {
				return nil, false
			}
			escaped = true
		case '"':
			quoted = !quoted
		case ',':
			if !quoted {
				parts = append(parts, value[start:i])
				start = i + 1
			}
		}
	}
	if quoted || escaped {
		return nil, false
	}
	parts = append(parts, value[start:])
	var result []cacheDirective
	for _, part := range parts {
		part = strings.Trim(part, " \t")
		if part == "" {
			continue
		}
		name, argument, hasValue := strings.Cut(part, "=")
		if !cacheToken(name) {
			return nil, false
		}
		if hasValue {
			var valid bool
			argument, valid = cacheArgument(argument)
			if !valid {
				return nil, false
			}
		}
		result = append(result, cacheDirective{strings.ToLower(name), argument})
	}
	return result, true
}

func cacheToken(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		character := value[i]
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(character)) {
			continue
		}
		return false
	}
	return true
}

func cacheArgument(value string) (string, bool) {
	if cacheToken(value) {
		return value, true
	}
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false
	}
	decoded := make([]byte, 0, len(value)-2)
	for i := 1; i < len(value)-1; i++ {
		character := value[i]
		if character == '\\' {
			i++
			if i >= len(value)-1 {
				return "", false
			}
			character = value[i]
		} else if character == '"' {
			return "", false
		}
		if character < 32 && character != '\t' || character == 127 {
			return "", false
		}
		decoded = append(decoded, character)
	}
	return string(decoded), true
}
