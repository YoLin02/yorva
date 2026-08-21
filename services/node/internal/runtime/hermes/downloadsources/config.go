package downloadsources

import (
	"errors"
	"net/url"
	"strings"
)

const (
	defaultHermesArchiveURL = "https://github.com/NousResearch/hermes-agent/archive/df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"
	defaultNodeArchiveURL   = "https://npmmirror.com/mirrors/node/v22.23.1/node-v22.23.1-win-x64.zip"
	defaultNPMArchiveURL    = "https://registry.npmmirror.com/npm/-/npm-12.0.2.tgz"
	defaultPythonIndexURL   = "https://pypi.tuna.tsinghua.edu.cn/simple"
	defaultNPMRegistryURL   = "https://registry.npmmirror.com"
)

var ErrInvalid = errors.New("invalid Hermes download source settings")

type Config struct {
	HermesArchiveURL string `json:"hermesArchiveUrl"`
	NodeArchiveURL   string `json:"nodeArchiveUrl"`
	NPMArchiveURL    string `json:"npmArchiveUrl"`
	PythonIndexURL   string `json:"pythonIndexUrl"`
	NPMRegistryURL   string `json:"npmRegistryUrl"`
}

func Default() Config {
	return Config{
		HermesArchiveURL: defaultHermesArchiveURL,
		NodeArchiveURL:   defaultNodeArchiveURL,
		NPMArchiveURL:    defaultNPMArchiveURL,
		PythonIndexURL:   defaultPythonIndexURL,
		NPMRegistryURL:   defaultNPMRegistryURL,
	}
}

func Normalize(config Config) (Config, error) {
	config.HermesArchiveURL = strings.TrimSpace(config.HermesArchiveURL)
	config.NodeArchiveURL = strings.TrimSpace(config.NodeArchiveURL)
	config.NPMArchiveURL = strings.TrimSpace(config.NPMArchiveURL)
	config.PythonIndexURL = strings.TrimSpace(config.PythonIndexURL)
	config.NPMRegistryURL = strings.TrimSpace(config.NPMRegistryURL)
	for _, value := range []string{
		config.HermesArchiveURL,
		config.NodeArchiveURL,
		config.NPMArchiveURL,
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
