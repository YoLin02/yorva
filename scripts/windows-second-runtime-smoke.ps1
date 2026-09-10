param(
    [Parameter(Mandatory = $true)][string]$SidecarPath,
    [Parameter(Mandatory = $true)][string]$WorkRoot,
    [switch]$DisposableUserProfile
)
$ErrorActionPreference = 'Stop'
if (-not $DisposableUserProfile) { throw 'Run only in a disposable Windows user or VM with both real Runtimes installed.' }
$p9Groups = (whoami.exe /groups) -join "`n"
if ($p9Groups -notmatch 'S-1-16-8192' -or $p9Groups -match 'S-1-16-12288') { throw 'Medium integrity is required.' }
$p9Root = [IO.Path]::GetFullPath($WorkRoot)
if (Test-Path -LiteralPath $p9Root) { throw 'The smoke WorkRoot must be new.' }
[IO.Directory]::CreateDirectory($p9Root) | Out-Null
$p9Sidecar = (Resolve-Path -LiteralPath $SidecarPath).Path
$script:p9Process = $null
$script:p9Base = ''
$script:p9Headers = @{}
$script:p9DaemonLog = $null

function Say([string]$message) { Write-Output "P9_G1 $message" }
function Start-P9Daemon {
    $info = [Diagnostics.ProcessStartInfo]::new()
    $info.FileName = $p9Sidecar
    $info.Arguments = '--bootstrap-stdio'
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.RedirectStandardInput = $true
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $script:p9Process = [Diagnostics.Process]::new()
    $script:p9Process.StartInfo = $info
    if (-not $script:p9Process.Start()) { throw 'daemon start failed' }
    $script:p9DaemonLog = $script:p9Process.StandardError.ReadToEndAsync()
    $random = [byte[]]::new(32)
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($random) } finally { $rng.Dispose() }
    $token = [Convert]::ToBase64String($random).TrimEnd('=').Replace('+','-').Replace('/','_')
    $script:p9Headers = @{ Authorization = "Bearer $token" }
    $bootstrap = @{protocolVersion='1';token=$token;dataDir=(Join-Path $p9Root 'data')} | ConvertTo-Json -Compress
    $script:p9Process.StandardInput.WriteLine($bootstrap)
    $script:p9Process.StandardInput.Flush()
    $handshake = $script:p9Process.StandardOutput.ReadLineAsync()
    if (-not $handshake.Wait([TimeSpan]::FromSeconds(45))) { throw 'daemon handshake exceeded Desktop deadline' }
    $ready = $handshake.Result | ConvertFrom-Json
    if ($ready.protocolVersion -ne '1' -or $ready.pid -ne $script:p9Process.Id -or $ready.port -le 0) { throw 'invalid daemon handshake' }
    $script:p9Base = "http://127.0.0.1:$($ready.port)/api/v1"
}
function Stop-P9Daemon {
    if ($script:p9Process -and -not $script:p9Process.HasExited) {
        $script:p9Process.StandardInput.WriteLine('{"type":"shutdown"}')
        $script:p9Process.StandardInput.Flush()
        if (-not $script:p9Process.WaitForExit(10000)) { $script:p9Process.Kill(); throw 'daemon shutdown timeout' }
        if ($script:p9Process.ExitCode -ne 0) { throw 'daemon exited unsuccessfully' }
    }
    if ($script:p9DaemonLog) { [IO.File]::WriteAllText((Join-Path $p9Root 'daemon.log'), $script:p9DaemonLog.Result) }
    if ($script:p9Process) { $script:p9Process.Dispose(); $script:p9Process = $null }
}
function Request-P9([string]$method, [string]$path, $body = $null, [string]$key = '') {
    $headers = @{} + $script:p9Headers
    if ($key) { $headers['Idempotency-Key'] = $key }
    $args = @{ Method=$method; Uri=($script:p9Base+$path); Headers=$headers; TimeoutSec=100 }
    if ($null -ne $body) { $args.ContentType='application/json'; $args.Body=($body | ConvertTo-Json -Depth 12 -Compress) }
    return Invoke-RestMethod @args
}
function Wait-P9Operation($operation, [string]$label) {
    $deadline = [DateTime]::UtcNow.AddSeconds(280)
    $current = $operation
    while ($current.status -in @('PENDING','RUNNING')) {
        if ([DateTime]::UtcNow -gt $deadline) { throw "$label operation timeout" }
        Start-Sleep -Seconds 1
        $current = Request-P9 'GET' "/operations/$($operation.id)"
    }
    Say "$label status=$($current.status) code=$($current.errorCode)"
    if ($current.status -ne 'SUCCEEDED') { throw "$label operation failed: $($current.errorCode)" }
}
function Lifecycle-P9($item, [string]$action) {
    $op = Request-P9 'POST' "/instances/$($item.instanceId)/$action" @{} ([Guid]::NewGuid().ToString('N'))
    Wait-P9Operation $op "$($item.name) $action"
}
function Assert-P9State($item, [string]$state) {
    $actual = Request-P9 'GET' "/instances/$($item.instanceId)/lifecycle"
    if ($actual.state -ne $state -or $actual.errorCode) { throw "$($item.name) state=$($actual.state) code=$($actual.errorCode), expected=$state" }
}
function Assert-P9CapabilityRejected([string]$path) {
    try { Request-P9 'GET' $path | Out-Null; throw 'unsupported capability was accepted' }
    catch [Net.WebException] {
        $json = $_.ErrorDetails.Message
        if (-not $json) {
            $reader = [IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
            try { $json = $reader.ReadToEnd() } finally { $reader.Dispose() }
        }
        $body = $json | ConvertFrom-Json
        if ($body.error.code -notin @('CAPABILITY_NOT_SUPPORTED','CHANNEL_NOT_SUPPORTED')) { throw "wrong capability error for ${path}: status=$([int]$_.Exception.Response.StatusCode) code=$($body.error.code)" }
    }
}
try {
    Say 'MEDIUM_INTEGRITY_PASS'
    Start-P9Daemon
    foreach ($kind in @('hermes','openclaw')) {
        $detected = Request-P9 'POST' "/runtimes/$kind/detect"
        if ($detected.state -ne 'SUPPORTED') { throw "$kind discovery=$($detected.state)" }
        Say "$kind version=$($detected.selected.version) executable=$($detected.selected.path)"
        $initial = Request-P9 'GET' "/runtimes/$kind/instances"
        if ($initial.freshness -ne 'FRESH' -or ($initial.instances | Where-Object name -in @('yorvap9same','yorvap9other'))) { throw "$kind inventory is not new and fresh: freshness=$($initial.freshness) code=$($initial.errorCode)" }
    }
    foreach ($target in @(@{kind='hermes';name='yorvap9same'}, @{kind='openclaw';name='yorvap9same'}, @{kind='openclaw';name='yorvap9other'})) {
        $op = Request-P9 'POST' "/runtimes/$($target.kind)/instances" @{name=$target.name} ([Guid]::NewGuid().ToString('N'))
        Wait-P9Operation $op "$($target.kind) create $($target.name)"
    }
    $hermesList = Request-P9 'GET' '/runtimes/hermes/instances'
    $clawList = Request-P9 'GET' '/runtimes/openclaw/instances'
    $hermes = @($hermesList.instances | Where-Object name -eq 'yorvap9same')[0]
    $clawA = @($clawList.instances | Where-Object name -eq 'yorvap9same')[0]
    $clawB = @($clawList.instances | Where-Object name -eq 'yorvap9other')[0]
    if (-not $hermes -or -not $clawA -or -not $clawB -or $hermes.instanceId -eq $clawA.instanceId -or $hermes.runtimeInstallationId -eq $clawA.runtimeInstallationId) { throw 'Runtime identities collided' }
    if ($clawA.capabilities.models -or $clawA.capabilities.channels -or $clawA.protected -or -not $clawA.capabilities.lifecycle) { throw 'OpenClaw capability/ownership mismatch' }
    foreach ($path in @("/instances/$($clawA.instanceId)/config", "/instances/$($clawA.instanceId)/channels", '/runtimes/openclaw/model-provider-presets')) { Assert-P9CapabilityRejected $path }
    Say 'SAME_NAME_AND_CAPABILITIES_PASS'
    Lifecycle-P9 $hermes 'start'
    Lifecycle-P9 $clawA 'start'
    Lifecycle-P9 $clawB 'start'
    foreach ($item in @($hermes,$clawA,$clawB)) { Assert-P9State $item 'RUNNING' }
    foreach ($item in @($clawA,$clawB)) {
        $health = Request-P9 'GET' "/instances/$($item.instanceId)/health"
        if ($health.state -ne 'HEALTHY' -or $health.partial) { throw 'Gateway authenticated health failed' }
    }
    Say 'THREE_LIVE_TWO_GATEWAYS_AUTHENTICATED'
    Lifecycle-P9 $clawA 'restart'
    foreach ($item in @($hermes,$clawA,$clawB)) { Assert-P9State $item 'RUNNING' }
    Stop-P9Daemon
    Start-P9Daemon
    foreach ($item in @($hermes,$clawA,$clawB)) { Assert-P9State $item 'RUNNING' }
    $again = Request-P9 'GET' '/runtimes/openclaw/instances'
    if (@($again.instances | Where-Object instanceId -eq $clawA.instanceId).Count -ne 1) { throw 'identity changed on reconnect' }
    $recovery = Request-P9 'GET' '/node/recovery'
    if ($recovery.state -ne 'READY' -or $recovery.errorCode) { throw "Node did not recover: state=$($recovery.state) code=$($recovery.errorCode)" }
    Say 'DAEMON_RECONNECT_RUNTIME_LIFETIME_PASS'
    Lifecycle-P9 $clawA 'stop'
    Assert-P9State $clawB 'RUNNING'
    Assert-P9State $hermes 'RUNNING'
    $op = Request-P9 'DELETE' "/instances/$($clawA.instanceId)" @{confirmationName=$clawA.name} ([Guid]::NewGuid().ToString('N'))
    Wait-P9Operation $op 'OpenClaw delete A'
    if (Test-Path -LiteralPath (Join-Path $env:USERPROFILE '.openclaw-yorvap9same')) { throw 'deleted Gateway state remains' }
    Assert-P9State $clawB 'RUNNING'
    Assert-P9State $hermes 'RUNNING'
    foreach ($item in @($clawB,$hermes)) {
        Lifecycle-P9 $item 'stop'
        $op = Request-P9 'DELETE' "/instances/$($item.instanceId)" @{confirmationName=$item.name} ([Guid]::NewGuid().ToString('N'))
        Wait-P9Operation $op "delete $($item.instanceId)"
    }
    if (Get-ScheduledTask | Where-Object TaskName -match 'OpenClaw') { throw 'unexpected OpenClaw login task' }
    if (Get-ChildItem -LiteralPath (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Startup') -Filter '*openclaw*') { throw 'unexpected OpenClaw Startup entry' }
    Say 'PASS create identity start restart auth reconnect stop delete isolation no-login-entry'
}
finally { Stop-P9Daemon }
