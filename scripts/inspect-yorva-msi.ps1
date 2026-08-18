param(
    [string]$MsiPath,
    [string]$ExtractedRoot,
    [switch]$AsLibrary
)

$ErrorActionPreference = "Stop"
$script:InspectionComplete = $false

function Get-YorvaMsiPayloadCatalog {
    return @(
        @{ Name = "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip"; Size = 71869305; SHA256 = "2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA" },
        @{ Name = "node-v22.23.1-win-x64.zip"; Size = 35682836; SHA256 = "7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29" },
        @{ Name = "npm-12.0.2.tgz"; Size = 3045132; SHA256 = "5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1" },
        @{ Name = "LICENSE"; Size = 1070; SHA256 = "821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6" },
        @{ Name = "NODE-LICENSE"; Size = 148217; SHA256 = "8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576" },
        @{ Name = "NPM-LICENSE"; Size = 9742; SHA256 = "7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257" }
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
    $view.GetType().InvokeMember("Execute", "InvokeMethod", $null, $view, $null)
    $rows = @()
    while ($true) {
        $record = $view.GetType().InvokeMember("Fetch", "InvokeMethod", $null, $view, $null)
        if ($null -eq $record) {
            break
        }
        $raw = $record.GetType().InvokeMember("StringData", "GetProperty", $null, $record, 1)
        $size = [int64]$record.GetType().InvokeMember("IntegerData", "GetProperty", $null, $record, 2)
        $rows += [pscustomobject]@{ RawName = $raw; Name = (Get-MsiLongFileName $raw); Size = $size }
    }
    return $rows
}

function Assert-ExactMsiIdentities($rows, $catalog) {
    $grouped = @{}
    foreach ($row in $rows) {
        if (-not $grouped.ContainsKey($row.Name)) {
            $grouped[$row.Name] = @()
        }
        $grouped[$row.Name] += $row
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

function Invoke-MsiAdministrativeExtract([string]$msi, [string]$target) {
    New-Item -ItemType Directory -Force -Path $target | Out-Null
    $process = Start-Process -FilePath "$env:SystemRoot\System32\msiexec.exe" -ArgumentList @("/a", $msi, "TARGETDIR=$target", "/qn") -Wait -PassThru -NoNewWindow
    if ($null -eq $process -or $process.ExitCode -ne 0) {
        $code = if ($null -eq $process) { "null" } else { $process.ExitCode }
        throw "msiexec administrative extract failed with exit $code"
    }
}

function Invoke-YorvaMsiInspection {
    param(
        [string]$MsiPath,
        [string]$ExtractedRoot
    )
    $script:InspectionComplete = $false
    $catalog = Get-YorvaMsiPayloadCatalog
    $tempRoot = $null
    try {
        if ($ExtractedRoot) {
            if (-not (Test-Path -LiteralPath $ExtractedRoot)) {
                throw "extracted root not found: $ExtractedRoot"
            }
            Assert-ExtractedPayloads (Resolve-Path $ExtractedRoot).Path $catalog
        } else {
            if (-not $MsiPath -or -not (Test-Path -LiteralPath $MsiPath)) {
                throw "MSI not found: $MsiPath"
            }
            $resolved = (Resolve-Path $MsiPath).Path
            $rows = Read-MsiFileTable $resolved
            Assert-ExactMsiIdentities $rows $catalog
            $tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("yorva-msi-" + [guid]::NewGuid().ToString("N"))
            New-Item -ItemType Directory -Path $tempRoot | Out-Null
            $extractDir = Join-Path $tempRoot "extract"
            Invoke-MsiAdministrativeExtract $resolved $extractDir
            Assert-ExtractedPayloads $extractDir $catalog
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
    Invoke-YorvaMsiInspection -MsiPath $MsiPath -ExtractedRoot $ExtractedRoot
}
