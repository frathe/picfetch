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

func TestDirectNodeTargetsDoNotInheritDeepLCredential(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test Node fixture uses a POSIX shell script")
	}
	repo := repositoryRoot(t)
	fixtureRoot := t.TempDir()
	fakeNode := filepath.Join(fixtureRoot, "node")
	script := "#!/bin/sh\n" +
		"if [ -n \"${DEEPL_API_KEY+x}\" ]; then\n" +
		"  echo 'DeepL credential leaked to Node target' >&2\n" +
		"  exit 41\n" +
		"fi\n"
	if err := os.WriteFile(fakeNode, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake Node executable: %v", err)
	}
	environment := withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "PATH")
	environment = append(environment,
		"DEEPL_API_KEY="+strings.Repeat("z", 24),
		"PATH="+fixtureRoot+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	for _, target := range []string{"validate-amp", "test-browser"} {
		t.Run(target, func(t *testing.T) {
			cmd := exec.Command("make", target)
			cmd.Dir = repo
			cmd.Env = environment
			if combined, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s exposed a DeepL credential to Node: %v\n%s", target, err, combined)
			}
		})
	}
}
