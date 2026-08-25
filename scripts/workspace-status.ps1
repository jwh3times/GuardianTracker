#requires -Version 5.1

[CmdletBinding()]
param(
    [switch]$IncludePrivateBranch,
    [Parameter(DontShow = $true)]
    [string]$RepositoryRoot,
    [Parameter(DontShow = $true)]
    [string]$GitExecutable = "git"
)

$ErrorActionPreference = "Stop"
$DebugPreference = "SilentlyContinue"
$VerbosePreference = "SilentlyContinue"
$script:UnsafeTargets = $false
$script:RepositoryErrors = $false

trap {
    Write-Error "Workspace status could not complete because an unexpected local error occurred." -ErrorAction Continue
    exit 1
}

function Get-FullPath {
    param([string]$Path)

    return [IO.Path]::GetFullPath($Path).TrimEnd(
        [IO.Path]::DirectorySeparatorChar,
        [IO.Path]::AltDirectorySeparatorChar
    )
}

function Invoke-GitCapture {
    param(
        [string]$WorkingTree,
        [string[]]$Arguments
    )

    $traceNames = @(
        "GIT_TRACE",
        "GIT_TRACE_PACKET",
        "GIT_TRACE_CURL",
        "GIT_CURL_VERBOSE",
        "GIT_TRACE_SETUP",
        "GIT_TRACE_PERFORMANCE",
        "GIT_TRACE_SHALLOW",
        "GIT_TRACE2",
        "GIT_TRACE2_EVENT",
        "GIT_TRACE2_PERF"
    )
    $onePasswordNames = @(
        [Environment]::GetEnvironmentVariables("Process").Keys |
            Where-Object { ([string]$_).StartsWith("OP_", [StringComparison]::OrdinalIgnoreCase) }
    )
    $saved = @{}
    $savedErrorPreference = $ErrorActionPreference

    try {
        foreach ($name in $traceNames) {
            $saved[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
            $traceOverride = if ($name.StartsWith("GIT_TRACE2", [StringComparison]::Ordinal)) { "0" } else { $null }
            [Environment]::SetEnvironmentVariable($name, $traceOverride, "Process")
        }
        foreach ($name in $onePasswordNames) {
            $saved[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
            [Environment]::SetEnvironmentVariable($name, $null, "Process")
        }
        $ErrorActionPreference = "SilentlyContinue"
        $output = & $GitExecutable -c core.excludesFile= -C $WorkingTree @Arguments 2>$null
        $exitCode = $LASTEXITCODE
        return @{
            ExitCode = $exitCode
            Output = @($output)
        }
    }
    finally {
        $ErrorActionPreference = $savedErrorPreference
        foreach ($name in $saved.Keys) {
            [Environment]::SetEnvironmentVariable($name, $saved[$name], "Process")
        }
    }
}

function Get-RepositoryStatus {
    param(
        [string]$Label,
        [string]$Path,
        [switch]$RedactBranch
    )

    $valid = Invoke-GitCapture -WorkingTree $Path -Arguments @("rev-parse", "--show-toplevel")
    if ($valid.ExitCode -ne 0 -or @($valid.Output).Count -ne 1 -or (Get-FullPath ([string]$valid.Output[0])) -ne (Get-FullPath $Path)) {
        Write-Output "$Label repository: invalid or not independent"
        $script:RepositoryErrors = $true
        return
    }

    $branchResult = Invoke-GitCapture -WorkingTree $Path -Arguments @("branch", "--show-current")
    if ($branchResult.ExitCode -ne 0) {
        $branchStatus = "unavailable"
        $script:RepositoryErrors = $true
    }
    elseif ([string]::IsNullOrWhiteSpace([string]$branchResult.Output[0])) {
        $branchStatus = "detached HEAD"
    }
    elseif ($RedactBranch) {
        $branchStatus = "checked out (name redacted)"
    }
    else {
        $branchStatus = [string]$branchResult.Output[0]
    }

    $dirtyResult = Invoke-GitCapture -WorkingTree $Path -Arguments @("status", "--porcelain", "--untracked-files=normal")
    if ($dirtyResult.ExitCode -ne 0) {
        $dirty = "working tree unavailable"
        $script:RepositoryErrors = $true
    }
    elseif (@($dirtyResult.Output).Count -eq 0) {
        $dirty = "clean"
    }
    else {
        $dirty = "dirty"
    }

    $upstream = Invoke-GitCapture -WorkingTree $Path -Arguments @("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
    if ($upstream.ExitCode -ne 0) {
        $tracking = "no upstream configured"
    }
    else {
        $ahead = Invoke-GitCapture -WorkingTree $Path -Arguments @("rev-list", "--count", "@{upstream}..HEAD")
        $behind = Invoke-GitCapture -WorkingTree $Path -Arguments @("rev-list", "--count", "HEAD..@{upstream}")
        if ($ahead.ExitCode -ne 0 -or $behind.ExitCode -ne 0) {
            $tracking = "local tracking status unavailable"
        }
        else {
            $tracking = "ahead $([string]$ahead.Output[0]), behind $([string]$behind.Output[0]) (local tracking refs)"
        }
    }

    Write-Output "$Label repository: branch $branchStatus; $dirty; $tracking"
}

function Test-RootGitignoreProtection {
    param(
        [string]$Root,
        [string]$RelativePath
    )

    $ignored = Invoke-GitCapture -WorkingTree $Root -Arguments @("check-ignore", "-v", "--no-index", "--", $RelativePath)
    if ($ignored.ExitCode -ne 0 -or @($ignored.Output).Count -ne 1) {
        return $false
    }
    $record = [string]$ignored.Output[0]
    $source = ($record -split "`t", 2)[0]
    if ($source -notmatch '^\.gitignore:\d+:((?!!).+)$') {
        return $false
    }

    $committedIgnore = Invoke-GitCapture -WorkingTree $Root -Arguments @("show", "HEAD:.gitignore")
    if ($committedIgnore.ExitCode -ne 0) {
        return $false
    }

    $gitDirectory = Invoke-GitCapture -WorkingTree $Root -Arguments @("rev-parse", "--absolute-git-dir")
    if ($gitDirectory.ExitCode -ne 0 -or @($gitDirectory.Output).Count -ne 1) {
        return $false
    }

    $temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) "guardian-ignore-$([Guid]::NewGuid().ToString('N'))"
    $temporaryIgnore = Join-Path $temporaryRoot ".gitignore"
    try {
        [IO.Directory]::CreateDirectory($temporaryRoot) | Out-Null
        [IO.File]::WriteAllLines($temporaryIgnore, @($committedIgnore.Output), [Text.UTF8Encoding]::new($false))
        $committedResult = Invoke-GitCapture -WorkingTree $temporaryRoot -Arguments @(
            "--git-dir=$([string]$gitDirectory.Output[0])",
            "--work-tree=$temporaryRoot",
            "-c",
            "core.excludesFile=",
            "check-ignore",
            "-v",
            "--no-index",
            "--",
            $RelativePath
        )
        if ($committedResult.ExitCode -ne 0 -or @($committedResult.Output).Count -ne 1) {
            return $false
        }
        $committedSource = ([string]$committedResult.Output[0] -split "`t", 2)[0]
        return $committedSource -match '^\.gitignore:\d+:((?!!).+)$'
    }
    finally {
        if (Test-Path -LiteralPath $temporaryIgnore -PathType Leaf) {
            [IO.File]::Delete($temporaryIgnore)
        }
        if (Test-Path -LiteralPath $temporaryRoot -PathType Container) {
            [IO.Directory]::Delete($temporaryRoot, $false)
        }
    }
}

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $RepositoryRoot = Join-Path $PSScriptRoot ".."
}
$root = Get-FullPath $RepositoryRoot

if (-not (Test-Path -LiteralPath (Join-Path $root ".git"))) {
    Write-Error "Public repository: unavailable" -ErrorAction Continue
    exit 1
}

Get-RepositoryStatus -Label "Public" -Path $root

$privatePath = Join-Path $root "private"
$privateGit = Join-Path $privatePath ".git"
if (-not (Test-Path -LiteralPath $privatePath)) {
    Write-Output "Private repository: missing"
}
elseif ((Get-Item -LiteralPath $privatePath -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
    Write-Output "Private repository: invalid or not independent"
    $script:RepositoryErrors = $true
}
elseif (-not (Test-Path -LiteralPath $privateGit)) {
    Write-Output "Private repository: present but not initialized"
}
elseif (-not (Test-Path -LiteralPath $privateGit -PathType Container) -or ((Get-Item -LiteralPath $privateGit -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    Write-Output "Private repository: invalid or not independent"
    $script:RepositoryErrors = $true
}
else {
    Get-RepositoryStatus -Label "Private" -Path $privatePath -RedactBranch:(-not $IncludePrivateBranch)
}

$targets = @(
    ".env",
    "backend/api-service/.env",
    "frontend/.env.local",
    "k8s/api-service-secret.yaml"
)

Write-Output "Plaintext target status:"
foreach ($relativePath in $targets) {
    $fullPath = Join-Path $root ($relativePath.Replace('/', [IO.Path]::DirectorySeparatorChar))
    $presence = if (Test-Path -LiteralPath $fullPath -PathType Leaf) { "present" } else { "missing" }
    $tracked = Invoke-GitCapture -WorkingTree $root -Arguments @("ls-files", "--error-unmatch", "--", $relativePath)
    if ($tracked.ExitCode -eq 0) {
        $protection = "UNSAFE: tracked"
        $script:UnsafeTargets = $true
    }
    elseif (Test-RootGitignoreProtection -Root $root -RelativePath $relativePath) {
        $protection = "ignored by repository rule"
    }
    else {
        $protection = "UNSAFE: no repository ignore rule"
        $script:UnsafeTargets = $true
    }
    Write-Output "- ${relativePath}: $presence; $protection"
}

if ($script:UnsafeTargets -or $script:RepositoryErrors) {
    Write-Error "Workspace status found an invalid repository or an unprotected plaintext target." -ErrorAction Continue
    exit 1
}
exit 0
