#requires -Version 5.1

[CmdletBinding()]
param(
    [switch]$PrivateFromPrompt,
    [switch]$PrivateFromOnePassword,
    [Parameter(DontShow = $true)]
    [switch]$InternalCloneFromEnvironment,
    [Parameter(DontShow = $true)]
    [string]$RepositoryRoot,
    [Parameter(DontShow = $true)]
    [string]$GitExecutable = "git",
    [Parameter(DontShow = $true)]
    [string]$OpExecutable = "op"
)

$ErrorActionPreference = "Stop"
$DebugPreference = "SilentlyContinue"
$VerbosePreference = "SilentlyContinue"

trap {
    Write-Error "Private workspace setup stopped because an unexpected local error occurred. Existing workspace files were preserved." -ErrorAction Continue
    exit 1
}

function Stop-Safely {
    param([string]$Message)

    Write-Error $Message -ErrorAction Continue
    exit 1
}

function Get-FullPath {
    param([string]$Path)

    return [IO.Path]::GetFullPath($Path).TrimEnd(
        [IO.Path]::DirectorySeparatorChar,
        [IO.Path]::AltDirectorySeparatorChar
    )
}

function Test-ValidPrivateRepositoryUrl {
    param([string]$Value)

    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -match '[\x00-\x1f\x7f]') {
        return $false
    }

    $uri = $null
    if (-not [Uri]::TryCreate($Value, [UriKind]::Absolute, [ref]$uri)) {
        return $false
    }
    if ($uri.Host -ne "github.com" -or $uri.Query -or $uri.Fragment -or $uri.IsDefaultPort -eq $false) {
        return $false
    }
    if ($uri.AbsolutePath -match '%' -or $uri.AbsolutePath -notmatch '^/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:\.git)?$') {
        return $false
    }

    if ($uri.Scheme -eq "https") {
        return [string]::IsNullOrEmpty($uri.UserInfo)
    }
    if ($uri.Scheme -eq "ssh") {
        return $uri.UserInfo -eq "git"
    }
    return $false
}

function ConvertTo-GitConfigValue {
    param([string]$Value)

    $escaped = $Value.Replace('\', '\\').Replace('"', '\"')
    return '"' + $escaped + '"'
}

function Invoke-GitCapture {
    param(
        [string[]]$Arguments,
        [hashtable]$ExtraEnvironment
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
        if ($ExtraEnvironment) {
            foreach ($name in $ExtraEnvironment.Keys) {
                $saved[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
                [Environment]::SetEnvironmentVariable($name, $ExtraEnvironment[$name], "Process")
            }
        }

        $ErrorActionPreference = "SilentlyContinue"
        $output = & $GitExecutable @Arguments 2>$null
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

function Test-RootGitignoreProtection {
    param(
        [string]$Root,
        [string]$RelativePath
    )

    $ignored = Invoke-GitCapture -Arguments @(
        "-c",
        "core.excludesFile=",
        "-C",
        $Root,
        "check-ignore",
        "-v",
        "--no-index",
        "--",
        $RelativePath
    )
    if ($ignored.ExitCode -ne 0 -or @($ignored.Output).Count -ne 1) {
        return $false
    }
    $source = ([string]$ignored.Output[0] -split "`t", 2)[0]
    if ($source -notmatch '^\.gitignore:\d+:((?!!).+)$') {
        return $false
    }

    $committedIgnore = Invoke-GitCapture -Arguments @("-c", "core.excludesFile=", "-C", $Root, "show", "HEAD:.gitignore")
    if ($committedIgnore.ExitCode -ne 0) {
        return $false
    }

    $gitDirectory = Invoke-GitCapture -Arguments @("-C", $Root, "rev-parse", "--absolute-git-dir")
    if ($gitDirectory.ExitCode -ne 0 -or @($gitDirectory.Output).Count -ne 1) {
        return $false
    }

    $temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) "guardian-ignore-$([Guid]::NewGuid().ToString('N'))"
    $temporaryIgnore = Join-Path $temporaryRoot ".gitignore"
    try {
        [IO.Directory]::CreateDirectory($temporaryRoot) | Out-Null
        [IO.File]::WriteAllLines($temporaryIgnore, @($committedIgnore.Output), [Text.UTF8Encoding]::new($false))
        $committedResult = Invoke-GitCapture -Arguments @(
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

function Test-ReferenceFileStructure {
    param([string]$Path)

    $assignments = 0
    foreach ($line in [IO.File]::ReadAllLines($Path)) {
        $trimmed = $line.Trim()
        if ($trimmed.Length -eq 0 -or $trimmed.StartsWith("#")) {
            continue
        }
        $assignmentPrefix = "GUARDIAN_PRIVATE_REPOSITORY_URL="
        if (-not $trimmed.StartsWith($assignmentPrefix, [StringComparison]::Ordinal)) {
            return $false
        }
        $reference = $trimmed.Substring($assignmentPrefix.Length)
        if ($reference.StartsWith('"', [StringComparison]::Ordinal)) {
            if ($reference.Length -lt 2 -or -not $reference.EndsWith('"', [StringComparison]::Ordinal)) {
                return $false
            }
            $reference = $reference.Substring(1, $reference.Length - 2)
            if ($reference.Contains('"')) {
                return $false
            }
        }
        elseif ($reference.Contains('"') -or $reference -match '\s') {
            return $false
        }
        if (-not $reference.StartsWith('op://', [StringComparison]::OrdinalIgnoreCase)) {
            return $false
        }
        $parts = @($reference.Substring('op://'.Length) -split '/')
        if (
            $parts.Count -notin @(3, 4) -or
            @($parts | Where-Object { $_ -notmatch '^[A-Za-z0-9._ -]+$' -or $_ -notmatch '[A-Za-z0-9._-]' }).Count -ne 0
        ) {
            return $false
        }
        $assignments++
    }
    return $assignments -eq 1
}

function Test-PathHasReparsePoint {
    param(
        [string]$Root,
        [string]$Path
    )

    $rootPath = Get-FullPath $Root
    $rootPrefix = $rootPath + [IO.Path]::DirectorySeparatorChar
    $current = Get-FullPath $Path
    if (-not $current.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        return $true
    }

    while ($current.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        if (Test-Path -LiteralPath $current) {
            $item = Get-Item -LiteralPath $current -Force
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                return $true
            }
        }
        $parent = Split-Path -Parent $current
        if ($parent -eq $current) {
            break
        }
        $current = $parent
    }
    return $false
}

function Set-OriginUrlWithoutCommandArgument {
    param(
        [string]$GitConfigPath,
        [string]$RepositoryUrl
    )

    $lines = [IO.File]::ReadAllLines($GitConfigPath)
    $insideOrigin = $false
    $updated = $false
    $quotedUrl = ConvertTo-GitConfigValue $RepositoryUrl

    for ($index = 0; $index -lt $lines.Length; $index++) {
        if ($lines[$index] -match '^\s*\[') {
            $insideOrigin = $lines[$index] -match '^\s*\[remote\s+"origin"\]\s*$'
            continue
        }
        if ($insideOrigin -and $lines[$index] -match '^\s*url\s*=') {
            $lines[$index] = "`turl = $quotedUrl"
            $updated = $true
            break
        }
    }

    if (-not $updated) {
        Stop-Safely "The private clone did not contain the expected origin configuration. No workspace was installed."
    }
    [IO.File]::WriteAllLines($GitConfigPath, $lines, [Text.UTF8Encoding]::new($false))
}

function Install-PrivateRepository {
    param(
        [string]$Root,
        [string]$RepositoryUrl
    )

    if (-not (Test-ValidPrivateRepositoryUrl $RepositoryUrl)) {
        Stop-Safely "The private repository location must be a credential-free GitHub HTTPS or ssh://git@github.com URL."
    }

    $privatePath = Join-Path $Root "private"
    if (Test-Path -LiteralPath $privatePath) {
        Stop-Safely "The private workspace path already exists. Preserve and migrate its contents before cloning."
    }

    $stagingName = ".private-bootstrap-$([Guid]::NewGuid().ToString('N'))"
    $stagingPath = Join-Path $Root $stagingName
    $temporaryConfig = Join-Path ([IO.Path]::GetTempPath()) "guardian-tracker-git-$([Guid]::NewGuid().ToString('N')).config"
    $alias = "guardian-private:"

    try {
        $quotedUrl = ConvertTo-GitConfigValue $RepositoryUrl
        $configText = "[url $quotedUrl]`r`n`tinsteadOf = $alias`r`n"
        [IO.File]::WriteAllText($temporaryConfig, $configText, [Text.UTF8Encoding]::new($false))

        $clone = Invoke-GitCapture -Arguments @(
            "-c",
            "include.path=$temporaryConfig",
            "clone",
            "--origin",
            "origin",
            "--",
            $alias,
            $stagingPath
        )
        if ($clone.ExitCode -ne 0) {
            Stop-Safely "The private repository could not be cloned. Existing workspace files were not changed."
        }

        $gitDirectory = Join-Path $stagingPath ".git"
        if (-not (Test-Path -LiteralPath $gitDirectory -PathType Container)) {
            Stop-Safely "The clone did not produce an independent Git repository. No workspace was installed."
        }
        if ((Get-Item -LiteralPath $gitDirectory -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
            Stop-Safely "The cloned Git metadata failed the workspace safety check. No workspace was installed."
        }

        $topLevel = Invoke-GitCapture -Arguments @("-C", $stagingPath, "rev-parse", "--show-toplevel")
        if ($topLevel.ExitCode -ne 0 -or @($topLevel.Output).Count -ne 1) {
            Stop-Safely "The cloned repository could not be validated. No workspace was installed."
        }
        if ((Get-FullPath ([string]$topLevel.Output[0])) -ne (Get-FullPath $stagingPath)) {
            Stop-Safely "The cloned repository root failed validation. No workspace was installed."
        }

        Set-OriginUrlWithoutCommandArgument -GitConfigPath (Join-Path $gitDirectory "config") -RepositoryUrl $RepositoryUrl
        [IO.Directory]::Move($stagingPath, $privatePath)
        Write-Output "Private workspace clone installed."
    }
    finally {
        $RepositoryUrl = $null
        if (Test-Path -LiteralPath $temporaryConfig -PathType Leaf) {
            [IO.File]::Delete($temporaryConfig)
        }
        if (Test-Path -LiteralPath $stagingPath -PathType Container) {
            $resolvedStaging = Get-FullPath $stagingPath
            $expectedPrefix = (Get-FullPath $Root) + [IO.Path]::DirectorySeparatorChar + ".private-bootstrap-"
            if ($resolvedStaging.StartsWith($expectedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
                [IO.Directory]::Delete($resolvedStaging, $true)
            }
        }
    }
}

if ($PrivateFromPrompt -and $PrivateFromOnePassword) {
    Stop-Safely "Choose either interactive private setup or 1Password private setup, not both."
}
if ($InternalCloneFromEnvironment -and ($PrivateFromPrompt -or $PrivateFromOnePassword)) {
    Stop-Safely "The requested bootstrap modes cannot be combined."
}

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $RepositoryRoot = Join-Path $PSScriptRoot ".."
}
$root = Get-FullPath $RepositoryRoot
$internalRepositoryUrl = $null
if ($InternalCloneFromEnvironment) {
    foreach ($name in @([Environment]::GetEnvironmentVariables("Process").Keys)) {
        if (([string]$name).StartsWith("OP_", [StringComparison]::OrdinalIgnoreCase)) {
            [Environment]::SetEnvironmentVariable([string]$name, $null, "Process")
        }
    }
    $internalRepositoryUrl = [Environment]::GetEnvironmentVariable("GUARDIAN_PRIVATE_REPOSITORY_URL", "Process")
    [Environment]::SetEnvironmentVariable("GUARDIAN_PRIVATE_REPOSITORY_URL", $null, "Process")
    if ([string]::IsNullOrWhiteSpace($internalRepositoryUrl)) {
        Stop-Safely "1Password did not provide the private repository location. No workspace was installed."
    }
}

if (-not (Test-Path -LiteralPath (Join-Path $root ".git"))) {
    Stop-Safely "Run this helper from a Guardian Tracker public repository clone."
}
$publicCheck = Invoke-GitCapture -Arguments @("-C", $root, "rev-parse", "--show-toplevel")
if ($publicCheck.ExitCode -ne 0 -or @($publicCheck.Output).Count -ne 1 -or (Get-FullPath ([string]$publicCheck.Output[0])) -ne $root) {
    Stop-Safely "The public Guardian Tracker repository could not be validated."
}

$privatePath = Join-Path $root "private"
$privateGit = Join-Path $privatePath ".git"
if ((Test-Path -LiteralPath $privatePath) -and ((Get-Item -LiteralPath $privatePath -Force).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    Stop-Safely "The private workspace path failed the reparse-point safety check."
}
if (Test-Path -LiteralPath $privateGit) {
    if (-not (Test-Path -LiteralPath $privateGit -PathType Container)) {
        Stop-Safely "The private workspace Git metadata is not an independent repository directory."
    }
    if ((Get-Item -LiteralPath $privateGit -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
        Stop-Safely "The private workspace Git metadata failed the reparse-point safety check."
    }
    $privateCheck = Invoke-GitCapture -Arguments @("-C", $privatePath, "rev-parse", "--show-toplevel")
    if ($privateCheck.ExitCode -ne 0 -or @($privateCheck.Output).Count -ne 1 -or (Get-FullPath ([string]$privateCheck.Output[0])) -ne (Get-FullPath $privatePath)) {
        Stop-Safely "The private workspace repository could not be validated."
    }
    Write-Output "Private workspace clone already present."
    exit 0
}

if (($PrivateFromPrompt -or $PrivateFromOnePassword -or $InternalCloneFromEnvironment) -and (Test-Path -LiteralPath $privatePath)) {
    Stop-Safely "The private workspace path already exists. Preserve and migrate its contents before cloning."
}

if ($InternalCloneFromEnvironment) {
    try {
        Install-PrivateRepository -Root $root -RepositoryUrl $internalRepositoryUrl
    }
    finally {
        $internalRepositoryUrl = $null
    }
    exit 0
}

if ($PrivateFromPrompt) {
    $secureUrl = Read-Host "Private GitHub repository URL" -AsSecureString
    $bstr = [IntPtr]::Zero
    try {
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureUrl)
        $repositoryUrl = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
        Install-PrivateRepository -Root $root -RepositoryUrl $repositoryUrl
    }
    finally {
        $repositoryUrl = $null
        if ($bstr -ne [IntPtr]::Zero) {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
        }
    }
    exit 0
}

if ($PrivateFromOnePassword) {
    $referenceFile = Join-Path $root ".private-workspace\repository.env.ref"
    if (-not (Test-Path -LiteralPath $referenceFile -PathType Leaf)) {
        Stop-Safely "Create the ignored private repository reference file before using 1Password setup."
    }
    if (-not (Test-RootGitignoreProtection -Root $root -RelativePath ".private-workspace/repository.env.ref")) {
        Stop-Safely "The private repository reference file is not protected by the committed public ignore rules."
    }
    $referenceTracked = Invoke-GitCapture -Arguments @("-c", "core.excludesFile=", "-C", $root, "ls-files", "--error-unmatch", "--", ".private-workspace/repository.env.ref")
    if ($referenceTracked.ExitCode -eq 0 -or (Test-PathHasReparsePoint -Root $root -Path $referenceFile)) {
        Stop-Safely "The private repository reference file failed the local path safety check."
    }
    if (-not (Test-ReferenceFileStructure -Path $referenceFile)) {
        Stop-Safely "The private repository reference file must contain exactly one approved variable mapping."
    }
    if (-not (Get-Command -Name $OpExecutable -ErrorAction SilentlyContinue)) {
        Stop-Safely "1Password CLI is not available. Public-only setup remains available."
    }

    $shellExecutable = (Get-Process -Id $PID).Path
    $savedErrorPreference = $ErrorActionPreference
    $removedSecretReferences = @{}
    try {
        foreach ($entry in [Environment]::GetEnvironmentVariables("Process").GetEnumerator()) {
            if ([string]$entry.Value -match 'op://') {
                $removedSecretReferences[[string]$entry.Key] = [string]$entry.Value
                [Environment]::SetEnvironmentVariable([string]$entry.Key, $null, "Process")
            }
        }
        $ErrorActionPreference = "SilentlyContinue"
        $opOutput = & $OpExecutable run --env-file $referenceFile -- $shellExecutable -NoProfile -File $PSCommandPath -InternalCloneFromEnvironment -RepositoryRoot $root -GitExecutable $GitExecutable 2>$null
        $opExitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $savedErrorPreference
        foreach ($name in $removedSecretReferences.Keys) {
            [Environment]::SetEnvironmentVariable($name, $removedSecretReferences[$name], "Process")
        }
    }
    $opOutput = $null
    if ($opExitCode -ne 0 -or -not (Test-Path -LiteralPath $privateGit)) {
        Stop-Safely "1Password authorization or private workspace setup failed. Existing workspace files were not changed."
    }
    Write-Output "Private workspace clone installed through 1Password."
    exit 0
}

if (Test-Path -LiteralPath (Join-Path $root "private")) {
    Write-Output "Public workspace is ready. Existing private files are not yet an independent clone."
}
else {
    Write-Output "Public workspace is ready. Private restoration was not requested."
}
Write-Output "Use -PrivateFromPrompt or -PrivateFromOnePassword when private access is available."
