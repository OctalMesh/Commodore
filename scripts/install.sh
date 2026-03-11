#!/usr/bin/env bash

# ============================================================================ #
# install.sh - install command for the Commodore CLI tool.                     #
#                                                                              #
# Provides:                                                                    #
#   cmd_install   Build a Go binary and install it with optional aliases.      #
#                                                                              #
# Required variables (set by the root script):                                 #
#   INSTALL_DIR   Destination directory.                                       #
# ============================================================================ #

# Guard against double-sourcing
[[ -n "${_COMMODORE_INSTALL_LOADED:-}" ]] && return 0
readonly _COMMODORE_INSTALL_LOADED=1

# ============================================================================ #
#                                  Functions                                   #
# ============================================================================ #

# Returns the platform-correct binary filename (appends .exe on Windows).
#
# Arguments:
#   $1 - Base binary name.
function _bin_name() {
    local base="$1"

    if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
        echo "${base}.exe"
    else
        echo "${base}"
    fi
}

# Returns true (0) if INSTALL_DIR is already present in the system PATH.
function _install_dir_on_path() {
    local dir="$1"
    # Normalise separators for Windows paths running under bash.
    local normalised
    normalised="$(cygpath -u "${dir}" 2>/dev/null || echo "${dir}")"
    IFS=':' read -ra _path_parts <<< "${PATH}"

    for part in "${_path_parts[@]}"; do
        local normalised_part
        normalised_part="$(cygpath -u "${part}" 2>/dev/null || echo "${part}")"
        [[ "${normalised_part}" == "${normalised}" ]] && return 0
    done

    return 1
}

# Prompt the user to add INSTALL_DIR to PATH, or print manual instructions.
#
# Arguments:
#   $1 - The directory to add.
function _ensure_on_path() {
    local dir="$1"

    _install_dir_on_path "${dir}" && return 0

    echo
    _print_warn "${dir} is not in PATH"
    printf '  Automatically add it? [Y/n] '

    local answer
    read -r answer
    answer="${answer:-Y}"

    if [[ "${answer}" =~ ^[Yy]$ ]]; then
        if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
            # Windows - add to User PATH via PowerShell
            local win_dir
            win_dir="$(cygpath -w "${dir}" 2>/dev/null || echo "${dir}")"

            if powershell.exe -NoProfile -Command "
                \$cur = [Environment]::GetEnvironmentVariable('PATH','User')
                if (\$cur -notlike '*${win_dir}*') {
                    [Environment]::SetEnvironmentVariable('PATH', \$cur + ';${win_dir}', 'User')
                }
            " 2>/dev/null; then
                _print_ok "Added to User PATH (Windows registry)."
                _print_info "Restart your terminal for the change to take effect."
            else
                _print_fail "Could not update PATH automatically."
                _print_info "Add manually: System Settings -> Environment Variables -> User PATH -> append ${win_dir}"
            fi
        else
            # Unix - append to shell profile
            local profile
            profile="$(_detect_shell_profile)"
            local line="export PATH=\"${dir}:\${PATH}\""

            if ! grep -qF "${dir}" "${profile}" 2>/dev/null; then
                printf '\n# Added by Commodore\n%s\n' "${line}" >> "${profile}"
                _print_ok "Added to ${profile}"
                _print_info "Run: source ${profile}"
            else
                _print_ok "Already present in ${profile} (PATH entry exists)."
            fi
        fi
    else
        echo
        _print_info "Manual setup:"

        if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
            local win_dir
            win_dir="$(cygpath -w "${dir}" 2>/dev/null || echo "${dir}")"
            printf '  System Settings -> Environment Variables -> User PATH -> append:\n'
            printf '    %s\n' "${win_dir}"
        else
            printf '  Add to your shell profile (~/.bashrc, ~/.zshrc, etc.):\n'
            printf '    export PATH="%s:${PATH}"\n' "${dir}"
        fi
    fi
    echo
}

# Detect the most appropriate shell profile file to edit.
function _detect_shell_profile() {
    local shell_name
    shell_name="$(basename "${SHELL:-bash}")"

    case "${shell_name}" in
        zsh)  echo "${HOME}/.zshrc"   ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        *)    echo "${HOME}/.bashrc" ;;
    esac
}

# Build a Go binary and install it, creating symlinks/hardlinks for aliases.
#
# Arguments:
#   $1        Path to the Go package directory (e.g. ./cli).
#   $2        Primary binary name   (e.g. octalweb).
#   $3 ..$N   Optional alias names  (e.g. ow).
function cmd_install() {
    local src="${1:?Usage: commodore install <path> <name> [alias...]}"
    local name="${2:?Usage: commodore install <path> <name> [alias...]}"
    shift 2

    local bin_name
    bin_name="$(_bin_name "${name}")"

    _print_info  "Building     ${bin_name}"
    _print_info  "Source       ${src}"
    _print_info  "Destination  ${INSTALL_DIR}"
    echo

    # Build
    if ! go build -o "${bin_name}" "${src}/main.go"; then
        _print_fail "Build failed: ${bin_name}"
        exit 1
    fi
    _print_ok "Built        ${bin_name}"

    # Install
    mkdir -p "${INSTALL_DIR}"
    mv "${bin_name}" "${INSTALL_DIR}/${bin_name}"
    _print_ok "Installed    ${INSTALL_DIR}/${bin_name}"

    # Aliases
    for alias_name in "$@"; do
        local alias_bin
        alias_bin="$(_bin_name "${alias_name}")"
        # On Windows, symbolic links require elevated privileges; use a hard link instead.
        if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
            cp "${INSTALL_DIR}/${bin_name}" "${INSTALL_DIR}/${alias_bin}"
            _print_ok "Copy         ${INSTALL_DIR}/${alias_bin}  ->  ${bin_name}"
        else
            ln -sf "${bin_name}" "${INSTALL_DIR}/${alias_bin}"
            _print_ok "Symlink      ${INSTALL_DIR}/${alias_bin}  ->  ${bin_name}"
        fi
    done

    # PATH check
    _ensure_on_path "${INSTALL_DIR}"
}
