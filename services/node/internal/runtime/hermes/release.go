package hermes

import "time"

const (
	officialRepository     = "NousResearch/hermes-agent"
	officialRelease        = "v2026.8.16"
	officialPackageVersion = "0.20.2"
	officialCommit         = "df4b65147d7ddd74dd449f9067aabbca5aef0ec7"
	officialScriptPath     = "scripts/install.ps1"
	officialGitRemote      = "https://github.com/NousResearch/hermes-agent.git"
	officialRawURL         = "https://raw.githubusercontent.com/NousResearch/hermes-agent/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/scripts/install.ps1"
	officialScriptSize     = 233712
	officialScriptSHA256   = "2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3"
	officialArchiveURL     = "https://github.com/NousResearch/hermes-agent/archive/df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"
	officialArchiveSize    = 71869305
	officialArchiveSHA256  = "2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba"
	officialArchiveRoot    = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7"
	officialLicenseSize    = 1070
	officialLicenseSHA256  = "821556e6336796450ab852d375117b48a4887e71d255794fd6318d99982a5ab6"
	officialProtocol       = 1
	sourceDownloadLimit    = 512 * 1024
	sourceConnectTimeout   = 30 * time.Second
	sourceFetchTimeout     = 120 * time.Second
	archiveConnectTimeout  = 30 * time.Second
	archiveFetchTimeout    = 180 * time.Second
	archiveDownloadLimit   = 80 << 20
	archiveMaxEntries      = 20000
	archiveMaxFiles        = 16000
	archiveMaxUncompressed = 256 << 20
	archiveMaxMember       = 16 << 20
	archiveMaxRatio        = 8
	archiveDiskBudget      = 256 << 20
	archiveDiskMargin      = 64 << 20

	officialNodeVersion    = "22.23.1"
	officialNodeMinVersion = "22.22.0"
	officialNodeArchiveURL = "https://nodejs.org/dist/v22.23.1/node-v22.23.1-win-x64.zip"
	officialNodeArchiveSize = 35682836
	officialNodeArchiveSHA = "7df0bc9375723f4a86b3aa1b7cc73342423d9677a8df4538aca31a049e309c29"
	officialNodeZipRoot    = "node-v22.23.1-win-x64"
	officialNpmVersion     = "12.0.2"
	officialNpmMinVersion  = "12.0.0"
	officialNpmArchiveURL  = "https://registry.npmjs.org/npm/-/npm-12.0.2.tgz"
	officialNpmArchiveSize = 3045132
	officialNpmArchiveSHA  = "5dbb86c71d07a1957f2e90734092dd6a58bdcd9ebc2d8d41ca1c6e6a21d364e1"
	nodeDepsTimeout        = 15 * time.Minute
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
