param(
    [Parameter(Mandatory = $true)]
    [string]$OutputDirectory
)

$ErrorActionPreference = "Stop"
$outputRoot = [IO.Path]::GetFullPath($OutputDirectory)
[IO.Directory]::CreateDirectory($outputRoot) | Out-Null
$wixRoot = Join-Path $env:LOCALAPPDATA "tauri\WixTools314"
$candle = Join-Path $wixRoot "candle.exe"
$light = Join-Path $wixRoot "light.exe"
if (-not (Test-Path -LiteralPath $candle -PathType Leaf) -or -not (Test-Path -LiteralPath $light -PathType Leaf)) {
    throw "reviewed WiX 3.14 tools are unavailable"
}

$source = Join-Path $outputRoot "failure-marker.txt"
$wxs = Join-Path $outputRoot "failure.wxs"
$object = Join-Path $outputRoot "failure.wixobj"
$msi = Join-Path $outputRoot "YORVA_0.4.0_x64_en-US.msi"
[IO.File]::WriteAllText($source, "qualification-only failure fixture", [Text.UTF8Encoding]::new($false))
$productCode = ([guid]::NewGuid().ToString("B")).ToUpperInvariant()
$componentCode = ([guid]::NewGuid().ToString("B")).ToUpperInvariant()
$xml = @"
<?xml version="1.0" encoding="utf-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
  <Product Id="$productCode" Name="YORVA" Language="1033" Version="0.4.0" Manufacturer="yolin" UpgradeCode="{E793918B-37EB-5E2F-B866-5FC4AA3AC75A}">
    <Package InstallerVersion="450" Compressed="yes" InstallScope="perUser" InstallPrivileges="limited" />
    <MajorUpgrade Schedule="afterInstallInitialize" DowngradeErrorMessage="A newer YORVA is installed." />
    <Media Id="1" Cabinet="fixture.cab" EmbedCab="yes" />
    <Directory Id="TARGETDIR" Name="SourceDir">
      <Directory Id="LocalAppDataFolder">
        <Directory Id="LocalProgramsFolder" Name="Programs">
          <Directory Id="INSTALLDIR" Name="YORVA">
            <Component Id="FailureMarker" Guid="$componentCode" Win64="yes">
              <File Id="FailureMarkerFile" Source="$source" KeyPath="yes" />
            </Component>
          </Directory>
        </Directory>
      </Directory>
    </Directory>
    <CustomAction Id="AlwaysFail" Error="Intentional disposable-VM updater failure." />
    <InstallExecuteSequence>
      <Custom Action="AlwaysFail" Before="RemoveExistingProducts">1</Custom>
    </InstallExecuteSequence>
    <Feature Id="Main" Level="1"><ComponentRef Id="FailureMarker" /></Feature>
  </Product>
</Wix>
"@
[IO.File]::WriteAllText($wxs, $xml, [Text.UTF8Encoding]::new($false))
& $candle -nologo -arch x64 -out $object $wxs
if ($LASTEXITCODE -ne 0) { throw "failure-fixture candle failed" }
& $light -nologo -sice:ICE38 -sice:ICE64 -out $msi $object
if ($LASTEXITCODE -ne 0) { throw "failure-fixture light failed" }

$manifest = [ordered]@{
    schemaVersion = 1
    productVersion = "0.4.0"
    sourceCommit = (git rev-parse HEAD).Trim()
    workingTreeDirty = $true
    qualificationFixture = $true
    qualificationBuild = $false
    fileName = [IO.Path]::GetFileName($msi)
    sizeBytes = (Get-Item -LiteralPath $msi).Length
    sha256 = (Get-FileHash -LiteralPath $msi -Algorithm SHA256).Hash
    signatureStatus = "NotSigned"
}
[IO.File]::WriteAllText((Join-Path $outputRoot "artifact.json"), ($manifest | ConvertTo-Json -Depth 3), [Text.UTF8Encoding]::new($false))
Write-Output "qualification failure MSI $msi"
