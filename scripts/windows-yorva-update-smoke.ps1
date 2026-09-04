param(
    [Parameter(Mandatory = $true)]
    [string]$BaselineMsi,
    [Parameter(Mandatory = $true)]
    [string]$HermesFixturePath,
    [ValidateSet("Happy", "Tamper", "Interrupted", "InstallerFailure")]
    [string]$Scenario,
    [string]$EvidencePath = (Join-Path $env:TEMP "yorva-update-smoke.json")
)

$ErrorActionPreference = "Stop"
$baseline = (Resolve-Path -LiteralPath $BaselineMsi).Path
$hermesFixture = (Resolve-Path -LiteralPath $HermesFixturePath).Path
$installRoot = Join-Path $env:LOCALAPPDATA "Programs\YORVA"
$desktopPath = Join-Path $installRoot "yorva-desktop.exe"
$daemonPath = Join-Path $installRoot "yorvad.exe"
$appDataRoot = Join-Path $env:APPDATA "com.yorva.desktop"
$updateState = Join-Path $appDataRoot "update-staging\state.json"
$hermesRoot = Join-Path $env:LOCALAPPDATA "hermes"
$result = [ordered]@{
    schemaVersion = 1
    scenario = $Scenario
    baselineVersion = "0.3.2"
    candidateVersion = "0.4.0"
    startedAtUtc = [DateTime]::UtcNow.ToString("O")
    checks = [System.Collections.Generic.List[string]]::new()
}

function Assert-Condition([bool]$condition, [string]$message) {
    if (-not $condition) { throw $message }
}

function Add-Pass([string]$name) {
    $result.checks.Add($name)
    Write-Output "PASS $name"
}

function Get-YorvaVersion {
    $root = "Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Uninstall"
    $entries = @(Get-ChildItem -LiteralPath $root -ErrorAction SilentlyContinue | ForEach-Object {
        Get-ItemProperty -LiteralPath $_.PSPath
    } | Where-Object { $_.DisplayName -eq "YORVA" })
    Assert-Condition ($entries.Count -eq 1) "expected exactly one YORVA uninstall entry"
    return [string]$entries[0].DisplayVersion
}

function Get-ExactProcesses([string]$path) {
    return @(Get-CimInstance Win32_Process | Where-Object {
        $_.ExecutablePath -and [string]::Equals(
            [System.IO.Path]::GetFullPath($_.ExecutablePath),
            [System.IO.Path]::GetFullPath($path),
            [StringComparison]::OrdinalIgnoreCase
        )
    })
}

function Wait-UpdateTerminal([int]$timeoutSeconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        if (Test-Path -LiteralPath $updateState -PathType Leaf) {
            try {
                $state = Get-Content -LiteralPath $updateState -Raw | ConvertFrom-Json
                if ($state.phase -in @("SUCCEEDED", "FAILED")) { return $state }
            }
            catch {}
        }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "update did not reach a terminal state"
}

function Wait-Reconcile([int]$timeoutSeconds) {
    $logPath = Join-Path $appDataRoot "logs\install.ndjson"
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        if (Test-Path -LiteralPath $logPath -PathType Leaf) {
            $tail = Get-Content -LiteralPath $logPath -Tail 500 -ErrorAction SilentlyContinue
            if (($tail -match 'runtime discovery completed').Count -gt 0 -and ($tail -match 'startup Instance inventory reconciled').Count -gt 0) {
                return
            }
        }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Runtime and Instance authoritative readback did not complete"
}

$groups = whoami.exe /groups /fo csv /nh
Assert-Condition (($groups -join "`n") -match 'S-1-16-8192') "worker is not at medium integrity"
Assert-Condition (($groups -join "`n") -notmatch 'S-1-16-12288') "worker is unexpectedly elevated"
Add-Pass "non-elevated-user-context"

$install = Start-Process -FilePath "$env:SystemRoot\System32\msiexec.exe" -ArgumentList @('/i', $baseline, '/qn', '/norestart') -Wait -PassThru -WindowStyle Hidden
Assert-Condition ($install.ExitCode -eq 0) "baseline MSI install failed with $($install.ExitCode)"
Assert-Condition ((Get-YorvaVersion) -eq "0.3.2") "baseline version was not installed"
Add-Pass "baseline-install"

[System.IO.Directory]::CreateDirectory((Join-Path $hermesRoot "bin")) | Out-Null
[System.IO.Directory]::CreateDirectory((Join-Path $hermesRoot "profiles\sentinel")) | Out-Null
[System.IO.Directory]::CreateDirectory($appDataRoot) | Out-Null
Copy-Item -LiteralPath $hermesFixture -Destination (Join-Path $hermesRoot "bin\hermes.exe") -Force
[System.IO.File]::WriteAllText((Join-Path $hermesRoot "profiles\sentinel\profile.txt"), "hermes-profile", [Text.UTF8Encoding]::new($false))
[System.IO.File]::WriteAllText((Join-Path $appDataRoot "update-sentinel.txt"), "yorva-data", [Text.UTF8Encoding]::new($false))

Start-Process -FilePath $desktopPath -ArgumentList @('--hidden', '--qualify-fixed-update') -WindowStyle Hidden | Out-Null
$terminal = Wait-UpdateTerminal 360
Assert-Condition ((Get-Content -LiteralPath (Join-Path $appDataRoot "update-sentinel.txt") -Raw) -eq "yorva-data") "YORVA data sentinel changed"
Assert-Condition ((Get-Content -LiteralPath (Join-Path $hermesRoot "profiles\sentinel\profile.txt") -Raw) -eq "hermes-profile") "Hermes data sentinel changed"

if ($Scenario -eq "Happy") {
    Assert-Condition ($terminal.phase -eq "SUCCEEDED") "happy update ended in $($terminal.phase)"
    Assert-Condition ([string]::IsNullOrEmpty([string]$terminal.errorCode)) "happy update retained an error"
    Assert-Condition ((Get-YorvaVersion) -eq "0.4.0") "candidate version was not installed"
    Assert-Condition (@(Get-ExactProcesses $desktopPath).Count -gt 0) "candidate Desktop did not relaunch"
    Assert-Condition (@(Get-ExactProcesses $daemonPath).Count -gt 0) "candidate daemon did not relaunch"
    Wait-Reconcile 120
    Add-Pass "older-to-candidate-update"
    Add-Pass "installed-version-readback"
    Add-Pass "runtime-instance-authoritative-readback"
} else {
    $expectedError = if ($Scenario -eq "InstallerFailure") { "UPDATE_INSTALL_FAILED" } elseif ($Scenario -eq "Tamper") { "UPDATE_INTEGRITY_FAILED" } else { "UPDATE_DOWNLOAD_FAILED" }
    Assert-Condition ($terminal.phase -eq "FAILED") "$Scenario did not fail closed"
    Assert-Condition ($terminal.errorCode -eq $expectedError) "$Scenario error $($terminal.errorCode) != $expectedError"
    Assert-Condition ((Get-YorvaVersion) -eq "0.3.2") "$Scenario replaced the baseline version"
    Assert-Condition (@(Get-ExactProcesses $desktopPath).Count -gt 0) "$Scenario left the old Desktop unusable"
    Assert-Condition (@(Get-ExactProcesses $daemonPath).Count -gt 0) "$Scenario left the old daemon unusable"
    Add-Pass ($Scenario.ToLowerInvariant() + "-old-version-usable")
}

$result["terminalPhase"] = [string]$terminal.phase
$result["errorCode"] = [string]$terminal.errorCode
$result["completedAtUtc"] = [DateTime]::UtcNow.ToString("O")
[System.IO.Directory]::CreateDirectory((Split-Path -Parent ([IO.Path]::GetFullPath($EvidencePath)))) | Out-Null
[System.IO.File]::WriteAllText([IO.Path]::GetFullPath($EvidencePath), ($result | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
Write-Output "Windows YORVA update smoke: PASS $Scenario"
