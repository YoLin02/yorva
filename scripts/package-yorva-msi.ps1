# Fail-closed Demo MSI packaging entry point.
# Requires the pinned Hermes source, Node, npm, Python, and license files before Tauri bundling.

[CmdletBinding()]
param(
    [switch]$UpdateQualification
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

$configPath = Join-Path $repoRoot "apps\desktop\src-tauri\tauri.conf.json"
$desktopPackagePath = Join-Path $repoRoot "apps\desktop\package.json"
$cargoPath = Join-Path $repoRoot "apps\desktop\src-tauri\Cargo.toml"
$configBackup = [System.IO.File]::ReadAllText($configPath)
$config = $configBackup | ConvertFrom-Json
$desktopPackage = Get-Content -LiteralPath $desktopPackagePath -Raw | ConvertFrom-Json
$cargoText = Get-Content -LiteralPath $cargoPath -Raw
$cargoVersionMatch = [regex]::Match($cargoText, '(?ms)^\[package\]\s+name\s*=\s*"yorva-desktop"\s+version\s*=\s*"([^\"]+)"')
if (-not $cargoVersionMatch.Success) {
    throw "could not read the Desktop Cargo package version"
}
$productVersion = [string]$config.version
if ($productVersion -notmatch '^\d+\.\d+\.\d+$' -or $desktopPackage.version -ne $productVersion -or $cargoVersionMatch.Groups[1].Value -ne $productVersion) {
    throw "Desktop package, Tauri bundle, and Rust crate versions must match a release MAJOR.MINOR.PATCH"
}
$sourceCommit = (git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $sourceCommit -notmatch '^[0-9a-f]{40}$') {
    throw "could not resolve the source commit"
}
$workingTreeDirty = [bool](git status --porcelain)

pwsh -NoProfile -File (Join-Path $PSScriptRoot "prepare-hermes-embedded-source.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Hermes source preparation failed"
}
pwsh -NoProfile -File (Join-Path $PSScriptRoot "prepare-hermes-node-prerequisites.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Node/npm payload preparation failed"
}
pwsh -NoProfile -File (Join-Path $PSScriptRoot "prepare-hermes-python-prerequisite.ps1") -RequirePresent
if ($LASTEXITCODE -ne 0) {
    throw "Python payload preparation failed"
}

$resourceDir = Join-Path $repoRoot "apps\desktop\src-tauri\resources\hermes\source"
$required = @(
    @{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347; SHA256 = "4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54" },
    @{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836; SHA256 = "7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29" },
    @{ Name = "npm-12.0.2.tgz"; Size = 3045132; SHA256 = "5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1" },
    @{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832; SHA256 = "64A804111830C5329BFC5A4D95D6CBCBB377CAA2C02195101EDF85D15FC53099" },
    @{ Name = "LICENSE"; Size = 1070; SHA256 = "821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6" },
    @{ Name = "NODE-LICENSE"; Size = 148217; SHA256 = "8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576" },
    @{ Name = "NPM-LICENSE"; Size = 9742; SHA256 = "7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257" }
)
$productResources = @(
    @{ Path = "resources/installer/DATA_RETENTION.txt"; Size = 1313; SHA256 = "C8A330453B4E794D3B3407D9C5F9D2A0F2463F5711A1B9AE47C12FBB38FDCF0B" }
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

foreach ($item in $productResources) {
    $path = Join-Path (Join-Path $repoRoot "apps\desktop\src-tauri") $item.Path
    $info = Get-Item -LiteralPath $path -ErrorAction Stop
    if ($info.Length -ne $item.Size -or (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash -ne $item.SHA256) {
        throw "$($item.Path) does not match the reviewed installer resource"
    }
}

try {
    $config.bundle.resources = @($productResources | ForEach-Object { $_.Path }) + @($required | ForEach-Object { "resources/hermes/source/$($_.Name)" })
    $utf8 = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($configPath, ($config | ConvertTo-Json -Depth 20), $utf8)
    $buildStartedUtc = [DateTime]::UtcNow.AddSeconds(-1)
    $tauriArguments = @("--filter", "@yorva/desktop", "tauri", "build", "--bundles", "msi")
    if ($UpdateQualification) {
        $tauriArguments += @("--features", "update-qualification")
    }
    $tauriOutput = @(& pnpm @tauriArguments 2>&1)
    $tauriExitCode = $LASTEXITCODE
    $tauriOutput | ForEach-Object { Write-Host $_ }
    if ($tauriExitCode -ne 0) {
        # Tauri 2.11 does not expose light.exe arguments. Its generated resource
        # components deliberately use file key paths beneath LocalAppData, which
        # WiX ICE38/ICE64 reject even for a per-user package. Re-link the exact
        # generated wixobj files while suppressing only those two ICE rules. The
        # MSI contract inspector and disposable-Windows lifecycle test remain the
        # authoritative checks for repair, removal and data retention behavior.
        if (($tauriOutput -join "`n") -notmatch 'failed to run .*light\.exe') {
            throw "tauri MSI build failed before WiX linking"
        }
        $wixBuildDir = Join-Path $repoRoot "apps\desktop\src-tauri\target\release\wix\x64"
        $wixToolsDir = Join-Path $env:LOCALAPPDATA "tauri\WixTools314"
        $lightPath = Join-Path $wixToolsDir "light.exe"
        $localePath = Join-Path $wixBuildDir "locale.wxl"
        $mainObject = Join-Path $wixBuildDir "main.wixobj"
        $wixObjects = @(Get-ChildItem -LiteralPath $wixBuildDir -Filter "*.wixobj" -File -ErrorAction Stop)
        if (
            -not (Test-Path -LiteralPath $lightPath) -or
            -not (Test-Path -LiteralPath $localePath) -or
            -not (Test-Path -LiteralPath $mainObject) -or
            (Get-Item -LiteralPath $mainObject).LastWriteTimeUtc -lt $buildStartedUtc -or
            $wixObjects.Count -eq 0
        ) {
            throw "tauri MSI build failed before a reviewed WiX relink was possible"
        }
        $manualOutput = Join-Path $wixBuildDir "output.msi"
        if (Test-Path -LiteralPath $manualOutput) {
            Remove-Item -LiteralPath $manualOutput -Force
        }
        $lightArguments = @(
            "-sice:ICE38",
            "-sice:ICE64",
            "-ext", (Join-Path $wixToolsDir "WixUtilExtension.dll"),
            "-ext", (Join-Path $wixToolsDir "WixUIExtension.dll"),
            "-o", $manualOutput,
            "-cultures:en-us",
            "-loc", $localePath
        ) + @($wixObjects | ForEach-Object { $_.FullName })
        & $lightPath @lightArguments
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $manualOutput)) {
            throw "reviewed WiX relink failed"
        }
        $bundleDir = Join-Path $repoRoot "apps\desktop\src-tauri\target\release\bundle\msi"
        [System.IO.Directory]::CreateDirectory($bundleDir) | Out-Null
        $bundlePath = Join-Path $bundleDir ("YORVA_{0}_x64_en-US.msi" -f $productVersion)
        Move-Item -LiteralPath $manualOutput -Destination $bundlePath -Force
        Write-Host "Tauri generated the MSI inputs; linked per-user MSI with reviewed ICE38/ICE64 exceptions."
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
& (Join-Path $PSScriptRoot "inspect-yorva-msi.ps1") -MsiPath $msi.FullName -ExpectedVersion $productVersion
if ($LASTEXITCODE -ne 0) {
    throw "MSI inspection failed"
}
$inspected = $true
if (-not $inspected) {
    throw "MSI inspection did not execute"
}
$msiHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $msi.FullName).Hash
$signature = Get-AuthenticodeSignature -LiteralPath $msi.FullName
$artifactRoot = Join-Path $repoRoot ".tools\p8-msi\$productVersion"
[System.IO.Directory]::CreateDirectory($artifactRoot) | Out-Null
$artifactPath = Join-Path $artifactRoot $msi.Name
Copy-Item -LiteralPath $msi.FullName -Destination $artifactPath -Force
$summary = [ordered]@{
    schemaVersion = 1
    productVersion = $productVersion
    sourceCommit = $sourceCommit
    workingTreeDirty = $workingTreeDirty
    fileName = $msi.Name
    sizeBytes = $msi.Length
    sha256 = $msiHash
    signatureStatus = [string]$signature.Status
    qualificationBuild = [bool]$UpdateQualification
    upgradeCode = "e793918b-37eb-5e2f-b866-5fc4aa3ac75a"
    installScope = "perUser"
    generatedAtUtc = [DateTime]::UtcNow.ToString("O")
}
[System.IO.File]::WriteAllText(
    (Join-Path $artifactRoot "artifact.json"),
    ($summary | ConvertTo-Json -Depth 3),
    [System.Text.UTF8Encoding]::new($false)
)
Write-Host ("MSI {0} SHA-256 {1} size {2} signature {3}" -f $msi.Name, $msiHash, $msi.Length, $signature.Status)
