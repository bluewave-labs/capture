param(
    [Parameter(Mandatory)] [string]$Binary
)

# Runs Capture as a Windows service and checks start, stop and startup failures.
# Every scenario prints what it saw; the script exits 1 if any check failed.

$ErrorActionPreference = 'Continue'
$svc      = 'capture'
$port     = 59232
$secret   = 's3cret'
$good     = @("API_SECRET=$secret", "PORT=$port", 'GIN_MODE=release')
$failures = [System.Collections.Generic.List[string]]::new()

# ERROR_SERVICE_SPECIFIC_ERROR: the service stopped and reported its own exit code
$serviceSpecificError = 1066

function Check([string]$name, [bool]$ok, [string]$detail = '') {
    $mark = if ($ok) { 'PASS' } else { 'FAIL' }
    Write-Host "  [$mark] $name $detail"
    if (-not $ok) { $failures.Add("$($script:scenario): $name $detail") }
}

function Section([string]$name) {
    $script:scenario = $name
    Write-Host "`n==================== $name ===================="
}

function Remove-Capture {
    if (Get-Service $svc -ErrorAction SilentlyContinue) {
        & sc.exe stop $svc | Out-Null
        Start-Sleep 2
        Get-Process capture -ErrorAction SilentlyContinue | Stop-Process -Force
        & sc.exe delete $svc | Out-Null
        Start-Sleep 2
    }
}

function Install-Capture([string[]]$envs) {
    Remove-Capture
    $null = New-Service -Name $svc -BinaryPathName "`"$Binary`"" -DisplayName 'Capture Monitoring Agent' -StartupType Manual
    Set-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Services\$svc" -Name Environment -Value $envs -Type MultiString
    # Same restart actions as deployment/windows/install.ps1
    & sc.exe failure $svc reset= 86400 actions= restart/5000/restart/10000/restart/30000 | Out-Null
}

function Start-Capture {
    $script:since = Get-Date
    Start-Sleep -Milliseconds 500
    & sc.exe start $svc | Out-Null
}

function Wait-State([string]$want, [int]$seconds) {
    $sw = [Diagnostics.Stopwatch]::StartNew()
    while ($sw.Elapsed.TotalSeconds -lt $seconds) {
        if ("$((Get-Service $svc).Status)" -eq $want) { return [math]::Round($sw.Elapsed.TotalSeconds, 1) }
        Start-Sleep -Milliseconds 250
    }
    return -1
}

function Get-ExitCodes {
    $query = (& sc.exe query $svc) -join "`n"
    $win32 = if ($query -match 'WIN32_EXIT_CODE\s*:\s*(\d+)') { [int]$Matches[1] } else { -1 }
    $specific = if ($query -match 'SERVICE_EXIT_CODE\s*:\s*(\d+)') { [int]$Matches[1] } else { -1 }
    return @{ Win32 = $win32; Specific = $specific }
}

function Get-Status([string]$path, [switch]$Auth) {
    $headers = @{}
    if ($Auth) { $headers = @{ Authorization = "Bearer $secret" } }
    try {
        return (Invoke-WebRequest "http://127.0.0.1:$port$path" -Headers $headers -SkipHttpErrorCheck -TimeoutSec 10).StatusCode
    } catch { return 0 }
}

function Get-CaptureEvents {
    @(Get-WinEvent -FilterHashtable @{ LogName = 'Application'; ProviderName = 'capture'; StartTime = $script:since } -ErrorAction SilentlyContinue |
        Sort-Object TimeCreated)
}

# Service Control Manager events 7031 and 7034 mean the service terminated unexpectedly
function Get-CrashEvents {
    @(Get-WinEvent -FilterHashtable @{ LogName = 'System'; ProviderName = 'Service Control Manager'; Id = 7031, 7034; StartTime = $script:since } -ErrorAction SilentlyContinue |
        Where-Object { $_.Message -match 'Capture' })
}

function Show-Events {
    Get-CaptureEvents | ForEach-Object { Write-Host "    [event $($_.LevelDisplayName)] $($_.Message)" }
    @(Get-WinEvent -FilterHashtable @{ LogName = 'System'; ProviderName = 'Service Control Manager'; StartTime = $script:since } -ErrorAction SilentlyContinue |
        Where-Object { $_.Message -match 'Capture' } | Sort-Object TimeCreated) |
        ForEach-Object { Write-Host "    [scm $($_.Id)] $($_.Message)" }
}

# A startup error must stop the service with Capture's own exit code, log an error event,
# and not be restarted by the failure actions.
# A bind error is retried for 60s before the start fails, so that scenario passes a longer wait.
function Test-StartFailure([string]$expectedMessage, [int]$stopSeconds = 20) {
    Start-Capture
    $stopped = Wait-State 'Stopped' $stopSeconds
    Check 'service stops instead of hanging in Start pending' ($stopped -ge 0) "(after ${stopped}s)"
    $codes = Get-ExitCodes
    Check 'reports a service-specific error' ($codes.Win32 -eq $serviceSpecificError -and $codes.Specific -eq 1) "(WIN32_EXIT_CODE=$($codes.Win32), SERVICE_EXIT_CODE=$($codes.Specific))"

    # The first restart action fires after 5s; wait past it to prove nothing restarts
    Start-Sleep 12
    Check 'is not restarted by the failure actions' ("$((Get-Service $svc).Status)" -eq 'Stopped') "(status $((Get-Service $svc).Status))"
    Check 'is not recorded as a crash' ((Get-CrashEvents).Count -eq 0) "($((Get-CrashEvents).Count) crash events)"

    $errors = @(Get-CaptureEvents | Where-Object { $_.LevelDisplayName -eq 'Error' })
    Check 'logs one error event' ($errors.Count -eq 1) "($($errors.Count) error events)"
    Check 'error event names the cause' (($errors | Where-Object { $_.Message -match $expectedMessage }).Count -ge 1) "(expected /$expectedMessage/)"
    Show-Events
}

Section 'install.ps1 parses'
$parseErrors = $null
$null = [System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path 'deployment/windows/install.ps1'), [ref]$null, [ref]$parseErrors)
Check 'no syntax errors' ($parseErrors.Count -eq 0) ($parseErrors -join '; ')

Section 'valid configuration: start, serve, stop'
Install-Capture $good
Start-Capture
$running = Wait-State 'Running' 30
Check 'reaches Running' ($running -ge 0) "(after ${running}s)"
Check '/health returns 200' ((Get-Status '/health') -eq 200)
$metrics = Get-Status '/api/v1/metrics' -Auth
Check '/api/v1/metrics with token returns 200 or 207' ($metrics -in 200, 207) "(got $metrics)"
Check '/api/v1/metrics without token returns 401' ((Get-Status '/api/v1/metrics') -eq 401)
Check 'logs that the server started' ((Get-CaptureEvents | Where-Object { $_.Message -match 'server started' }).Count -ge 1)
Check 'logs the metrics URL hint' ((Get-CaptureEvents | Where-Object { $_.Message -match '/api/v1/metrics' }).Count -ge 1)
& sc.exe stop $svc | Out-Null
$stopped = Wait-State 'Stopped' 20
Check 'stops cleanly' ($stopped -ge 0) "(after ${stopped}s)"
$codes = Get-ExitCodes
Check 'exit code is 0' ($codes.Win32 -eq 0) "(WIN32_EXIT_CODE=$($codes.Win32))"
Check 'port is released' ((Get-Status '/health') -eq 0)
Check 'no crash recorded' ((Get-CrashEvents).Count -eq 0)
Show-Events
Remove-Capture

Section 'port already in use'
# Create() makes a dual-mode (IPv4 and IPv6) listener. Capture's default 0.0.0.0 also binds
# dual-stack, so an IPv4-only listener would not always conflict with it.
$listener = [Net.Sockets.TcpListener]::Create($port)
$listener.Start()
Install-Capture $good
Test-StartFailure 'could not start: listen tcp .*bind' 90
Check 'logs that it retried the bind' ((Get-CaptureEvents | Where-Object { $_.Message -match 'will retry' }).Count -ge 1)
$listener.Stop()
Remove-Capture

Section 'port freed while retrying'
# At boot the LISTEN address or the port may not be available yet; Capture must wait for it
# instead of failing for good.
$listener = [Net.Sockets.TcpListener]::Create($port)
$listener.Start()
Install-Capture $good
Start-Capture
Start-Sleep 5
Check 'stays Start pending while the port is taken' ("$((Get-Service $svc).Status)" -eq 'StartPending') "(status $((Get-Service $svc).Status))"
$listener.Stop()
$running = Wait-State 'Running' 30
Check 'reaches Running once the port is free' ($running -ge 0) "(after ${running}s)"
Check '/health returns 200' ((Get-Status '/health') -eq 200)
Check 'logs no error event' (@(Get-CaptureEvents | Where-Object { $_.LevelDisplayName -eq 'Error' }).Count -eq 0)
& sc.exe stop $svc | Out-Null
$stopped = Wait-State 'Stopped' 20
Check 'stops cleanly' ($stopped -ge 0) "(after ${stopped}s)"
Show-Events
Remove-Capture

Section 'API_SECRET missing'
Install-Capture @("PORT=$port", 'GIN_MODE=release')
Test-StartFailure 'API_SECRET'
Remove-Capture

Section 'invalid SMART_DEVICES'
Install-Capture ($good + 'SMART_DEVICES=bad')
Test-StartFailure 'SMART_DEVICES'
Remove-Capture

Write-Host ''
if ($failures.Count -gt 0) {
    Write-Host "$($failures.Count) check(s) failed:" -ForegroundColor Red
    $failures | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
    exit 1
}
Write-Host 'All Windows service checks passed.' -ForegroundColor Green
exit 0
