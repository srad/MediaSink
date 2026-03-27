param(
    [switch]$UsePub,
    [switch]$AllowDebugFallback,
    [string]$Flutter = "flutter"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$keyPropertiesPath = Join-Path $scriptDir "android\key.properties"
$apkPath = Join-Path $scriptDir "build\app\outputs\flutter-apk\app-release.apk"
$staleRegistrantPath = Join-Path $scriptDir "android\app\src\main\java\io\flutter\plugins\GeneratedPluginRegistrant.java"
$pubspecPath = Join-Path $scriptDir "pubspec.yaml"

function Get-KeyProperties {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $values = @{}
    if (-not (Test-Path -LiteralPath $Path)) {
        return $values
    }

    foreach ($line in Get-Content -LiteralPath $Path) {
        if ([string]::IsNullOrWhiteSpace($line) -or $line.TrimStart().StartsWith("#")) {
            continue
        }

        $parts = $line -split "=", 2
        if ($parts.Length -ne 2) {
            continue
        }

        $values[$parts[0].Trim()] = $parts[1].Trim()
    }

    return $values
}

function Test-HasReleaseSigning {
    param(
        [Parameter(Mandatory = $true)]
        [hashtable]$Properties
    )

    foreach ($key in @("storeFile", "storePassword", "keyAlias", "keyPassword")) {
        if (-not $Properties.ContainsKey($key) -or [string]::IsNullOrWhiteSpace([string]$Properties[$key])) {
            return $false
        }
    }

    return Test-Path -LiteralPath $Properties["storeFile"]
}

function Get-AppVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PubspecPath
    )

    if (-not (Test-Path -LiteralPath $PubspecPath)) {
        return "release"
    }

    foreach ($line in Get-Content -LiteralPath $PubspecPath) {
        if ($line -match '^\s*version\s*:\s*([^\s#]+)') {
            $rawVersion = $matches[1].Trim()
            $buildName = ($rawVersion -split '\+')[0]
            if (-not [string]::IsNullOrWhiteSpace($buildName)) {
                return ($buildName -replace '[^0-9A-Za-z._-]', '-')
            }
        }
    }

    return "release"
}

Push-Location $scriptDir
try {
    $keyProperties = Get-KeyProperties -Path $keyPropertiesPath
    $hasReleaseSigning = Test-HasReleaseSigning -Properties $keyProperties

    if (-not $hasReleaseSigning -and -not $AllowDebugFallback) {
        throw "Release signing is not fully configured in android\key.properties. Fill storeFile, storePassword, keyAlias, and keyPassword, or pass -AllowDebugFallback if you intentionally want the Gradle debug-signing fallback."
    }

    if (Test-Path -LiteralPath $staleRegistrantPath) {
        Write-Host "Removing stale Android GeneratedPluginRegistrant.java..."
        Remove-Item -LiteralPath $staleRegistrantPath -Force
    }

    $args = @("build", "apk", "--release")
    if (-not $UsePub) {
        $args += "--no-pub"
    }

    Write-Host "Building release APK..."
    & $Flutter @args
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    if (-not (Test-Path -LiteralPath $apkPath)) {
        throw "Flutter reported success, but the APK was not found at $apkPath"
    }

    $version = Get-AppVersion -PubspecPath $pubspecPath
    $renamedApkPath = Join-Path (Split-Path -Parent $apkPath) "mediasink-$version.apk"
    Move-Item -LiteralPath $apkPath -Destination $renamedApkPath -Force

    $apk = Get-Item -LiteralPath $renamedApkPath
    Write-Host ""
    Write-Host "Release APK ready:"
    Write-Host $apk.FullName
    Write-Host ("Size: {0:N2} MB" -f ($apk.Length / 1MB))
}
finally {
    Pop-Location
}
