@echo off
setlocal
title Deraah Pathways

set "PROJECT=%~dp0"
if not exist "%PROJECT%Setup.ps1" set "PROJECT=%USERPROFILE%\Documents\Codex\2026-11-05\new-chat-8\work\family-update\deraah-pathways\"
if not exist "%PROJECT%Setup.ps1" goto missing_project
pushd "%PROJECT%" || goto missing_project

where docker >nul 2>&1
if errorlevel 1 goto missing_docker
docker info --format "{{.ServerVersion}}" >nul 2>&1
if not errorlevel 1 goto build

echo Starting Docker Desktop. Please wait...
if exist "%LOCALAPPDATA%\Programs\DockerDesktop\Docker Desktop.exe" (
    start "" "%LOCALAPPDATA%\Programs\DockerDesktop\Docker Desktop.exe"
) else (
    docker desktop start --detach
)
powershell -NoProfile -Command "for ($attempt=0; $attempt -lt 36; $attempt++) { docker info --format '{{.ServerVersion}}' *> $null; if ($LASTEXITCODE -eq 0) { exit 0 }; Start-Sleep -Seconds 5 }; exit 1"
if errorlevel 1 goto docker_unavailable

:build
echo Building and starting Deraah Pathways...
powershell -NoProfile -ExecutionPolicy Bypass -File ".\Setup.ps1"
if errorlevel 1 goto failed

echo Waiting for the portal and database...
powershell -NoProfile -Command "for ($attempt=0; $attempt -lt 30; $attempt++) { try { $health = Invoke-RestMethod -Uri 'http://localhost:8093/health' -TimeoutSec 3 -ErrorAction Stop; if ($health.status -eq 'ok') { exit 0 } } catch {}; Start-Sleep -Seconds 2 }; exit 1"
if errorlevel 1 goto failed

start "" "http://localhost:8093/login"
echo.
echo Portal ready. Local admin settings:
findstr /B /C:"ADMIN_EMAIL=" /C:"ADMIN_PASSWORD=" .env
popd
pause
exit /b 0

:missing_project
echo Could not find the Deraah Pathways project folder.
pause
exit /b 1

:missing_docker
echo Docker command was not found on this Windows account.
goto failed

:docker_unavailable
echo Docker Desktop did not become reachable for this Windows account.
docker info --format "{{.ServerVersion}}"

:failed
echo Deraah Pathways did not start. The error is shown above.
popd
pause
exit /b 1
