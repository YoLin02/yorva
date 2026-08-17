param(
    [Parameter(Mandatory = $true)]
    [string]$MsiPath
)

$ErrorActionPreference = "Stop"
if (-not (Test-Path -LiteralPath $MsiPath)) {
    throw "MSI not found: $MsiPath"
}

$installer = New-Object -ComObject WindowsInstaller.Installer
$database = $installer.GetType().InvokeMember("OpenDatabase", "InvokeMethod", $null, $installer, @((Resolve-Path $MsiPath).Path, 0))
$view = $database.GetType().InvokeMember("OpenView", "InvokeMethod", $null, $database, @("SELECT FileName, FileSize FROM File"))
$view.GetType().InvokeMember("Execute", "InvokeMethod", $null, $view, $null)

$files = @{}
while ($true) {
    $record = $view.GetType().InvokeMember("Fetch", "InvokeMethod", $null, $view, $null)
    if ($null -eq $record) {
        break
    }
    $name = $record.GetType().InvokeMember("StringData", "GetProperty", $null, $record, 1)
    $size = [int64]$record.GetType().InvokeMember("IntegerData", "GetProperty", $null, $record, 2)
    if ($name -match '\|') {
        $name = ($name -split '\|', 2)[1]
    }
    $files[$name] = $size
}

$required = @{
    "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip" = 71869305
    "node-v22.23.1-win-x64.zip"                                = 35682836
    "npm-12.0.2.tgz"                                           = 3045132
    "LICENSE"                                                  = 1
    "NODE-LICENSE"                                             = 1
    "NPM-LICENSE"                                              = 1
}

foreach ($name in $required.Keys) {
    $match = $files.Keys | Where-Object { $_ -eq $name -or $_ -like "*$name" } | Select-Object -First 1
    if (-not $match) {
        throw "MSI File table missing $name"
    }
    if ($required[$name] -gt 1 -and $files[$match] -ne $required[$name]) {
        throw "MSI $match size $($files[$match]) != $($required[$name])"
    }
    if ($files[$match] -lt $required[$name]) {
        throw "MSI $match is too small"
    }
    Write-Host "msi contains $match $($files[$match])"
}
