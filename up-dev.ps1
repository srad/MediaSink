# up-dev.ps1 - Windows equivalent of up-dev.sh

# Load .env.local variables into the current session
Get-Content .env.local | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
        [System.Environment]::SetEnvironmentVariable($Matches[1].Trim(), $Matches[2].Trim(), 'Process')
    }
}

$dataPath = [System.Environment]::GetEnvironmentVariable('DATA_PATH', 'Process')
Write-Host "Creating directory: $dataPath ..."
New-Item -ItemType Directory -Path $dataPath -Force | Out-Null

docker compose down
docker compose --env-file=.env.local up --build
