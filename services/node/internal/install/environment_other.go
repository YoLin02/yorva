//go:build !windows

package install

func readUserEnvironment() (ObservedEnvironment, error) {
	return ObservedEnvironment{}, nil
}

func writeUserHermesHome(string) error { return nil }

func writeUserPath([]string) error { return nil }

func broadcastEnvironmentChange() error { return nil }
