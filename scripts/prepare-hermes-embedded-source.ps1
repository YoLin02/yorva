# Prepare the verified official Hermes commit archive for Demo MSI packaging.
# Does not commit the payload. Ordinary unit tests must not invoke this script.

[CmdletBinding()]
param(
    [switch]$RequirePresent
)

$ErrorActionPreference = "Stop"

$commit = "df4b65147d7ddd74dd449f9067aabbca5aef0ec7"
$expectedSize = 71869305
$expectedSha = "2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA"
$url = "https://codeload.github.com/NousResearch/hermes-agent/zip/$commit"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$cacheDir = Join-Path $repoRoot ".cache\hermes-source"
$resourceDir = Join-Path $repoRoot "apps\desktop\src-tauri\resources\hermes\source"
$cacheFile = Join-Path $cacheDir "hermes-agent-$commit.zip"
$resourceFile = Join-Path $resourceDir "hermes-agent-$commit.zip"

function Test-OfficialArchive([string]$path) {
    if (-not (Test-Path -LiteralPath $path)) {
        return $false
    }
    $item = Get-Item -LiteralPath $path
    if ($item.Length -ne $expectedSize) {
        return $false
    }
    $hash = (Get-FileHash -Algorithm SHA256 -Path $path).Hash
    return $hash -eq $expectedSha
}

New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null
New-Item -ItemType Directory -Force -Path $resourceDir | Out-Null

function Publish-OfficialLicense([string]$archivePath, [string]$licensePath) {
    try { Add-Type -AssemblyName System.IO.Compression.FileSystem } catch { }
    $zip = [System.IO.Compression.ZipFile]::OpenRead($archivePath)
    try {
        $entry = $zip.Entries | Where-Object { $_.FullName -eq "hermes-agent-$commit/LICENSE" } | Select-Object -First 1
        if (-not $entry) {
            throw "official archive is missing LICENSE"
        }
        if (Test-Path -LiteralPath $licensePath) {
            Remove-Item -LiteralPath $licensePath -Force
        }
        [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $licensePath, $true)
    } finally {
        $zip.Dispose()
    }
    $licenseHash = (Get-FileHash -Algorithm SHA256 -Path $licensePath).Hash
    if ((Get-Item -LiteralPath $licensePath).Length -ne 1070 -or $licenseHash -ne "821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6") {
        throw "extracted Hermes LICENSE failed size/SHA-256 verification"
    }
    Write-Host "verified LICENSE 1070 $licenseHash"
}

if (Test-OfficialArchive $resourceFile) {
    Write-Host "Hermes embedded source already verified at $resourceFile"
    Publish-OfficialLicense $resourceFile (Join-Path $resourceDir "LICENSE")
    exit 0
}

if (-not (Test-OfficialArchive $cacheFile)) {
    $knownTemp = Join-Path $env:TEMP "hermes-df4b651.zip"
    if (Test-OfficialArchive $knownTemp) {
        Copy-Item -LiteralPath $knownTemp -Destination $cacheFile -Force
    } else {
        if ($RequirePresent) {
            Write-Host "Downloading official Hermes archive..."
            $part = "$cacheFile.part"
            if (Test-Path -LiteralPath $part) {
                Remove-Item -LiteralPath $part -Force
            }
            & curl.exe -L --fail --retry 5 --retry-all-errors --connect-timeout 30 --output $part $url
            if ($LASTEXITCODE -ne 0) {
                throw "Hermes archive download failed with exit $LASTEXITCODE"
            }
            if (-not (Test-OfficialArchive $part)) {
                Remove-Item -LiteralPath $part -ErrorAction SilentlyContinue
                throw "Downloaded Hermes archive failed size/SHA-256 verification."
            }
            Move-Item -LiteralPath $part -Destination $cacheFile -Force
        } else {
            Write-Host "Hermes embedded source is absent; skipping for non-MSI workflows."
            exit 0
        }
    }
}

if (-not (Test-OfficialArchive $cacheFile)) {
    throw "Cached Hermes archive failed size/SHA-256 verification."
}

Copy-Item -LiteralPath $cacheFile -Destination $resourceFile -Force
if (-not (Test-OfficialArchive $resourceFile)) {
    throw "Copied Hermes archive failed size/SHA-256 verification."
}

Write-Host "Verified Hermes embedded source ready: $resourceFile"
Publish-OfficialLicense $resourceFile (Join-Path $resourceDir "LICENSE")
