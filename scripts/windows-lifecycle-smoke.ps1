param(
    [string]$SidecarPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\binaries\yorvad-x86_64-pc-windows-msvc.exe")
)

$ErrorActionPreference = "Stop"

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

    $bootstrap = @{
        protocolVersion = "1"
        token = New-SessionToken
        dataDir = $dataDir
    } | ConvertTo-Json -Compress
    Write-ControlLine $process $bootstrap

    $handshakeTask = $process.StandardOutput.ReadLineAsync()
    if (-not $handshakeTask.Wait([TimeSpan]::FromSeconds(45))) {
        $process.Kill()
        throw "Timed out waiting for the yorvad handshake."
    }
    $handshakeLine = $handshakeTask.Result
    if ([string]::IsNullOrWhiteSpace($handshakeLine)) {
        $stderr = $process.StandardError.ReadToEnd()
        if (-not $process.HasExited) {
            $process.Kill()
        }
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
        $stderr = $process.StandardError.ReadToEnd()
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
        $process.Dispose()
    }
    $resolvedTemp = [System.IO.Path]::GetFullPath($tempRoot)
    $resolvedSystemTemp = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    if ($resolvedTemp.StartsWith($resolvedSystemTemp, [StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $resolvedTemp)) {
        [System.IO.Directory]::Delete($resolvedTemp, $true)
    }
}
