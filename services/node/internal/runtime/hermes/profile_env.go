package hermes

import (
	"os"
	"strings"
)

func profileCommandEnvironment(home string) []string {
	result := make([]string, 0, 24)
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
