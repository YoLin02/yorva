param(
    [ValidateSet("Prepare", "Verify")]
    [string]$Mode,
    [string]$DesktopPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\target\release\yorva-desktop.exe"),
    [string]$DaemonPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\target\release\yorvad.exe"),
    [string]$StatePath = (Join-Path $PSScriptRoot "..\.tools\p8-reboot-smoke\state.json")
)

$ErrorActionPreference = "Stop"

$desktop = (Resolve-Path -LiteralPath $DesktopPath).Path
$daemon = (Resolve-Path -LiteralPath $DaemonPath).Path
$state = [System.IO.Path]::GetFullPath($StatePath)
$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$stateRoot = [System.IO.Path]::GetFullPath((Join-Path $repositoryRoot ".tools\p8-reboot-smoke"))
$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$dataRoot = Join-Path $env:APPDATA "com.yorva.desktop"
$diagnosticLog = Join-Path $dataRoot "logs\install.ndjson"

if (-not $state.StartsWith($stateRoot + [System.IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -and $state -ne (Join-Path $stateRoot "state.json")) {
    throw "StatePath must stay inside the repository .tools/p8-reboot-smoke directory."
}

function Get-ExactProcesses([string]$executable) {
    return @(Get-CimInstance Win32_Process | Where-Object {
        $_.ExecutablePath -and [string]::Equals(
            [System.IO.Path]::GetFullPath($_.ExecutablePath),
            [System.IO.Path]::GetFullPath($executable),
            [StringComparison]::OrdinalIgnoreCase
        )
    })
}

function Wait-ExactProcessCount([string]$executable, [int]$count, [int]$timeoutSeconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        $matches = @(Get-ExactProcesses $executable)
        if ($matches.Count -eq $count) {
            return $matches
        }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for exact process count $count for $executable."
}

function Wait-StableOwnedCandidate([int]$desktopProcessID, [int]$timeoutSeconds, [int]$stableSeconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    $stableSince = $null
    $lastDesktop = @()
    $lastDaemon = @()
    do {
        $lastDesktop = @(Get-ExactProcesses $desktop)
        $lastDaemon = @(Get-ExactProcesses $daemon)
        $owned = $lastDesktop.Count -eq 1 -and
            $lastDesktop[0].ProcessId -eq $desktopProcessID -and
            $lastDaemon.Count -eq 1 -and
            $lastDaemon[0].ParentProcessId -eq $desktopProcessID
        if ($owned) {
            if ($null -eq $stableSince) {
                $stableSince = [DateTime]::UtcNow
            }
            if (([DateTime]::UtcNow - $stableSince).TotalSeconds -ge $stableSeconds) {
                return $lastDaemon[0]
            }
        } else {
            $stableSince = $null
        }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)

    $desktopSummary = ($lastDesktop | ForEach-Object { "pid=$($_.ProcessId)" }) -join ","
    $daemonSummary = ($lastDaemon | ForEach-Object { "pid=$($_.ProcessId),parent=$($_.ParentProcessId)" }) -join ","
    throw "Timed out waiting for stable candidate ownership. Desktop=[$desktopSummary] daemon=[$daemonSummary]."
}

function Start-Candidate {
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $desktop
    $startInfo.Arguments = "--hidden"
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    if (-not $process.Start()) {
        throw "Failed to start the reboot candidate."
    }
    return $process
}

function Get-LoginCommand {
    if (-not (Test-Path -LiteralPath $runKey)) {
        return $null
    }
    return (Get-ItemProperty -LiteralPath $runKey -Name "Yorva" -ErrorAction SilentlyContinue).Yorva
}

function Assert-LoginCommand {
    $loginCommand = Get-LoginCommand
    if ([string]::IsNullOrWhiteSpace($loginCommand)) {
        throw "The per-user YORVA login item is missing."
    }
    $normalized = $loginCommand.Trim().Trim('"')
    if (-not $normalized.StartsWith($desktop, [StringComparison]::OrdinalIgnoreCase) -or -not $normalized.EndsWith("--hidden", [StringComparison]::OrdinalIgnoreCase)) {
        throw "The per-user login item does not target the exact candidate with --hidden."
    }
}

function Stop-OwnedCandidate([System.Diagnostics.Process]$process) {
    if (-not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
        [void]$process.WaitForExit(10000)
    }
    [void](Wait-ExactProcessCount $desktop 0 10)
    [void](Wait-ExactProcessCount $daemon 0 10)
}

if ($Mode -eq "Prepare") {
    if ((Get-ExactProcesses $desktop).Count -ne 0 -or (Get-ExactProcesses $daemon).Count -ne 0) {
        throw "The exact release candidate is already running; close it before preparation."
    }
    $candidate = Start-Candidate
    try {
        $daemonProcess = Wait-StableOwnedCandidate $candidate.Id 90 4
        Assert-LoginCommand
        [System.IO.Directory]::CreateDirectory($stateRoot) | Out-Null
        $record = [ordered]@{
            schemaVersion = 1
            preparedAtUtc = [DateTime]::UtcNow.ToString("O")
            preparedAtUnixMilliseconds = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
            bootTimeFileTimeUtc = (Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToUniversalTime().ToFileTimeUtc()
            desktopPath = $desktop
            desktopSha256 = (Get-FileHash -LiteralPath $desktop -Algorithm SHA256).Hash
            daemonPath = $daemon
            daemonSha256 = (Get-FileHash -LiteralPath $daemon -Algorithm SHA256).Hash
            loginCommandVerified = $true
        }
        [System.IO.File]::WriteAllText(
            $state,
            ($record | ConvertTo-Json -Depth 3),
            [System.Text.UTF8Encoding]::new($false)
        )
    }
    finally {
        Stop-OwnedCandidate $candidate
        $candidate.Dispose()
    }
    Write-Output "Windows reboot recovery smoke prepared. Reboot, sign in, then run Verify."
    exit 0
}

if (-not (Test-Path -LiteralPath $state -PathType Leaf)) {
    throw "The bounded reboot preparation record is missing."
}
$stateInfo = Get-Item -LiteralPath $state
if ($stateInfo.Length -le 0 -or $stateInfo.Length -gt 8192) {
    throw "The reboot preparation record has an invalid size."
}
$record = Get-Content -LiteralPath $state -Raw | ConvertFrom-Json
if ($record.schemaVersion -ne 1 -or -not $record.loginCommandVerified) {
    throw "The reboot preparation record is invalid."
}
$preparedBootTime = [int64]$record.bootTimeFileTimeUtc
$currentBootTime = (Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToUniversalTime().ToFileTimeUtc()
if ($currentBootTime -le $preparedBootTime) {
    throw "Windows has not rebooted since this preparation record was created."
}
if (-not [string]::Equals($record.desktopPath, $desktop, [StringComparison]::OrdinalIgnoreCase) -or
    -not [string]::Equals($record.daemonPath, $daemon, [StringComparison]::OrdinalIgnoreCase)) {
    throw "The reboot candidate paths changed after preparation."
}
if ((Get-FileHash -LiteralPath $desktop -Algorithm SHA256).Hash -ne $record.desktopSha256 -or
    (Get-FileHash -LiteralPath $daemon -Algorithm SHA256).Hash -ne $record.daemonSha256) {
    throw "The reboot candidate bytes changed after preparation."
}
Assert-LoginCommand
$desktopProcesses = @(Wait-ExactProcessCount $desktop 1 30)
$daemonProcesses = @(Wait-StableOwnedCandidate $desktopProcesses[0].ProcessId 90 4)

$preparedAt = [int64]$record.preparedAtUnixMilliseconds
$recentRecords = if (Test-Path -LiteralPath $diagnosticLog -PathType Leaf) {
    @(Get-Content -LiteralPath $diagnosticLog -Tail 500 | ForEach-Object {
        try { $_ | ConvertFrom-Json } catch { $null }
    } | Where-Object { $_ })
} else {
    @()
}
$discovery = @($recentRecords | Where-Object {
    $_.msg -eq "runtime discovery completed" -and
    [DateTimeOffset]::Parse([string]$_.time).ToUnixTimeMilliseconds() -gt $preparedAt
})
$inventory = @($recentRecords | Where-Object {
    $_.msg -eq "startup Instance inventory reconciled" -and
    [DateTimeOffset]::Parse([string]$_.time).ToUnixTimeMilliseconds() -gt $preparedAt
})
if ($discovery.Count -eq 0 -or $inventory.Count -eq 0) {
    throw "Post-reboot authoritative Runtime/Instance reconciliation evidence is missing."
}

$secondLaunch = Start-Candidate
try {
    if (-not $secondLaunch.WaitForExit(10000)) {
        throw "The post-reboot second launch did not yield to the existing instance."
    }
    $settledDaemon = Wait-StableOwnedCandidate $desktopProcesses[0].ProcessId 20 2
    if ($settledDaemon.ProcessId -ne $daemonProcesses[0].ProcessId) {
        throw "The post-reboot second launch replaced the incumbent daemon process."
    }
}
finally {
    if (-not $secondLaunch.HasExited) {
        Stop-Process -Id $secondLaunch.Id -Force
    }
    $secondLaunch.Dispose()
}

Remove-Item -LiteralPath $state -Force
Write-Output "Windows reboot/login recovery smoke: PASS"
