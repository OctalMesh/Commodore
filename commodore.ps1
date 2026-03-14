[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Arguments
)

$ErrorActionPreference = 'Stop'

$COMMODORE_VERSION = '0.1.0'
$script:INSTALL_DIR = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $HOME '.local\bin' }

function Write-Ok {
    param([string]$Message)
    Write-Host '[  OK  ] ' -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Info {
    param([string]$Message)
    Write-Host "[ INFO ] $Message"
}

function Write-WarnMessage {
    param([string]$Message)
    Write-Host '[ WARN ] ' -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Fail {
    param([string]$Message)
    Write-Host '[ FAIL ] ' -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Get-BinName {
    param([string]$Base)

    if ($Base.EndsWith('.exe', [System.StringComparison]::OrdinalIgnoreCase)) {
        return $Base
    }

    return "$Base.exe"
}

function Test-InstallDirOnPath {
    param([string]$Directory)

    $target = [System.IO.Path]::GetFullPath($Directory).TrimEnd('\')
    $pathEntries = [Environment]::GetEnvironmentVariable('PATH', 'User'), [Environment]::GetEnvironmentVariable('PATH', 'Process')

    foreach ($pathValue in $pathEntries) {
        if ([string]::IsNullOrWhiteSpace($pathValue)) {
            continue
        }

        foreach ($entry in ($pathValue -split ';')) {
            if ([string]::IsNullOrWhiteSpace($entry)) {
                continue
            }

            try {
                $candidate = [System.IO.Path]::GetFullPath($entry).TrimEnd('\')
                if ($candidate.Equals($target, [System.StringComparison]::OrdinalIgnoreCase)) {
                    return $true
                }
            }
            catch {
            }
        }
    }

    return $false
}

function Ensure-OnPath {
    param([string]$Directory)

    if (Test-InstallDirOnPath -Directory $Directory) {
        return
    }

    Write-Host ''
    Write-WarnMessage "$Directory is not in PATH"
    $answer = Read-Host '  Automatically add it? [Y/n]'
    if ([string]::IsNullOrWhiteSpace($answer)) {
        $answer = 'Y'
    }

    if ($answer -match '^[Yy]$') {
        $current = [Environment]::GetEnvironmentVariable('PATH', 'User')
        $normalizedCurrent = if ([string]::IsNullOrWhiteSpace($current)) { @() } else { $current -split ';' }

        if (-not ($normalizedCurrent | Where-Object { $_.TrimEnd('\').Equals($Directory.TrimEnd('\'), [System.StringComparison]::OrdinalIgnoreCase) })) {
            $updated = if ([string]::IsNullOrWhiteSpace($current)) { $Directory } else { "$current;$Directory" }
            [Environment]::SetEnvironmentVariable('PATH', $updated, 'User')
            Write-Ok 'Added to User PATH (Windows registry).'
            Write-Info 'Restart your terminal for the change to take effect.'
        }
        else {
            Write-Ok 'Already present in User PATH.'
        }
    }
    else {
        Write-Host ''
        Write-Info 'Manual setup:'
        Write-Host '  System Settings -> Environment Variables -> User PATH -> append:'
        Write-Host "    $Directory"
    }

    Write-Host ''
}

function Resolve-AbsoluteDirectory {
    param([string]$PathValue)

    return (Resolve-Path -LiteralPath $PathValue).Path
}

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

function Get-PackagePathFromRoot {
    param(
        [string]$ModuleRoot,
        [string]$PackageDirectory
    )

    $moduleUri = [System.Uri]((Resolve-AbsoluteDirectory -PathValue $ModuleRoot).TrimEnd('\') + '\')
    $packageUri = [System.Uri]((Resolve-AbsoluteDirectory -PathValue $PackageDirectory).TrimEnd('\') + '\')

    if ($moduleUri.AbsoluteUri -eq $packageUri.AbsoluteUri) {
        return '.'
    }

    $relativeUri = $moduleUri.MakeRelativeUri($packageUri)
    $relativePath = [System.Uri]::UnescapeDataString($relativeUri.ToString()).Replace('/', '\').TrimEnd('\')

    if ([string]::IsNullOrWhiteSpace($relativePath)) {
        return '.'
    }

    return ('./' + $relativePath.Replace('\', '/'))
}

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
    $packagePath = Get-PackagePathFromRoot -ModuleRoot $moduleRoot -PackageDirectory $packageDirectory
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
    }
    finally {
        Pop-Location
    }
}

function Remove-One {
    param(
        [string]$BaseName,
        [string]$Label
    )

    $candidates = @(
        (Join-Path $script:INSTALL_DIR $BaseName)
        (Join-Path $script:INSTALL_DIR (Get-BinName -Base $BaseName))
    )

    $found = $candidates | Select-Object -Unique | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
    if ($found) {
        Remove-Item -LiteralPath $found -Force
        Write-Ok "Removed      $found  ($Label)"
    }
    else {
        Write-WarnMessage "Not found:   $(Join-Path $script:INSTALL_DIR $BaseName)  ($Label, skipped)"
    }
}

function Show-Usage {
    Write-Host "Commodore CLI $COMMODORE_VERSION"
    Write-Host ''
    Write-Host 'Usage:'
    Write-Host '  commodore install   [path] [name] [alias...]   Build and install a CLI binary.'
    Write-Host '  commodore uninstall [name] [alias...]          Remove an installed CLI binary.'
    Write-Host '  commodore help                                 Show this help message.'
    Write-Host ''
    Write-Host 'Examples:'
    Write-Host '  .\commodore.ps1 install'
    Write-Host '  .\commodore.ps1 install .\cli octalweb ow'
    Write-Host '  .\commodore.ps1 uninstall'
    Write-Host '  .\commodore.ps1 uninstall octalweb ow'
    Write-Host ''
    Write-Host 'Environment:'
    Write-Host "  INSTALL_DIR   Destination directory (default: $script:INSTALL_DIR)"
}

function Invoke-Install {
    param([string[]]$RemainingArgs)

    $source = '.'
    $name = 'commodore'
    $aliases = @()

    switch ($RemainingArgs.Count) {
        0 { }
        1 {
            $source = $RemainingArgs[0]
        }
        default {
            $source = $RemainingArgs[0]
            $name = $RemainingArgs[1]
            if ($RemainingArgs.Count -gt 2) {
                $aliases = $RemainingArgs[2..($RemainingArgs.Count - 1)]
            }
        }
    }

    $binName = Get-BinName -Base $name
    $buildContext = Resolve-BuildContext -Source $source -Name $name
    if (-not $buildContext) {
        Write-Fail "Build target not found for source '$source' and binary '$name'."
        Write-Info "Expected one of: $source/main.go or $source/cmd/$name/main.go"
        exit 1
    }

    $buildArtifact = Join-Path (Get-Location).Path $binName

    Write-Info "Building     $binName"
    Write-Info "Source       $source"
    Write-Info "Target       $($buildContext.BuildTarget)"
    Write-Info "Module Root  $($buildContext.ModuleRoot)"
    Write-Info "Package      $($buildContext.PackagePath)"
    Write-Info "Destination  $script:INSTALL_DIR"
    Write-Host ''

    Invoke-GoBuild -ModuleRoot $buildContext.ModuleRoot -PackagePath $buildContext.PackagePath -OutputPath $buildArtifact
    Write-Ok "Built        $binName"

    New-Item -ItemType Directory -Path $script:INSTALL_DIR -Force | Out-Null
    Move-Item -LiteralPath $buildArtifact -Destination (Join-Path $script:INSTALL_DIR $binName) -Force
    Write-Ok "Installed    $(Join-Path $script:INSTALL_DIR $binName)"

    foreach ($aliasName in $aliases) {
        $aliasBin = Get-BinName -Base $aliasName
        Copy-Item -LiteralPath (Join-Path $script:INSTALL_DIR $binName) -Destination (Join-Path $script:INSTALL_DIR $aliasBin) -Force
        Write-Ok "Copy         $(Join-Path $script:INSTALL_DIR $aliasBin)  ->  $binName"
    }

    Ensure-OnPath -Directory $script:INSTALL_DIR
}

function Invoke-Uninstall {
    param([string[]]$RemainingArgs)

    $name = 'commodore'
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

    Write-Host ''
}

function Main {
    param([string[]]$CliArgs)

    $command = if ($CliArgs.Count -gt 0) { $CliArgs[0] } else { '' }
    $remaining = if ($CliArgs.Count -gt 1) { $CliArgs[1..($CliArgs.Count - 1)] } else { @() }

    switch ($command) {
        'install' { Invoke-Install -RemainingArgs $remaining }
        'uninstall' { Invoke-Uninstall -RemainingArgs $remaining }
        'help' { Show-Usage }
        '' { Show-Usage }
        default {
            Write-Fail "Unknown command: '$command'"
            Write-Host ''
            Show-Usage
            exit 1
        }
    }
}

Main -CliArgs $Arguments