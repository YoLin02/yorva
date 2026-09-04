param(
    [string]$DesktopPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\target\debug\yorva-desktop.exe"),
    [string]$DaemonPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\target\debug\yorvad.exe"),
    [string]$WorkRoot = "",
    [switch]$DisposableUserProfile
)

$ErrorActionPreference = "Stop"

function Test-PathEntry([string]$path) {
    try {
        [void](Get-Item -LiteralPath $path -Force -ErrorAction Stop)
        return $true
    }
    catch [System.Management.Automation.ItemNotFoundException] {
        return $false
    }
}

$desktop = (Resolve-Path -LiteralPath $DesktopPath).Path
$daemon = (Resolve-Path -LiteralPath $DaemonPath).Path
if (-not $DisposableUserProfile) {
    throw "This smoke uses the current Windows Known Folder app-data path. Run it only in a disposable Windows user/VM/CI environment and pass -DisposableUserProfile."
}
$productDataRoot = Join-Path $env:APPDATA "com.yorva.desktop"
$legacyProductDataRoot = Join-Path $env:APPDATA "com.yorva.desktop.dev"
if ((Test-PathEntry $productDataRoot) -or (Test-PathEntry $legacyProductDataRoot)) {
    throw "The disposable Windows profile already contains YORVA product data; refusing to mix smoke ownership."
}
if ([string]::IsNullOrWhiteSpace($WorkRoot)) {
    $WorkRoot = Join-Path $env:TEMP ("yorva-p8-recovery-smoke-" + [Guid]::NewGuid().ToString("N"))
}
$root = [IO.Path]::GetFullPath($WorkRoot)
if (Test-PathEntry $root) {
    throw "The recovery-smoke WorkRoot already exists: $root"
}
$hermesHome = Join-Path $root "hermes"
[IO.Directory]::CreateDirectory($hermesHome) | Out-Null
$diagnosticLog = Join-Path $productDataRoot "logs\install.ndjson"

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
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for exact process count $count for $executable."
}

function Wait-ExactProcessReplacement([string]$executable, [uint32]$previousProcessID, [int]$timeoutSeconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        $matches = @(Get-ExactProcesses $executable)
        if ($matches.Count -eq 1 -and $matches[0].ProcessId -ne $previousProcessID) {
            return $matches[0]
        }
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for the exact replacement process for $executable."
}

function Wait-DaemonReady([int64]$afterUnixMilliseconds, [int]$timeoutSeconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        if (Test-Path -LiteralPath $diagnosticLog -PathType Leaf) {
            $ready = @(Get-Content -LiteralPath $diagnosticLog -Tail 500 | ForEach-Object {
                try { $_ | ConvertFrom-Json } catch { $null }
            } | Where-Object {
                $_ -and $_.msg -eq "daemon listening" -and
                [DateTimeOffset]::Parse([string]$_.time).ToUnixTimeMilliseconds() -ge $afterUnixMilliseconds
            })
            if ($ready.Count -gt 0) {
                return
            }
        }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for the authenticated daemon readiness record."
}

function Start-SmokeDesktop {
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $desktop
    $startInfo.Arguments = "--hidden"
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.EnvironmentVariables["HERMES_HOME"] = $hermesHome
    $startInfo.EnvironmentVariables["PATH"] = "$env:SystemRoot\System32;$env:SystemRoot"
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    if (-not $process.Start()) {
        throw "Failed to start the YORVA Desktop candidate."
    }
    return $process
}

$ownedDesktopIds = [System.Collections.Generic.HashSet[uint32]]::new()
$ownedDaemonIds = [System.Collections.Generic.HashSet[uint32]]::new()

try {
    if ((Get-ExactProcesses $desktop).Count -ne 0 -or (Get-ExactProcesses $daemon).Count -ne 0) {
        throw "The exact candidate is already running; refusing to mix smoke ownership."
    }

    $firstStartBoundary = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    $firstDesktop = Start-SmokeDesktop
    [void]$ownedDesktopIds.Add([uint32]$firstDesktop.Id)
    [void](Wait-ExactProcessCount $desktop 1 10)
    [void](Wait-ExactProcessCount $daemon 1 50)
    Wait-DaemonReady $firstStartBoundary 50
    $firstDaemon = (Wait-ExactProcessCount $daemon 1 10)[0]
    [void]$ownedDaemonIds.Add([uint32]$firstDaemon.ProcessId)

    $restartBoundary = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    Stop-Process -Id $firstDaemon.ProcessId -Force
    $restartedDaemon = Wait-ExactProcessReplacement $daemon $firstDaemon.ProcessId 50
    Wait-DaemonReady $restartBoundary 50
    $restartedDaemon = (Wait-ExactProcessCount $daemon 1 10)[0]
    if ($restartedDaemon.ProcessId -eq $firstDaemon.ProcessId) {
        throw "The daemon identity did not change after the forced crash."
    }
    [void]$ownedDaemonIds.Add([uint32]$restartedDaemon.ProcessId)
    [void](Wait-ExactProcessCount $desktop 1 10)

    $secondLaunch = Start-SmokeDesktop
    [void]$ownedDesktopIds.Add([uint32]$secondLaunch.Id)
    if (-not $secondLaunch.WaitForExit(10000)) {
        throw "The second Desktop launch did not yield to the existing instance."
    }
    [void](Wait-ExactProcessCount $desktop 1 10)
    [void](Wait-ExactProcessCount $daemon 1 10)

    Stop-Process -Id $firstDesktop.Id -Force
    [void](Wait-ExactProcessCount $desktop 0 10)
    [void](Wait-ExactProcessCount $daemon 0 10)

    $reopenBoundary = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    $reopenedDesktop = Start-SmokeDesktop
    [void]$ownedDesktopIds.Add([uint32]$reopenedDesktop.Id)
    [void](Wait-ExactProcessCount $daemon 1 50)
    Wait-DaemonReady $reopenBoundary 50
    $reopenedDaemon = (Wait-ExactProcessCount $daemon 1 10)[0]
    [void]$ownedDaemonIds.Add([uint32]$reopenedDaemon.ProcessId)
    [void](Wait-ExactProcessCount $desktop 1 10)

    Stop-Process -Id $reopenedDesktop.Id -Force
    [void](Wait-ExactProcessCount $desktop 0 10)
    [void](Wait-ExactProcessCount $daemon 0 10)
    Write-Output "Windows Desktop recovery smoke: PASS"
    Write-Output "Work root: $root"
}
finally {
    foreach ($processId in $ownedDesktopIds) {
        $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($process -and [string]::Equals($process.Path, $desktop, [StringComparison]::OrdinalIgnoreCase)) {
            Stop-Process -Id $processId -Force
        }
    }
    foreach ($processId in $ownedDaemonIds) {
        $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($process -and [string]::Equals($process.Path, $daemon, [StringComparison]::OrdinalIgnoreCase)) {
            Stop-Process -Id $processId -Force
        }
    }
    foreach ($process in @(Get-ExactProcesses $daemon)) {
        if ($ownedDesktopIds.Contains([uint32]$process.ParentProcessId)) {
            Stop-Process -Id $process.ProcessId -Force
        }
    }
}
