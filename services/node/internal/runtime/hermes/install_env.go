package hermes

import (
	"os"
	"strings"
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
	"PYTHON", "PIP_", "UV_", "OPENAI_", "ANTHROPIC_", "GOOGLE_", "GEMINI_", "HERMES_",
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
	result = append(result, "HERMES_HOME="+home)
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
