param(
    [Parameter(Mandatory = $true)]
    [string]$PayloadRoot,
    [Parameter(Mandatory = $true)]
    [string]$Scenario
)

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
    $evidence = "C:\ProgramData\YorvaDisposable\b4\update-$($Scenario.ToLowerInvariant()).json"
    $output = & (Join-Path $PayloadRoot "windows-yorva-update-smoke.ps1") `
        -BaselineMsi (Join-Path $PayloadRoot "YORVA_0.3.2_x64_en-US.msi") `
        -HermesFixturePath (Join-Path $PayloadRoot "hermes.exe") `
        -Scenario $Scenario -DisposableWindowsProfile `
        -EvidencePath $evidence 2>&1
    $output | ForEach-Object { Write-Serial "YORVA_B4_OUTPUT $_" }
    if (-not (Test-Path -LiteralPath $evidence -PathType Leaf)) {
        throw "update smoke evidence was not written"
    }
    Write-Serial "YORVA_B4_UPDATE_PASS $Scenario"
}
catch {
    Write-Serial "YORVA_B4_UPDATE_FAIL $Scenario $($_.Exception.Message)"
}
finally {
    shutdown.exe /s /t 10 /f | Out-Null
}
