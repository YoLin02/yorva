param(
    [string]$SidecarPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\binaries\yorvad-x86_64-pc-windows-msvc.exe")
)

$ErrorActionPreference = "Stop"
$script:smokeStderr = @{}

function Read-SmokeFailure([System.Diagnostics.Process]$process) {
    $task = $script:smokeStderr[$process.Id]
    if (-not $task -or -not $task.Wait(5000)) { return 'stderr drain did not complete' }
    $text = $task.Result
    if ($text.Length -gt 8192) { $text = $text.Substring($text.Length - 8192) }
    return $text
}

function New-SessionToken {
    $bytes = [byte[]]::new(32)
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($bytes)
    }
    finally {
        $generator.Dispose()
    }
    return [Convert]::ToBase64String($bytes).TrimEnd("=").Replace("+", "-").Replace("/", "_")
}

function Start-SmokeDaemon([string]$dataDir) {
    $scenario = Split-Path -Leaf $dataDir
    $elapsed = [System.Diagnostics.Stopwatch]::StartNew()
    Write-Host "Windows lifecycle smoke: starting $scenario"
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = (Resolve-Path -LiteralPath $SidecarPath).Path
    $startInfo.Arguments = "--bootstrap-stdio"
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardInput = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    if (-not $process.Start()) {
        throw "Failed to start yorvad."
    }
    # Drain diagnostics while waiting for stdout; a redirected pipe must not
    # block the child before it can write the handshake or exit gracefully.
    $script:smokeStderr[$process.Id] = $process.StandardError.ReadToEndAsync()

    $bootstrap = @{
        protocolVersion = "1"
        token = New-SessionToken
        dataDir = $dataDir
    } | ConvertTo-Json -Compress
    Write-ControlLine $process $bootstrap

    $handshakeTask = $process.StandardOutput.ReadLineAsync()
    if (-not $handshakeTask.Wait([TimeSpan]::FromSeconds(45))) {
        $process.Kill()
        $null = $process.WaitForExit(5000)
        $stderr = Read-SmokeFailure $process
        $process.Dispose()
        throw "Timed out waiting for the yorvad handshake ($scenario): $stderr"
    }
    $handshakeLine = $handshakeTask.Result
    if ([string]::IsNullOrWhiteSpace($handshakeLine)) {
        if (-not $process.HasExited) {
            $process.Kill()
        }
        $null = $process.WaitForExit(5000)
        $stderr = Read-SmokeFailure $process
        $process.Dispose()
        throw "yorvad exited before the handshake: $stderr"
    }
    $handshake = $handshakeLine | ConvertFrom-Json
    if ($handshake.protocolVersion -ne "1" -or $handshake.pid -ne $process.Id -or $handshake.port -le 0) {
        $process.Kill()
        throw "Invalid yorvad handshake."
    }

    $health = Invoke-RestMethod -Uri "http://127.0.0.1:$($handshake.port)/api/v1/health" -TimeoutSec 5
    if ($health.status -ne "ok") {
        $process.Kill()
        throw "yorvad health check failed."
    }
    Write-Host "Windows lifecycle smoke: $scenario handshake and health in $($elapsed.ElapsedMilliseconds) ms"
    return $process
}

function Write-ControlLine([System.Diagnostics.Process]$process, [string]$line) {
    $bytes = [System.Text.UTF8Encoding]::new($false).GetBytes($line + "`n")
    $process.StandardInput.BaseStream.Write($bytes, 0, $bytes.Length)
    $process.StandardInput.BaseStream.Flush()
}

function Wait-GracefulExit([System.Diagnostics.Process]$process, [string]$scenario) {
    if (-not $process.WaitForExit(5000)) {
        $process.Kill()
        throw "yorvad did not exit after $scenario."
    }
    if ($process.ExitCode -ne 0) {
        $stderr = Read-SmokeFailure $process
        throw "yorvad exited with code $($process.ExitCode) after ${scenario}: $stderr"
    }
}

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("yorva-lifecycle-" + [Guid]::NewGuid().ToString("N"))
[System.IO.Directory]::CreateDirectory($tempRoot) | Out-Null
$processes = [System.Collections.Generic.List[System.Diagnostics.Process]]::new()

try {
    $shutdownProcess = Start-SmokeDaemon (Join-Path $tempRoot "shutdown")
    $processes.Add($shutdownProcess)
    Write-ControlLine $shutdownProcess '{"type":"shutdown"}'
    Wait-GracefulExit $shutdownProcess "shutdown control"

    $reopenData = Join-Path $tempRoot "desktop-reopen"
    $eofProcess = Start-SmokeDaemon $reopenData
    $processes.Add($eofProcess)
    $eofProcess.StandardInput.Close()
    Wait-GracefulExit $eofProcess "parent stdin EOF"

    $reopenedProcess = Start-SmokeDaemon $reopenData
    $processes.Add($reopenedProcess)
    Write-ControlLine $reopenedProcess '{"type":"shutdown"}'
    Wait-GracefulExit $reopenedProcess "Desktop reopen"

    $restartData = Join-Path $tempRoot "daemon-restart"
    $crashedProcess = Start-SmokeDaemon $restartData
    $processes.Add($crashedProcess)
    $crashedProcess.Kill()
    if (-not $crashedProcess.WaitForExit(5000)) {
        throw "Killed yorvad did not terminate."
    }

    $restartedProcess = Start-SmokeDaemon $restartData
    $processes.Add($restartedProcess)
    Write-ControlLine $restartedProcess '{"type":"shutdown"}'
    Wait-GracefulExit $restartedProcess "daemon crash restart"

    Write-Output "Windows lifecycle smoke: PASS"
}
finally {
    foreach ($process in $processes) {
        if (-not $process.HasExited) {
            $process.Kill()
            $process.WaitForExit()
        }
        $stderrTask = $script:smokeStderr[$process.Id]
        if ($stderrTask) { $null = $stderrTask.Wait(5000) }
        $process.Dispose()
    }
    $resolvedTemp = [System.IO.Path]::GetFullPath($tempRoot)
    $resolvedSystemTemp = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    if ($resolvedTemp.StartsWith($resolvedSystemTemp, [StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $resolvedTemp)) {
        [System.IO.Directory]::Delete($resolvedTemp, $true)
    }
}
