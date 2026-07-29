package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Credentials a real user plausibly exports before running asc. None of them
// belong to a PATH-resolved skills or npx helper.
var skillsCheckCredentialSentinels = map[string]string{
	"ASC_PRIVATE_KEY":                    "asc-red-sentinel-private-key-2ba917",
	"ASC_KEY_ID":                         "asc-red-sentinel-key-id-8cd034",
	"ASC_ISSUER_ID":                      "asc-red-sentinel-issuer-6f1e52",
	"ASC_PRIVATE_KEY_PATH":               "/tmp/asc-red-sentinel-key-path-4a70bd.p8",
	"ASC_SLACK_WEBHOOK":                  "https://hooks.slack.com/services/asc-red-sentinel-webhook-19e6cf",
	"GITHUB_TOKEN":                       "asc-red-sentinel-github-token-d5b283",
	"MATCH_PASSWORD":                     "asc-red-sentinel-signing-pw-7e40ca",
	"NPM_TOKEN":                          "asc-red-sentinel-npm-token-b921fe",
	"AWS_SECRET_ACCESS_KEY":              "asc-red-sentinel-aws-secret-3cd158",
	"SOME_UNRECOGNIZED_INTERNAL_SETTING": "asc-red-sentinel-unknown-var-0f7a62",
}

func seedSkillsCheckCredentials(t *testing.T) {
	t.Helper()
	for key, value := range skillsCheckCredentialSentinels {
		t.Setenv(key, value)
	}
}

func assertNoCredentialSentinels(t *testing.T, label string, env []string) {
	t.Helper()
	joined := strings.Join(env, "\n")
	for key, value := range skillsCheckCredentialSentinels {
		if strings.Contains(joined, value) {
			t.Fatalf("%s received credential %s: %s", label, key, joined)
		}
	}
}

func TestSkillsCheckWorkerEnvironmentExcludesCredentials(t *testing.T) {
	seedSkillsCheckCredentials(t)

	env := skillsCheckWorkerEnvironment(os.Environ(), skillsCheckWorkerSpec{
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
		lockPath:  filepath.Join(t.TempDir(), "lock"),
		token:     "worker-token",
	})

	assertNoCredentialSentinels(t, "detached worker", env)

	values := envMap(env)
	if values["PATH"] != os.Getenv("PATH") {
		t.Fatalf("worker PATH = %q, want the parent PATH", values["PATH"])
	}
	if values[skillsWorkerEnvVar] != "1" {
		t.Fatalf("worker coordination variable %s = %q, want 1", skillsWorkerEnvVar, values[skillsWorkerEnvVar])
	}
	if values[skillsWorkerTokenEnvVar] != "worker-token" {
		t.Fatalf("worker token = %q, want worker-token", values[skillsWorkerTokenEnvVar])
	}
}

func TestSkillsCheckHelperEnvironmentExcludesWorkerCoordinationVariables(t *testing.T) {
	seedSkillsCheckCredentials(t)
	t.Setenv(skillsWorkerEnvVar, "1")
	t.Setenv(skillsWorkerTokenEnvVar, "worker-token")

	env := skillsCheckHelperEnvironment(os.Environ())

	assertNoCredentialSentinels(t, "skills helper", env)

	values := envMap(env)
	if _, ok := values[skillsWorkerEnvVar]; ok {
		t.Fatalf("helper env exposed %s", skillsWorkerEnvVar)
	}
	if _, ok := values[skillsWorkerTokenEnvVar]; ok {
		t.Fatalf("helper env exposed %s", skillsWorkerTokenEnvVar)
	}
	if values["PATH"] != os.Getenv("PATH") {
		t.Fatal("helper env dropped PATH, breaking executable discovery")
	}
}

func TestDefaultRunSkillsCheckCommandGivesDirectHelperOnlyAllowlistedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	seedSkillsCheckCredentials(t)

	mockSkills := writeExecutable(t, "#!/bin/sh\nenv\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return mockSkills, nil
	}
	lookupNpx = func(string) (string, error) {
		t.Fatal("lookupNpx should not run when skills is available")
		return "", nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}

	assertNoCredentialSentinels(t, "direct skills helper", strings.Split(output, "\n"))
	if !strings.Contains(output, "PATH=") {
		t.Fatalf("direct skills helper lost PATH: %q", output)
	}
}

func TestDefaultRunSkillsCheckCommandGivesNpxFallbackOnlyAllowlistedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	seedSkillsCheckCredentials(t)

	mockNpx := writeExecutable(t, "#!/bin/sh\nenv\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return "", os.ErrNotExist
	}
	lookupNpx = func(string) (string, error) {
		return mockNpx, nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}

	assertNoCredentialSentinels(t, "npx fallback", strings.Split(output, "\n"))
	if !strings.Contains(output, "npm_config_offline=true") {
		t.Fatalf("npx fallback lost the offline cache setting: %q", output)
	}
	if !strings.Contains(output, "PATH=") {
		t.Fatalf("npx fallback lost PATH: %q", output)
	}
}

func TestSkillsCheckEnvironmentAllowlistIsPlatformAware(t *testing.T) {
	unix := skillsCheckEnvAllowlistFor("darwin")
	windows := skillsCheckEnvAllowlistFor("windows")

	for _, name := range []string{"PATH", "HOME", "TMPDIR"} {
		if _, ok := unix[name]; !ok {
			t.Fatalf("unix allowlist is missing %s", name)
		}
	}
	for _, name := range []string{"PATH", "SYSTEMROOT", "PATHEXT", "USERPROFILE", "APPDATA"} {
		if _, ok := windows[name]; !ok {
			t.Fatalf("windows allowlist is missing %s", name)
		}
	}
	if _, ok := windows["SHELL"]; ok {
		t.Fatal("windows allowlist should not carry Unix-only variables")
	}
	if _, ok := unix["SYSTEMROOT"]; ok {
		t.Fatal("unix allowlist should not carry Windows-only variables")
	}
}

func TestFilterSkillsCheckEnvironmentMatchesWindowsCaseInsensitively(t *testing.T) {
	base := []string{
		"SystemRoot=C:\\Windows",
		"Path=C:\\Windows\\System32",
		"NPM_TOKEN=" + skillsCheckCredentialSentinels["NPM_TOKEN"],
		"malformed-entry",
	}

	filtered := filterSkillsCheckEnvironment(base, "windows")

	values := envMap(filtered)
	if values["SystemRoot"] != "C:\\Windows" {
		t.Fatalf("windows filter dropped SystemRoot: %v", filtered)
	}
	if values["Path"] != "C:\\Windows\\System32" {
		t.Fatalf("windows filter dropped Path: %v", filtered)
	}
	assertNoCredentialSentinels(t, "windows filter", filtered)
	if len(filtered) != 2 {
		t.Fatalf("windows filter kept unexpected entries: %v", filtered)
	}
}

func envMap(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	return values
}
