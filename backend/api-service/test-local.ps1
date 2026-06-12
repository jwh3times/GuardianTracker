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

  This script closes both gaps: it spins up a throwaway Postgres container on a
  non-default port (5433, so it won't collide with the docker-compose Postgres on
  5432), exports CGO_ENABLED=1 + TEST_DATABASE_URL, then runs
  `go test -race -coverprofile`. The container is left running for fast re-runs;
  pass -Down to remove it.

  The test DB needs no manual setup — the migrations are embedded in the binary
  and applied automatically by the test harness (db.Migrate), and each test
  creates/cleans its own rows.

.PARAMETER Port      Host port to map Postgres to (default 5433).
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
  [int]$Port = 5433,
  [switch]$Down,
  [switch]$Fresh,
  [switch]$Html,
  [switch]$NoRace
)

$ErrorActionPreference = 'Stop'

$Container = 'gt-test-pg'
$DbUser    = 'test_user'
$DbPass    = 'test_password'
$DbName    = 'test_db'
$Image     = 'postgres:18-alpine'   # already pulled by docker-compose

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

# --- teardown ---------------------------------------------------------------
if ($Down) {
  docker rm -f $Container *> $null
  Write-Host "Removed container '$Container'." -ForegroundColor Yellow
  return
}

Assert-Docker
if ($Fresh) { docker rm -f $Container *> $null }

# --- ensure the container is up (idempotent) --------------------------------
$running = (docker ps    --format '{{.Names}}') -contains $Container
$exists  = (docker ps -a --format '{{.Names}}') -contains $Container

if (-not $exists) {
  Write-Host "Starting Postgres ($Image) on localhost:$Port ..." -ForegroundColor Cyan
  docker run -d --name $Container `
    -e POSTGRES_USER=$DbUser -e POSTGRES_PASSWORD=$DbPass -e POSTGRES_DB=$DbName `
    -p "${Port}:5432" $Image | Out-Null
} elseif (-not $running) {
  Write-Host "Starting existing container '$Container' ..." -ForegroundColor Cyan
  docker start $Container | Out-Null
} else {
  Write-Host "Reusing running container '$Container'." -ForegroundColor DarkGray
}

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
