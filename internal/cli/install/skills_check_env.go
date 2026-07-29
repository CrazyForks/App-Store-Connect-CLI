package install

import (
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
		key, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if _, ok := allowed[normalizeSkillsCheckEnvKey(key, goos)]; ok {
			filtered = append(filtered, entry)
		}
	}
	return filtered
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
