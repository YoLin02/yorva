[CmdletBinding()]
param([switch]$RequirePresent)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$cacheDir = Join-Path $repoRoot ".cache\hermes-node"
$resourceDir = Join-Path $repoRoot "apps\desktop\src-tauri\resources\hermes\source"

$artifacts = @(
    @{
        Name     = "node-v22.23.1-win-x64.zip"
        Url      = "https://nodejs.org/dist/v22.23.1/node-v22.23.1-win-x64.zip"
        Size     = 35682836
        SHA256   = "7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29"
    },
    @{
        Name     = "npm-12.0.2.tgz"
        Url      = "https://registry.npmjs.org/npm/-/npm-12.0.2.tgz"
        Size     = 3045132
        SHA256   = "5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1"
    }
)

function Test-Artifact($path, $size, $sha) {
    if (-not (Test-Path -LiteralPath $path)) { return $false }
    if ((Get-Item -LiteralPath $path).Length -ne $size) { return $false }
    return (Get-FileHash -Algorithm SHA256 -Path $path).Hash -eq $sha
}

New-Item -ItemType Directory -Force -Path $cacheDir | Out-Null
New-Item -ItemType Directory -Force -Path $resourceDir | Out-Null

foreach ($item in $artifacts) {
    $cacheFile = Join-Path $cacheDir $item.Name
    $resourceFile = Join-Path $resourceDir $item.Name
    if (Test-Artifact $resourceFile $item.Size $item.SHA256) {
        Write-Host "verified $($item.Name)"
        continue
    }
    if (-not (Test-Artifact $cacheFile $item.Size $item.SHA256)) {
        if (-not $RequirePresent) {
            Write-Host "skipping $($item.Name) for non-MSI workflow"
            continue
        }
        $part = "$cacheFile.part"
        Invoke-WebRequest -Uri $item.Url -OutFile $part -UseBasicParsing
        if (-not (Test-Artifact $part $item.Size $item.SHA256)) {
            Remove-Item $part -ErrorAction SilentlyContinue
            throw "$($item.Name) failed size/SHA-256 verification"
        }
        Move-Item $part $cacheFile -Force
    }
    Copy-Item $cacheFile $resourceFile -Force
    if (-not (Test-Artifact $resourceFile $item.Size $item.SHA256)) {
        throw "copied $($item.Name) failed verification"
    }
    Write-Host "prepared $($item.Name)"
}
