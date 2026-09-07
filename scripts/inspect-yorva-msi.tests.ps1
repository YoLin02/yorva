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

function New-LifecycleProperties {
    return @(
        [pscustomobject]@{ Property = "ProductName"; Value = "YORVA" },
        [pscustomobject]@{ Property = "Manufacturer"; Value = "yolin" },
        [pscustomobject]@{ Property = "UpgradeCode"; Value = "{E793918B-37EB-5E2F-B866-5FC4AA3AC75A}" },
        [pscustomobject]@{ Property = "ProductVersion"; Value = "0.3.2" },
        [pscustomobject]@{ Property = "REINSTALLMODE"; Value = "amus" },
        [pscustomobject]@{ Property = "WixUIRMOption"; Value = "UseRM" },
        [pscustomobject]@{ Property = "ARPCOMMENTS"; Value = "Uninstall removes YORVA program files, shortcuts and login startup. YORVA user data, encrypted backups, Hermes Runtime and Profiles are preserved." }
    )
}

$lifecycleDirectories = @(
    [pscustomobject]@{ Directory = "LocalProgramsFolder"; Parent = "LocalAppDataFolder"; DefaultDir = "Programs" },
    [pscustomobject]@{ Directory = "INSTALLDIR"; Parent = "LocalProgramsFolder"; DefaultDir = "YORVA" }
)
$lifecycleRegistry = @(
    [pscustomobject]@{ Root = "1"; Key = "Software\yolin\YORVA"; Name = "InstallDir"; Value = "[INSTALLDIR]" },
    [pscustomobject]@{ Root = "1"; Key = "Software\Microsoft\Windows\CurrentVersion\Run"; Name = "Yorva"; Value = '"[INSTALLDIR]yorva-desktop.exe" --hidden' }
)
$lifecycleRemove = @([pscustomobject]@{ Directory = "INSTALLDIR" })
$lifecycleUpgrade = @([pscustomobject]@{ UpgradeCode = "{E793918B-37EB-5E2F-B866-5FC4AA3AC75A}" })
$lifecycleActions = @([pscustomobject]@{ Target = "[LAUNCHAPPARGS]" })

Assert-YorvaMsiLifecycleContract (New-LifecycleProperties) $lifecycleDirectories $lifecycleRegistry $lifecycleRemove $lifecycleUpgrade $lifecycleActions "0.3.2"
Write-Host "PASS per-user repairable lifecycle contract"

Assert-Throws "per-machine package" {
    Assert-YorvaMsiLifecycleContract @((New-LifecycleProperties) + [pscustomobject]@{ Property = "ALLUSERS"; Value = "1" }) $lifecycleDirectories $lifecycleRegistry $lifecycleRemove $lifecycleUpgrade $lifecycleActions "0.3.2"
} "per-user installation"

Assert-Throws "repair hidden" {
    Assert-YorvaMsiLifecycleContract @((New-LifecycleProperties) + [pscustomobject]@{ Property = "ARPNOREPAIR"; Value = "1" }) $lifecycleDirectories $lifecycleRegistry $lifecycleRemove $lifecycleUpgrade $lifecycleActions "0.3.2"
} "expose Repair"

Assert-Throws "unstable upgrade identity" {
    Assert-YorvaMsiLifecycleContract (New-LifecycleProperties) $lifecycleDirectories $lifecycleRegistry $lifecycleRemove @([pscustomobject]@{ UpgradeCode = "{00000000-0000-0000-0000-000000000000}" }) $lifecycleActions "0.3.2"
} "stable YORVA UpgradeCode"

Assert-Throws "user data removal" {
    Assert-YorvaMsiLifecycleContract (New-LifecycleProperties) $lifecycleDirectories $lifecycleRegistry @([pscustomobject]@{ Directory = "AppDataFolder" }) $lifecycleUpgrade $lifecycleActions "0.3.2"
} "user data root"

Assert-Throws "login startup retained" {
    Assert-YorvaMsiLifecycleContract (New-LifecycleProperties) $lifecycleDirectories @([pscustomobject]@{ Root = "1"; Key = "Software\yolin\YORVA"; Name = "InstallDir"; Value = "[INSTALLDIR]" }) $lifecycleRemove $lifecycleUpgrade $lifecycleActions "0.3.2"
} "login startup value"

Assert-Throws "deletion custom action" {
    Assert-YorvaMsiLifecycleContract (New-LifecycleProperties) $lifecycleDirectories $lifecycleRegistry $lifecycleRemove $lifecycleUpgrade @([pscustomobject]@{ Target = "powershell Remove-Item $env:APPDATA" }) "0.3.2"
} "forbidden data-deletion"

Assert-Throws "missing Hermes LICENSE" {
    $rows = @(
        [pscustomobject]@{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 },
        [pscustomobject]@{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "missing exact name LICENSE"

Assert-Throws "suffix collision" {
    $rows = @(
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
        [pscustomobject]@{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "missing exact name LICENSE"

Assert-Throws "duplicate payload" {
    $rows = @(
        [pscustomobject]@{ Name = "LICENSE"; Size = 1070 },
        [pscustomobject]@{ Name = "LICENSE"; Size = 1070 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
        [pscustomobject]@{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "2 entries named LICENSE"

Assert-Throws "wrong filename" {
    $rows = @(
        [pscustomobject]@{ Name = "LICENSE.txt"; Size = 1070 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
        [pscustomobject]@{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832 }
    )
    Assert-ExactMsiIdentities $rows $catalog
} "missing exact name LICENSE"

Assert-Throws "wrong size" {
    $rows = @(
        [pscustomobject]@{ Name = "LICENSE"; Size = 20 },
        [pscustomobject]@{ Name = "NODE-LICENSE"; Size = 148217 },
        [pscustomobject]@{ Name = "NPM-LICENSE"; Size = 9742 },
        [pscustomobject]@{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347 },
        [pscustomobject]@{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836 },
        [pscustomobject]@{ Name = "npm-12.0.2.tgz"; Size = 3045132 }
        [pscustomobject]@{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832 }
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

    [System.IO.File]::WriteAllBytes((Join-Path $payload "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"), (New-Object byte[] 73798347))
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
