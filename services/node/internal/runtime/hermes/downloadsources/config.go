package downloadsources

import (
	"errors"
	"net/url"
	"strings"
)

const (
	PreferenceBundledFirst = "bundled-first"
	PreferenceOnlineFirst  = "online-first"

	defaultHermesArchiveURL = "https://github.com/NousResearch/hermes-agent/archive/a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"
	legacyHermesArchiveURL  = "https://github.com/NousResearch/hermes-agent/archive/df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"
	defaultNodeArchiveURL   = "https://npmmirror.com/mirrors/node/v22.23.1/node-v22.23.1-win-x64.zip"
	defaultNPMArchiveURL    = "https://registry.npmmirror.com/npm/-/npm-12.0.2.tgz"
	defaultPythonArchiveURL = "https://github.com/astral-sh/python-build-standalone/releases/download/20260728/cpython-3.11.15%2B20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"
	defaultPythonIndexURL   = "https://pypi.tuna.tsinghua.edu.cn/simple"
	defaultNPMRegistryURL   = "https://registry.npmmirror.com"
)

var ErrInvalid = errors.New("invalid Hermes download source settings")

type Config struct {
	ArtifactPreference string `json:"artifactPreference"`
	HermesArchiveURL   string `json:"hermesArchiveUrl"`
	NodeArchiveURL     string `json:"nodeArchiveUrl"`
	NPMArchiveURL      string `json:"npmArchiveUrl"`
	PythonArchiveURL   string `json:"pythonArchiveUrl"`
	PythonIndexURL     string `json:"pythonIndexUrl"`
	NPMRegistryURL     string `json:"npmRegistryUrl"`
}

func Default() Config {
	return Config{
		ArtifactPreference: PreferenceBundledFirst,
		HermesArchiveURL:   defaultHermesArchiveURL,
		NodeArchiveURL:     defaultNodeArchiveURL,
		NPMArchiveURL:      defaultNPMArchiveURL,
		PythonArchiveURL:   defaultPythonArchiveURL,
		PythonIndexURL:     defaultPythonIndexURL,
		NPMRegistryURL:     defaultNPMRegistryURL,
	}
}

func Normalize(config Config) (Config, error) {
	config.ArtifactPreference = strings.TrimSpace(config.ArtifactPreference)
	config.HermesArchiveURL = strings.TrimSpace(config.HermesArchiveURL)
	config.NodeArchiveURL = strings.TrimSpace(config.NodeArchiveURL)
	config.NPMArchiveURL = strings.TrimSpace(config.NPMArchiveURL)
	config.PythonArchiveURL = strings.TrimSpace(config.PythonArchiveURL)
	config.PythonIndexURL = strings.TrimSpace(config.PythonIndexURL)
	config.NPMRegistryURL = strings.TrimSpace(config.NPMRegistryURL)
	if config.ArtifactPreference != PreferenceBundledFirst && config.ArtifactPreference != PreferenceOnlineFirst {
		return Config{}, ErrInvalid
	}
	for _, value := range []string{
		config.HermesArchiveURL,
		config.NodeArchiveURL,
		config.NPMArchiveURL,
		config.PythonArchiveURL,
		config.PythonIndexURL,
		config.NPMRegistryURL,
	} {
		if err := validateHTTPSURL(value); err != nil {
			return Config{}, err
		}
	}
	return config, nil
}

func validateHTTPSURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Hostname() == "" {
		return ErrInvalid
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return ErrInvalid
	}
	return nil
}
