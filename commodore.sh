#!/usr/bin/env bash

# ============================================================================ #
# Commodore CLI Tool                                                           #
# Description: Single-file, curl-friendly installer for Commodore CLI binaries.#
# Usage: ./commodore <command> [args]                                          #
#                                                                              #
# Commands:                                                                    #
#   install   <path> <name> [alias...]   Build and install a CLI binary.       #
#   uninstall <name> [alias...]          Remove an installed CLI binary.       #
#   help                                 Show this help message.               #
# ============================================================================ #

set -euo pipefail

# ============================================================================ #
#                                  Variables                                   #
# ============================================================================ #

declare -rx COMMODORE_VERSION="0.1.0"

export INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# ============================================================================ #
#                                  Logging                                     #
# ============================================================================ #

readonly _CLR_GREEN='\033[0;32m'
readonly _CLR_YELLOW='\033[1;33m'
readonly _CLR_RED='\033[0;31m'
readonly _CLR_RESET='\033[0m'

function _print_ok()   { printf "${_CLR_GREEN}[  OK  ]${_CLR_RESET} %s\n" "$*"; }
function _print_info() { printf "[ INFO ] %s\n" "$*"; }
function _print_warn() { printf "${_CLR_YELLOW}[ WARN ]${_CLR_RESET} %s\n" "$*" >&2; }
function _print_fail() { printf "${_CLR_RED}[ FAIL ]${_CLR_RESET} %s\n" "$*" >&2; }

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
        zsh)  echo "${HOME}/.zshrc" ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        *)    echo "${HOME}/.bashrc" ;;
    esac
}

# Resolve a directory to an absolute filesystem path.
#
# Arguments:
#   $1 - Directory path.
function _abs_dir() {
    local dir="$1"

    (
        cd "${dir}" >/dev/null 2>&1
        pwd -P
    )
}

# Find the nearest Go module root by walking up to the filesystem root.
#
# Arguments:
#   $1 - Directory to start from.
function _find_go_module_root() {
    local dir
    dir="$(_abs_dir "$1")" || return 1

    while true; do
        if [[ -f "${dir}/go.mod" ]]; then
            echo "${dir}"
            return 0
        fi

        local parent
        parent="$(dirname "${dir}")"
        [[ "${parent}" == "${dir}" ]] && break
        dir="${parent}"
    done

    return 1
}

# Convert an absolute child directory into a Go package path relative to its module root.
#
# Arguments:
#   $1 - Absolute module root.
#   $2 - Absolute package directory.
function _package_path_from_root() {
    local module_root="$1"
    local package_dir="$2"

    if [[ "${package_dir}" == "${module_root}" ]]; then
        echo "."
        return 0
    fi

    local relative="${package_dir#"${module_root}/"}"
    if [[ "${relative}" == "${package_dir}" ]]; then
        return 1
    fi

    echo "./${relative}"
}

# Resolves a Go build target for installation.
# Priority:
# 1) <src>/main.go (custom single-binary project)
# 2) <src>/cmd/<name>/main.go (standard multi-command repository layout)
#
# Arguments:
#   $1 - Source root/path from install command.
#   $2 - Binary name.
function _resolve_build_target() {
    local src="$1"
    local name="$2"

    local cleaned_src="${src%/}"
    [[ -z "${cleaned_src}" ]] && cleaned_src="."

    if [[ -f "${cleaned_src}/main.go" ]]; then
        echo "${cleaned_src}"
        return 0
    fi

    if [[ -f "${cleaned_src}/cmd/${name}/main.go" ]]; then
        echo "${cleaned_src}/cmd/${name}"
        return 0
    fi

    return 1
}

# Resolve the Go module root and package path to build.
#
# Arguments:
#   $1 - Source root/path from install command.
#   $2 - Binary name.
function _resolve_build_context() {
    local src="$1"
    local name="$2"

    local build_target
    if ! build_target="$(_resolve_build_target "${src}" "${name}")"; then
        return 1
    fi

    local module_root
    if ! module_root="$(_find_go_module_root "${build_target}")"; then
        _print_fail "Go module root not found for '${build_target}'."
        _print_info "Add a go.mod file to the project root or pass a path inside an existing Go module."
        exit 1
    fi

    local package_dir
    package_dir="$(_abs_dir "${build_target}")" || return 1

    local package_path
    if ! package_path="$(_package_path_from_root "${module_root}" "${package_dir}")"; then
        _print_fail "Could not determine Go package path for '${build_target}'."
        exit 1
    fi

    printf '%s|%s|%s\n' "${build_target}" "${module_root}" "${package_path}"
}

# Build a Go binary and install it, creating symlinks/hardlinks for aliases.
#
# Arguments:
#   $1        Path to the Go package directory (e.g. ./cli).
#   $2        Primary binary name   (e.g. octalweb).
#   $3 ..$N   Optional alias names  (e.g. ow).
function cmd_install() {
    local src
    local name

    case $# in
        0)
            src="./"
            name="commodore"
            ;;
        1)
            src="$1"
            name="commodore"
            shift 1
            ;;
        *)
            src="${1:?Usage: commodore install <path> <name> [alias...]}"
            name="${2:?Usage: commodore install <path> <name> [alias...]}"
            shift 2
            ;;
    esac

    local bin_name
    bin_name="$(_bin_name "${name}")"

    local build_context
    if ! build_context="$(_resolve_build_context "${src}" "${name}")"; then
        _print_fail "Build target not found for source '${src}' and binary '${name}'."
        _print_info "Expected one of: ${src}/main.go or ${src}/cmd/${name}/main.go"
        exit 1
    fi

    local build_target module_root package_path
    IFS='|' read -r build_target module_root package_path <<< "${build_context}"

    local build_artifact
    build_artifact="$(pwd -P)/${bin_name}"

    _print_info  "Building     ${bin_name}"
    _print_info  "Source       ${src}"
    _print_info  "Target       ${build_target}"
    _print_info  "Module Root  ${module_root}"
    _print_info  "Package      ${package_path}"
    _print_info  "Destination  ${INSTALL_DIR}"
    echo

    if ! (
        cd "${module_root}" &&
        go build -o "${build_artifact}" "${package_path}"
    ); then
        _print_fail "Build failed: ${bin_name}"
        exit 1
    fi
    _print_ok "Built        ${bin_name}"

    mkdir -p "${INSTALL_DIR}"
    mv "${build_artifact}" "${INSTALL_DIR}/${bin_name}"
    _print_ok "Installed    ${INSTALL_DIR}/${bin_name}"

    for alias_name in "$@"; do
        local alias_bin
        alias_bin="$(_bin_name "${alias_name}")"
        if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
            cp "${INSTALL_DIR}/${bin_name}" "${INSTALL_DIR}/${alias_bin}"
            _print_ok "Copy         ${INSTALL_DIR}/${alias_bin}  ->  ${bin_name}"
        else
            ln -sf "${bin_name}" "${INSTALL_DIR}/${alias_bin}"
            _print_ok "Symlink      ${INSTALL_DIR}/${alias_bin}  ->  ${bin_name}"
        fi
    done

    _ensure_on_path "${INSTALL_DIR}"
}

# Remove an installed binary and any symlink/copy aliases.
# On Windows the .exe suffix is appended automatically.
#
# Arguments:
#   $1        Primary binary name to remove (e.g. octalweb).
#   $2 ..$N   Optional alias names to remove (e.g. ow).
function cmd_uninstall() {
    local name="commodore"
    if [[ $# -gt 0 ]]; then
        name="$1"
        shift 1
    fi

    _remove_one "${name}" "binary"

    for alias_name in "$@"; do
        _remove_one "${alias_name}" "alias"
    done

    echo
}

# Remove a single installed file, trying both bare name and .exe variant.
#
# Arguments:
#   $1 - Base name (without .exe).
#   $2 - Label for output ("binary" or "alias").
function _remove_one() {
    local base="$1"
    local label="$2"

    local found=""
    for candidate in "${INSTALL_DIR}/${base}" "${INSTALL_DIR}/${base}.exe"; do
        if [[ -f "${candidate}" || -L "${candidate}" ]]; then
            found="${candidate}"
            break
        fi
    done

    if [[ -n "${found}" ]]; then
        rm -f "${found}"
        _print_ok "Removed      ${found}  (${label})"
    else
        _print_warn "Not found:   ${INSTALL_DIR}/${base}  (${label}, skipped)"
    fi
}

function _usage() {
    printf 'Commodore CLI %s\n\n' "${COMMODORE_VERSION}"
    printf 'Usage:\n'
    printf '  commodore install   [path] [name] [alias...]   Build and install a CLI binary.\n'
    printf '  commodore uninstall [name] [alias...]          Remove an installed CLI binary.\n'
    printf '  commodore help                                 Show this help message.\n'
    printf '\nExamples:\n'
    printf '  commodore install                              Install standard Commodore CLI from this repo\n'
    printf '  commodore install ./cli octalweb ow\n'
    printf '  commodore uninstall                            Uninstall standard Commodore CLI\n'
    printf '  commodore uninstall octalweb ow\n'
    printf '\nEnvironment:\n'
    printf '  INSTALL_DIR   Destination directory (default: %s)\n' "${INSTALL_DIR}"
}

function main() {
    local cmd="${1:-}"
    shift 1 2>/dev/null || true

    case "${cmd}" in
        install)   cmd_install   "$@" ;;
        uninstall) cmd_uninstall "$@" ;;
        help|"")   _usage ;;
        *)
            _print_fail "Unknown command: '${cmd}'"
            printf '\n' >&2
            _usage >&2
            exit 1
            ;;
    esac
}

main "$@"
