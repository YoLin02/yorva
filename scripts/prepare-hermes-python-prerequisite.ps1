[CmdletBinding()]
param([switch]$RequirePresent)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$cacheDir = Join-Path $repoRoot ".cache\hermes-python-prerequisites"
$resourceDir = Join-Path $repoRoot "apps\desktop\src-tauri\resources\hermes\source"
$artifact = @{
    Name   = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"
    Url    = "https://github.com/astral-sh/python-build-standalone/releases/download/20260728/cpython-3.11.15%2B20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"
    Size   = 25676832
    SHA256 = "64A804111830C5329BFC5A4D95D6CBCBB377CAA2C02195101EDF85D15FC53099"
}

function Test-Artifact($path) {
    if (-not (Test-Path -LiteralPath $path)) { return $false }
    if ((Get-Item -LiteralPath $path).Length -ne $artifact.Size) { return $false }
    return (Get-FileHash -Algorithm SHA256 -Path $path).Hash -eq $artifact.SHA256
}

New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null
New-Item -ItemType Directory -Force -Path $resourceDir | Out-Null
$cacheFile = Join-Path $cacheDir $artifact.Name
$resourceFile = Join-Path $resourceDir $artifact.Name
if (-not (Test-Artifact $resourceFile)) {
    if (-not (Test-Artifact $cacheFile)) {
        if (-not $RequirePresent) {
            Write-Host "skipping $($artifact.Name) for non-MSI workflow"
            exit 0
        }
        $part = "$cacheFile.part"
        if (Test-Path -LiteralPath $part) { Remove-Item -LiteralPath $part -Force }
        & curl.exe -L --fail --retry 5 --retry-all-errors --connect-timeout 30 --output $part $artifact.Url
        if ($LASTEXITCODE -ne 0) { throw "$($artifact.Name) download failed with exit $LASTEXITCODE" }
        if (-not (Test-Artifact $part)) {
            Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue
            throw "$($artifact.Name) failed size/SHA-256 verification"
        }
        Move-Item -LiteralPath $part -Destination $cacheFile -Force
    }
    Copy-Item -LiteralPath $cacheFile -Destination $resourceFile -Force
}
if (-not (Test-Artifact $resourceFile)) { throw "copied $($artifact.Name) failed verification" }
Write-Host "verified $($artifact.Name) $($artifact.Size) $($artifact.SHA256)"
