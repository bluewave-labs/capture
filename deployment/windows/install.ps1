#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Installs the latest Capture release from GitHub and registers it as a Windows service.

.PARAMETER InstallDir
    Directory where the Capture binary will be installed. Defaults to: C:\Program Files\Capture

.PARAMETER ServiceName
    Name of the Windows service to create. Defaults to: capture

.PARAMETER APISecret
    Authentication secret for the Capture API. Prompted if not provided.

.PARAMETER Port
    Port the Capture server listens on. Defaults to: 59232

.EXAMPLE
    .\install.ps1

.EXAMPLE
    .\install.ps1 -APISecret "my-secret" -Port 8080

.EXAMPLE
    .\install.ps1 -InstallDir "C:\capture" -ServiceName "capture-agent" -APISecret "my-secret"
#>
[CmdletBinding()]
param(
    [string] $InstallDir  = 'C:\Program Files\Capture',
    [string] $ServiceName = 'capture',
    [string] $APISecret   = '',
    [int]    $Port        = 59232
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Detect architecture
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}
Write-Host "Architecture: $arch"

# Fetch latest release from GitHub
$release    = Invoke-RestMethod -Uri 'https://api.github.com/repos/bluewave-labs/capture/releases/latest' `
                                -Headers @{ 'User-Agent' = 'capture-install-script' }
$version    = $release.tag_name
$versionNum = $version.TrimStart('v')
Write-Host "Latest version: $version"

# Find the download asset
$assetName = "capture_${versionNum}_windows_${arch}.zip"
$asset     = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
if (-not $asset) {
    Write-Error "Asset '$assetName' not found in the latest release."
    exit 1
}
Write-Host "Downloading: $($asset.browser_download_url)"

# Download and extract
$tmpDir  = Join-Path $env:TEMP "capture_install_$([System.IO.Path]::GetRandomFileName())"
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
$zipPath = Join-Path $tmpDir $assetName
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath -UseBasicParsing

$extractDir = Join-Path $tmpDir 'extracted'
Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

$binaryPath = Get-ChildItem -Path $extractDir -Filter 'capture.exe' -Recurse |
              Select-Object -First 1 -ExpandProperty FullName
if (-not $binaryPath) {
    Write-Error "capture.exe not found in the extracted archive."
    exit 1
}

# Install binary
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
$targetBinary = Join-Path $InstallDir 'capture.exe'

$existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existingService) {
    Write-Host "Stopping existing service '$ServiceName'..."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

Copy-Item -Path $binaryPath -Destination $targetBinary -Force
Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "Installed: $targetBinary"

# Collect API secret
if ([string]::IsNullOrWhiteSpace($APISecret)) {
    $secureSecret = Read-Host 'Enter API_SECRET' -AsSecureString
    $bstr         = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureSecret)
    $APISecret    = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
}
if ([string]::IsNullOrWhiteSpace($APISecret)) {
    Write-Error 'API_SECRET must not be empty.'
    exit 1
}

# Create or update the Windows service
if ($existingService) {
    & sc.exe config $ServiceName binPath= "`"$targetBinary`"" | Out-Null
    Write-Host "Updated service '$ServiceName'."
} else {
    New-Service `
        -Name           $ServiceName `
        -BinaryPathName $targetBinary `
        -DisplayName    'Capture Monitoring Agent' `
        -Description    'Capture hardware monitoring agent (https://github.com/bluewave-labs/capture)' `
        -StartupType    Automatic | Out-Null
    Write-Host "Created service '$ServiceName'."
}

# Write environment variables into the service registry key
Set-ItemProperty `
    -Path  "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName" `
    -Name  'Environment' `
    -Value @("API_SECRET=$APISecret", "PORT=$Port", "GIN_MODE=release") `
    -Type  MultiString

# Start the service
Start-Service -Name $ServiceName
Write-Host "Service status: $((Get-Service -Name $ServiceName).Status)"

Write-Host ""
Write-Host "Capture $version installed successfully."
Write-Host "  Install directory : $InstallDir"
Write-Host "  Service name      : $ServiceName"
Write-Host "  Port              : $Port"
Write-Host ""
Write-Host "  Get-Service $ServiceName"
Write-Host "  Stop-Service $ServiceName"
Write-Host "  Start-Service $ServiceName"
Write-Host "  Restart-Service $ServiceName"
Write-Host ""
Write-Host "  Invoke-RestMethod http://localhost:$Port/health"
