//go:build windows

package install

import (
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	wmSettingChange = 0x001A
	smtoAbortIfHung = 0x0002
	hwndBroadcast   = 0xffff
)

func readUserEnvironment() (ObservedEnvironment, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return ObservedEnvironment{}, err
	}
	defer key.Close()
	home, _, _ := key.GetStringValue("HERMES_HOME")
	path, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return ObservedEnvironment{}, err
	}
	var entries []string
	if strings.TrimSpace(path) != "" {
		entries = strings.Split(path, string(filepath.ListSeparator))
	}
	return ObservedEnvironment{HermesHome: home, PathEntries: entries}, nil
}

func writeUserHermesHome(home string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue("HERMES_HOME", home)
}

func writeUserPath(entries []string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue("Path", strings.Join(entries, string(filepath.ListSeparator)))
}

func broadcastEnvironmentChange() error {
	var result uintptr
	env, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return err
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	send := user32.NewProc("SendMessageTimeoutW")
	r1, _, callErr := send.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung,
		5000,
		uintptr(unsafe.Pointer(&result)),
	)
	if r1 == 0 {
		return callErr
	}
	return nil
}
