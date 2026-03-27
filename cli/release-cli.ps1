param(
    [string]$Cargo = "cargo",
    [switch]$Unlocked
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$cargoTomlPath = Join-Path $scriptDir "Cargo.toml"

function Get-CargoPackageInfo {
    param(
        [Parameter(Mandatory = $true)]
        [string]$CargoTomlPath
    )

    if (-not (Test-Path -LiteralPath $CargoTomlPath)) {
        throw "Cargo.toml was not found at $CargoTomlPath"
    }

    $inPackageSection = $false
    $packageName = $null
    $packageVersion = $null

    foreach ($line in Get-Content -LiteralPath $CargoTomlPath) {
        $trimmed = $line.Trim()

        if ($trimmed -match '^\[(.+)\]$') {
            $inPackageSection = ($matches[1] -eq "package")
            continue
        }

        if (-not $inPackageSection) {
            continue
        }

        if (-not $packageName -and $trimmed -match '^name\s*=\s*"([^"]+)"') {
            $packageName = $matches[1]
            continue
        }

        if (-not $packageVersion -and $trimmed -match '^version\s*=\s*"([^"]+)"') {
            $packageVersion = $matches[1]
            continue
        }
    }

    if ([string]::IsNullOrWhiteSpace($packageName) -or [string]::IsNullOrWhiteSpace($packageVersion)) {
        throw "Could not read package name/version from $CargoTomlPath"
    }

    return @{
        Name = $packageName
        Version = ($packageVersion -replace '[^0-9A-Za-z._-]', '-')
    }
}

Push-Location $scriptDir
try {
    $package = Get-CargoPackageInfo -CargoTomlPath $cargoTomlPath
    $sourceBinaryPath = Join-Path $scriptDir ("target\release\{0}.exe" -f $package.Name)
    $releaseBinaryPath = Join-Path $scriptDir ("target\release\mediasink-{0}.exe" -f $package.Version)

    $args = @("build", "--release")
    if (-not $Unlocked) {
        $args += "--locked"
    }

    Write-Host "Building CLI release binary..."
    & $Cargo @args
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    if (-not (Test-Path -LiteralPath $sourceBinaryPath)) {
        throw "Cargo reported success, but the CLI binary was not found at $sourceBinaryPath"
    }

    Copy-Item -LiteralPath $sourceBinaryPath -Destination $releaseBinaryPath -Force

    $binary = Get-Item -LiteralPath $releaseBinaryPath
    Write-Host ""
    Write-Host "CLI release binary ready:"
    Write-Host $binary.FullName
    Write-Host ("Size: {0:N2} MB" -f ($binary.Length / 1MB))
}
finally {
    Pop-Location
}
