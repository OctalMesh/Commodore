[CmdletBinding(PositionalBinding = $false)]
param(
    [Alias('e')]
    [string]$EnvFile,

    [Alias('n')]
    [string]$Name,

    [Alias('s')]
    [string]$Src,

    [Alias('o')]
    [string]$Out,

    [Alias('d')]
    [string]$Dir,

    [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
    [string[]]$Arguments
)

# ============================================================================ #
#                                 Admiral CLI                                  #
# ============================================================================ #
# Build and install tooling for Commodore CLI and projects built on top of the #
# Commodore SDK. Supports cross-platform builds, binary installation, alias    #
# linking, and automatic PATH management on Linux, macOS and Windows.          #
# ============================================================================ #
# Usage:                                                                       #
#   admiral [command] [flags]                                                  #
#                                                                              #
# Global Flags:                                                                #
#   -e, --env  <path>     Use custom dotenv file                               #
#   -n, --name <name>     Binary name          (overrides ADMIRAL_BINARY_NAME) #
#   -s, --src  <path>     Source path          (overrides ADMIRAL_SOURCE_DIR)  #
#   -o, --out  <path>     Build output dir     (overrides ADMIRAL_BUILD_DIR)   #
#   -d, --dir  <path>     Install directory    (overrides ADMIRAL_INSTALL_DIR) #
#                                                                              #
# Commands:                                                                    #
#   help                  Show this help message                               #
#                                                                              #
#   lint                  Run 'golangci-lint' via go tool                      #
#     -s, --src  <path>     Source path (module root)                          #
#                                                                              #
#   build                 Build cross-platform binaries                        #
#     -n, --name <name>     Binary name                                        #
#     -s, --src  <path>     Source path                                        #
#     -o, --out  <path>     Build output directory                             #
#                                                                              #
#   install               Build and install a CLI binary                       #
#     -n, --name  <name>    Binary name                                        #
#     -s, --src   <path>    Source path                                        #
#     -d, --dir   <path>    Install directory                                  #
#     -a, --alias <name>    Alias (repeatable)                                 #
#                                                                              #
#   uninstall             Remove an installed CLI binary                       #
#     -n, --name  <name>    Binary name                                        #
#     -d, --dir   <path>    Install directory                                  #
#     -a, --alias <name>    Alias (repeatable)                                 #
#                                                                              #
# Environment:                                                                 #
#   ADMIRAL_BINARY_NAME   Primary binary name                                  #
#   ADMIRAL_SOURCE_DIR    Default source path                                  #
#   ADMIRAL_BUILD_DIR     Cross-build output directory                         #
#   ADMIRAL_INSTALL_DIR   Destination directory                                #
# ============================================================================ #

$ErrorActionPreference = 'Stop'

# ============================================================================ #
#                                  Variables                                   #
# ============================================================================ #

$script:LAUNCH_DIR  = (Get-Location).Path
$script:BINARY_NAME = ''
$script:SOURCE_DIR  = ''
$script:BUILD_DIR   = ''
$script:INSTALL_DIR = ''

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
            $key   = $matches[1]
            $value = $matches[2].Trim()

            if ($value.StartsWith('"') -and $value.EndsWith('"') -and $value.Length -ge 2) {
                $value = $value.Substring(1, $value.Length - 2)
            } elseif ($value.StartsWith("'") -and $value.EndsWith("'") -and $value.Length -ge 2) {
                $value = $value.Substring(1, $value.Length - 2)
            }

            Set-Item -Path "Env:$key" -Value $value
        }
    }
}

# Returns env variable value or provided default when env value is empty.
#
# Arguments:
#   Name         - Environment variable name.
#   DefaultValue - Fallback value.
function Get-EnvOrDefault {
    param(
        [string]$Name,
        [string]$DefaultValue
    )

    $value = [Environment]::GetEnvironmentVariable($Name, 'Process')
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $DefaultValue
    }

    return $value
}

# Loads runtime configuration from environment and dotenv files.
# Flag overrides are applied last, after all env sources.
#
# Value precedence (lowest -> highest):
#   1) Built-in script defaults
#   2) .env.example
#   3) .env
#   4) Custom file from --env/-e
#   5) CLI flags (-n/--name, -s/--src, -o/--out, -d/--dir)
#
# Arguments:
#   CustomEnvPath - Optional custom .env file path.
#   FlagName      - Optional binary name override (from --name flag).
#   FlagSrc       - Optional source dir override  (from --src  flag).
#   FlagOut       - Optional build dir override   (from --out  flag).
#   FlagDir       - Optional install dir override (from --dir  flag).
function Initialize-EnvConfig {
    param(
        [string]$CustomEnvPath = '',
        [string]$FlagName      = '',
        [string]$FlagSrc       = '',
        [string]$FlagOut       = '',
        [string]$FlagDir       = ''
    )

    # 1) Built-in defaults
    $defaultBinaryName = 'commodore'
    $defaultSourceDir  = $script:LAUNCH_DIR
    $defaultBuildDir   = Join-Path $script:LAUNCH_DIR 'build'
    $defaultInstallDir = Join-Path $HOME '.local\bin'

    # 2) .env.example
    Import-DotenvFile -FilePath (Join-Path $script:LAUNCH_DIR '.env.example')

    # 3) .env
    Import-DotenvFile -FilePath (Join-Path $script:LAUNCH_DIR '.env')

    # 4) Custom file from --env/-e
    if (-not [string]::IsNullOrWhiteSpace($CustomEnvPath)) {
        $CustomEnvPath = Resolve-AbsolutePathValue -PathValue $CustomEnvPath -BasePath $script:LAUNCH_DIR
        Import-DotenvFile -FilePath $CustomEnvPath
    }

    # 5) Flag overrides (highest priority)
    if (-not [string]::IsNullOrWhiteSpace($FlagName)) { $env:ADMIRAL_BINARY_NAME = $FlagName }
    if (-not [string]::IsNullOrWhiteSpace($FlagSrc))  { $env:ADMIRAL_SOURCE_DIR  = $FlagSrc  }
    if (-not [string]::IsNullOrWhiteSpace($FlagOut))  { $env:ADMIRAL_BUILD_DIR   = $FlagOut  }
    if (-not [string]::IsNullOrWhiteSpace($FlagDir))  { $env:ADMIRAL_INSTALL_DIR = $FlagDir  }

    $script:BINARY_NAME = Get-EnvOrDefault -Name 'ADMIRAL_BINARY_NAME' -DefaultValue $defaultBinaryName
    $script:SOURCE_DIR  = Get-EnvOrDefault -Name 'ADMIRAL_SOURCE_DIR'  -DefaultValue $defaultSourceDir
    $script:BUILD_DIR   = Get-EnvOrDefault -Name 'ADMIRAL_BUILD_DIR'   -DefaultValue $defaultBuildDir
    $script:INSTALL_DIR = Get-EnvOrDefault -Name 'ADMIRAL_INSTALL_DIR' -DefaultValue $defaultInstallDir

    foreach ($pathVar in @('SOURCE_DIR', 'BUILD_DIR', 'INSTALL_DIR')) {
        $current    = Get-Variable -Name $pathVar -Scope Script -ValueOnly
        $normalized = Resolve-AbsolutePathValue -PathValue $current -BasePath $script:LAUNCH_DIR

        Set-Variable -Name $pathVar -Scope Script -Value $normalized
    }

    $env:ADMIRAL_BINARY_NAME = $script:BINARY_NAME
    $env:ADMIRAL_SOURCE_DIR  = $script:SOURCE_DIR
    $env:ADMIRAL_BUILD_DIR   = $script:BUILD_DIR
    $env:ADMIRAL_INSTALL_DIR = $script:INSTALL_DIR
}

# ============================================================================ #
#                                Flag Parsing                                  #
# ============================================================================ #

# Parse a known set of flags from an argument list.
# Returns a PSCustomObject with Name, Src, Out, Dir, Aliases, Rest.
#
# Recognised flags (all optional):
#   -n | --name  <value> -> Name
#   -s | --src   <value> -> Src
#   -o | --out   <value> -> Out
#   -d | --dir   <value> -> Dir
#   -a | --alias <value> -> Aliases (array, repeatable)
#
# Unknown positional args are collected into Rest.
# '--' stops flag parsing; everything after goes to Rest.
function Invoke-ParseFlags {
    param([string[]]$CliArgs)

    $result = [pscustomobject]@{
        Name    = ''
        Src     = ''
        Out     = ''
        Dir     = ''
        Aliases = [System.Collections.Generic.List[string]]::new()
        Rest    = [System.Collections.Generic.List[string]]::new()
    }

    $i = 0
    :whileLoop while ($i -lt $CliArgs.Count) {
        $arg = $CliArgs[$i]

        # End-of-flags sentinel
        if ($arg -eq '--') {
            $i++
            while ($i -lt $CliArgs.Count) {
                $result.Rest.Add($CliArgs[$i])
                $i++
            }
            break
        }

        switch -Regex ($arg) {
            # Binary name
            '^(-n|--name)$' {
                if ($i + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $result.Name = $CliArgs[$i + 1]; $i += 2; continue whileLoop
            }
            '^--name=(.+)$' {
                $result.Name = $Matches[1]; $i++; continue whileLoop
            }
            # Source path
            '^(-s|--src)$' {
                if ($i + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $result.Src = $CliArgs[$i + 1]; $i += 2; continue whileLoop
            }
            '^--src=(.+)$' {
                $result.Src = $Matches[1]; $i++; continue whileLoop
            }
            # Build output path
            '^(-o|--out)$' {
                if ($i + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $result.Out = $CliArgs[$i + 1]; $i += 2; continue whileLoop
            }
            '^--out=(.+)$' {
                $result.Out = $Matches[1]; $i++; continue whileLoop
            }
            # Install directory
            '^(-d|--dir)$' {
                if ($i + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $result.Dir = $CliArgs[$i + 1]; $i += 2; continue whileLoop
            }
            '^--dir=(.+)$' {
                $result.Dir = $Matches[1]; $i++; continue whileLoop
            }
            # Binary aliases (repeatable)
            '^(-a|--alias)$' {
                if ($i + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $result.Aliases.Add($CliArgs[$i + 1]); $i += 2; continue whileLoop
            }
            '^--alias=(.+)$' {
                $result.Aliases.Add($Matches[1]); $i++; continue whileLoop
            }
            # Unknown flag
            '^-' {
                Write-Fail "Unknown flag: $arg"; exit 1
            }
            # Positional / rest
            default {
                $result.Rest.Add($arg); $i++; continue whileLoop
            }
        }
    }

    return $result
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

# Returns $true when the directory is already listed in User/Process PATH.
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

# Resolves a path to absolute form, using BasePath for relative input.
# Supports '~' and '~/...' expansion.
#
# Arguments:
#   PathValue - Path to normalize.
#   BasePath  - Base directory for relative paths.
function Resolve-AbsolutePathValue {
    param(
        [string]$PathValue,
        [string]$BasePath = $script:LAUNCH_DIR
    )

    if ([string]::IsNullOrWhiteSpace($PathValue)) {
        return $PathValue
    }

    $expanded = [Environment]::ExpandEnvironmentVariables($PathValue)
    if ($expanded -eq '~') {
        $expanded = $HOME
    } elseif ($expanded.StartsWith('~/') -or $expanded.StartsWith('~\')) {
        $expanded = Join-Path $HOME $expanded.Substring(2)
    }

    if ([System.IO.Path]::IsPathRooted($expanded)) {
        return [System.IO.Path]::GetFullPath($expanded)
    }

    return [System.IO.Path]::GetFullPath((Join-Path $BasePath $expanded))
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

    $cleanedSource = (Resolve-AbsolutePathValue -PathValue $Source -BasePath $script:LAUNCH_DIR).TrimEnd('/', '\')
    if ([string]::IsNullOrWhiteSpace($cleanedSource)) {
        $cleanedSource = '.'
    }

    $candidates = @(
        $cleanedSource
        (Join-Path $cleanedSource "cmd/$Name")
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath (Join-Path $candidate 'main.go') -PathType Leaf) {
            return $candidate
        }
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
        BuildTarget      = $buildTarget
        ModuleRoot       = $moduleRoot
        PackageDirectory = $packageDirectory
        PackagePath      = $packagePath
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
#   BaseName   - Base executable name.
#   Label      - Output label (binary|alias).
#   InstallDir - Directory to remove from.
function Remove-One {
    param(
        [string]$BaseName,
        [string]$Label,
        [string]$InstallDir
    )

    $candidates = @(
        (Join-Path $InstallDir $BaseName)
        (Join-Path $InstallDir (Get-BinName -Base $BaseName))
    )

    $found = $candidates |
        Select-Object -Unique |
        Where-Object  { Test-Path -LiteralPath $_ } |
        Select-Object -First 1

    if ($found) {
        Remove-Item -LiteralPath $found -Force
        Write-Succ "Removed      $found  ($Label)"
    } else {
        Write-Warn "Not found:   $(Join-Path $InstallDir $BaseName)  ($Label, skipped)"
    }
}

# ============================================================================ #
#                              Table Formatting                                #
# ============================================================================ #

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

    $value = if ($null -eq $Text) { '' }
             else                 { $Text }
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
#   Length - Optional line width (default: 72).
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
        $text = if ($i -lt $Values.Count) { $Values[$i] }
                else                      { '' }
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

    $ok   = @($Rows | Where-Object { $_.Status -eq 'OK'   }).Count
    $fail = @($Rows | Where-Object { $_.Status -eq 'FAIL' }).Count
    $skip = @($Rows | Where-Object { $_.Status -eq 'SKIP' }).Count

    $widths = @(14, 8, 8, 36)

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

# Adds a directory to User PATH with confirmation,
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
    $existingEntries = if ([string]::IsNullOrWhiteSpace($current)) { @() }
                       else                                        { $current -split ';' }

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

    $updated = if ([string]::IsNullOrWhiteSpace($current)) { $Directory }
               else                                        { "$current;$Directory" }
    [Environment]::SetEnvironmentVariable('PATH', $updated, 'User')

    Write-Succ 'Added to User PATH (Windows registry).'
    Write-Info 'Restart your terminal for the change to take effect.'
}

# Removes a directory from User PATH after confirmation.
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

# Prints full usage information and examples.
function Show-Usage {
    @"

  Admiral CLI

  Usage:
    admiral [command] [flags]

  Global Flags:
    -e, --env  <path>     Use custom dotenv file
    -n, --name <name>     Binary name            (overrides ADMIRAL_BINARY_NAME)
    -s, --src  <path>     Source path            (overrides ADMIRAL_SOURCE_DIR)
    -o, --out  <path>     Build output dir       (overrides ADMIRAL_BUILD_DIR)
    -d, --dir  <path>     Install directory      (overrides ADMIRAL_INSTALL_DIR)

  Commands:
    help                  Show this help message

    lint                  Run 'golangci-lint' via go tool
      -s, --src  <path>     Source path (module root)

    build                 Build cross-platform binaries
      -n, --name <name>     Binary name
      -s, --src  <path>     Source path
      -o, --out  <path>     Build output directory

    install               Build and install a CLI binary
      -n, --name  <name>    Binary name
      -s, --src   <path>    Source path
      -d, --dir   <path>    Install directory
      -a, --alias <name>    Alias (repeatable)

    uninstall             Remove an installed CLI binary
      -n, --name  <name>    Binary name
      -d, --dir   <path>    Install directory
      -a, --alias <name>    Alias (repeatable)

  Environment:
    ADMIRAL_BINARY_NAME   Primary binary name          (default: $env:ADMIRAL_BINARY_NAME)
    ADMIRAL_SOURCE_DIR    Default source path          (default: $env:ADMIRAL_SOURCE_DIR)
    ADMIRAL_BUILD_DIR     Cross-build output directory (default: $env:ADMIRAL_BUILD_DIR)
    ADMIRAL_INSTALL_DIR   Destination directory        (default: $env:ADMIRAL_INSTALL_DIR)

"@ | Write-Host
}

# Handles 'lint' command.
#
# Flags:
#   -s, --src <path>   Source path to resolve module root from.
function Invoke-Lint {
    param([string[]]$RemainingArgs)

    $fp = Invoke-ParseFlags -CliArgs $RemainingArgs

    $src = if (-not [string]::IsNullOrWhiteSpace($fp.Src)) { $fp.Src }
           else                                            { $script:SOURCE_DIR }
    $src = Resolve-AbsolutePathValue -PathValue $src -BasePath $script:LAUNCH_DIR

    $moduleRoot = Find-GoModuleRoot -StartDirectory $src
    if (-not $moduleRoot) {
        Write-Fail "Go module root not found for '$src'."
        Write-Info 'Run lint inside a Go module or pass a path with --src.'
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

# Handles 'build' command.
#
# Flags:
#   -n, --name <name>   Binary name.
#   -s, --src  <path>   Source path.
#   -o, --out  <path>   Build output directory.
function Invoke-Build {
    param([string[]]$RemainingArgs)

    $fp = Invoke-ParseFlags -CliArgs $RemainingArgs

    $name = if (-not [string]::IsNullOrWhiteSpace($fp.Name)) { $fp.Name }
            else                                             { $script:BINARY_NAME }
    $src  = if (-not [string]::IsNullOrWhiteSpace($fp.Src))  { $fp.Src  }
            else                                             { $script:SOURCE_DIR  }
    $src  = Resolve-AbsolutePathValue -PathValue $src -BasePath $script:LAUNCH_DIR

    $buildContext = Resolve-BuildContext -Source $src -Name $name
    if (-not $buildContext) {
        Write-Fail "Build target not found for --src '$src' and --name '$name'."
        Write-Info "Expected one of: $src/main.go or $src/cmd/$name/main.go"
        exit 1
    }

    $outputDir = if (-not [string]::IsNullOrWhiteSpace($fp.Out)) { $fp.Out }
                 else                                            { $script:BUILD_DIR }
    $outputDir = Resolve-AbsolutePathValue -PathValue $outputDir -BasePath $script:LAUNCH_DIR

    New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

    Write-InfoDivider
    Write-Info "Building     $name"
    Write-Info "Source       $src"
    Write-Info "Target       $($buildContext.BuildTarget)"
    Write-Info "Module Root  $($buildContext.ModuleRoot)"
    Write-Info "Package Dir  $($buildContext.PackageDirectory)"
    Write-Info "Output Dir   $outputDir"
    Write-InfoDivider

    $targets = @(
        @{ GOOS = 'linux';   GOARCH = 'amd64' }
        @{ GOOS = 'darwin';  GOARCH = 'amd64' }
        @{ GOOS = 'darwin';  GOARCH = 'arm64' }
        @{ GOOS = 'windows'; GOARCH = 'amd64' }
    )

    $rows        = @()
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
        $status    = 'OK'

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

# Handles 'install' command.
#
# Flags:
#   -n, --name  <name>   Binary name.
#   -s, --src   <path>   Source path.
#   -d, --dir   <path>   Install directory.
#   -a, --alias <name>   Alias (repeatable).
function Invoke-Install {
    param([string[]]$RemainingArgs)

    $fp = Invoke-ParseFlags -CliArgs $RemainingArgs

    $name = if (-not [string]::IsNullOrWhiteSpace($fp.Name)) { $fp.Name }
            else                                             { $script:BINARY_NAME }
    $src  = if (-not [string]::IsNullOrWhiteSpace($fp.Src))  { $fp.Src  }
            else                                             { $script:SOURCE_DIR  }
    $src  = Resolve-AbsolutePathValue -PathValue $src -BasePath $script:LAUNCH_DIR

    $installDir = if (-not [string]::IsNullOrWhiteSpace($fp.Dir)) { $fp.Dir }
                  else                                            { $script:INSTALL_DIR }
    $installDir = Resolve-AbsolutePathValue -PathValue $installDir -BasePath $script:LAUNCH_DIR

    $binName      = Get-BinName -Base $name
    $buildContext = Resolve-BuildContext -Source $src -Name $name

    if (-not $buildContext) {
        Write-Fail "Build target not found for --src '$src' and --name '$name'."
        Write-Info "Expected one of: $src/main.go or $src/cmd/$name/main.go"
        exit 1
    }

    $buildArtifact = Join-Path (Get-Location).Path $binName

    Write-InfoDivider
    Write-Info "Building     $binName"
    Write-Info "Source       $src"
    Write-Info "Target       $($buildContext.BuildTarget)"
    Write-Info "Module Root  $($buildContext.ModuleRoot)"
    Write-Info "Package Dir  $($buildContext.PackageDirectory)"
    Write-Info "Destination  $installDir"
    Write-InfoDivider

    Invoke-GoBuild -ModuleRoot $buildContext.ModuleRoot -PackagePath $buildContext.PackagePath -OutputPath $buildArtifact
    Write-Succ     "Built        $binName"

    New-Item  -ItemType Directory -Path $installDir -Force | Out-Null
    Move-Item -LiteralPath $buildArtifact -Destination (Join-Path $installDir $binName) -Force
    Write-Succ "Installed    $(Join-Path $installDir $binName)"

    foreach ($aliasName in $fp.Aliases) {
        $aliasBin = Get-BinName -Base $aliasName

        Copy-Item  -LiteralPath (Join-Path $installDir $binName) -Destination (Join-Path $installDir $aliasBin) -Force
        Write-Succ "Copy         $(Join-Path $installDir $aliasBin)  ->  $binName"
    }

    Add-InstallDirToPath -Directory $installDir
}

# Handles 'uninstall' command.
#
# Flags:
#   -n, --name  <name>   Binary name.
#   -d, --dir   <path>   Install directory.
#   -a, --alias <name>   Alias (repeatable).
function Invoke-Uninstall {
    param([string[]]$RemainingArgs)

    $fp = Invoke-ParseFlags -CliArgs $RemainingArgs

    $name = if (-not [string]::IsNullOrWhiteSpace($fp.Name)) { $fp.Name }
            else                                             { $script:BINARY_NAME }

    $installDir = if (-not [string]::IsNullOrWhiteSpace($fp.Dir)) { $fp.Dir }
                  else                                            { $script:INSTALL_DIR }
    $installDir = Resolve-AbsolutePathValue -PathValue $installDir -BasePath $script:LAUNCH_DIR

    Remove-One -BaseName $name -Label 'binary' -InstallDir $installDir

    foreach ($aliasName in $fp.Aliases) {
        Remove-One -BaseName $aliasName -Label 'alias' -InstallDir $installDir
    }

    Remove-InstallDirFromPath -Directory $installDir
}

# ============================================================================ #
#                                  Entry Point                                 #
# ============================================================================ #

# Main CLI entrypoint.
function Main {
    param([string[]]$CliArgs)

    # Parse global flags that appear before the subcommand
    $gEnvFile = ''
    $gName    = ''
    $gSrc     = ''
    $gOut     = ''
    $gDir     = ''
    $index    = 0

    :whileLoop while ($index -lt $CliArgs.Count) {
        $arg = $CliArgs[$index]

        switch -Regex ($arg) {
            '^(-e|--env)$' {
                if ($index + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $gEnvFile = $CliArgs[$index + 1]; $index += 2; continue whileLoop
            }
            '^--env=(.+)$' {
                $gEnvFile = $Matches[1]; $index++; continue whileLoop
            }
            '^(-n|--name)$' {
                if ($index + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $gName = $CliArgs[$index + 1]; $index += 2; continue whileLoop
            }
            '^--name=(.+)$' {
                $gName = $Matches[1]; $index++; continue whileLoop
            }
            '^(-s|--src)$' {
                if ($index + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $gSrc = $CliArgs[$index + 1]; $index += 2; continue whileLoop
            }
            '^--src=(.+)$' {
                $gSrc = $Matches[1]; $index++; continue whileLoop
            }
            '^(-o|--out)$' {
                if ($index + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $gOut = $CliArgs[$index + 1]; $index += 2; continue whileLoop
            }
            '^--out=(.+)$' {
                $gOut = $Matches[1]; $index++; continue whileLoop
            }
            '^(-d|--dir)$' {
                if ($index + 1 -ge $CliArgs.Count) { Write-Fail "Missing value for $arg"; exit 1 }
                $gDir = $CliArgs[$index + 1]; $index += 2; continue whileLoop
            }
            '^--dir=(.+)$' {
                $gDir = $Matches[1]; $index++; continue whileLoop
            }
            default {
                break whileLoop
            }
        }
    }

    Initialize-EnvConfig `
        -CustomEnvPath $gEnvFile `
        -FlagName      $gName    `
        -FlagSrc       $gSrc     `
        -FlagOut       $gOut     `
        -FlagDir       $gDir

    $remaining   = [string[]]@(if ($index -lt $CliArgs.Count) { $CliArgs[$index..($CliArgs.Count - 1)] })
    $command     = if ($remaining.Count -gt 0) { [string]$remaining[0] } else { '' }
    $commandArgs = [string[]]@(if ($remaining.Count -gt 1) { $remaining[1..($remaining.Count - 1)] })

    switch ($command) {
        'install'   { Invoke-Install   -RemainingArgs $commandArgs }
        'build'     { Invoke-Build     -RemainingArgs $commandArgs }
        'lint'      { Invoke-Lint      -RemainingArgs $commandArgs }
        'uninstall' { Invoke-Uninstall -RemainingArgs $commandArgs }
        'help'      { Show-Usage }
        ''          { Show-Usage }
        default {
            Write-Fail "Unknown command: '$command'"
            Show-Usage
            exit 1
        }
    }
}

# Assemble entry args from PowerShell's own param() block

$entryArgs = [System.Collections.Generic.List[string]]::new()

if (-not [string]::IsNullOrWhiteSpace($EnvFile)) { $entryArgs.Add('--env');  $entryArgs.Add($EnvFile) }
if (-not [string]::IsNullOrWhiteSpace($Name))    { $entryArgs.Add('--name'); $entryArgs.Add($Name) }
if (-not [string]::IsNullOrWhiteSpace($Src))     { $entryArgs.Add('--src');  $entryArgs.Add($Src) }
if (-not [string]::IsNullOrWhiteSpace($Out))     { $entryArgs.Add('--out');  $entryArgs.Add($Out) }
if (-not [string]::IsNullOrWhiteSpace($Dir))     { $entryArgs.Add('--dir');  $entryArgs.Add($Dir) }

if ($Arguments) {
    foreach ($a in $Arguments) {
        $entryArgs.Add($a)
    }
}

Main -CliArgs $entryArgs.ToArray()

if ($MyInvocation.InvocationName -eq '&' -or [string]::IsNullOrWhiteSpace($MyInvocation.InvocationName)) {
    Read-Host "`nPress Enter to exit"
}
