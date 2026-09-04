$ErrorActionPreference = "Stop"

function Write-Serial([string]$message) {
    $port = [System.IO.Ports.SerialPort]::new("COM1", 115200, "None", 8, "One")
    try {
        $port.Open()
        $port.WriteLine("$([DateTime]::UtcNow.ToString('O')) $message")
    }
    finally {
        if ($port.IsOpen) { $port.Close() }
        $port.Dispose()
    }
}

try {
    $sourceRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
    $workerRoot = "C:\ProgramData\YorvaDisposable\b4"
    $workerPath = Join-Path $workerRoot "worker.ps1"
    [IO.Directory]::CreateDirectory($workerRoot) | Out-Null
    Copy-Item -LiteralPath (Join-Path $sourceRoot "worker.ps1") -Destination $workerPath -Force

    $scenario = (Get-Content -LiteralPath (Join-Path $sourceRoot "payload\scenario.txt") -Raw).Trim()
    $taskName = "YorvaB4UpdateLifecycle"
    $user = [Security.Principal.WindowsIdentity]::GetCurrent().Name
    $arguments = '-NoProfile -ExecutionPolicy Bypass -File "{0}" -PayloadRoot "{1}" -Scenario "{2}"' -f $workerPath, (Join-Path $sourceRoot "payload"), $scenario
    $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $arguments
    $principal = New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Limited
    $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit (New-TimeSpan -Minutes 15) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    Register-ScheduledTask -TaskName $taskName -Action $action -Principal $principal -Settings $settings -Force | Out-Null
    Start-ScheduledTask -TaskName $taskName
    Write-Serial "YORVA_B4_LIMITED_TASK_STARTED $scenario"
}
catch {
    Write-Serial "YORVA_B4_LAUNCH_FAIL $($_.Exception.Message)"
    shutdown.exe /s /t 5 /f | Out-Null
}
