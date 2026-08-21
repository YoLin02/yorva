package hermes

import (
	"os"
	"strings"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
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

func installerEnvironment(home string, sources downloadsources.Config) []string {
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
		"UV_DEFAULT_INDEX="+sources.PythonIndexURL,
		"UV_INDEX_STRATEGY=first-index",
		"PIP_INDEX_URL="+sources.PythonIndexURL,
		"PIP_CONFIG_FILE=NUL",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"NPM_CONFIG_REGISTRY="+sources.NPMRegistryURL,
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
