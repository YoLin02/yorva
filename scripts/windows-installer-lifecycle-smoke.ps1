param(
    [Parameter(Mandatory = $true)]
    [string]$BaselineMsi,
    [Parameter(Mandatory = $true)]
    [string]$CandidateMsi,
    [Parameter(Mandatory = $true)]
    [string]$HermesFixturePath,
    [string]$EvidencePath = (Join-Path $env:TEMP "yorva-installer-lifecycle.json")
)

$ErrorActionPreference = "Stop"
$baseline = (Resolve-Path -LiteralPath $BaselineMsi).Path
$candidate = (Resolve-Path -LiteralPath $CandidateMsi).Path
$hermesFixture = (Resolve-Path -LiteralPath $HermesFixturePath).Path
$installRoot = Join-Path $env:LOCALAPPDATA "Programs\YORVA"
$desktopPath = Join-Path $installRoot "yorva-desktop.exe"
$daemonPath = Join-Path $installRoot "yorvad.exe"
$appDataRoot = Join-Path $env:APPDATA "com.yorva.desktop"
$backupRoot = Join-Path $appDataRoot "backups"
$hermesRoot = Join-Path $env:LOCALAPPDATA "hermes"
$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$uninstallRoot = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall"
$startMenuShortcut = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\YORVA\YORVA.lnk"
$desktopShortcut = Join-Path ([Environment]::GetFolderPath("Desktop")) "YORVA.lnk"
$results = [ordered]@{
    schemaVersion = 1
    startedAtUtc = [DateTime]::UtcNow.ToString("O")
    baselineVersion = "0.3.2"
    candidateVersion = "0.4.0"
    checks = [System.Collections.Generic.List[string]]::new()
}

function Add-Pass([string]$name) {
    $results.checks.Add($name)
    Write-Output "PASS $name"
}

function Assert-Condition([bool]$condition, [string]$message) {
    if (-not $condition) {
        throw $message
    }
}

function Invoke-Msi([string[]]$arguments, [string]$name) {
    $process = Start-Process -FilePath "$env:SystemRoot\System32\msiexec.exe" -ArgumentList $arguments -Wait -PassThru -WindowStyle Hidden
    if ($process.ExitCode -ne 0) {
        throw "$name failed with Windows Installer exit code $($process.ExitCode)"
    }
}

function Get-YorvaArpEntry {
    $found = [ordered]@{}
    foreach ($root in @(
        "Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Uninstall",
        "Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Uninstall"
    )) {
        if (Test-Path -LiteralPath $root) {
            foreach ($key in @(Get-ChildItem -LiteralPath $root)) {
                $value = Get-ItemProperty -LiteralPath $key.PSPath
                if ($value.DisplayName -eq "YORVA" -and -not $found.Contains($key.PSChildName)) {
                    $found[$key.PSChildName] = [pscustomobject]@{
                        ProductCode = $key.PSChildName
                        DisplayVersion = [string]$value.DisplayVersion
                        InstallLocation = [string]$value.InstallLocation
                    }
                }
            }
        }
    }
    return @($found.Values)
}

function Get-ProductInstallContext([string]$productCode) {
    $installer = New-Object -ComObject WindowsInstaller.Installer
    $products = $installer.GetType().InvokeMember("ProductsEx", "GetProperty", $null, $installer, @("", "", 7))
    foreach ($product in $products) {
        $code = [string]$product.GetType().InvokeMember("ProductCode", "GetProperty", $null, $product, $null)
        if ($code -eq $productCode) {
            return [uint32]$product.GetType().InvokeMember("Context", "GetProperty", $null, $product, $null)
        }
    }
    throw "Windows Installer did not enumerate product $productCode"
}

function Assert-Installed([string]$version) {
    Assert-Condition (Test-Path -LiteralPath $desktopPath -PathType Leaf) "installed Desktop executable is missing"
    Assert-Condition (Test-Path -LiteralPath $daemonPath -PathType Leaf) "installed daemon executable is missing"
    Assert-Condition (Test-Path -LiteralPath $startMenuShortcut -PathType Leaf) "Start Menu shortcut is missing"
    Assert-Condition (Test-Path -LiteralPath $desktopShortcut -PathType Leaf) "Desktop shortcut is missing"
    $entries = @(Get-YorvaArpEntry)
    Assert-Condition ($entries.Count -eq 1) "expected one per-user YORVA uninstall entry, found $($entries.Count)"
    $context = Get-ProductInstallContext $entries[0].ProductCode
    Assert-Condition ($context -in @(1, 2)) "YORVA was registered in machine installation context $context"
    Assert-Condition ($entries[0].DisplayVersion -eq $version) "installed version $($entries[0].DisplayVersion) != $version"
    $reportedInstallRoot = [System.IO.Path]::GetFullPath($entries[0].InstallLocation).TrimEnd('\')
    $expectedInstallRoot = [System.IO.Path]::GetFullPath($installRoot).TrimEnd('\')
    Assert-Condition ($reportedInstallRoot -eq $expectedInstallRoot) "unexpected per-user install location"
    return $entries[0]
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

function Wait-ExactProcess([string]$path, [int]$timeoutSeconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        $found = @(Get-ExactProcesses $path)
        if ($found.Count -gt 0) {
            return $found[0]
        }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "timed out waiting for $path"
}

function Stop-YorvaProcesses {
    $deadline = [DateTime]::UtcNow.AddSeconds(30)
    $stableSince = $null
    do {
        $running = @()
        foreach ($path in @($desktopPath, $daemonPath)) {
            $running += @(Get-ExactProcesses $path)
        }
        if ($running.Count -eq 0) {
            if ($null -eq $stableSince) {
                $stableSince = [DateTime]::UtcNow
            }
            if (([DateTime]::UtcNow - $stableSince).TotalSeconds -ge 3) {
                return
            }
        } else {
            $stableSince = $null
            foreach ($process in $running) {
                Stop-Process -Id $process.ProcessId -Force -ErrorAction SilentlyContinue
            }
        }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "YORVA processes did not remain stopped after Restart Manager handling"
}

function Assert-Sentinels {
    Assert-Condition ((Get-Content -LiteralPath (Join-Path $appDataRoot "sentinel-user.txt") -Raw) -eq "user-data") "YORVA user data was not preserved"
    Assert-Condition ((Get-Content -LiteralPath (Join-Path $backupRoot "sentinel-backup.txt") -Raw) -eq "encrypted-backup") "YORVA backup was not preserved"
    Assert-Condition ((Get-Content -LiteralPath (Join-Path $hermesRoot "profiles\sentinel\profile.txt") -Raw) -eq "hermes-profile") "Hermes Profile was not preserved"
}

function Start-AndVerifyRuntime {
    $process = Start-Process -FilePath $desktopPath -ArgumentList "--hidden" -PassThru -WindowStyle Hidden
    [void](Wait-ExactProcess $desktopPath 45)
    [void](Wait-ExactProcess $daemonPath 90)
    $logPath = Join-Path $appDataRoot "logs\install.ndjson"
    $deadline = [DateTime]::UtcNow.AddSeconds(90)
    do {
        if (Test-Path -LiteralPath $logPath -PathType Leaf) {
            $tail = Get-Content -LiteralPath $logPath -Tail 500 -ErrorAction SilentlyContinue
            if (($tail -match 'runtime discovery completed').Count -gt 0 -and ($tail -match 'startup Instance inventory reconciled').Count -gt 0) {
                return $process
            }
        }
        Start-Sleep -Milliseconds 500
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "authoritative Runtime/Profile reconciliation evidence is missing"
}

$groupOutput = whoami.exe /groups /fo csv /nh
Assert-Condition ($LASTEXITCODE -eq 0) "could not inspect the current Windows token"
Assert-Condition (($groupOutput -join "`n") -match 'S-1-16-8192') "installer lifecycle worker is not running at medium integrity"
Assert-Condition (($groupOutput -join "`n") -notmatch 'S-1-16-12288') "installer lifecycle worker is unexpectedly elevated"
Add-Pass "non-elevated-user-context"

Assert-Condition (@(Get-YorvaArpEntry).Count -eq 0) "disposable guest is not in a fresh YORVA state"
Assert-Condition (-not (Test-Path -LiteralPath $installRoot)) "YORVA program directory already exists"

$logRoot = Split-Path -Parent ([System.IO.Path]::GetFullPath($EvidencePath))
[System.IO.Directory]::CreateDirectory($logRoot) | Out-Null
Invoke-Msi -arguments @('/i', $baseline, '/qn', '/norestart', '/l*v', (Join-Path $logRoot 'fresh.log')) -name "fresh install"
$baselineEntry = Assert-Installed "0.3.2"
Add-Pass "fresh-install"

[System.IO.Directory]::CreateDirectory($backupRoot) | Out-Null
[System.IO.Directory]::CreateDirectory((Join-Path $hermesRoot "profiles\sentinel")) | Out-Null
[System.IO.Directory]::CreateDirectory((Join-Path $hermesRoot "bin")) | Out-Null
[System.IO.File]::WriteAllText((Join-Path $appDataRoot "sentinel-user.txt"), "user-data", [System.Text.UTF8Encoding]::new($false))
[System.IO.File]::WriteAllText((Join-Path $backupRoot "sentinel-backup.txt"), "encrypted-backup", [System.Text.UTF8Encoding]::new($false))
[System.IO.File]::WriteAllText((Join-Path $hermesRoot "profiles\sentinel\profile.txt"), "hermes-profile", [System.Text.UTF8Encoding]::new($false))
Copy-Item -LiteralPath $hermesFixture -Destination (Join-Path $hermesRoot "bin\hermes.exe") -Force
$runningBaseline = Start-AndVerifyRuntime
Set-ItemProperty -LiteralPath $runKey -Name "Yorva" -Value ('"{0}" --hidden' -f $desktopPath)

Invoke-Msi -arguments @('/i', $candidate, '/qn', '/norestart', '/l*v', (Join-Path $logRoot 'upgrade.log')) -name "running-process upgrade"
$candidateEntry = Assert-Installed "0.4.0"
Assert-Condition ($candidateEntry.ProductCode -ne $baselineEntry.ProductCode) "major upgrade retained the baseline ProductCode"
Assert-Sentinels
Add-Pass "running-process-upgrade"

Stop-YorvaProcesses
$candidateHash = (Get-FileHash -LiteralPath $desktopPath -Algorithm SHA256).Hash
Remove-Item -LiteralPath $desktopPath -Force
Assert-Condition (-not (Test-Path -LiteralPath $desktopPath)) "repair fixture could not remove the exact program file"
Invoke-Msi -arguments @('/fa', $candidateEntry.ProductCode, '/qn', '/norestart', '/l*v', (Join-Path $logRoot 'repair.log')) -name "repair"
[void](Assert-Installed "0.4.0")
Assert-Condition ((Get-FileHash -LiteralPath $desktopPath -Algorithm SHA256).Hash -eq $candidateHash) "repair did not restore the exact candidate Desktop file"
Assert-Sentinels
Add-Pass "repair"

Invoke-Msi -arguments @('/x', $candidateEntry.ProductCode, '/qn', '/norestart', '/l*v', (Join-Path $logRoot 'uninstall.log')) -name "uninstall"
Assert-Condition (@(Get-YorvaArpEntry).Count -eq 0) "uninstall entry remains after uninstall"
Assert-Condition (-not (Test-Path -LiteralPath $desktopPath)) "Desktop executable remains after uninstall"
Assert-Condition (-not (Test-Path -LiteralPath $daemonPath)) "daemon executable remains after uninstall"
Assert-Condition (-not (Test-Path -LiteralPath $startMenuShortcut)) "Start Menu shortcut remains after uninstall"
Assert-Condition (-not (Test-Path -LiteralPath $desktopShortcut)) "Desktop shortcut remains after uninstall"
$startup = (Get-ItemProperty -LiteralPath $runKey -Name "Yorva" -ErrorAction SilentlyContinue).Yorva
Assert-Condition ([string]::IsNullOrWhiteSpace([string]$startup)) "per-user login startup remains after uninstall"
Assert-Sentinels
Add-Pass "uninstall-preserves-data"

Invoke-Msi -arguments @('/i', $candidate, '/qn', '/norestart', '/l*v', (Join-Path $logRoot 'reinstall.log')) -name "reinstall"
[void](Assert-Installed "0.4.0")
Assert-Sentinels
$reinstalled = Start-AndVerifyRuntime
Add-Pass "reinstall-authoritative-recovery"
Stop-YorvaProcesses

$results["completedAtUtc"] = [DateTime]::UtcNow.ToString("O")
$results["baselineMsiSha256"] = (Get-FileHash -LiteralPath $baseline -Algorithm SHA256).Hash
$results["candidateMsiSha256"] = (Get-FileHash -LiteralPath $candidate -Algorithm SHA256).Hash
[System.IO.File]::WriteAllText(
    [System.IO.Path]::GetFullPath($EvidencePath),
    ($results | ConvertTo-Json -Depth 5),
    [System.Text.UTF8Encoding]::new($false)
)
Write-Output "Windows installer lifecycle smoke: PASS"
