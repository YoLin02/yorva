param(
    [Parameter(Mandatory = $true)]
    [string]$ReconnectPressureRoot,
    [Parameter(Mandatory = $true)]
    [string]$SingleDaemonContinuityRoot,
    [double]$MinimumWindowSeconds = 14400
)

$ErrorActionPreference = "Stop"

function Read-Qualification([string]$path, [string]$label) {
    $root = (Resolve-Path -LiteralPath $path).Path
    $summaryPath = Join-Path $root "summary.json"
    $samplePath = Join-Path $root "samples.ndjson"
    if (-not (Test-Path -LiteralPath $summaryPath -PathType Leaf) -or -not (Test-Path -LiteralPath $samplePath -PathType Leaf)) {
        throw "$label qualification is missing summary.json or samples.ndjson."
    }
    $summary = Get-Content -Raw -LiteralPath $summaryPath | ConvertFrom-Json
    $samples = @(Get-Content -LiteralPath $samplePath | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | ForEach-Object { $_ | ConvertFrom-Json })
    if ($summary.result -ne "PASS" -or [double]$summary.durationSeconds -lt $MinimumWindowSeconds) {
        throw "$label qualification did not pass the minimum duration."
    }
    if ([int]$summary.instanceCount -ne 3 -or [bool]$summary.hostRebooted) {
        throw "$label qualification did not preserve the three-Instance, no-host-reboot contract."
    }
    if ([int]$summary.cycles -lt 20 -or [int]$summary.operations -lt 100 -or [int]$summary.diagnostics -lt 1) {
        throw "$label qualification does not contain the required sustained workload coverage."
    }
    if ($samples.Count -lt 20 -or $samples.Count -ne [int]$summary.sampleCount) {
        throw "$label qualification does not contain a complete sample series."
    }
    if ([int]$samples[0].cycle -ne 0 -or [int]$samples[-1].cycle -ne [int]$summary.cycles) {
        throw "$label qualification sample boundaries do not match its summary."
    }
    return [pscustomobject]@{ Root = $root; Summary = $summary; Samples = $samples }
}

function Get-Median([double[]]$values) {
    if ($values.Count -eq 0) { throw "Cannot calculate a median for an empty series." }
    $ordered = @($values | Sort-Object)
    $middle = [Math]::Floor($ordered.Count / 2)
    if ($ordered.Count % 2 -eq 1) { return [double]$ordered[$middle] }
    return ([double]$ordered[$middle - 1] + [double]$ordered[$middle]) / 2
}

function Measure-ContinuityTrend([object[]]$samples, [string]$property, [double]$maximumMedianGrowth) {
    $window = [Math]::Max(5, [Math]::Floor($samples.Count / 5))
    $first = @($samples | Select-Object -First $window | ForEach-Object { [double]$_.$property })
    $last = @($samples | Select-Object -Last $window | ForEach-Object { [double]$_.$property })
    $firstMedian = Get-Median $first
    $lastMedian = Get-Median $last
    $growth = $lastMedian - $firstMedian
    if ($growth -gt $maximumMedianGrowth) {
        throw "Continuity $property median grew by $growth, above the allowed $maximumMedianGrowth."
    }
    return [pscustomobject]@{
        firstWindowMedian = $firstMedian
        lastWindowMedian = $lastMedian
        medianGrowth = $growth
        maximumAllowedGrowth = $maximumMedianGrowth
    }
}

$pressure = Read-Qualification $ReconnectPressureRoot "Reconnect-pressure"
$continuity = Read-Qualification $SingleDaemonContinuityRoot "Single-daemon continuity"

$pressurePids = @($pressure.Samples | ForEach-Object { [int]$_.daemonPid } | Sort-Object -Unique)
$continuityPids = @($continuity.Samples | ForEach-Object { [int]$_.daemonPid } | Sort-Object -Unique)
if ([int]$pressure.Summary.daemonReconnects -lt 3 -or $pressurePids.Count -lt 4) {
    throw "Reconnect-pressure did not prove at least three daemon replacements."
}
if ([int]$continuity.Summary.daemonReconnects -ne 0 -or $continuityPids.Count -ne 1) {
    throw "Single-daemon continuity did not retain exactly one daemon identity."
}

$totalDuration = [double]$pressure.Summary.durationSeconds + [double]$continuity.Summary.durationSeconds
if ($totalDuration -lt (2 * $MinimumWindowSeconds)) {
    throw "Combined qualification duration did not reach the two-window target."
}

$workingSet = Measure-ContinuityTrend $continuity.Samples "workingSetBytes" 67108864
$privateMemory = Measure-ContinuityTrend $continuity.Samples "privateMemoryBytes" 67108864
$handles = Measure-ContinuityTrend $continuity.Samples "handles" 128
$threads = Measure-ContinuityTrend $continuity.Samples "threads" 64

foreach ($qualification in @($pressure, $continuity)) {
    if ([int64]$qualification.Summary.peakWorkingSetBytes -gt 536870912 -or [int64]$qualification.Summary.peakPrivateMemoryBytes -gt 536870912) {
        throw "Qualification exceeded the 512 MiB process-memory ceiling."
    }
    if ([int]$qualification.Summary.peakHandles -gt 2048 -or [int]$qualification.Summary.peakThreads -gt 256) {
        throw "Qualification exceeded the handle or thread ceiling."
    }
    if ([int64]$qualification.Summary.peakDataBytes -gt 134217728 -or [int64]$qualification.Summary.peakWorkRootBytes -gt 268435456) {
        throw "Qualification data exceeded its bounded storage ceiling."
    }
}

$result = [ordered]@{
    result = "PASS"
    hostRebooted = $false
    combinedDurationSeconds = $totalDuration
    reconnectPressure = [ordered]@{
        cycles = [int]$pressure.Summary.cycles
        operations = [int]$pressure.Summary.operations
        daemonReconnects = [int]$pressure.Summary.daemonReconnects
        daemonPids = $pressurePids
    }
    singleDaemonContinuity = [ordered]@{
        cycles = [int]$continuity.Summary.cycles
        operations = [int]$continuity.Summary.operations
        daemonPid = $continuityPids[0]
        workingSet = $workingSet
        privateMemory = $privateMemory
        handles = $handles
        threads = $threads
    }
}

$result | ConvertTo-Json -Depth 8
