package hermes

import "time"

const (
	officialRepository     = "NousResearch/hermes-agent"
	officialRelease        = "v2026.8.19"
	officialPackageVersion = "0.20.5"
	officialCommit         = "a0ca7c19204e514f9590ce3b812e029b315ab9e9"
	officialScriptPath     = "scripts/install.ps1"
	officialGitRemote      = "https://github.com/NousResearch/hermes-agent.git"
	officialRawURL         = "https://raw.githubusercontent.com/NousResearch/hermes-agent/a0ca7c19204e514f9590ce3b812e029b315ab9e9/scripts/install.ps1"
	officialScriptSize     = 243941
	officialScriptSHA256   = "28391a336eb3e2c3e88027faa99d0a90873d3a5bf070a63b52788dc361b4eaa1"
	officialArchiveURL     = "https://github.com/NousResearch/hermes-agent/archive/a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"
	officialArchiveSize    = 73798347
	officialArchiveSHA256  = "4e63f39bd992b79bf364a4e22204a22b8e520e2365b53c909f6f15d42c917c54"
	officialArchiveRoot    = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9"
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

	officialNodeVersion             = "22.23.1"
	officialNodeMinVersion          = "22.22.0"
	officialNodeArchiveURL          = "https://nodejs.org/dist/v22.23.1/node-v22.23.1-win-x64.zip"
	officialNodeArchiveSize         = 35682836
	officialNodeArchiveSHA          = "7df0bc9375723f4a86b3aa1b7cc73342423d9677a8df4538aca31a049e309c29"
	officialNodeZipRoot             = "node-v22.23.1-win-x64"
	officialNpmVersion              = "12.0.2"
	officialNpmMinVersion           = "12.0.0"
	officialNpmArchiveURL           = "https://registry.npmjs.org/npm/-/npm-12.0.2.tgz"
	officialNpmArchiveSize          = 3045132
	officialNpmArchiveSHA           = "5dbb86c71d07a1957f2e90734092dd6a58bdcd9ebc2d8d41ca1c6e6a21d364e1"
	officialPythonVersion           = "3.11.15"
	officialPythonRelease           = "20260728"
	officialPythonArchiveName       = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"
	officialPythonArchiveSize int64 = 25676832
	officialPythonArchiveSHA        = "64a804111830c5329bfc5a4d95d6cbcbb377caa2c02195101edf85d15fc53099"
	nodeDepsTimeout                 = 15 * time.Minute
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
