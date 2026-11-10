$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot

function New-PathwaysSecret {
    $bytes = New-Object byte[] 24
    $generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $generator.GetBytes($bytes) } finally { $generator.Dispose() }
    return [BitConverter]::ToString($bytes).Replace('-', '').ToLowerInvariant()
}

if (!(Test-Path -LiteralPath '.env')) {
    $password = New-PathwaysSecret
    @(
        'APP_ORIGIN=http://localhost:8093'
        'ADMIN_EMAIL=admin@deraah.local'
        "ADMIN_PASSWORD=$password"
        "POSTGRES_PASSWORD=$(New-PathwaysSecret)"
    ) | Set-Content -LiteralPath '.env' -Encoding ascii
    Write-Host 'First-start account saved to .env (ADMIN_EMAIL and ADMIN_PASSWORD).'
}

docker info --format '{{.ServerVersion}}' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Open Docker Desktop and wait for the Linux engine to start, then run Setup.ps1 again.' }
docker compose up -d --build
if ($LASTEXITCODE -ne 0) { throw 'Docker Compose could not start the project.' }
Write-Host 'Open http://localhost:8093'
