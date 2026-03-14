@echo off
:: Commodore CLI - Windows launcher
:: Requires bash (Git Bash, WSL, or Cygwin) to be available on PATH.

where bash >nul 2>&1
if errorlevel 1 (
    echo [ERROR] bash not found on PATH.
    echo Install Git for Windows ^(https://git-scm.com^) or enable WSL to use this tool.
    exit /b 1
)

bash "%~dp0commodore" %*
