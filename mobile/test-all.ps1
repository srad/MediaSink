param(
    [switch]$SkipIntegration,
    [switch]$UsePub,
    [string]$Flutter = "flutter"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$commonArgs = @("test")

if (-not $UsePub) {
    $commonArgs += "--no-pub"
}

function Test-HasSupportedAndroidDevice {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FlutterBinary
    )

    $devicesJson = & $FlutterBinary "devices" "--machine" 2>$null
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($devicesJson)) {
        return $false
    }

    try {
        $devices = $devicesJson | ConvertFrom-Json
    }
    catch {
        return $false
    }

    foreach ($device in @($devices)) {
        $platform = [string]($device.targetPlatform ?? "")
        if ($platform.StartsWith("android-", [System.StringComparison]::OrdinalIgnoreCase)) {
            return $true
        }
    }

    return $false
}

Push-Location $scriptDir
try {
    Write-Host "Running unit and widget tests..."
    & $Flutter @commonArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    if (-not $SkipIntegration) {
        if (Test-HasSupportedAndroidDevice -FlutterBinary $Flutter) {
            Write-Host "Running integration tests..."
            & $Flutter "test" "integration_test" @($commonArgs | Select-Object -Skip 1)
            if ($LASTEXITCODE -ne 0) {
                exit $LASTEXITCODE
            }
        }
        else {
            Write-Host "Skipping integration tests: no supported Android device detected."
        }
    }

    Write-Host "All requested tests passed."
}
finally {
    Pop-Location
}
