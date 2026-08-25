#requires -Version 5.1

[CmdletBinding()]
param(
    [ValidateSet("all", "root", "api", "frontend", "k8s")]
    [string[]]$Target = @("all"),
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
$currentOperation = "initial validation"

trap {
    $failureType = $_.Exception.GetType().Name
    $failureLine = $_.InvocationInfo.ScriptLineNumber
    Write-Error "Secret restoration stopped because an unexpected local error occurred during $currentOperation ($failureType at script line $failureLine). No existing target was overwritten." -ErrorAction Continue
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

function Invoke-ToolCapture {
    param(
        [string]$Executable,
        [string[]]$Arguments,
        [switch]$StripOnePasswordEnvironment
    )
    $onePasswordNames = if ($StripOnePasswordEnvironment) {
        @(
            [Environment]::GetEnvironmentVariables("Process").Keys |
                Where-Object { ([string]$_).StartsWith("OP_", [StringComparison]::OrdinalIgnoreCase) }
        )
    }
    else {
        @()
    }

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
        $output = & $Executable @Arguments 2>$null
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

    $ignored = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @(
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

    $committedIgnore = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @("-c", "core.excludesFile=", "-C", $Root, "show", "HEAD:.gitignore")
    if ($committedIgnore.ExitCode -ne 0) {
        return $false
    }

    $gitDirectory = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @("-C", $Root, "rev-parse", "--absolute-git-dir")
    if ($gitDirectory.ExitCode -ne 0 -or @($gitDirectory.Output).Count -ne 1) {
        return $false
    }

    $temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) "guardian-ignore-$([Guid]::NewGuid().ToString('N'))"
    $temporaryIgnore = Join-Path $temporaryRoot ".gitignore"
    try {
        [IO.Directory]::CreateDirectory($temporaryRoot) | Out-Null
        [IO.File]::WriteAllLines($temporaryIgnore, @($committedIgnore.Output), [Text.UTF8Encoding]::new($false))
        $committedResult = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @(
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

function Assert-SafePath {
    param(
        [string]$Root,
        [string]$Path
    )

    $rootPrefix = (Get-FullPath $Root) + [IO.Path]::DirectorySeparatorChar
    $fullPath = Get-FullPath $Path
    if (-not $fullPath.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        Stop-Safely "A secret target failed the repository containment check. Nothing was written."
    }

    $current = Split-Path -Parent $fullPath
    while ($current.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        if (Test-Path -LiteralPath $current) {
            $item = Get-Item -LiteralPath $current -Force
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                Stop-Safely "A secret target parent failed the reparse-point safety check. Nothing was written."
            }
        }
        $parent = Split-Path -Parent $current
        if ($parent -eq $current) {
            break
        }
        $current = $parent
    }
}

function Test-ResolvedStructure {
    param(
        [string]$Path,
        [string[]]$RequiredKeys,
        [ValidateSet("env", "yaml")]
        [string]$Format
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return $false
    }

    $lines = [IO.File]::ReadAllLines($Path)
    if ($lines.Count -eq 0) {
        return $false
    }
    if (@($lines | Where-Object { $_ -match 'op://|<vault>|<item>|<field>' }).Count -ne 0) {
        return $false
    }

    if ($Format -eq "env") {
        $values = @{}
        foreach ($line in $lines) {
            $trimmed = $line.Trim()
            if ($trimmed.Length -eq 0 -or $trimmed.StartsWith("#")) {
                continue
            }
            if ($line -notmatch '^([A-Za-z_][A-Za-z0-9_]*)=(.*)$') {
                return $false
            }
            $key = $Matches[1]
            if ($values.ContainsKey($key)) {
                return $false
            }
            $value = $Matches[2].Trim()
            $startsDouble = $value.StartsWith('"')
            $startsSingle = $value.StartsWith("'")
            if ($startsDouble -or $startsSingle) {
                $quote = if ($startsDouble) { '"' } else { "'" }
                if ($value.Length -lt 2 -or -not $value.EndsWith($quote)) {
                    return $false
                }
                $value = $value.Substring(1, $value.Length - 2)
                if ($value.Contains($quote)) {
                    return $false
                }
            }
            elseif ($value -match '["'']' -or $value -match '\s') {
                return $false
            }
            $values[$key] = $value
        }
        foreach ($key in $RequiredKeys) {
            if (-not $values.ContainsKey($key) -or [string]::IsNullOrWhiteSpace([string]$values[$key]) -or [string]$values[$key] -match '\s') {
                return $false
            }
        }
        return $true
    }

    $contentLines = @($lines | Where-Object { $_.Trim().Length -ne 0 })
    if (@($contentLines | Where-Object { $_ -match "`t" }).Count -ne 0) {
        return $false
    }
    $expectedPrefix = @(
        "apiVersion: v1",
        "kind: Secret",
        "metadata:",
        "  name: api-service-secrets",
        "  namespace: default",
        "type: Opaque",
        "stringData:"
    )
    if ($contentLines.Count -ne $expectedPrefix.Count + $RequiredKeys.Count) {
        return $false
    }
    for ($index = 0; $index -lt $expectedPrefix.Count; $index++) {
        if ($contentLines[$index] -cne $expectedPrefix[$index]) {
            return $false
        }
    }
    for ($index = 0; $index -lt $RequiredKeys.Count; $index++) {
        $line = $contentLines[$expectedPrefix.Count + $index]
        $pattern = '^  ' + [Regex]::Escape($RequiredKeys[$index]) + ': "([^"\\\x00-\x1f]+)"$'
        if ($line -notmatch $pattern) {
            return $false
        }
        if ([string]::IsNullOrWhiteSpace([string]$Matches[1])) {
            return $false
        }
    }
    return $true
}

if ([string]::IsNullOrWhiteSpace($RepositoryRoot)) {
    $RepositoryRoot = Join-Path $PSScriptRoot ".."
}
$root = Get-FullPath $RepositoryRoot

if (-not (Test-Path -LiteralPath (Join-Path $root ".git"))) {
    Stop-Safely "The public Guardian Tracker repository could not be validated."
}
$publicCheck = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @("-c", "core.excludesFile=", "-C", $root, "rev-parse", "--show-toplevel")
if ($publicCheck.ExitCode -ne 0 -or @($publicCheck.Output).Count -ne 1 -or (Get-FullPath ([string]$publicCheck.Output[0])) -ne $root) {
    Stop-Safely "The public Guardian Tracker repository could not be validated."
}

$privateRoot = Join-Path $root "private"
$privateGit = Join-Path $privateRoot ".git"
if (-not (Test-Path -LiteralPath $privateGit -PathType Container)) {
    Stop-Safely "Restore the independent private companion repository before restoring secret files."
}
if ((Get-Item -LiteralPath $privateRoot -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
    Stop-Safely "The private companion repository path failed the reparse-point safety check."
}
if ((Get-Item -LiteralPath $privateGit -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
    Stop-Safely "The private companion repository metadata failed validation."
}
$privateCheck = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @("-c", "core.excludesFile=", "-C", $privateRoot, "rev-parse", "--show-toplevel")
if ($privateCheck.ExitCode -ne 0 -or @($privateCheck.Output).Count -ne 1 -or (Get-FullPath ([string]$privateCheck.Output[0])) -ne (Get-FullPath $privateRoot)) {
    Stop-Safely "The private companion repository is invalid or not independent."
}
if (-not (Get-Command -Name $OpExecutable -ErrorAction SilentlyContinue)) {
    Stop-Safely "1Password CLI is not available. No secret files were written."
}

$definitions = @(
    @{
        Name = "root"
        Template = "private/bootstrap/templates/root.env.tpl"
        Output = ".env"
        Format = "env"
        RequiredKeys = @("GO_ENV", "BUNGIE_API_KEY", "BUNGIE_CLIENT_ID", "BUNGIE_CLIENT_SECRET", "JWT_SECRET", "POSTGRES_PASSWORD", "TOKEN_ENCRYPTION_KEY", "TOKEN_ENCRYPTION_KEY_VERSION")
    },
    @{
        Name = "api"
        Template = "private/bootstrap/templates/api-service.env.tpl"
        Output = "backend/api-service/.env"
        Format = "env"
        RequiredKeys = @("GO_ENV", "BUNGIE_API_KEY", "BUNGIE_CLIENT_ID", "BUNGIE_CLIENT_SECRET", "JWT_SECRET", "DATABASE_URL", "TOKEN_ENCRYPTION_KEY", "TOKEN_ENCRYPTION_KEY_VERSION")
    },
    @{
        Name = "frontend"
        Template = "private/bootstrap/templates/frontend.env.local.tpl"
        Output = "frontend/.env.local"
        Format = "env"
        RequiredKeys = @("VITE_API_URL")
    },
    @{
        Name = "k8s"
        Template = "private/bootstrap/templates/api-service-secret.yaml.tpl"
        Output = "k8s/api-service-secret.yaml"
        Format = "yaml"
        RequiredKeys = @("BUNGIE_API_KEY", "BUNGIE_CLIENT_ID", "BUNGIE_CLIENT_SECRET", "JWT_SECRET", "TOKEN_ENCRYPTION_KEY")
    }
)

$selected = if ($Target -contains "all") {
    $definitions
}
else {
    @($definitions | Where-Object { $Target -contains $_.Name })
}

$plans = @()
$currentOperation = "restore planning"
foreach ($definition in $selected) {
    $templatePath = Join-Path $root $definition.Template.Replace('/', [IO.Path]::DirectorySeparatorChar)
    $outputPath = Join-Path $root $definition.Output.Replace('/', [IO.Path]::DirectorySeparatorChar)
    Assert-SafePath -Root $root -Path $templatePath
    Assert-SafePath -Root $root -Path $outputPath

    if (-not (Test-Path -LiteralPath $templatePath -PathType Leaf)) {
        Stop-Safely "A required private restoration template is missing. Nothing was written for $($definition.Output)."
    }
    if ((Get-Item -LiteralPath $templatePath -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) {
        Stop-Safely "A private restoration template failed the reparse-point safety check. Nothing was written."
    }
    if (Test-Path -LiteralPath $outputPath) {
        Stop-Safely "$($definition.Output) already exists. It was not inspected or overwritten."
    }

    $tracked = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @(
        "-c",
        "core.excludesFile=",
        "-C",
        $root,
        "ls-files",
        "--error-unmatch",
        "--",
        $definition.Output
    )
    if ($tracked.ExitCode -eq 0 -or -not (Test-RootGitignoreProtection -Root $root -RelativePath $definition.Output)) {
        Stop-Safely "$($definition.Output) is not protected by the committed public ignore rules. Nothing was written."
    }

    $parent = Split-Path -Parent $outputPath
    $temporaryName = if ($definition.Name -eq "k8s") {
        "api-service-$([Guid]::NewGuid().ToString('N'))-secret.yaml"
    }
    else {
        ".env.guardian-inject-$([Guid]::NewGuid().ToString('N'))"
    }
    $temporaryPath = Join-Path $parent $temporaryName
    $relativeParent = Split-Path -Parent $definition.Output
    $temporaryRelativePath = if ([string]::IsNullOrWhiteSpace($relativeParent)) {
        $temporaryName
    }
    else {
        ($relativeParent.TrimEnd('/', '\') + "/" + $temporaryName).Replace('\', '/')
    }
    Assert-SafePath -Root $root -Path $temporaryPath

    if (Test-Path -LiteralPath $temporaryPath) {
        Stop-Safely "A script-owned restoration path already exists. No plaintext files were written."
    }
    $temporaryTracked = Invoke-ToolCapture -Executable $GitExecutable -StripOnePasswordEnvironment -Arguments @(
        "-c",
        "core.excludesFile=",
        "-C",
        $root,
        "ls-files",
        "--error-unmatch",
        "--",
        $temporaryRelativePath
    )
    if ($temporaryTracked.ExitCode -eq 0 -or -not (Test-RootGitignoreProtection -Root $root -RelativePath $temporaryRelativePath)) {
        Stop-Safely "A script-owned plaintext path is not protected by the committed public ignore rules. Nothing was written."
    }

    $plans += @{
        Definition = $definition
        TemplatePath = $templatePath
        OutputPath = $outputPath
        TemporaryPath = $temporaryPath
    }
}

$installed = @()
$completed = $false
try {
    foreach ($plan in $plans) {
        $definition = $plan.Definition
        $currentOperation = "1Password injection"
        $inject = Invoke-ToolCapture -Executable $OpExecutable -Arguments @(
            "inject",
            "--file-mode",
            "0600",
            "-i",
            $plan.TemplatePath,
            "-o",
            $plan.TemporaryPath
        )
        if ($inject.ExitCode -ne 0) {
            Stop-Safely "1Password could not restore $($definition.Output). No target file was installed."
        }
        $inject.Output = $null
        $currentOperation = "restored structure validation"
        if (-not (Test-ResolvedStructure -Path $plan.TemporaryPath -RequiredKeys $definition.RequiredKeys -Format $definition.Format)) {
            Stop-Safely "The restored structure for $($definition.Output) failed validation. No target file was installed."
        }
    }

    foreach ($plan in $plans) {
        $currentOperation = "validated target installation"
        [IO.File]::Move($plan.TemporaryPath, $plan.OutputPath)
        $installed += $plan.OutputPath
    }
    $completed = $true
    foreach ($plan in $plans) {
        Write-Output "$($plan.Definition.Output): restored"
    }
}
finally {
    foreach ($plan in $plans) {
        if (Test-Path -LiteralPath $plan.TemporaryPath -PathType Leaf) {
            [IO.File]::Delete($plan.TemporaryPath)
        }
    }
    if (-not $completed) {
        foreach ($outputPath in $installed) {
            if (Test-Path -LiteralPath $outputPath -PathType Leaf) {
                [IO.File]::Delete($outputPath)
            }
        }
    }
}
