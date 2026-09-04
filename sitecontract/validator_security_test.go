package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeUpdateDoesNotExposeDeepLCredentialToAMPValidator(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test validator fixture uses a POSIX shell script")
	}
	repo := repositoryRoot(t)
	fixtureRoot := t.TempDir()
	validator := filepath.Join(fixtureRoot, "validator")
	validatorScript := "#!/bin/sh\n" +
		"if [ -n \"${DEEPL_API_KEY+x}\" ]; then\n" +
		"  echo 'DeepL credential leaked to validator' >&2\n" +
		"  exit 41\n" +
		"fi\n"
	if err := os.WriteFile(validator, []byte(validatorScript), 0o700); err != nil {
		t.Fatalf("write fake AMP validator: %v", err)
	}

	server := echoingDeepLServer(t)
	defer server.Close()
	output := filepath.Join(fixtureRoot, "docs")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatalf("create deployment fixture: %v", err)
	}
	writeStaticSiteFiles(t, output)
	cmd := exec.Command("make", "update",
		"SITE_TRANSLATIONS="+filepath.Join(fixtureRoot, "de.json"),
		"SITE_OUTPUT_DIR="+output,
		"NODE="+validator,
		"AMP_VALIDATOR=ignored",
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env,
		"DEEPL_API_KEY="+strings.Repeat("e", 24),
		"DEEPL_API_URL="+server.URL,
	)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validator received a credential that it does not need: %v\n%s", err, combined)
	}
}

func TestDirectNodeTargetsHonorOverridesWithoutInheritingDeepLCredential(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test Node fixture uses a POSIX shell script")
	}
	repo := repositoryRoot(t)
	fixtureRoot := t.TempDir()
	fakeNode := filepath.Join(fixtureRoot, "custom-node")
	script := "#!/bin/sh\n" +
		"if [ -n \"${DEEPL_API_KEY+x}\" ]; then\n" +
		"  echo 'DeepL credential leaked to Node target' >&2\n" +
		"  exit 41\n" +
		"fi\n" +
		"printf 'custom Node arguments: %s\\n' \"$*\"\n"
	if err := os.WriteFile(fakeNode, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake Node executable: %v", err)
	}
	environment := append(withoutEnvironment(os.Environ(), "DEEPL_API_KEY"), "DEEPL_API_KEY="+strings.Repeat("z", 24))
	tests := []struct {
		name         string
		target       string
		extraSetting string
		wantArgument string
	}{
		{name: "validate AMP", target: "validate-amp", extraSetting: "AMP_VALIDATOR=custom-validator-wrapper", wantArgument: "custom-validator-wrapper"},
		{name: "test browser", target: "test-browser", wantArgument: "site/tools/test-language.cjs"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			arguments := []string{testCase.target, "NODE=" + fakeNode}
			if testCase.extraSetting != "" {
				arguments = append(arguments, testCase.extraSetting)
			}
			cmd := exec.Command("make", arguments...)
			cmd.Dir = repo
			cmd.Env = environment
			combined, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s failed through configured Node command: %v\n%s", testCase.target, err, combined)
			}
			if want := "custom Node arguments: " + testCase.wantArgument; !strings.Contains(string(combined), want) {
				t.Fatalf("%s ignored a configured command: want %q in output\n%s", testCase.target, want, combined)
			}
		})
	}
}
