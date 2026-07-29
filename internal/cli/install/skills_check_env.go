package install

import (
	"net/url"
	"runtime"
	"strings"
)

// The automatic skills update check launches a detached worker plus a
// PATH-resolved `skills` or `npx` helper, so none of them may inherit the
// caller's credentials. The environment is built from an allowlist rather than
// a secret denylist: an unrecognized variable is dropped instead of judged.
//
// Only variables that executable discovery, the OS loader, and language
// runtimes genuinely need are forwarded.
var skillsCheckSharedEnvAllowlist = []string{
	"HOME",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	"PATH",
	"TEMP",
	"TMP",
	"TMPDIR",
	"TZ",
}

var skillsCheckUnixEnvAllowlist = []string{
	"LOGNAME",
	"SHELL",
	"USER",
	"XDG_CACHE_HOME",
	"XDG_CONFIG_HOME",
	"XDG_DATA_HOME",
}

// Proxy configuration is connectivity, not identity: the update check is a
// network operation, so without these variables it silently fails behind a
// corporate proxy. Proxy URLs may embed credentials in their userinfo, and a
// credential must not cross the helper boundary, so HTTP_PROXY and HTTPS_PROXY
// are forwarded with their userinfo stripped; a value that cannot be provably
// sanitized is dropped. Authenticated proxies therefore still fail closed —
// the check degrades to its cached result rather than forwarding a secret.
// NO_PROXY is a host list with no credential form and is forwarded verbatim.
// Names are matched case-insensitively on every platform because both the
// upper- and lowercase spellings are conventional.
var skillsCheckProxyEnvAllowlist = []string{
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
}

// Windows names are stored uppercase because Windows environment variable names
// are case-insensitive.
var skillsCheckWindowsEnvAllowlist = []string{
	"APPDATA",
	"COMSPEC",
	"HOMEDRIVE",
	"HOMEPATH",
	"LOCALAPPDATA",
	"NUMBER_OF_PROCESSORS",
	"OS",
	"PATHEXT",
	"PROCESSOR_ARCHITECTURE",
	"PROGRAMDATA",
	"PROGRAMFILES",
	"PROGRAMFILES(X86)",
	"PROGRAMW6432",
	"SYSTEMDRIVE",
	"SYSTEMROOT",
	"USERPROFILE",
	"WINDIR",
}

// skillsCheckWorkerEnvironment builds the detached worker environment: the
// allowlisted runtime variables plus the private worker coordination values.
func skillsCheckWorkerEnvironment(base []string, spec skillsCheckWorkerSpec) []string {
	return append(
		filterSkillsCheckEnvironment(base, runtime.GOOS),
		skillsWorkerEnvVar+"=1",
		skillsWorkerCacheEnvVar+"="+spec.cachePath,
		skillsWorkerLockEnvVar+"="+spec.lockPath,
		skillsWorkerTokenEnvVar+"="+spec.token,
	)
}

// skillsCheckHelperEnvironment builds the environment for the PATH-resolved
// `skills` or `npx` helper. Worker coordination values are not allowlisted, so
// they stay inside the worker.
func skillsCheckHelperEnvironment(base []string) []string {
	return filterSkillsCheckEnvironment(base, runtime.GOOS)
}

func filterSkillsCheckEnvironment(base []string, goos string) []string {
	allowed := skillsCheckEnvAllowlistFor(goos)
	filtered := make([]string, 0, len(allowed))
	for _, entry := range base {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if proxyName, isProxy := skillsCheckProxyEnvName(key); isProxy {
			if safe, ok := sanitizeSkillsCheckProxyValue(proxyName, value); ok {
				filtered = append(filtered, key+"="+safe)
			}
			continue
		}
		if _, ok := allowed[normalizeSkillsCheckEnvKey(key, goos)]; ok {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// skillsCheckProxyEnvName reports whether key names proxy configuration and
// returns its canonical uppercase form.
func skillsCheckProxyEnvName(key string) (string, bool) {
	upper := strings.ToUpper(key)
	for _, name := range skillsCheckProxyEnvAllowlist {
		if upper == name {
			return name, true
		}
	}
	return "", false
}

// sanitizeSkillsCheckProxyValue returns the proxy value safe to forward. A
// proxy URL keeps its scheme, host, and port; userinfo, query, and fragment are
// stripped because they can carry credentials. A value containing "@" that
// cannot be parsed into a URL with explicit userinfo is dropped entirely: the
// credential position cannot be proven, so the value is not forwarded.
func sanitizeSkillsCheckProxyValue(canonicalName, value string) (string, bool) {
	if canonicalName == "NO_PROXY" {
		return value, true
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || !strings.Contains(trimmed, "@") {
		return value, true
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.User == nil || parsed.Host == "" {
		return "", false
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""
	return parsed.String(), true
}

func skillsCheckEnvAllowlistFor(goos string) map[string]struct{} {
	platform := skillsCheckUnixEnvAllowlist
	if goos == "windows" {
		platform = skillsCheckWindowsEnvAllowlist
	}

	allowed := make(map[string]struct{}, len(skillsCheckSharedEnvAllowlist)+len(platform))
	for _, name := range skillsCheckSharedEnvAllowlist {
		allowed[normalizeSkillsCheckEnvKey(name, goos)] = struct{}{}
	}
	for _, name := range platform {
		allowed[normalizeSkillsCheckEnvKey(name, goos)] = struct{}{}
	}
	return allowed
}

func normalizeSkillsCheckEnvKey(key, goos string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}
