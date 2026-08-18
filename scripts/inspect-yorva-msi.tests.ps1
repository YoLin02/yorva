$ErrorActionPreference = "Stop"
$here = $PSScriptRoot
. (Join-Path $here "inspect-yorva-msi.ps1") -AsLibrary

$failed = 0
function Assert-Throws([string]$name, [scriptblock]$body, [string]$want) {
    try {
        & $body
        Write-Host "FAIL ${name}: expected throw containing $want"
        $script:failed++
    } catch {
        if ("$($_.Exception.Message)" -notlike "*$want*") {
            Write-Host "FAIL ${name}: $($_.Exception.Message)"
            $script:failed++
        } else {
            Write-Host "PASS ${name}"
        }
    }
}

$catalog = Get-YorvaMsiPayloadCatalog

Assert-Throws "missing Hermes LICENSE" {
    $rows = @(
        [pscustomobject]@{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "missing exact name LICENSE"

Assert-Throws "suffix collision" {
    $rows = @(
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "missing exact name LICENSE"

Assert-Throws "duplicate payload" {
    $rows = @(
        [pscustomobject]@{ Name = "LICENSE"; Size = 1070 },
        [pscustomobject]@{ Name = "LICENSE"; Size = 1070 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "2 entries named LICENSE"

Assert-Throws "wrong filename" {
    $rows = @(
        [pscustomobject]@{ Name = "LICENSE.txt"; Size = 1070 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "missing exact name LICENSE"

Assert-Throws "wrong size" {
    $rows = @(
        [pscustomobject]@{ Name = "LICENSE"; Size = 20 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "size 20 != 1070"

$root = Join-Path ([System.IO.Path]::GetTempPath()) ("yorva-msi-neg-" + [guid]::NewGuid().ToString("N"))
$payload = Join-Path $root "resources\hermes\source"
New-Item -ItemType Directory -Force -Path $payload | Out-Null
try {
    foreach ($item in $catalog) {
        $bytes = New-Object byte[] $item.Size
        [System.IO.File]::WriteAllBytes((Join-Path $payload $item.Name), $bytes)
    }
    Assert-Throws "same size but wrong hash" {
        Assert-ExtractedPayloads $root $catalog
    } "SHA-256"

    Set-Content -LiteralPath (Join-Path $payload "LICENSE") -Value ("x" * 1070) -NoNewline
    Assert-Throws "wrong license content" {
        Assert-ExtractedPayloads $root $catalog
    } "SHA-256"

    [System.IO.File]::WriteAllBytes((Join-Path $payload "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"), (New-Object byte[] 71869305))
    Assert-Throws "substituted archive" {
        Assert-ExtractedPayloads $root $catalog
    } "SHA-256"

    Set-Content -LiteralPath (Join-Path $payload "extra.exe") -Value "MZ"
    Assert-Throws "unexpected extra executable" {
        Assert-NoUnexpectedPayloads $payload $catalog
    } "unexpected extra resource"

    Assert-Throws "extraction failure" {
        Invoke-YorvaMsiInspection -MsiPath (Join-Path $root "missing.msi")
    } "MSI not found"
}
finally {
    Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue
}

if ($failed -ne 0) {
    throw "$failed MSI inspector negative tests failed"
}
Write-Host "all MSI inspector negative tests passed"
