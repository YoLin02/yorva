param(
    [string]$MsiPath,
    [string]$ExtractedRoot,
    [string]$ExpectedVersion,
    [switch]$AsLibrary
)

$ErrorActionPreference = "Stop"
$script:InspectionComplete = $false

function Get-YorvaMsiPayloadCatalog {
    return @(
        @{ Name = "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip"; Size = 73798347; SHA256 = "4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54" },
        @{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836; SHA256 = "7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29" },
        @{ Name = "npm-12.0.2.tgz"; Size = 3045132; SHA256 = "5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1" },
        @{ Name = "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz"; Size = 25676832; SHA256 = "64A804111830C5329BFC5A4D95D6CBCBB377CAA2C02195101EDF85D15FC53099" },
        @{ Name = "LICENSE"; Size = 1070; SHA256 = "821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6" },
        @{ Name = "NODE-LICENSE"; Size = 148217; SHA256 = "8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576" },
        @{ Name = "NPM-LICENSE"; Size = 9742; SHA256 = "7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257" }
    )
}

function Get-YorvaMsiProductResourceCatalog {
    return @(
        @{ Name = "DATA_RETENTION.txt"; Size = 1313; SHA256 = "C8A330453B4E794D3B3407D9C5F9D2A0F2463F5711A1B9AE47C12FBB38FDCF0B" }
    )
}

function Get-MsiLongFileName([string]$raw) {
    if ([string]::IsNullOrWhiteSpace($raw)) {
        throw "MSI FileName is empty"
    }
    if ($raw -match '\|') {
        return ($raw -split '\|', 2)[1]
    }
    return $raw
}

function Read-MsiFileTable([string]$msi) {
    $installer = New-Object -ComObject WindowsInstaller.Installer
    $database = $installer.GetType().InvokeMember("OpenDatabase", "InvokeMethod", $null, $installer, @((Resolve-Path $msi).Path, 0))
    $view = $database.GetType().InvokeMember("OpenView", "InvokeMethod", $null, $database, @("SELECT FileName, FileSize FROM File"))
    $null = $view.GetType().InvokeMember("Execute", "InvokeMethod", $null, $view, $null)
    $rows = New-Object System.Collections.Generic.List[object]
    while ($true) {
        $record = $view.GetType().InvokeMember("Fetch", "InvokeMethod", $null, $view, $null)
        if ($null -eq $record) {
            break
        }
        $raw = [string]$record.GetType().InvokeMember("StringData", "GetProperty", $null, $record, 1)
        $sizeText = [string]$record.GetType().InvokeMember("StringData", "GetProperty", $null, $record, 2)
        if (-not $sizeText) {
            $sizeText = [string]$record.GetType().InvokeMember("IntegerData", "GetProperty", $null, $record, 2)
        }
        $name = Get-MsiLongFileName $raw
        if ([string]::IsNullOrWhiteSpace($name)) {
            throw "MSI File table row has an empty decoded name (raw='$raw')"
        }
        $rows.Add([pscustomobject]@{ RawName = $raw; Name = $name; Size = [int64]$sizeText })
    }
    return ,$rows.ToArray()
}

function Read-MsiQuery([string]$msi, [string]$query, [string[]]$columns) {
    $installer = New-Object -ComObject WindowsInstaller.Installer
    $database = $installer.GetType().InvokeMember("OpenDatabase", "InvokeMethod", $null, $installer, @((Resolve-Path $msi).Path, 0))
    $view = $database.GetType().InvokeMember("OpenView", "InvokeMethod", $null, $database, @($query))
    $null = $view.GetType().InvokeMember("Execute", "InvokeMethod", $null, $view, $null)
    $rows = [System.Collections.Generic.List[object]]::new()
    while ($true) {
        $record = $view.GetType().InvokeMember("Fetch", "InvokeMethod", $null, $view, $null)
        if ($null -eq $record) {
            break
        }
        $row = [ordered]@{}
        for ($index = 0; $index -lt $columns.Count; $index++) {
            $row[$columns[$index]] = [string]$record.GetType().InvokeMember("StringData", "GetProperty", $null, $record, $index + 1)
        }
        $rows.Add([pscustomobject]$row)
    }
    return ,$rows.ToArray()
}

function Assert-YorvaMsiLifecycleContract($properties, $directories, $registryRows, $removeRows, $upgradeRows, $customActions, [string]$expectedVersion) {
    $values = @{}
    foreach ($row in @($properties)) {
        $values[[string]$row.Property] = [string]$row.Value
    }
    $required = @{
        ProductName = "YORVA"
        Manufacturer = "yolin"
        UpgradeCode = "{E793918B-37EB-5E2F-B866-5FC4AA3AC75A}"
        REINSTALLMODE = "amus"
        WixUIRMOption = "UseRM"
        ARPCOMMENTS = "Uninstall removes YORVA program files, shortcuts and login startup. YORVA user data, encrypted backups, Hermes Runtime and Profiles are preserved."
    }
    foreach ($item in $required.GetEnumerator()) {
        if (-not $values.ContainsKey($item.Key) -or $values[$item.Key] -ne $item.Value) {
            throw "MSI property $($item.Key) does not match the YORVA lifecycle contract"
        }
    }
    if ($expectedVersion -and $values.ProductVersion -ne $expectedVersion) {
        throw "MSI ProductVersion $($values.ProductVersion) != $expectedVersion"
    }
    if ($values.ContainsKey("ALLUSERS") -and -not [string]::IsNullOrWhiteSpace($values.ALLUSERS)) {
        throw "MSI must use per-user installation without ALLUSERS"
    }
    if ($values.ContainsKey("ARPNOREPAIR")) {
        throw "MSI must expose Repair instead of setting ARPNOREPAIR"
    }

    $directoryByID = @{}
    foreach ($row in @($directories)) {
        $directoryByID[[string]$row.Directory] = $row
    }
    if (-not $directoryByID.ContainsKey("INSTALLDIR") -or $directoryByID.INSTALLDIR.Parent -ne "LocalProgramsFolder") {
        throw "MSI INSTALLDIR is not under the per-user local Programs folder"
    }
    if (-not $directoryByID.ContainsKey("LocalProgramsFolder") -or $directoryByID.LocalProgramsFolder.Parent -ne "LocalAppDataFolder") {
        throw "MSI local Programs folder is not rooted in LocalAppDataFolder"
    }

    foreach ($row in @($registryRows)) {
        if ([string]$row.Root -ne "1") {
            throw "MSI Registry table writes outside HKCU"
        }
    }
    $loginRegistration = @($registryRows | Where-Object {
        $_.Root -eq "1" -and
        $_.Key -eq "Software\Microsoft\Windows\CurrentVersion\Run" -and
        $_.Name -eq "Yorva" -and
        $_.Value -eq '"[INSTALLDIR]yorva-desktop.exe" --hidden'
    })
    if ($loginRegistration.Count -ne 1) {
        throw "MSI does not own the exact YORVA per-user login startup value"
    }
    foreach ($row in @($removeRows)) {
        if ([string]$row.Directory -in @("LocalAppDataFolder", "AppDataFolder")) {
            throw "MSI RemoveFile table targets a user data root"
        }
    }
    $expectedUpgradeCode = "{E793918B-37EB-5E2F-B866-5FC4AA3AC75A}"
    if (@($upgradeRows).Count -eq 0 -or @($upgradeRows | Where-Object { $_.UpgradeCode -ne $expectedUpgradeCode }).Count -ne 0) {
        throw "MSI Upgrade table does not exclusively use the stable YORVA UpgradeCode"
    }
    foreach ($action in @($customActions)) {
        $command = [string]$action.Target
        if ($command -match '(?i)Remove-Item|\brmdir\b|\bdel\s|%APPDATA%|\\hermes') {
            throw "MSI custom action contains a forbidden data-deletion command"
        }
    }
}

function Assert-ExactMsiIdentities($rows, $catalog) {
    $grouped = @{}
    foreach ($row in @($rows)) {
        $name = [string]$row.Name
        if ([string]::IsNullOrWhiteSpace($name)) {
            throw "MSI File table row has an empty name"
        }
        if (-not $grouped.ContainsKey($name)) {
            $grouped[$name] = @()
        }
        $grouped[$name] += $row
    }
    foreach ($item in $catalog) {
        if (-not $grouped.ContainsKey($item.Name)) {
            throw "MSI File table missing exact name $($item.Name)"
        }
        if ($grouped[$item.Name].Count -ne 1) {
            throw "MSI File table has $($grouped[$item.Name].Count) entries named $($item.Name)"
        }
        $actual = $grouped[$item.Name][0].Size
        if ($actual -ne $item.Size) {
            throw "MSI File table $($item.Name) size $actual != $($item.Size)"
        }
    }
}

function Resolve-ExactPayloadFile([string]$root, [string]$name) {
    $hits = @(Get-ChildItem -LiteralPath $root -Recurse -File -Force | Where-Object {
            $_.Name -eq $name -and (($_.DirectoryName -replace '\\', '/') -match '/hermes/source$')
        })
    if ($hits.Count -eq 0) {
        throw "extracted MSI is missing hermes/source/$name"
    }
    if ($hits.Count -ne 1) {
        throw "extracted MSI has $($hits.Count) hermes/source/$name entries"
    }
    return $hits[0]
}

function Assert-NoUnexpectedPayloads([string]$payloadDir, $catalog) {
    $allowed = @($catalog | ForEach-Object { $_.Name })
    foreach ($extra in Get-ChildItem -LiteralPath $payloadDir -Force) {
        if ($allowed -notcontains $extra.Name) {
            throw "unexpected extra resource in hermes/source: $($extra.Name)"
        }
    }
}

function Assert-ExtractedPayloads([string]$root, $catalog) {
    $payloadDir = $null
    foreach ($item in $catalog) {
        $file = Resolve-ExactPayloadFile $root $item.Name
        if ($file.Length -ne $item.Size) {
            throw "extracted $($item.Name) size $($file.Length) != $($item.Size)"
        }
        $hash = (Get-FileHash -Algorithm SHA256 -Path $file.FullName).Hash
        if ($hash -ne $item.SHA256) {
            throw "extracted $($item.Name) SHA-256 $hash != $($item.SHA256)"
        }
        Write-Host ("payload {0} {1} {2}" -f $item.Name, $file.Length, $hash)
        if ($null -eq $payloadDir) {
            $payloadDir = $file.Directory.FullName
            Assert-NoUnexpectedPayloads $payloadDir $catalog
        } elseif ($payloadDir -ne $file.Directory.FullName) {
            throw "payload $($item.Name) is not beside the other Hermes source payloads"
        }
    }
}

function Assert-ExtractedProductResources([string]$root, $catalog) {
    foreach ($item in $catalog) {
        $hits = @(Get-ChildItem -LiteralPath $root -Recurse -File -Force | Where-Object {
            $_.Name -eq $item.Name -and (($_.DirectoryName -replace '\\', '/') -match '/resources/installer$')
        })
        if ($hits.Count -ne 1) {
            throw "extracted MSI must contain exactly one resources/installer/$($item.Name)"
        }
        if ($hits[0].Length -ne $item.Size) {
            throw "extracted $($item.Name) size $($hits[0].Length) != $($item.Size)"
        }
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $hits[0].FullName).Hash
        if ($hash -ne $item.SHA256) {
            throw "extracted $($item.Name) SHA-256 $hash != $($item.SHA256)"
        }
        Write-Host ("product resource {0} {1} {2}" -f $item.Name, $hits[0].Length, $hash)
    }
}

function Invoke-MsiAdministrativeExtract([string]$msi, [string]$target) {
    New-Item -ItemType Directory -Force -Path $target | Out-Null
    $process = Start-Process -FilePath "$env:SystemRoot\System32\msiexec.exe" -ArgumentList @("/a", "`"$msi`"", "TARGETDIR=`"$target`"", "/qn") -Wait -PassThru -NoNewWindow
    if ($null -eq $process -or $process.ExitCode -ne 0) {
        $code = if ($null -eq $process) { "null" } else { $process.ExitCode }
        throw "msiexec administrative extract failed with exit $code"
    }
}

function Invoke-YorvaMsiInspection {
    param(
        [string]$MsiPath,
        [string]$ExtractedRoot,
        [string]$ExpectedVersion
    )
    $script:InspectionComplete = $false
    $catalog = Get-YorvaMsiPayloadCatalog
    $productCatalog = Get-YorvaMsiProductResourceCatalog
    $tempRoot = $null
    try {
        if ($ExtractedRoot) {
            if (-not (Test-Path -LiteralPath $ExtractedRoot)) {
                throw "extracted root not found: $ExtractedRoot"
            }
            Assert-ExtractedPayloads (Resolve-Path $ExtractedRoot).Path $catalog
            Assert-ExtractedProductResources (Resolve-Path $ExtractedRoot).Path $productCatalog
        } else {
            if (-not $MsiPath -or -not (Test-Path -LiteralPath $MsiPath)) {
                throw "MSI not found: $MsiPath"
            }
            $resolved = (Resolve-Path $MsiPath).Path
            $rows = Read-MsiFileTable $resolved
            Assert-ExactMsiIdentities $rows @($catalog + $productCatalog)
            $properties = Read-MsiQuery $resolved 'SELECT `Property`,`Value` FROM `Property`' @("Property", "Value")
            $directories = Read-MsiQuery $resolved 'SELECT `Directory`,`Directory_Parent`,`DefaultDir` FROM `Directory`' @("Directory", "Parent", "DefaultDir")
            $registryRows = Read-MsiQuery $resolved 'SELECT `Registry`,`Root`,`Key`,`Name`,`Value`,`Component_` FROM `Registry`' @("Registry", "Root", "Key", "Name", "Value", "Component")
            $removeRows = Read-MsiQuery $resolved 'SELECT `FileKey`,`Component_`,`FileName`,`DirProperty`,`InstallMode` FROM `RemoveFile`' @("FileKey", "Component", "FileName", "Directory", "InstallMode")
            $upgradeRows = Read-MsiQuery $resolved 'SELECT `UpgradeCode`,`VersionMin`,`VersionMax`,`Language`,`Attributes`,`Remove`,`ActionProperty` FROM `Upgrade`' @("UpgradeCode", "VersionMin", "VersionMax", "Language", "Attributes", "Remove", "ActionProperty")
            $customActions = Read-MsiQuery $resolved 'SELECT `Action`,`Type`,`Source`,`Target` FROM `CustomAction`' @("Action", "Type", "Source", "Target")
            Assert-YorvaMsiLifecycleContract $properties $directories $registryRows $removeRows $upgradeRows $customActions $ExpectedVersion
            $tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("yorva-msi-" + [guid]::NewGuid().ToString("N"))
            New-Item -ItemType Directory -Path $tempRoot | Out-Null
            $extractDir = Join-Path $tempRoot "extract"
            Invoke-MsiAdministrativeExtract $resolved $extractDir
            Assert-ExtractedPayloads $extractDir $catalog
            Assert-ExtractedProductResources $extractDir $productCatalog
            $msiHash = (Get-FileHash -Algorithm SHA256 -Path $resolved).Hash
            Write-Host ("MSI {0} SHA-256 {1} size {2}" -f (Split-Path $resolved -Leaf), $msiHash, (Get-Item $resolved).Length)
        }
        $script:InspectionComplete = $true
    }
    finally {
        if ($tempRoot -and (Test-Path -LiteralPath $tempRoot)) {
            Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    if (-not $script:InspectionComplete) {
        throw "MSI inspection did not complete"
    }
}

if (-not $AsLibrary) {
    Invoke-YorvaMsiInspection -MsiPath $MsiPath -ExtractedRoot $ExtractedRoot -ExpectedVersion $ExpectedVersion
}
