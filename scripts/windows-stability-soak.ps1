param(
    [string]$SidecarPath = (Join-Path $PSScriptRoot "..\apps\desktop\src-tauri\target\release\yorvad.exe"),
    [Parameter(Mandatory = $true)]
    [string]$HermesFixturePath,
    [string]$WorkRoot = (Join-Path $PSScriptRoot "..\.tools\p8-soak\current"),
    [double]$DurationMinutes = 240,
    [int]$CycleIntervalSeconds = 60,
    [int]$HeavyEveryCycles = 10,
    [int]$DiagnosticsEveryCycles = 5,
    [int]$DaemonRestartEveryCycles = 30
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Net.Http

if ($DurationMinutes -le 0 -or $CycleIntervalSeconds -lt 1 -or $HeavyEveryCycles -lt 1 -or $DiagnosticsEveryCycles -lt 1 -or $DaemonRestartEveryCycles -lt 1) {
    throw "Soak duration and intervals must be positive."
}

$sidecar = (Resolve-Path -LiteralPath $SidecarPath).Path
$fixture = (Resolve-Path -LiteralPath $HermesFixturePath).Path
$root = [IO.Path]::GetFullPath($WorkRoot)
if (Test-Path -LiteralPath $root) {
    throw "The soak WorkRoot already exists: $root"
}

$localAppData = Join-Path $root "local-app-data"
$roamingAppData = Join-Path $root "roaming-app-data"
$dataDir = Join-Path $root "yorva-data"
$hermesRoot = Join-Path $localAppData "hermes"
$hermesBin = Join-Path $hermesRoot "bin"
$diagnosticRoot = Join-Path $root "diagnostics"
$samplePath = Join-Path $root "samples.ndjson"
$summaryPath = Join-Path $root "summary.json"
$daemonStdoutPath = Join-Path $root "daemon-stdout.log"
$daemonStderrPath = Join-Path $root "daemon-stderr.ndjson"

foreach ($directory in @($hermesBin, (Join-Path $hermesRoot "skills"), $roamingAppData, $dataDir, $diagnosticRoot)) {
    [IO.Directory]::CreateDirectory($directory) | Out-Null
}
[IO.File]::WriteAllText((Join-Path $hermesRoot "config.yaml"), "{}`n", [Text.UTF8Encoding]::new($false))
Copy-Item -LiteralPath $fixture -Destination (Join-Path $hermesBin "hermes.exe")

$script:daemon = $null
$script:stdoutDrain = $null
$script:stderrDrain = $null
$script:token = ""
$script:baseUri = ""
$script:sessionTokens = [Collections.Generic.List[string]]::new()
$httpHandler = [Net.Http.HttpClientHandler]::new()
$httpHandler.UseProxy = $false
$script:http = [Net.Http.HttpClient]::new($httpHandler, $true)
$script:http.Timeout = [TimeSpan]::FromSeconds(45)
$secretMarker = "p8-soak-secret-marker"
$profileNames = @("default", "p8_soak_a", "p8_soak_b")
$instanceIds = @()
$samples = [Collections.Generic.List[object]]::new()
$startedAt = [DateTime]::UtcNow
$deadline = $startedAt.AddMinutes($DurationMinutes)
$cycle = 0
$operationCount = 0
$diagnosticCount = 0
$daemonRestartCount = 0

function New-Token {
    $bytes = [byte[]]::new(32)
    $generator = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $generator.GetBytes($bytes) } finally { $generator.Dispose() }
    return [Convert]::ToBase64String($bytes).TrimEnd("=").Replace("+", "-").Replace("/", "_")
}

function Write-ControlLine([Diagnostics.Process]$process, [string]$line) {
    $bytes = [Text.UTF8Encoding]::new($false).GetBytes($line + "`n")
    $process.StandardInput.BaseStream.Write($bytes, 0, $bytes.Length)
    $process.StandardInput.BaseStream.Flush()
}

function Save-DaemonOutput([string]$path, [string]$content, [int64]$maximumBytes) {
    if ([string]::IsNullOrEmpty($content)) { return }
    if ($content.Contains($secretMarker)) { throw "Daemon output leaked a qualification secret." }
    foreach ($sessionToken in $script:sessionTokens) {
        if ($content.Contains($sessionToken)) { throw "Daemon output leaked a bootstrap token." }
    }
    $encoding = [Text.UTF8Encoding]::new($false)
    $existingBytes = if (Test-Path -LiteralPath $path) { [int64](Get-Item -LiteralPath $path).Length } else { 0 }
    $newBytes = [int64]$encoding.GetByteCount($content)
    if ($existingBytes + $newBytes -gt $maximumBytes) { throw "Daemon output exceeded the qualification bound." }
    [IO.File]::AppendAllText($path, $content, $encoding)
}

function Start-SoakDaemon {
    $info = [Diagnostics.ProcessStartInfo]::new()
    $info.FileName = $sidecar
    $info.Arguments = "--bootstrap-stdio"
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.RedirectStandardInput = $true
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $utf8 = [Text.UTF8Encoding]::new($false)
    if ($null -ne $info.PSObject.Properties["StandardInputEncoding"]) {
        $info.StandardInputEncoding = $utf8
        $info.StandardOutputEncoding = $utf8
        $info.StandardErrorEncoding = $utf8
    }
    else {
        [Console]::InputEncoding = $utf8
    }
    $info.EnvironmentVariables["LOCALAPPDATA"] = $localAppData
    $info.EnvironmentVariables["APPDATA"] = $roamingAppData
    $info.EnvironmentVariables["PATH"] = "$env:SystemRoot\System32;$env:SystemRoot"

    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $info
    if (-not $process.Start()) { throw "Failed to start the soak daemon." }
    $script:daemon = $process
    $script:stderrDrain = $process.StandardError.ReadToEndAsync()
    $script:token = New-Token
    $script:sessionTokens.Add($script:token)
    $bootstrap = @{ protocolVersion = "1"; token = $script:token; dataDir = $dataDir } | ConvertTo-Json -Compress
    Write-ControlLine $process $bootstrap
    $handshakeTask = $process.StandardOutput.ReadLineAsync()
    if (-not $handshakeTask.Wait([TimeSpan]::FromSeconds(60))) {
        throw "Timed out waiting for the soak daemon handshake."
    }
    $line = $handshakeTask.Result
    if ([string]::IsNullOrWhiteSpace($line)) {
        throw "The soak daemon exited before its handshake."
    }
    $handshake = $line | ConvertFrom-Json
    if ($handshake.protocolVersion -ne "1" -or $handshake.pid -ne $process.Id -or $handshake.port -le 0) {
        throw "The soak daemon returned an invalid handshake."
    }
    # The daemon logs to stderr after the bootstrap handshake. Both redirected
    # streams must remain drained or the OS pipe can fill and stall an otherwise
    # completed Operation.
    $script:stdoutDrain = $process.StandardOutput.ReadToEndAsync()
    $script:baseUri = "http://127.0.0.1:$($handshake.port)"
    $health = Invoke-Api "GET" "/api/v1/health"
    if ($health.status -ne "ok") { throw "The soak daemon health check failed." }
}

function Stop-SoakDaemon {
    if ($null -eq $script:daemon) { return }
    if (-not $script:daemon.HasExited) {
        try { Write-ControlLine $script:daemon '{"type":"shutdown"}' } catch {}
        if (-not $script:daemon.WaitForExit(10000)) {
            $script:daemon.Kill()
            if (-not $script:daemon.WaitForExit(10000)) { throw "The owned soak daemon did not exit." }
        }
    }
    $stdout = if ($null -ne $script:stdoutDrain) { $script:stdoutDrain.GetAwaiter().GetResult() } else { "" }
    $stderr = if ($null -ne $script:stderrDrain) { $script:stderrDrain.GetAwaiter().GetResult() } else { "" }
    Save-DaemonOutput $daemonStdoutPath $stdout 65536
    Save-DaemonOutput $daemonStderrPath $stderr 2097152
    $script:daemon.Dispose()
    $script:daemon = $null
    $script:stdoutDrain = $null
    $script:stderrDrain = $null
    $script:token = ""
    $script:baseUri = ""
}

function Wait-NoOwnedHermesProcesses {
    $deadlineAt = [DateTime]::UtcNow.AddSeconds(10)
    $stableEmptySnapshots = 0
    do {
        $owned = @(Get-CimInstance Win32_Process -Filter "Name = 'hermes.exe'" -ErrorAction SilentlyContinue | Where-Object {
            -not [string]::IsNullOrWhiteSpace($_.ExecutablePath) -and
            [IO.Path]::GetFullPath($_.ExecutablePath).StartsWith($root + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)
        })
        if ($owned.Count -eq 0) {
            $stableEmptySnapshots++
            if ($stableEmptySnapshots -ge 2) { return }
        }
        else {
            $stableEmptySnapshots = 0
        }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $deadlineAt)
    throw "The soak left an owned Hermes fixture process running."
}

function Invoke-Api([string]$method, [string]$path, [object]$body = $null, [string]$idempotencyKey = "") {
    $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::new($method), $script:baseUri + $path)
    try {
        $request.Headers.Authorization = [Net.Http.Headers.AuthenticationHeaderValue]::new("Bearer", $script:token)
        if ($idempotencyKey -ne "") { [void]$request.Headers.TryAddWithoutValidation("Idempotency-Key", $idempotencyKey) }
        if ($null -ne $body) {
            $json = $body | ConvertTo-Json -Compress -Depth 10
            $request.Content = [Net.Http.StringContent]::new($json, [Text.Encoding]::UTF8, "application/json")
        }
        $response = $script:http.SendAsync($request).GetAwaiter().GetResult()
        try {
            $content = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            if (-not $response.IsSuccessStatusCode) {
                if ($content.Length -gt 2048) { $content = $content.Substring(0, 2048) }
                throw "API $method $path failed with $([int]$response.StatusCode): $content"
            }
            if ([string]::IsNullOrWhiteSpace($content)) { return $null }
            return $content | ConvertFrom-Json
        }
        finally { $response.Dispose() }
    }
    finally { $request.Dispose() }
}

function Export-Diagnostic([string]$path) {
    $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::Post, $script:baseUri + "/api/v1/diagnostics/bundle")
    try {
        $request.Headers.Authorization = [Net.Http.Headers.AuthenticationHeaderValue]::new("Bearer", $script:token)
        $response = $script:http.SendAsync($request).GetAwaiter().GetResult()
        try {
            if (-not $response.IsSuccessStatusCode) { throw "Diagnostic export failed with $([int]$response.StatusCode)." }
            $bytes = $response.Content.ReadAsByteArrayAsync().GetAwaiter().GetResult()
            [IO.File]::WriteAllBytes($path, $bytes)
        }
        finally { $response.Dispose() }
    }
    finally { $request.Dispose() }

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [IO.Compression.ZipFile]::OpenRead($path)
    try {
        $names = @($archive.Entries | ForEach-Object FullName)
        foreach ($required in @("manifest.json", "node-summary.json", "runtime-summary.json", "instance-summary.json", "operations.json", "schema.json", "redaction-report.json")) {
            if ($names -notcontains $required) { throw "Diagnostic bundle is missing $required." }
        }
        foreach ($entry in $archive.Entries) {
            if ($entry.Length -gt 2097152) { throw "Diagnostic entry exceeds the qualification bound." }
            $stream = $entry.Open()
            $reader = [IO.StreamReader]::new($stream, [Text.Encoding]::UTF8, $true, 4096, $false)
            try {
                $text = $reader.ReadToEnd()
                $containsSessionToken = $false
                foreach ($sessionToken in $script:sessionTokens) {
                    if ($text.Contains($sessionToken)) {
                        $containsSessionToken = $true
                        break
                    }
                }
                if ($text.Contains($secretMarker) -or $containsSessionToken) {
                    throw "Diagnostic bundle leaked a qualification secret."
                }
            }
            finally { $reader.Dispose(); $stream.Dispose() }
        }
    }
    finally { $archive.Dispose() }
}

function New-IdempotencyKey([string]$prefix) {
    return "$prefix-$([Guid]::NewGuid().ToString('N'))"
}

function Start-Operation([string]$method, [string]$path, [object]$body, [string]$prefix) {
    $operation = Invoke-Api $method $path $body (New-IdempotencyKey $prefix)
    if ([string]::IsNullOrWhiteSpace([string]$operation.id)) { throw "Operation response did not contain an id." }
    $script:operationCount++
    return $operation
}

function Wait-Operation([object]$operation, [int]$timeoutSeconds = 120) {
    $deadlineAt = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
    do {
        $current = Invoke-Api "GET" ("/api/v1/operations/" + [Uri]::EscapeDataString([string]$operation.id))
        if ($current.status -eq "SUCCEEDED") { return $current }
        if ($current.status -in @("FAILED", "CANCELLED")) {
            throw "Operation $($current.id) ended as $($current.status) with $($current.errorCode)."
        }
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadlineAt)
    throw "Operation $($operation.id) timed out."
}

function Wait-Operations([object[]]$operations, [int]$timeoutSeconds = 120) {
    foreach ($operation in $operations) { [void](Wait-Operation $operation $timeoutSeconds) }
}

function Get-Instances {
    return @((Invoke-Api "GET" "/api/v1/runtimes/hermes/instances").instances)
}

function Require-ProfileInstances {
    $instances = Get-Instances
    $resolved = @()
    foreach ($name in $profileNames) {
        $match = @($instances | Where-Object name -eq $name)
        if ($match.Count -ne 1 -or $match[0].availability -ne "AVAILABLE") {
            $observed = ($instances | Select-Object instanceId, name, availability | ConvertTo-Json -Compress)
            throw "Profile $name did not reconcile to one available Instance. Observed: $observed"
        }
        $resolved += [string]$match[0].instanceId
    }
    return $resolved
}

function Invoke-LifecycleWave([string]$verb) {
    $operations = @()
    foreach ($instanceId in $instanceIds) {
        $escaped = [Uri]::EscapeDataString($instanceId)
        $operations += Start-Operation "POST" "/api/v1/instances/$escaped/$verb" @{} "p8-$verb"
    }
    Wait-Operations $operations
    $expected = if ($verb -eq "stop") { "STOPPED" } else { "RUNNING" }
    foreach ($instanceId in $instanceIds) {
        $state = Invoke-Api "GET" ("/api/v1/instances/" + [Uri]::EscapeDataString($instanceId) + "/lifecycle")
        if ($state.state -ne $expected) { throw "Lifecycle $verb did not read back as $expected." }
    }
}

function Invoke-SkillLifecycle([string]$instanceId) {
    $escaped = [Uri]::EscapeDataString($instanceId)
    $skill = "yorva-managed-demo"
    [void](Wait-Operation (Start-Operation "POST" "/api/v1/instances/$escaped/skills/$skill/install" @{ sourceId = "yorva-demo" } "p8-skill-install"))
    [void](Wait-Operation (Start-Operation "POST" "/api/v1/instances/$escaped/skills/$skill/update" @{} "p8-skill-update"))
    [void](Wait-Operation (Start-Operation "POST" "/api/v1/instances/$escaped/skills/$skill/disable" @{} "p8-skill-disable"))
    [void](Wait-Operation (Start-Operation "POST" "/api/v1/instances/$escaped/skills/$skill/enable" @{} "p8-skill-enable"))
    $items = @((Invoke-Api "GET" "/api/v1/instances/$escaped/skills").items)
    $managed = @($items | Where-Object id -eq $skill)
    if ($managed.Count -ne 1 -or $managed[0].ownership -ne "YORVA_MANAGED" -or $managed[0].enabledState -ne "ENABLED") {
        throw "Managed Skill lifecycle did not authoritatively read back."
    }
    [void](Wait-Operation (Start-Operation "DELETE" "/api/v1/instances/$escaped/skills/$skill" @{} "p8-skill-remove"))
    $remaining = @((Invoke-Api "GET" "/api/v1/instances/$escaped/skills").items | Where-Object id -eq $skill)
    if ($remaining.Count -ne 0) { throw "Managed Skill remained after removal." }
}

function Invoke-MCPLifecycle([string]$instanceId) {
    $escaped = [Uri]::EscapeDataString($instanceId)
    $server = "yorva-mcp-test"
    [void](Wait-Operation (Start-Operation "PUT" "/api/v1/instances/$escaped/mcp-bindings/$server" @{ enabledToolIds = @("yorva_ping") } "p8-mcp-bind"))
    [void](Wait-Operation (Start-Operation "POST" "/api/v1/instances/$escaped/mcp-bindings/$server/test" @{} "p8-mcp-test"))
    $items = @((Invoke-Api "GET" "/api/v1/instances/$escaped/mcp-bindings").items | Where-Object id -eq $server)
    if ($items.Count -ne 1 -or $items[0].state -ne "READY" -or $items[0].ownership -ne "YORVA_MANAGED") {
        throw "MCP lifecycle did not authoritatively read back as READY. Observed: $($items | ConvertTo-Json -Compress)"
    }
    [void](Wait-Operation (Start-Operation "DELETE" "/api/v1/instances/$escaped/mcp-bindings/$server" @{} "p8-mcp-unbind"))
    $remaining = @((Invoke-Api "GET" "/api/v1/instances/$escaped/mcp-bindings").items | Where-Object id -eq $server)
    if ($remaining.Count -ne 0) { throw "MCP binding remained after removal." }
}

function Invoke-BackupLifecycle {
    $before = @((Invoke-Api "GET" "/api/v1/runtimes/hermes/backups").items | ForEach-Object backupId)
    [void](Wait-Operation (Start-Operation "POST" "/api/v1/runtimes/hermes/backups" @{} "p8-backup-create") 600)
    $after = @((Invoke-Api "GET" "/api/v1/runtimes/hermes/backups").items)
    $created = @($after | Where-Object { $before -notcontains $_.backupId })
    if ($created.Count -ne 1 -or $created[0].state -ne "AVAILABLE" -or $created[0].sizeBytes -le 0) {
        throw "Runtime backup did not create and verify exactly once."
    }
    $backupId = [Uri]::EscapeDataString([string]$created[0].backupId)
    $readback = Invoke-Api "GET" "/api/v1/runtimes/hermes/backups/$backupId"
    if ($readback.state -ne "AVAILABLE" -or $readback.checksumSha256 -ne $created[0].checksumSha256) {
        throw "Runtime backup authoritative readback failed."
    }
    [void](Wait-Operation (Start-Operation "DELETE" "/api/v1/backups/$backupId" @{} "p8-backup-delete") 180)
    $remaining = @((Invoke-Api "GET" "/api/v1/runtimes/hermes/backups").items | Where-Object backupId -eq $readback.backupId)
    if ($remaining.Count -ne 0) { throw "Runtime backup remained after deletion." }
}

function Add-Sample([int]$cycleNumber) {
    $process = Get-Process -Id $script:daemon.Id -ErrorAction Stop
    $dbPath = Join-Path $dataDir "yorva.db"
    $dataBytes = (Get-ChildItem -LiteralPath $dataDir -Recurse -File -Force -ErrorAction SilentlyContinue | Measure-Object Length -Sum).Sum
    if ($null -eq $dataBytes) { $dataBytes = 0 }
    $workRootBytes = (Get-ChildItem -LiteralPath $root -Recurse -File -Force -ErrorAction SilentlyContinue | Measure-Object Length -Sum).Sum
    if ($null -eq $workRootBytes) { $workRootBytes = 0 }
    $sample = [ordered]@{
        at = [DateTime]::UtcNow.ToString("o")
        cycle = $cycleNumber
        daemonPid = $process.Id
        workingSetBytes = [int64]$process.WorkingSet64
        privateMemoryBytes = [int64]$process.PrivateMemorySize64
        handles = [int]$process.HandleCount
        threads = [int]$process.Threads.Count
        cpuSeconds = [double]$process.CPU
        databaseBytes = if (Test-Path -LiteralPath $dbPath) { [int64](Get-Item -LiteralPath $dbPath).Length } else { 0 }
        dataBytes = [int64]$dataBytes
        workRootBytes = [int64]$workRootBytes
        operations = $script:operationCount
        diagnostics = $script:diagnosticCount
    }
    $script:samples.Add([pscustomobject]$sample)
    Add-Content -LiteralPath $samplePath -Value ($sample | ConvertTo-Json -Compress) -Encoding UTF8
}

function Test-SustainedGrowth([object[]]$values, [double]$minimumDelta) {
    if ($values.Count -lt 8) { return $false }
    $tail = @($values | Select-Object -Last 8)
    if (($tail[-1] - $tail[0]) -lt $minimumDelta) { return $false }
    for ($index = 1; $index -lt $tail.Count; $index++) {
        if ($tail[$index] -lt $tail[$index - 1]) { return $false }
    }
    return $true
}

try {
    Start-SoakDaemon
    foreach ($name in $profileNames | Where-Object { $_ -ne "default" }) {
        $existing = @(Get-Instances | Where-Object name -eq $name)
        if ($existing.Count -eq 0) {
            [void](Wait-Operation (Start-Operation "POST" "/api/v1/runtimes/hermes/instances" @{ name = $name } "p8-instance-create"))
        }
    }
    $instanceIds = Require-ProfileInstances

    foreach ($instanceId in $instanceIds) {
        $escaped = [Uri]::EscapeDataString($instanceId)
        $model = "deepseek-chat"
        $configured = Invoke-Api "PUT" "/api/v1/instances/$escaped/credentials/model-provider" @{
            providerPresetId = "deepseek"; modelId = $model; selectedModelIds = @($model); value = $secretMarker
        }
        if ($configured.state -ne "CONFIGURED" -or -not $configured.credentialConfigured) { throw "Model configuration write did not complete." }
        $readback = Invoke-Api "GET" "/api/v1/instances/$escaped/config"
        if ($readback.providerPresetId -ne "deepseek" -or $readback.modelId -ne $model -or -not $readback.credentialConfigured) {
            throw "Model configuration authoritative readback failed."
        }
    }

    Add-Sample 0
    while ([DateTime]::UtcNow -lt $deadline) {
        $cycle++
        Invoke-LifecycleWave "start"
        Invoke-LifecycleWave "restart"
        foreach ($instanceId in $instanceIds) {
            $escaped = [Uri]::EscapeDataString($instanceId)
            $channels = @((Invoke-Api "GET" "/api/v1/instances/$escaped/channels").channels)
            if ($channels.Count -ne 2) { throw "Channel status did not return the closed two-channel set." }
            $model = Invoke-Api "GET" "/api/v1/instances/$escaped/config"
            if ($model.state -ne "CONFIGURED" -or $model.modelId -ne "deepseek-chat") { throw "Model readback drifted during soak." }
        }
        if ($cycle % $HeavyEveryCycles -eq 0 -or $cycle -eq 1) {
            Invoke-SkillLifecycle $instanceIds[0]
            Invoke-MCPLifecycle $instanceIds[1]
        }
        Invoke-LifecycleWave "stop"
        if ($cycle % $HeavyEveryCycles -eq 0 -or $cycle -eq 1) { Invoke-BackupLifecycle }
        if ($cycle % $DiagnosticsEveryCycles -eq 0 -or $cycle -eq 1) {
            $script:diagnosticCount++
            Export-Diagnostic (Join-Path $diagnosticRoot ("diagnostic-{0:D5}.zip" -f $script:diagnosticCount))
        }
        Add-Sample $cycle
        if ($cycle % $DaemonRestartEveryCycles -eq 0) {
            Stop-SoakDaemon
            Start-SoakDaemon
            $script:daemonRestartCount++
            $instanceIds = Require-ProfileInstances
        }
        $remaining = ($deadline - [DateTime]::UtcNow).TotalSeconds
        if ($remaining -gt 0) { Start-Sleep -Seconds ([Math]::Min($CycleIntervalSeconds, [Math]::Ceiling($remaining))) }
    }

    Invoke-LifecycleWave "stop"
    $active = @((Invoke-Api "GET" "/api/v1/operations").operations | Where-Object status -in @("PENDING", "RUNNING"))
    if ($active.Count -ne 0) { throw "The soak ended with stale active Operations." }
    $instanceIds = Require-ProfileInstances
    if (Test-SustainedGrowth @($samples | ForEach-Object workingSetBytes) 268435456) { throw "Daemon working set shows sustained unbounded growth." }
    if (Test-SustainedGrowth @($samples | ForEach-Object privateMemoryBytes) 268435456) { throw "Daemon private memory shows sustained unbounded growth." }
    if (Test-SustainedGrowth @($samples | ForEach-Object handles) 200) { throw "Daemon handle count shows sustained unbounded growth." }
    if (Test-SustainedGrowth @($samples | ForEach-Object threads) 100) { throw "Daemon thread count shows sustained unbounded growth." }
    if ($samples[-1].dataBytes -gt 134217728) { throw "Soak product data exceeded the 128 MiB qualification bound." }
    if ($samples[-1].workRootBytes -gt 268435456) { throw "Soak work-root data exceeded the 256 MiB qualification bound." }

    # PASS is durable only after the owned daemon has exited, all redirected
    # output has been drained/scanned, and no fixture child remains.
    Stop-SoakDaemon
    Wait-NoOwnedHermesProcesses

    $finishedAt = [DateTime]::UtcNow
    $summary = [ordered]@{
        result = "PASS"
        startedAt = $startedAt.ToString("o")
        finishedAt = $finishedAt.ToString("o")
        durationSeconds = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
        cycles = $cycle
        instanceCount = $instanceIds.Count
        operations = $operationCount
        diagnostics = $diagnosticCount
        daemonReconnects = $daemonRestartCount
        hostRebooted = $false
        sampleCount = $samples.Count
        peakWorkingSetBytes = [int64](($samples | Measure-Object workingSetBytes -Maximum).Maximum)
        peakPrivateMemoryBytes = [int64](($samples | Measure-Object privateMemoryBytes -Maximum).Maximum)
        peakHandles = [int](($samples | Measure-Object handles -Maximum).Maximum)
        peakThreads = [int](($samples | Measure-Object threads -Maximum).Maximum)
        peakDatabaseBytes = [int64](($samples | Measure-Object databaseBytes -Maximum).Maximum)
        peakDataBytes = [int64](($samples | Measure-Object dataBytes -Maximum).Maximum)
        peakWorkRootBytes = [int64](($samples | Measure-Object workRootBytes -Maximum).Maximum)
        daemonStdoutBytes = if (Test-Path -LiteralPath $daemonStdoutPath) { [int64](Get-Item -LiteralPath $daemonStdoutPath).Length } else { 0 }
        daemonStderrBytes = if (Test-Path -LiteralPath $daemonStderrPath) { [int64](Get-Item -LiteralPath $daemonStderrPath).Length } else { 0 }
        firstSample = $samples[0]
        lastSample = $samples[-1]
    }
    [IO.File]::WriteAllText($summaryPath, ($summary | ConvertTo-Json -Depth 8), [Text.UTF8Encoding]::new($false))
    Write-Output "Windows stability soak: PASS"
    Write-Output "Summary: $summaryPath"
}
catch {
    $failure = [ordered]@{
        result = "FAIL"
        startedAt = $startedAt.ToString("o")
        failedAt = [DateTime]::UtcNow.ToString("o")
        cycles = $cycle
        operations = $operationCount
        diagnostics = $diagnosticCount
        hostRebooted = $false
        error = $_.Exception.Message
    }
    [IO.File]::WriteAllText($summaryPath, ($failure | ConvertTo-Json -Depth 5), [Text.UTF8Encoding]::new($false))
    throw
}
finally {
    Stop-SoakDaemon
    $script:http.Dispose()
}
