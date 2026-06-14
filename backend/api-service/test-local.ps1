#requires -Version 5.1
<#
.SYNOPSIS
  Run the api-service Go test suite locally with the same coverage CI produces.

.DESCRIPTION
  CI hits ~63% statement coverage; a plain local `go test ./...` only sees
  ~52% because two groups of tests self-skip:

    * sqlite-backed tests (services/manifest, services/search BuildIndex) need
      cgo — gated by a runtime `requireSQLite` probe.
    * the db package integration tests need a reachable Postgres — gated on the
      TEST_DATABASE_URL env var.

  This script closes both gaps: it starts the `test-postgres` service from the
  root docker-compose.yml (a "test"-profiled, throwaway Postgres on a non-default
  port — 5533, so it won't collide with the docker-compose Postgres on 5532),
  exports CGO_ENABLED=1 + TEST_DATABASE_URL, then runs `go test -race
  -coverprofile`. Driving it through Compose means Docker Desktop groups it with
  the rest of the "guardiantracker" stack. The container is left running for fast
  re-runs; pass -Down to remove it.

  The test DB needs no manual setup — the migrations are embedded in the binary
  and applied automatically by the test harness (db.Migrate), and each test
  creates/cleans its own rows.

.PARAMETER Port      Host port to map Postgres to (default 5533).
.PARAMETER Down      Stop and remove the test Postgres container, then exit.
.PARAMETER Fresh     Recreate the container from scratch (drops its data first).
.PARAMETER Html      Open the per-line HTML coverage report after the run.
.PARAMETER NoRace    Skip the race detector (slightly faster; CI uses -race).

.EXAMPLE
  ./test-local.ps1            # start pg if needed, run all tests, print coverage
.EXAMPLE
  ./test-local.ps1 -Html      # also open the HTML coverage report
.EXAMPLE
  ./test-local.ps1 -Down      # tear down the test Postgres container
#>
param(
  [int]$Port = 5533,
  [switch]$Down,
  [switch]$Fresh,
  [switch]$Html,
  [switch]$NoRace
)

$ErrorActionPreference = 'Stop'

$Container = 'gt-test-pg'        # container_name of the test-postgres service
$Service   = 'test-postgres'     # service name in the root docker-compose.yml
$DbUser    = 'test_user'
$DbPass    = 'test_password'
$DbName    = 'test_db'

# Manage the test DB through the root docker-compose.yml (rather than a bare
# `docker run`) so Docker Desktop groups it under the same "guardiantracker"
# project as the rest of the stack. The service is gated behind the "test"
# profile, so it never starts on a plain `docker compose up`.
$RepoRoot    = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$ComposeFile = Join-Path $RepoRoot 'docker-compose.yml'
$Compose     = @('compose', '-f', $ComposeFile, '--profile', 'test')

function Assert-Docker {
  # A missing `docker` executable throws (catchable); a stopped daemon only
  # sets a non-zero exit code, so check both.
  try { docker info *> $null } catch {
    throw 'Docker CLI not found on PATH. Install Docker Desktop.'
  }
  if ($LASTEXITCODE -ne 0) {
    throw 'Docker daemon is not responding. Start Docker Desktop and try again.'
  }
}

Assert-Docker

# One-time migration: older versions of this script started a standalone
# `gt-test-pg` via `docker run` (no Compose labels). Compose can't adopt that
# name, so remove such a leftover before handing the name to the Compose service.
$existing = (docker ps -a --format '{{.Names}}') -contains $Container
if ($existing) {
  $managed = (docker ps -a --filter "name=^/$Container$" --filter 'label=com.docker.compose.project' --format '{{.Names}}') -contains $Container
  if (-not $managed) {
    Write-Host "Removing legacy standalone '$Container' so Compose can manage it..." -ForegroundColor Yellow
    docker rm -f $Container *> $null
  }
}

# --- teardown ---------------------------------------------------------------
if ($Down) {
  docker @Compose rm -fsv $Service *> $null
  Write-Host "Removed test Postgres (service '$Service')." -ForegroundColor Yellow
  return
}

if ($Fresh) { docker @Compose rm -fsv $Service *> $null }

# --- ensure the service is up (idempotent) ----------------------------------
# `up -d` is a no-op if it's already running with this config, starts it if
# stopped, and recreates it (e.g. with a new -Port mapping) when config changes.
$env:TEST_POSTGRES_PORT = "$Port"
Write-Host "Starting Postgres service '$Service' on localhost:$Port (profile: test) ..." -ForegroundColor Cyan
docker @Compose up -d $Service
if ($LASTEXITCODE -ne 0) { throw "Failed to start the '$Service' Compose service." }

# --- wait for readiness -----------------------------------------------------
Write-Host 'Waiting for Postgres to accept connections' -NoNewline
$ready = $false
for ($i = 0; $i -lt 40; $i++) {
  docker exec $Container pg_isready -U $DbUser -d $DbName *> $null
  if ($LASTEXITCODE -eq 0) { $ready = $true; break }
  Start-Sleep -Milliseconds 500
  Write-Host '.' -NoNewline
}
Write-Host ''
if (-not $ready) { throw "Postgres did not become ready (container '$Container')." }

# --- run the suite exactly like CI ------------------------------------------
$env:CGO_ENABLED       = '1'
$env:TEST_DATABASE_URL = "postgres://${DbUser}:${DbPass}@localhost:${Port}/${DbName}?sslmode=disable"

Push-Location $PSScriptRoot
try {
  $goArgs = @('test')
  if (-not $NoRace) { $goArgs += '-race' }
  $goArgs += @('-coverprofile=coverage.out', './...')

  Write-Host "go $($goArgs -join ' ')" -ForegroundColor Cyan
  & go @goArgs
  $testExit = $LASTEXITCODE

  Write-Host ''
  if (Test-Path coverage.out) {
    # Quote the -flag=value args: PowerShell otherwise splits them on '=' when
    # passing to the native go.exe, which trips "too many arguments".
    go tool cover "-func=coverage.out" | Select-Object -Last 1
    if ($Html) { go tool cover "-html=coverage.out" }
  }
}
finally {
  Pop-Location
}

if ($testExit -ne 0) {
  Write-Host "Tests failed (exit $testExit)." -ForegroundColor Red
  exit $testExit
}
Write-Host 'All tests passed.' -ForegroundColor Green
