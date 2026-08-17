# Fail-closed Demo MSI packaging entry point.
# Requires the pinned Hermes source, Node, npm, and license files before Tauri bundling.

[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

& (Join-Path $PSScriptRoot "prepare-hermes-embedded-source.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Hermes source preparation failed"
}
& (Join-Path $PSScriptRoot "prepare-hermes-node-prerequisites.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Node/npm payload preparation failed"
}

$resourceDir = Join-Path $repoRoot "apps\desktop\src-tauri\resources\hermes\source"
$required = @(
    @{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305; SHA256 = "2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA" },
    @{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836; SHA256 = "7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29" },
    @{ Name = "npm-12.0.2.tgz"; Size = 3045132; SHA256 = "5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1" }
)
$licenses = @("LICENSE", "NODE-LICENSE", "NPM-LICENSE")

foreach ($item in $required) {
    $path = Join-Path $resourceDir $item.Name
    if (-not (Test-Path -LiteralPath $path)) {
        throw "missing required payload $($item.Name)"
    }
    $info = Get-Item -LiteralPath $path
    if ($info.Length -ne $item.Size) {
        throw "$($item.Name) size $($info.Length) != $($item.Size)"
    }
    $hash = (Get-FileHash -Algorithm SHA256 -Path $path).Hash
    if ($hash -ne $item.SHA256) {
        throw "$($item.Name) digest mismatch"
    }
    Write-Host "verified $($item.Name) $($item.Size) $hash"
}

foreach ($name in $licenses) {
    $path = Join-Path $resourceDir $name
    if (-not (Test-Path -LiteralPath $path) -or (Get-Item -LiteralPath $path).Length -lt 20) {
        throw "missing or empty license $name"
    }
}

$configPath = Join-Path $repoRoot "apps\desktop\src-tauri\tauri.conf.json"
$configBackup = [System.IO.File]::ReadAllText($configPath)
try {
    $config = $configBackup | ConvertFrom-Json
    $config.bundle.resources = @(
        "resources/hermes/source/LICENSE",
        "resources/hermes/source/NODE-LICENSE",
        "resources/hermes/source/NPM-LICENSE"
    ) + @($required | ForEach-Object { "resources/hermes/source/$($_.Name)" })
    $utf8 = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($configPath, ($config | ConvertTo-Json -Depth 20), $utf8)
    pnpm --filter @yorva/desktop tauri build --bundles msi
    if ($LASTEXITCODE -ne 0) {
        throw "tauri MSI build failed"
    }
}
finally {
    [System.IO.File]::WriteAllText($configPath, $configBackup)
}

$msi = Get-ChildItem -Path (Join-Path $repoRoot "apps\desktop\src-tauri\target\release\bundle\msi") -Filter "YORVA_*.msi" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $msi) {
    throw "MSI output was not produced"
}

& (Join-Path $PSScriptRoot "inspect-yorva-msi.ps1") -MsiPath $msi.FullName
Write-Host ("MSI {0} SHA-256 {1} size {2}" -f $msi.Name, (Get-FileHash -Algorithm SHA256 -Path $msi.FullName).Hash, $msi.Length)
