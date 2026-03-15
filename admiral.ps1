[CmdletBinding(PositionalBinding = $false)]
param(
    [Alias('e', 'env')]
    [string]$EnvFile,

    [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
    [string[]]$Arguments
)

# ============================================================================ #
# Admiral CLI Tool                                                             #
#                                                                              #
# Description:                                                                 #
#   Single-file installer for Commodore CLI binaries on Windows.               #
#                                                                              #
# Usage:                                                                       #
#   ./admiral <command> [args]                                                 #
#                                                                              #
# Commands:                                                                    #
#   install   [path] [name] [alias...] Build and install a CLI binary.         #
#   build     [path] [name]            Build binaries for multiple OS targets. #
#   lint      [path]                   Run golangci-lint via `go tool`.        #
#   uninstall [name] [alias...]        Remove an installed CLI binary.         #
#   help                               Show this help message.                 #
# ============================================================================ #

$ErrorActionPreference = 'Stop'

# ============================================================================ #
#                                  Variables                                   #
# ============================================================================ #

$script:INSTALL_DIR         = ''
$script:BUILD_DIR           = ''
$script:DEFAULT_BINARY_NAME = ''
$script:LAUNCH_DIR          = (Get-Location).Path

# Box-drawing characters for table formatting

$script:CHR_H  = [char]0x2500
$script:CHR_V  = [char]0x2502
$script:CHR_TL = [char]0x250C
$script:CHR_TR = [char]0x2510
$script:CHR_BL = [char]0x2514
$script:CHR_BR = [char]0x2518
$script:CHR_T  = [char]0x252C
$script:CHR_B  = [char]0x2534
$script:CHR_L  = [char]0x251C
$script:CHR_R  = [char]0x2524
$script:CHR_X  = [char]0x253C

# ============================================================================ #
#                                  Logging                                     #
# ============================================================================ #

function Write-Note {
    param([string]$Message)
    Write-Host "         $Message"
}

function Write-Ask {
    param([string]$Message)
    Write-Host "         $Message " -NoNewline
}

function Write-Succ {
    param([string]$Message)
    Write-Host '[ SUCC ] ' -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Info {
    param([string]$Message)
    Write-Host "[ INFO ] $Message"
}

function Write-Warn {
    param([string]$Message)
    Write-Host '[ WARN ] ' -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Fail {
    param([string]$Message)
    Write-Host '[ FAIL ] ' -ForegroundColor Red -NoNewline
    Write-Host $Message
}

# ============================================================================ #
#                               Env Configuration                              #
# ============================================================================ #

# Loads a dotenv file and exports all parsed key/value pairs into process env.
#
# Arguments:
#   FilePath - Path to dotenv file.
function Import-DotenvFile {
    param([string]$FilePath)

    if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf)) {
        return
    }

    foreach ($raw in (Get-Content -LiteralPath $FilePath)) {
        if ($raw -match '^\s*#' -or $raw -match '^\s*$') {
            continue
        }

        if ($raw -match '^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$') {
            $name  = $matches[1]
            $value = $matches[2].Trim()

            if ($value.StartsWith('"') -and $value.EndsWith('"') -and $value.Length -ge 2) {
                $value = $value.Substring(1, $value.Length - 2)
            } elseif ($value.StartsWith("'") -and $value.EndsWith("'") -and $value.Length -ge 2) {
                $value = $value.Substring(1, $value.Length - 2)
            }

            Set-Item -Path "Env:$name" -Value $value
        }
    }
}

# Applies runtime config values.
#
# Arguments:
#   CustomEnvPath - Optional custom .env path.
function Initialize-EnvConfig {
    param([string]$CustomEnvPath)

    # 1) Built-in defaults
    $defaultInstallDir = Join-Path $HOME '.local\bin'
    $defaultBuildDir   = Join-Path $script:LAUNCH_DIR 'build'
    $defaultBinaryName = 'commodore'

    $env:INSTALL_DIR         = $defaultInstallDir
    $env:BUILD_DIR           = $defaultBuildDir
    $env:DEFAULT_BINARY_NAME = $defaultBinaryName

    # 2) .env.example
    Import-DotenvFile -FilePath (Join-Path $script:LAUNCH_DIR '.env.example')

    # 3) .env
    Import-DotenvFile -FilePath (Join-Path $script:LAUNCH_DIR '.env')

    # 4) Custom file from --env/-e
    if (-not [string]::IsNullOrWhiteSpace($CustomEnvPath)) {
        Import-DotenvFile -FilePath $CustomEnvPath
    }

    $script:INSTALL_DIR         = if ([string]::IsNullOrWhiteSpace($env:INSTALL_DIR))         { $defaultInstallDir } else { $env:INSTALL_DIR }
    $script:BUILD_DIR           = if ([string]::IsNullOrWhiteSpace($env:BUILD_DIR))           { $defaultBuildDir }   else { $env:BUILD_DIR }
    $script:DEFAULT_BINARY_NAME = if ([string]::IsNullOrWhiteSpace($env:DEFAULT_BINARY_NAME)) { $defaultBinaryName } else { $env:DEFAULT_BINARY_NAME }

    $env:INSTALL_DIR         = $script:INSTALL_DIR
    $env:BUILD_DIR           = $script:BUILD_DIR
    $env:DEFAULT_BINARY_NAME = $script:DEFAULT_BINARY_NAME
}

# Parses global options before command dispatch.
#
# Arguments:
#   CliArgs - Raw CLI args.
function Get-GlobalArgs {
    param([string[]]$CliArgs)

    $customEnv = ''
    $index     = 0

    while ($index -lt $CliArgs.Count) {
        $arg = $CliArgs[$index]

        if ($arg -eq '-e' -or $arg -eq '--env') {
            if ($index + 1 -ge $CliArgs.Count) {
                Write-Fail "Missing value for $arg"
                exit 1
            }

            $customEnv = $CliArgs[$index + 1]
            $index += 2
            continue
        }

        if ($arg.StartsWith('--env=')) {
            $customEnv = $arg.Substring(6)
            $index += 1
            continue
        }

        break
    }

    $remaining = if ($index -lt $CliArgs.Count) { $CliArgs[$index..($CliArgs.Count - 1)] } else { @() }

    return [pscustomobject]@{
        CustomEnv = $customEnv
        Remaining = $remaining
    }
}

# ============================================================================ #
#                                  Utilities                                   #
# ============================================================================ #

# Returns executable file name for a CLI command.
# Always adds .exe suffix unless already present.
#
# Arguments:
#   Base - Base binary name.
function Get-BinName {
    param([string]$Base)

    if ($Base.EndsWith('.exe', [System.StringComparison]::OrdinalIgnoreCase)) {
        return $Base
    }

    return "$Base.exe"
}

# Returns $true when two directories point to the same location (case-insensitive).
#
# Arguments:
#   LeftPath  - First path.
#   RightPath - Second path.
function Test-SameDirectory {
    param(
        [string]$LeftPath,
        [string]$RightPath
    )

    try {
        $left  = [System.IO.Path]::GetFullPath($LeftPath).TrimEnd('\')
        $right = [System.IO.Path]::GetFullPath($RightPath).TrimEnd('\')

        return $left.Equals($right, [System.StringComparison]::OrdinalIgnoreCase)
    } catch {
        return $false
    }
}

# Returns $true when INSTALL_DIR is already listed in User/Process PATH.
#
# Arguments:
#   Directory - Directory to search for in PATH.
function Test-InstallDirOnPath {
    param([string]$Directory)

    $pathEntries = @(
        [Environment]::GetEnvironmentVariable('PATH', 'User')
        [Environment]::GetEnvironmentVariable('PATH', 'Process')
    )

    foreach ($pathValue in $pathEntries) {
        if ([string]::IsNullOrWhiteSpace($pathValue)) {
            continue
        }

        foreach ($entry in ($pathValue -split ';')) {
            if ([string]::IsNullOrWhiteSpace($entry)) {
                continue
            }

            if (Test-SameDirectory -LeftPath $entry -RightPath $Directory) {
                return $true
            }
        }
    }

    return $false
}

# Resolves a directory path to absolute filesystem path.
#
# Arguments:
#   PathValue - Path to resolve.
function Resolve-AbsoluteDirectory {
    param([string]$PathValue)

    return (Resolve-Path -LiteralPath $PathValue).Path
}

# Detects nearest Go module root by walking up directories.
#
# Arguments:
#   StartDirectory - Directory to start search from.
function Find-GoModuleRoot {
    param([string]$StartDirectory)

    $current = [System.IO.DirectoryInfo](Resolve-AbsoluteDirectory -PathValue $StartDirectory)

    while ($null -ne $current) {
        if (Test-Path -LiteralPath (Join-Path $current.FullName 'go.mod') -PathType Leaf) {
            return $current.FullName
        }

        $current = $current.Parent
    }

    return $null
}

# Builds a module-relative Go package path from absolute directories.
#
# Arguments:
#   ModuleRoot       - Absolute module root.
#   PackageDirectory - Absolute package directory.
function Get-PackagePathFromRoot {
    param(
        [string]$ModuleRoot,
        [string]$PackageDirectory
    )

    $moduleUri  = [System.Uri]((Resolve-AbsoluteDirectory -PathValue $ModuleRoot).TrimEnd('\') + '\')
    $packageUri = [System.Uri]((Resolve-AbsoluteDirectory -PathValue $PackageDirectory).TrimEnd('\') + '\')

    if ($moduleUri.AbsoluteUri -eq $packageUri.AbsoluteUri) {
        return '.'
    }

    $relativeUri  = $moduleUri.MakeRelativeUri($packageUri)
    $relativePath = [System.Uri]::UnescapeDataString($relativeUri.ToString()).Replace('/', '\').TrimEnd('\')

    if ([string]::IsNullOrWhiteSpace($relativePath)) {
        return '.'
    }

    return ('./' + $relativePath.Replace('\\', '/'))
}

# Resolves a Go build target for installation.
# Priority:
# 1) <src>/main.go
# 2) <src>/cmd/<name>/main.go
#
# Arguments:
#   Source - Source root/path from command.
#   Name   - Binary name.
function Resolve-BuildTarget {
    param(
        [string]$Source,
        [string]$Name
    )

    $cleanedSource = $Source.TrimEnd('/', '\')
    if ([string]::IsNullOrWhiteSpace($cleanedSource)) {
        $cleanedSource = '.'
    }

    if (Test-Path -LiteralPath (Join-Path $cleanedSource 'main.go') -PathType Leaf) {
        return $cleanedSource
    }

    $cmdTarget = Join-Path $cleanedSource "cmd/$Name"
    if (Test-Path -LiteralPath (Join-Path $cmdTarget 'main.go') -PathType Leaf) {
        return $cmdTarget
    }

    return $null
}

# Resolves build target, module root and module-relative package path.
#
# Arguments:
#   Source - Source root/path from command.
#   Name   - Binary name.
function Resolve-BuildContext {
    param(
        [string]$Source,
        [string]$Name
    )

    $buildTarget = Resolve-BuildTarget -Source $Source -Name $Name
    if (-not $buildTarget) {
        return $null
    }

    $moduleRoot = Find-GoModuleRoot -StartDirectory $buildTarget
    if (-not $moduleRoot) {
        Write-Fail "Go module root not found for '$buildTarget'."
        Write-Info 'Add a go.mod file to the project root or pass a path inside an existing Go module.'
        exit 1
    }

    $packageDirectory = Resolve-AbsoluteDirectory -PathValue $buildTarget
    $packagePath      = Get-PackagePathFromRoot -ModuleRoot $moduleRoot -PackageDirectory $packageDirectory
    if (-not $packagePath) {
        Write-Fail "Could not determine Go package path for '$buildTarget'."
        exit 1
    }

    return [pscustomobject]@{
        BuildTarget = $buildTarget
        ModuleRoot  = $moduleRoot
        PackagePath = $packagePath
    }
}

# Executes go build from module root.
#
# Arguments:
#   ModuleRoot  - Absolute Go module root.
#   PackagePath - Relative package path.
#   OutputPath  - Build artifact output path.
function Invoke-GoBuild {
    param(
        [string]$ModuleRoot,
        [string]$PackagePath,
        [string]$OutputPath
    )

    Push-Location $ModuleRoot
    try {
        & go build -o $OutputPath $PackagePath
        if ($LASTEXITCODE -ne 0) {
            exit $LASTEXITCODE
        }
    } finally {
        Pop-Location
    }
}

# Removes one installed binary/alias from INSTALL_DIR.
# Tries both plain and .exe variants.
#
# Arguments:
#   BaseName - Base executable name.
#   Label    - Output label (binary|alias).
function Remove-One {
    param(
        [string]$BaseName,
        [string]$Label
    )

    $candidates = @(
        (Join-Path $script:INSTALL_DIR $BaseName)
        (Join-Path $script:INSTALL_DIR (Get-BinName -Base $BaseName))
    )

    $found = $candidates |
        Select-Object -Unique |
        Where-Object  { Test-Path -LiteralPath $_ } |
        Select-Object -First 1

    if ($found) {
        Remove-Item -LiteralPath $found -Force
        Write-Succ "Removed      $found  ($Label)"
    } else {
        Write-Warn "Not found:   $(Join-Path $script:INSTALL_DIR $BaseName)  ($Label, skipped)"
    }
}

# Formats a fixed-width table cell with ellipsis truncation.
#
# Arguments:
#   Text  - Cell text.
#   Width - Cell width.
function Format-TableCell {
    param(
        [string]$Text,
        [int]$Width
    )

    $value = if ($null -eq $Text) { '' } else { $Text }
    if ($value.Length -gt $Width) {
        $value = $value.Substring(0, [Math]::Max(0, $Width - 3)) + '...'
    }

    return $value.PadRight($Width)
}

# Returns a repeated character sequence with the given length.
#
# Arguments:
#   Length - Number of characters.
#   Char   - Character to repeat.
function Get-RepeatedChar {
    param(
        [int]$Length,
        [string]$Char
    )

    if ($Length -le 0) {
        return ''
    }

    return ($Char * $Length)
}

# Prints a decorative info divider line.
#
# Arguments:
#   Length - Optional line width.
function Write-InfoDivider {
    param([int]$Length = 72)
    Write-Info (Get-RepeatedChar -Length $Length -Char $script:CHR_H)
}

# Prints a table border for provided column widths.
#
# Arguments:
#   Left   - Left border symbol.
#   Middle - Middle separator symbol.
#   Right  - Right border symbol.
#   Widths - Column widths.
function Write-TableBorder {
    param(
        [string]$Left,
        [string]$Middle,
        [string]$Right,
        [int[]]$Widths
    )

    $parts = foreach ($width in $Widths) {
        "$script:CHR_H$(Get-RepeatedChar -Length $width -Char $script:CHR_H)$script:CHR_H"
    }

    Write-Note ($Left + ($parts -join $Middle) + $Right)
}

# Prints one formatted table row.
#
# Arguments:
#   Values - Cell values.
#   Widths - Cell widths.
function Write-TableRow {
    param(
        [string[]]$Values,
        [int[]]$Widths
    )

    $cells = for ($i = 0; $i -lt $Widths.Count; $i++) {
        $text = if ($i -lt $Values.Count) { $Values[$i] } else { '' }
        Format-TableCell -Text $text -Width $Widths[$i]
    }

    $sep = " $script:CHR_V "
    Write-Note ("$script:CHR_V " + ($cells -join $sep) + " $script:CHR_V")
}

# Prints a build summary table for cross-platform artifacts.
#
# Arguments:
#   Rows - Collection with Target/Status/Duration/Artifact properties.
function Write-BuildSummaryTable {
    param([object[]]$Rows)

    if (-not $Rows -or $Rows.Count -eq 0) {
        return
    }

    $ok    = @($Rows | Where-Object { $_.Status -eq 'OK'   }).Count
    $fail  = @($Rows | Where-Object { $_.Status -eq 'FAIL' }).Count
    $skip  = @($Rows | Where-Object { $_.Status -eq 'SKIP' }).Count

    $wTarget   = 14
    $wStatus   = 8
    $wDuration = 8
    $wArtifact = 36

    $widths = @($wTarget, $wStatus, $wDuration, $wArtifact)

    Write-Note ''
    Write-TableBorder -Left $script:CHR_TL -Middle $script:CHR_T -Right $script:CHR_TR -Widths $widths
    Write-TableRow    -Values @('Target', 'Status', 'Duration', 'Artifact') -Widths $widths
    Write-TableBorder -Left $script:CHR_L  -Middle $script:CHR_X -Right $script:CHR_R  -Widths $widths

    foreach ($row in $Rows) {
        Write-TableRow -Values @($row.Target, $row.Status, $row.Duration, $row.Artifact) -Widths $widths
    }

    Write-TableBorder -Left $script:CHR_BL -Middle $script:CHR_B -Right $script:CHR_BR -Widths $widths
    Write-Note "  Totals: $ok built  $fail failed  $skip skipped"
    Write-Note ''
}

# ============================================================================ #
#                                PATH Management                               #
# ============================================================================ #

# Adds INSTALL_DIR to user PATH with confirmation,
# or prints manual setup instructions.
#
# Arguments:
#   Directory - Directory to add.
function Add-InstallDirToPath {
    param([string]$Directory)

    if (Test-InstallDirOnPath -Directory $Directory) {
        return
    }

    Write-Warn "$Directory is not in PATH"
    Write-Ask  'Automatically add it? [Y/n]'

    $answer = Read-Host
    if ([string]::IsNullOrWhiteSpace($answer)) {
        $answer = 'Y'
    }

    if ($answer -notmatch '^[Yy]$') {
        Write-Info "Manual setup required for: $Directory"
        return
    }

    $current         = [Environment]::GetEnvironmentVariable('PATH', 'User')
    $existingEntries = if ([string]::IsNullOrWhiteSpace($current)) { @() } else { $current -split ';' }

    $alreadyPresent = $false
    foreach ($entry in $existingEntries) {
        if (Test-SameDirectory -LeftPath $entry -RightPath $Directory) {
            $alreadyPresent = $true
            break
        }
    }

    if ($alreadyPresent) {
        Write-Succ 'Already present in User PATH.'
        return
    }

    $updated = if ([string]::IsNullOrWhiteSpace($current)) { $Directory } else { "$current;$Directory" }
    [Environment]::SetEnvironmentVariable('PATH', $updated, 'User')

    Write-Succ 'Added to User PATH (Windows registry).'
    Write-Info 'Restart your terminal for the change to take effect.'
}

# Removes INSTALL_DIR from User PATH after confirmation.
#
# Arguments:
#   Directory - Directory to remove.
function Remove-InstallDirFromPath {
    param([string]$Directory)

    if (-not (Test-InstallDirOnPath -Directory $Directory)) {
        return
    }

    Write-Info "$Directory is currently in PATH."
    Write-Ask  'Automatically remove it? [y/N]'

    $answer = Read-Host
    if ([string]::IsNullOrWhiteSpace($answer)) {
        $answer = 'N'
    }

    if ($answer -notmatch '^[Yy]$') {
        return
    }

    $current = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ([string]::IsNullOrWhiteSpace($current)) {
        return
    }

    $entries  = $current -split ';'
    $filtered = @()

    foreach ($entry in $entries) {
        if ([string]::IsNullOrWhiteSpace($entry)) {
            continue
        }

        if (-not (Test-SameDirectory -LeftPath $entry -RightPath $Directory)) {
            $filtered += $entry
        }
    }

    [Environment]::SetEnvironmentVariable('PATH', ($filtered -join ';'), 'User')
    Write-Succ 'Removed from User PATH (Windows registry).'
    Write-Info 'Restart your terminal for the change to take effect.'
}

# ============================================================================ #
#                               Commands                                       #
# ============================================================================ #

# Handles `install` command.
#
# Arguments:
#   RemainingArgs - Optional [path] [name] [alias...]
function Invoke-Install {
    param([string[]]$RemainingArgs)

    $source  = '.'
    $name    = $script:DEFAULT_BINARY_NAME
    $aliases = @()

    switch ($RemainingArgs.Count) {
        0 { }
        1 {
            $source = $RemainingArgs[0]
        }
        default {
            $source = $RemainingArgs[0]
            $name   = $RemainingArgs[1]

            if ($RemainingArgs.Count -gt 2) {
                $aliases = $RemainingArgs[2..($RemainingArgs.Count - 1)]
            }
        }
    }

    $binName      = Get-BinName -Base $name
    $buildContext = Resolve-BuildContext -Source $source -Name $name

    if (-not $buildContext) {
        Write-Fail "Build target not found for source '$source' and binary '$name'."
        Write-Info "Expected one of: $source/main.go or $source/cmd/$name/main.go"
        exit 1
    }

    $buildArtifact = Join-Path (Get-Location).Path $binName

    Write-InfoDivider
    Write-Info "Building     $binName"
    Write-Info "Source       $source"
    Write-Info "Target       $($buildContext.BuildTarget)"
    Write-Info "Module Root  $($buildContext.ModuleRoot)"
    Write-Info "Package      $($buildContext.PackagePath)"
    Write-Info "Destination  $script:INSTALL_DIR"
    Write-InfoDivider

    Invoke-GoBuild -ModuleRoot $buildContext.ModuleRoot -PackagePath $buildContext.PackagePath -OutputPath $buildArtifact
    Write-Succ     "Built        $binName"

    New-Item   -ItemType Directory -Path $script:INSTALL_DIR -Force | Out-Null
    Move-Item  -LiteralPath $buildArtifact -Destination (Join-Path $script:INSTALL_DIR $binName) -Force
    Write-Succ "Installed    $(Join-Path $script:INSTALL_DIR $binName)"

    foreach ($aliasName in $aliases) {
        $aliasBin = Get-BinName -Base $aliasName

        Copy-Item  -LiteralPath (Join-Path $script:INSTALL_DIR $binName) -Destination (Join-Path $script:INSTALL_DIR $aliasBin) -Force
        Write-Succ "Copy         $(Join-Path $script:INSTALL_DIR $aliasBin)  ->  $binName"
    }

    Add-InstallDirToPath -Directory $script:INSTALL_DIR
}

# Handles `uninstall` command.
#
# Arguments:
#   RemainingArgs - Optional [name] [alias...]
function Invoke-Uninstall {
    param([string[]]$RemainingArgs)

    $name    = $script:DEFAULT_BINARY_NAME
    $aliases = @()

    if ($RemainingArgs.Count -gt 0) {
        $name = $RemainingArgs[0]

        if ($RemainingArgs.Count -gt 1) {
            $aliases = $RemainingArgs[1..($RemainingArgs.Count - 1)]
        }
    }

    Remove-One -BaseName $name -Label 'binary'
    foreach ($aliasName in $aliases) {
        Remove-One -BaseName $aliasName -Label 'alias'
    }

    Remove-InstallDirFromPath -Directory $script:INSTALL_DIR
}

# Handles `build` command.
#
# Arguments:
#   RemainingArgs - Optional [path] [name]
function Invoke-Build {
    param([string[]]$RemainingArgs)

    $source = '.'
    $name   = $script:DEFAULT_BINARY_NAME

    switch ($RemainingArgs.Count) {
        0 { }
        1 {
            $source = $RemainingArgs[0]
        }
        default {
            $source = $RemainingArgs[0]
            $name   = $RemainingArgs[1]
        }
    }

    $buildContext = Resolve-BuildContext -Source $source -Name $name
    if (-not $buildContext) {
        Write-Fail "Build target not found for source '$source' and binary '$name'."
        Write-Info "Expected one of: $source/main.go or $source/cmd/$name/main.go"
        exit 1
    }

    $cwd       = (Get-Location).Path
    $outputDir = if ($script:BUILD_DIR) { $script:BUILD_DIR } elseif ($env:BUILD_DIR) { $env:BUILD_DIR } else { Join-Path $cwd 'build' }

    Write-InfoDivider
    Write-Info "Building     $name"
    Write-Info "Source       $source"
    Write-Info "Target       $($buildContext.BuildTarget)"
    Write-Info "Module Root  $($buildContext.ModuleRoot)"
    Write-Info "Package      $($buildContext.PackagePath)"
    Write-Info "Output Dir   $outputDir"
    Write-InfoDivider

    New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

    $targets = @(
        @{ GOOS = 'linux';   GOARCH = 'amd64' }
        @{ GOOS = 'darwin';  GOARCH = 'amd64' }
        @{ GOOS = 'darwin';  GOARCH = 'arm64' }
        @{ GOOS = 'windows'; GOARCH = 'amd64' }
    )

    $rows = @()
    $hasFailures = $false

    foreach ($target in $targets) {
        $goos   = $target.GOOS
        $goarch = $target.GOARCH

        $artifactName = "${name}_${goos}_${goarch}"
        if ($goos -eq 'windows') {
            $artifactName = "$artifactName.exe"
        }

        $artifactPath = Join-Path $outputDir $artifactName
        Write-Info "Cross-build  $goos/$goarch -> $artifactPath"

        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        $status = 'OK'

        Push-Location $buildContext.ModuleRoot
        try {
            $env:GOOS        = $goos
            $env:GOARCH      = $goarch
            $env:CGO_ENABLED = '0'

            & go build -o $artifactPath $buildContext.PackagePath
            if ($LASTEXITCODE -ne 0) {
                $status      = 'FAIL'
                $hasFailures = $true

                Write-Fail "Build failed for $goos/$goarch"
            } else {
                Write-Succ "Built        $artifactPath"
            }
        } finally {
            Remove-Item Env:GOOS        -ErrorAction SilentlyContinue
            Remove-Item Env:GOARCH      -ErrorAction SilentlyContinue
            Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
            Pop-Location
            $stopwatch.Stop()
        }

        $rows += [pscustomobject]@{
            Target   = "$goos/$goarch"
            Status   = $status
            Duration = ([Math]::Round($stopwatch.Elapsed.TotalMilliseconds, 0).ToString() + 'ms')
            Artifact = [System.IO.Path]::GetFileName($artifactPath)
        }
    }

    Write-BuildSummaryTable -Rows $rows

    if ($hasFailures) {
        exit 1
    }
}

# Handles `lint` command.
#
# Arguments:
#   RemainingArgs - Optional [path] used to resolve module root.
function Invoke-Lint {
    param([string[]]$RemainingArgs)

    $source     = if ($RemainingArgs.Count -gt 0) { $RemainingArgs[0] } else { '.' }
    $moduleRoot = Find-GoModuleRoot -StartDirectory $source

    if (-not $moduleRoot) {
        Write-Fail "Go module root not found for '$source'."
        Write-Info 'Run lint inside a Go module or pass a path inside one.'
        exit 1
    }

    Write-Info "Linting      $moduleRoot"
    Write-Info 'Command      go tool golangci-lint run ./...'

    Push-Location $moduleRoot
    try {
        & go tool golangci-lint run ./...
        if ($LASTEXITCODE -ne 0) {
            Write-Fail 'Lint failed.'
            exit $LASTEXITCODE
        }
    } finally {
        Pop-Location
    }

    Write-Succ 'Lint passed'
}

# Prints full usage information and examples.
function Show-Usage {
    @"

  Admiral CLI

  Usage:
    admiral [options] <command> [args]

  Options:
    -e, --env <path>                     Use custom dotenv file

  Commands:
    install   [path] [name] [alias...]   Build and install a CLI binary
    build     [path] [name]              Build Linux/macOS/Windows binaries
    lint      [path]                     Run 'golangci-lint' via go tool
    uninstall [name] [alias...]          Remove an installed CLI binary
    help                                 Show this help message

  Examples:
    ./admiral install                    Install standard CLI binary
    ./admiral build                      Cross-build into ./build/
    ./admiral lint                       Run linter from module root
    ./admiral install . commodore cdr    Install with an alias
    ./admiral install ./cmd commodore    Build from custom source root
    ./admiral uninstall                  Uninstall standard CLI binary
    ./admiral uninstall commodore cdr    Remove binary and alias

  Environment:
    INSTALL_DIR          Destination directory        (default: $INSTALL_DIR)
    BUILD_DIR            Cross-build output directory (default: $BUILD_DIR)
    DEFAULT_BINARY_NAME  Primary binary name          (default: $DEFAULT_BINARY_NAME)

"@ | Write-Host
}

# ============================================================================ #
#                                  Entry Point                                 #
# ============================================================================ #

# Main CLI entrypoint.
#
# Arguments:
#   CliArgs - Raw command-line arguments.
function Main {
    param([string[]]$CliArgs)

    $parsedGlobal = Get-GlobalArgs -CliArgs $CliArgs
    Initialize-EnvConfig -CustomEnvPath $parsedGlobal.CustomEnv

    $CliArgs = $parsedGlobal.Remaining

    $command   = if ($CliArgs.Count -gt 0) { $CliArgs[0] }                       else { '' }
    $remaining = if ($CliArgs.Count -gt 1) { $CliArgs[1..($CliArgs.Count - 1)] } else { @() }

    switch ($command) {
        'install'   { Invoke-Install   -RemainingArgs $remaining }
        'build'     { Invoke-Build     -RemainingArgs $remaining }
        'lint'      { Invoke-Lint      -RemainingArgs $remaining }
        'uninstall' { Invoke-Uninstall -RemainingArgs $remaining }
        'help'      { Show-Usage }
        ''          { Show-Usage }
        default {
            Write-Fail "Unknown command: '$command'";
            Show-Usage;
            exit 1;
        }
    }
}

$entryArgs = @()
if (-not [string]::IsNullOrWhiteSpace($EnvFile)) {
    $entryArgs += @('--env', $EnvFile)
}
if ($Arguments) {
    $entryArgs += $Arguments
}

Main -CliArgs $entryArgs
