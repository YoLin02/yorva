package hermes

import (
	"os"
	"strings"
)

const (
	// These endpoints are adapter-owned Phase 3 distribution policy. They
	// cannot be supplied by a request or inherited from the launching shell.
	hermesPythonIndex = "https://pypi.tuna.tsinghua.edu.cn/simple"
	hermesNPMRegistry = "https://registry.npmmirror.com"
)

var installerAllowedEnv = map[string]struct{}{
	"ALLUSERSPROFILE": {}, "APPDATA": {}, "COMMONPROGRAMFILES": {},
	"COMMONPROGRAMFILES(X86)": {}, "COMSPEC": {}, "COMPUTERNAME": {},
	"HOMEDRIVE": {}, "HOMEPATH": {}, "LOCALAPPDATA": {}, "LOGONSERVER": {},
	"NUMBER_OF_PROCESSORS": {}, "OS": {}, "PATHEXT": {}, "PROCESSOR_ARCHITECTURE": {},
	"PROGRAMDATA": {}, "PROGRAMFILES": {}, "PROGRAMFILES(X86)": {},
	"PUBLIC": {}, "SESSIONNAME": {}, "SYSTEMDRIVE": {}, "SYSTEMROOT": {},
	"TEMP": {}, "TMP": {}, "USERDOMAIN": {}, "USERNAME": {}, "USERPROFILE": {},
	"WINDIR": {}, "PATH": {}, "LANG": {},
}

var installerBlockedPrefixes = []string{
	"PYTHON", "PIP_", "UV_", "NPM_", "NODE_", "OPENAI_", "ANTHROPIC_", "GOOGLE_", "GEMINI_", "HERMES_",
}

func installerEnvironment(home string) []string {
	result := make([]string, 0, 32)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(name)
		if blockedEnvironment(upper) {
			continue
		}
		if _, allowed := installerAllowedEnv[upper]; !allowed && !strings.HasPrefix(upper, "LC_") {
			continue
		}
		result = append(result, entry)
	}
	result = append(result,
		"HERMES_HOME="+home,
		"UV_DEFAULT_INDEX="+hermesPythonIndex,
		"UV_INDEX_STRATEGY=first-index",
		"PIP_INDEX_URL="+hermesPythonIndex,
		"PIP_CONFIG_FILE=NUL",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"NPM_CONFIG_REGISTRY="+hermesNPMRegistry,
		"NPM_CONFIG_AUDIT=false",
		"NPM_CONFIG_FUND=false",
	)
	return result
}

func blockedEnvironment(upperName string) bool {
	if upperName == "HERMES_HOME" {
		return true
	}
	for _, prefix := range installerBlockedPrefixes {
		if strings.HasPrefix(upperName, prefix) {
			return true
		}
	}
	return false
}
