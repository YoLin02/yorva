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
        if (Test-Path -LiteralPath $part) {
            Remove-Item -LiteralPath $part -Force
        }
        & curl.exe -L --fail --retry 5 --retry-all-errors --connect-timeout 30 --output $part $item.Url
        if ($LASTEXITCODE -ne 0) {
            throw "$($item.Name) download failed with exit $LASTEXITCODE"
        }
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

$nodeZip = Join-Path $resourceDir "node-v22.23.1-win-x64.zip"
$npmTgz = Join-Path $resourceDir "npm-12.0.2.tgz"
if ((Test-Path -LiteralPath $nodeZip) -and (Test-Path -LiteralPath $npmTgz)) {
    try { Add-Type -AssemblyName System.IO.Compression.FileSystem } catch { }
    $nodeLicense = Join-Path $resourceDir "NODE-LICENSE"
    $zip = [System.IO.Compression.ZipFile]::OpenRead($nodeZip)
    try {
        $entry = $zip.Entries | Where-Object { $_.FullName -eq "node-v22.23.1-win-x64/LICENSE" } | Select-Object -First 1
        if (-not $entry) {
            throw "Node archive is missing LICENSE"
        }
        if (Test-Path -LiteralPath $nodeLicense) {
            Remove-Item -LiteralPath $nodeLicense -Force
        }
        [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $nodeLicense, $true)
    } finally {
        $zip.Dispose()
    }
    $nodeHash = (Get-FileHash -Algorithm SHA256 -Path $nodeLicense).Hash
    if ((Get-Item -LiteralPath $nodeLicense).Length -ne 148217 -or $nodeHash -ne "8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576") {
        throw "extracted NODE-LICENSE failed size/SHA-256 verification"
    }
    Write-Host "verified NODE-LICENSE 148217 $nodeHash"

    $tmp = Join-Path $cacheDir "npm-license"
    if (Test-Path -LiteralPath $tmp) {
        Remove-Item -LiteralPath $tmp -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tmp | Out-Null
    & tar.exe -xf $npmTgz -C $tmp package/LICENSE
    if ($LASTEXITCODE -ne 0) {
        throw "failed to extract npm LICENSE"
    }
    $npmLicense = Join-Path $resourceDir "NPM-LICENSE"
    Copy-Item -LiteralPath (Join-Path $tmp "package\LICENSE") -Destination $npmLicense -Force
    Remove-Item -LiteralPath $tmp -Recurse -Force
    $npmHash = (Get-FileHash -Algorithm SHA256 -Path $npmLicense).Hash
    if ((Get-Item -LiteralPath $npmLicense).Length -ne 9742 -or $npmHash -ne "7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257") {
        throw "extracted NPM-LICENSE failed size/SHA-256 verification"
    }
    Write-Host "verified NPM-LICENSE 9742 $npmHash"
}
