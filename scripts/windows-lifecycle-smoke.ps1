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
    $process.StandardInput.WriteLine($bootstrap)
    $process.StandardInput.Flush()

    $handshakeTask = $process.StandardOutput.ReadLineAsync()
    if (-not $handshakeTask.Wait([TimeSpan]::FromSeconds(10))) {
        $process.Kill()
        throw "Timed out waiting for the yorvad handshake."
    }
    $handshake = $handshakeTask.Result | ConvertFrom-Json
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
    $shutdownProcess.StandardInput.WriteLine('{"type":"shutdown"}')
    $shutdownProcess.StandardInput.Flush()
    Wait-GracefulExit $shutdownProcess "shutdown control"

    $eofProcess = Start-SmokeDaemon (Join-Path $tempRoot "parent-eof")
    $processes.Add($eofProcess)
    $eofProcess.StandardInput.Close()
    Wait-GracefulExit $eofProcess "parent stdin EOF"

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
