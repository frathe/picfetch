package wingettag

import "regexp"

// Pattern is the release-tag allowlist shared with .github/workflows/winget.yml.
// winget-releaser interpolates the tag into inline PowerShell, so this must stay
// a charset that cannot break out of a single-quoted pwsh string.
const Pattern = `^v[0-9]+\.[0-9]+\.[0-9]+$`

var pattern = regexp.MustCompile(Pattern)

// Valid reports whether tag is a vX.Y.Z release tag safe to pass to
// winget-releaser, which interpolates the value into inline PowerShell.
func Valid(tag string) bool {
	return pattern.MatchString(tag)
}
