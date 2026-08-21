# Fail-closed Demo MSI packaging entry point.
# Requires the pinned Hermes source, Node, npm, and license files before Tauri bundling.

[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

pwsh -NoProfile -File (Join-Path $PSScriptRoot "prepare-hermes-embedded-source.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Hermes source preparation failed"
}
pwsh -NoProfile -File (Join-Path $PSScriptRoot "prepare-hermes-node-prerequisites.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Node/npm payload preparation failed"
}

$resourceDir = Join-Path $repoRoot "apps\desktop\src-tauri\resources\hermes\source"
$required = @(
    @{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305; SHA256 = "2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA" },
    @{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836; SHA256 = "7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29" },
    @{ Name = "npm-12.0.2.tgz"; Size = 3045132; SHA256 = "5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1" },
    @{ Name = "LICENSE"; Size = 1070; SHA256 = "821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6" },
    @{ Name = "NODE-LICENSE"; Size = 148217; SHA256 = "8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576" },
    @{ Name = "NPM-LICENSE"; Size = 9742; SHA256 = "7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257" }
)

$verifiedInputs = $false
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
$verifiedInputs = $true
if (-not $verifiedInputs) {
    throw "payload verification did not complete"
}

$configPath = Join-Path $repoRoot "apps\desktop\src-tauri\tauri.conf.json"
$configBackup = [System.IO.File]::ReadAllText($configPath)
try {
    $config = $configBackup | ConvertFrom-Json
    $config.bundle.resources = @($required | ForEach-Object { "resources/hermes/source/$($_.Name)" })
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

$msi = Get-ChildItem -Path (Join-Path $repoRoot "apps\desktop\src-tauri\target\release\bundle\msi") -Filter "Yorva_*.msi" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $msi) {
    throw "MSI output was not produced"
}

$inspected = $false
& (Join-Path $PSScriptRoot "inspect-yorva-msi.ps1") -MsiPath $msi.FullName
if ($LASTEXITCODE -ne 0) {
    throw "MSI inspection failed"
}
$inspected = $true
if (-not $inspected) {
    throw "MSI inspection did not execute"
}
Write-Host ("MSI {0} SHA-256 {1} size {2}" -f $msi.Name, (Get-FileHash -Algorithm SHA256 -Path $msi.FullName).Hash, $msi.Length)
