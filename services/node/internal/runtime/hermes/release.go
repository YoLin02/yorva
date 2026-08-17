package hermes

import "time"

const (
	officialRepository     = "NousResearch/hermes-agent"
	officialRelease        = "v2026.8.16"
	officialPackageVersion = "0.20.2"
	officialCommit         = "df4b65147d7ddd74dd449f9067aabbca5aef0ec7"
	officialScriptPath     = "scripts/install.ps1"
	officialRawURL         = "https://raw.githubusercontent.com/NousResearch/hermes-agent/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/scripts/install.ps1"
	officialScriptSize     = 233712
	officialScriptSHA256   = "2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3"
	officialProtocol       = 1
	sourceDownloadLimit    = 512 * 1024
	sourceConnectTimeout   = 30 * time.Second
	sourceFetchTimeout     = 120 * time.Second
)

type officialSource struct {
	URL          string
	ExpectedSize int64
	ExpectedSHA  string
}

func pinnedOfficialSource() officialSource {
	return officialSource{
		URL:          officialRawURL,
		ExpectedSize: officialScriptSize,
		ExpectedSHA:  officialScriptSHA256,
	}
}
