#!/usr/bin/env bash

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

set -euo pipefail

# ============================================================================ #
#                                  Variables                                   #
# ============================================================================ #

_LAUNCH_DIR="$(pwd -P)"
readonly _LAUNCH_DIR

BINARY_NAME=""
SOURCE_DIR=""
BUILD_DIR=""
INSTALL_DIR=""

# ============================================================================ #
#                               Env Configuration                              #
# ============================================================================ #

# Sources a dotenv file and exports all loaded variables.
#
# Arguments:
#   $1 - Absolute file path.
function _dotenv_source_file() {
    local file="$1"
    [[ -f "${file}" ]] || return 0

    set -a
    # shellcheck disable=SC1090
    source "${file}"
    set +a
}

# Loads runtime configuration from environment and dotenv files.
# Flag overrides are applied last, after all env sources.
#
# Arguments:
#   $1 - Optional custom .env file path.
#   $2 - Optional binary name override (from --name flag).
#   $3 - Optional source dir override  (from --src  flag).
#   $4 - Optional build dir override   (from --out  flag).
#   $5 - Optional install dir override (from --dir  flag).
function _load_env_config() {
    local custom_env_file="${1:-}"
    local flag_name="${2:-}"
    local flag_src="${3:-}"
    local flag_out="${4:-}"
    local flag_dir="${5:-}"

    # 1) Built-in defaults
    local default_binary_name="commodore"
    local default_source_dir="${_LAUNCH_DIR}"
    local default_build_dir="${_LAUNCH_DIR}/build"
    local default_install_dir="${HOME}/.local/bin"

    # 2) .env.example
    _dotenv_source_file "${_LAUNCH_DIR}/.env.example"

    # 3) .env
    _dotenv_source_file "${_LAUNCH_DIR}/.env"

    # 4) Custom file from --env/-e
    if [[ -n "${custom_env_file}" ]]; then
        custom_env_file="$(_abs_path "${custom_env_file}" "${_LAUNCH_DIR}")"
        _dotenv_source_file "${custom_env_file}"
    fi

    # 5) Flag overrides (highest priority)
    [[ -n "${flag_name}" ]] && ADMIRAL_BINARY_NAME="${flag_name}"
    [[ -n "${flag_src}"  ]] && ADMIRAL_SOURCE_DIR="${flag_src}"
    [[ -n "${flag_out}"  ]] && ADMIRAL_BUILD_DIR="${flag_out}"
    [[ -n "${flag_dir}"  ]] && ADMIRAL_INSTALL_DIR="${flag_dir}"

    : "${ADMIRAL_BINARY_NAME:=${default_binary_name}}"
    : "${ADMIRAL_SOURCE_DIR:=${default_source_dir}}"
    : "${ADMIRAL_BUILD_DIR:=${default_build_dir}}"
    : "${ADMIRAL_INSTALL_DIR:=${default_install_dir}}"

    local dir_var dir_value
    for dir_var in ADMIRAL_SOURCE_DIR ADMIRAL_BUILD_DIR ADMIRAL_INSTALL_DIR; do
        dir_value="$(_abs_path "${!dir_var}" "${_LAUNCH_DIR}")"
        printf -v "${dir_var}" '%s' "${dir_value}"
    done

    BINARY_NAME="${ADMIRAL_BINARY_NAME}"
    SOURCE_DIR="${ADMIRAL_SOURCE_DIR}"
    BUILD_DIR="${ADMIRAL_BUILD_DIR}"
    INSTALL_DIR="${ADMIRAL_INSTALL_DIR}"

    export ADMIRAL_BINARY_NAME
    export ADMIRAL_SOURCE_DIR
    export ADMIRAL_BUILD_DIR
    export ADMIRAL_INSTALL_DIR
}

# ============================================================================ #
#                                  Logging                                     #
# ============================================================================ #

readonly _CLR_GRN='\033[0;32m'
readonly _CLR_YLW='\033[1;33m'
readonly _CLR_RED='\033[0;31m'
readonly _CLR_RST='\033[0m'

function ask()  { printf            "         %s " "$*"; }
function note() { printf            "         %s\n" "$*"; }
function succ() { printf "${_CLR_GRN}[ SUCC ]${_CLR_RST} %s\n" "$*"; }
function info() { printf            "[ INFO ] %s\n" "$*"; }
function warn() { printf "${_CLR_YLW}[ WARN ]${_CLR_RST} %s\n" "$*" >&2; }
function fail() { printf "${_CLR_RED}[ FAIL ]${_CLR_RST} %s\n" "$*" >&2; }

# ============================================================================ #
#                                  Utilities                                   #
# ============================================================================ #

# Returns 0 when running under a Windows-like POSIX shell (Git Bash/MSYS/Cygwin).
function _is_windows_runtime() {
    local os
    os="$(uname -s 2>/dev/null || echo '')"
    [[ "${os}" == MINGW* || "${os}" == CYGWIN* || "${os}" == MSYS* ]]
}

# Returns the platform-correct binary filename (adds .exe on Windows shells).
#
# Arguments:
#   $1 - Base binary name.
function _bin_name() {
    local base="$1"
    _is_windows_runtime && echo "${base}.exe" || echo "${base}"
}

# Converts a path to Unix form when cygpath exists; otherwise returns it as-is.
#
# Arguments:
#   $1 - Path to convert.
function _to_unix_path() {
    local path="$1"
    cygpath -u "${path}" 2>/dev/null || echo "${path}"
}

# Converts a path to Windows form when cygpath exists; otherwise returns it as-is.
#
# Arguments:
#   $1 - Path to convert.
function _to_windows_path() {
    local path="$1"
    cygpath -w "${path}" 2>/dev/null || echo "${path}"
}

# Returns 0 if the provided directory is already present in PATH.
#
# Arguments:
#   $1 - Directory to check.
function _install_dir_on_path() {
    local target_dir="$1"

    if ! command -v cygpath >/dev/null 2>&1; then
        [[ ":$PATH:" == *":$target_dir:"* ]] && return 0
        return 1
    fi

    local normalized unix_path_full
    normalized="$(_to_unix_path "$target_dir")"
    unix_path_full="$(cygpath -u -p "$PATH")"

    [[ ":$unix_path_full:" == *":$normalized:"* ]] && return 0
    return 1
}

# ============================================================================ #
#                                Flag Parsing                                  #
# ============================================================================ #

# Parse a known set of flags from an argument list, populating named variables.
# Unknown args are collected into the remainder array.
#
# Recognised flags (all optional):
#   -n | --name  <value> -> _fp_name
#   -s | --src   <value> -> _fp_src
#   -o | --out   <value> -> _fp_out
#   -d | --dir   <value> -> _fp_dir
#   -a | --alias <value> -> _fp_aliases (array, repeatable)
#
# Note: caller must declare all _fp_* variables before calling this function.
function _parse_flags() {
    while (( $# > 0 )); do
        case "$1" in
            # Binary name
            -n|--name)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                _fp_name="$2"; shift 2 ;;
            --name=*)
                _fp_name="${1#--name=}"; shift ;;
            # Source path
            -s|--src)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                _fp_src="$2"; shift 2 ;;
            --src=*)
                _fp_src="${1#--src=}"; shift ;;
            # Build output path
            -o|--out)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                _fp_out="$2"; shift 2 ;;
            --out=*)
                _fp_out="${1#--out=}"; shift ;;
            # Install directory
            -d|--dir)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                _fp_dir="$2"; shift 2 ;;
            --dir=*)
                _fp_dir="${1#--dir=}"; shift ;;
            # Binary aliases (repeatable)
            -a|--alias)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                _fp_aliases+=("$2"); shift 2 ;;
            --alias=*)
                _fp_aliases+=("${1#--alias=}"); shift ;;
            # End of flags
            --)
                shift; _fp_rest+=("$@"); break ;;
            # Unknown flag
            -*)
                fail "Unknown flag: $1"; exit 1 ;;
            # Positional / rest
            *)
                _fp_rest+=("$1"); shift ;;
        esac
    done
}

# ============================================================================ #
#                                PATH Management                               #
# ============================================================================ #

# Prompt the user to add INSTALL_DIR to PATH, or print manual instructions.
#
# Arguments:
#   $1 - The directory to add.
function _ensure_on_path() {
    local dir="$1"

    _install_dir_on_path "${dir}" && return 0

    warn "${dir} is not in PATH"
    ask  "Automatically add it? [Y/n]"

    local answer
    read -r answer
    answer="${answer:-Y}"

    if [[ ! "${answer}" =~ ^[Yy]$ ]]; then
        info "Manual setup required for: ${dir}"
        return 0
    fi

    if _is_windows_runtime; then
        local win_dir
        win_dir="$(_to_windows_path "${dir}")"

        if powershell.exe -NoProfile -Command "
                \$cur = [Environment]::GetEnvironmentVariable('PATH','User')
                if (\$cur -notlike '*${win_dir}*') {
                    [Environment]::SetEnvironmentVariable('PATH', \$cur + ';${win_dir}', 'User')
                }
            " 2>/dev/null; then
            succ "Added to User PATH (Windows Registry)."
            info "Restart your terminal to apply changes."
        else
            fail "Could not update PATH automatically."
            info "Add manually: System Settings -> Environment Variables -> User PATH -> append ${win_dir}"
        fi
    else
        local profile
        profile="$(_detect_shell_profile)"
        local line

        if [[ "${profile}" == *"config.fish" ]]; then
            line="fish_add_path ${dir}"
        else
            line="export PATH=\"${dir}:\${PATH}\""
        fi

        mkdir -p "$(dirname "$profile")"
        if ! grep -qF "${dir}" "${profile}" 2>/dev/null; then
            printf '\n# Added by Admiral\n%s\n' "${line}" >> "${profile}"
            succ   "Added to ${profile}"
            info   "Run: source ${profile}"
        else
            succ "Already present in ${profile}"
        fi
    fi
}

# Removes INSTALL_DIR from PATH after user confirmation.
#
# Arguments:
#   $1 - Directory to remove from PATH.
function _remove_from_path() {
    local dir="$1"

    _install_dir_on_path "${dir}" || return 0

    info "${dir} is currently in PATH."
    ask  "Automatically remove it? [y/N]"

    local answer
    read -r answer
    answer="${answer:-N}"

    [[ "${answer}" =~ ^[Yy]$ ]] || return 0

    if _is_windows_runtime; then
        local win_dir
        win_dir="$(_to_windows_path "${dir}")"

        if powershell.exe -NoProfile -Command "
                \$target = [System.IO.Path]::GetFullPath('${win_dir}').TrimEnd('\\')
                \$cur = [Environment]::GetEnvironmentVariable('PATH','User')
                if ([string]::IsNullOrWhiteSpace(\$cur)) { exit 0 }
                \$parts = \$cur -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace(\$_) }
                \$new = @()
                foreach (\$part in \$parts) {
                    try {
                        \$candidate = [System.IO.Path]::GetFullPath(\$part).TrimEnd('\\')
                    } catch {
                        \$candidate = \$part.TrimEnd('\\')
                    }

                    if (-not \$candidate.Equals(\$target, [System.StringComparison]::OrdinalIgnoreCase)) {
                        \$new += \$part
                    }
                }
                [Environment]::SetEnvironmentVariable('PATH', (\$new -join ';'), 'User')
            " 2>/dev/null; then
            succ "Removed from User PATH (Windows Registry)."
            info "Restart your terminal to apply changes."
        else
            fail "Could not remove PATH entry automatically."
            info "Remove manually from User PATH: ${win_dir}"
        fi
    else
        local profile
        profile="$(_detect_shell_profile)"

        if [[ -f "${profile}" ]]; then
            local tmp
            tmp="${profile}.admiral.tmp"

            grep -vF "${dir}" "${profile}" > "${tmp}" || true
            mv "${tmp}" "${profile}"

            succ "Removed PATH entry from ${profile}"
            info "Run: source ${profile}"
        else
            warn "Profile not found: ${profile}"
            info "Remove manually from PATH: ${dir}"
        fi
    fi
}

# Detect the most appropriate shell profile file to edit.
function _detect_shell_profile() {
    local shell_path="${SHELL:-}"
    local shell_name

    if [[ -z "$shell_path" ]]; then
        shell_name=$(ps -p $$ -o comm= | sed 's/^-//')
    else
        shell_name=$(basename "$shell_path")
    fi

    case "${shell_name}" in
        zsh)  echo "${HOME}/.zshrc" ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        *)    echo "${HOME}/.bashrc" ;;
    esac
}

# ============================================================================ #
#                               Build & Context                                #
# ============================================================================ #

# Resolve a directory to an absolute filesystem path.
#
# Arguments:
#   $1 - Directory path.
function _abs_dir() {
    local dir="$1"
    (cd "${dir}" >/dev/null 2>&1 && pwd -P)
}

# Resolve a path into an absolute path using a base directory when needed.
# Supports '~' and '~/...' expansion.
#
# Arguments:
#   $1 - Path to normalize.
#   $2 - Base directory for relative paths (optional, default: _LAUNCH_DIR).
function _abs_path() {
    local path="$1"
    local base_dir="${2:-${_LAUNCH_DIR}}"

    [[ -n "${path}" ]] || return 1

    if [[ "${path}" == "~" ]]; then
        path="${HOME}"
    elif [[ "${path}" == ~/* ]]; then
        path="${HOME}/${path#~/}"
    fi

    local candidate
    if [[ "${path}" == /* ]]; then
        candidate="${path}"
    else
        candidate="${base_dir%/}/${path}"
    fi

    local parent leaf parent_abs
    parent="$(dirname "${candidate}")"
    leaf="$(basename "${candidate}")"

    if parent_abs="$(_abs_dir "${parent}")"; then
        printf '%s/%s\n' "${parent_abs}" "${leaf}"
    else
        printf '%s\n' "${candidate}"
    fi
}

# Find the nearest Go module root by walking up to the filesystem root.
#
# Arguments:
#   $1 - Directory to start from.
function _find_go_module_root() {
    local dir
    dir="$(_abs_dir "$1")" || return 1

    while [[ "${dir}" != "/" ]]; do
        [[ -f "${dir}/go.mod" ]] && { echo "${dir}"; return 0; }
        dir="$(dirname "${dir}")"
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
#   $1 - Source root/path.
#   $2 - Binary name.
function _resolve_build_target() {
    local src="${1%/}"
    local name="$2"
    [[ -z "${src}" ]] && src="."

    local candidate
    for candidate in "${src}" "${src}/cmd/${name}"; do
        if [[ -f "${candidate}/main.go" ]]; then
            echo "${candidate}"
            return 0
        fi
    done

    return 1
}

# Resolve the Go module root and package path to build.
#
# Arguments:
#   $1 - Source root/path.
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
        fail "Go module root not found for '${build_target}'."
        info "Add a go.mod file to the project root or pass a path inside an existing Go module."
        exit 1
    fi

    local package_dir
    package_dir="$(_abs_dir "${build_target}")" || return 1

    local package_path
    if ! package_path="$(_package_path_from_root "${module_root}" "${package_dir}")"; then
        fail "Could not determine Go package path for '${build_target}'."
        exit 1
    fi

    printf '%s|%s|%s|%s\n' "${build_target}" "${module_root}" "${package_dir}" "${package_path}"
}

# ============================================================================ #
#                              Table Formatting                                #
# ============================================================================ #

# Format a table cell with fixed width and ellipsis truncation.
#
# Arguments:
#   $1 - Cell text.
#   $2 - Cell width.
function _tcell() {
    local text="$1"
    local width="$2"

    if (( ${#text} > width )); then
        text="${text:0:$((width - 3))}..."
    fi

    printf "%-${width}s" "${text}"
}

# Returns a repeated character sequence with the given length.
#
# Arguments:
#   $1 - Length.
#   $2 - Character to repeat.
function _repeat_char() {
    local length="$1"
    local char="$2"
    local buf

    printf -v buf '%*s' "${length}" ''
    buf="${buf// /${char}}"
    printf '%s' "${buf}"
}

# Prints a table border for a set of column widths.
#
# Arguments:
#   $1     - Left border symbol.
#   $2     - Middle separator symbol.
#   $3     - Right border symbol.
#   $4..$N - Column widths.
function _table_border() {
    local left="$1"
    local mid="$2"
    local right="$3"
    shift 3

    local widths=("$@")
    local line="${left}"
    local i

    for i in "${!widths[@]}"; do
        line+="─$(_repeat_char "${widths[$i]}" "─")─"
        if (( i < ${#widths[@]} - 1 )); then
            line+="${mid}"
        fi
    done

    line+="${right}"
    note "${line}"
}

# Prints one formatted table row for given widths and cell values.
#
# Arguments:
#   $1      - Name of widths array variable.
#   $2..$N  - Cell values.
function _table_row() {
    local -n _widths_ref="$1"
    shift

    local _vals_ref=("$@")
    local line="│"
    local i

    for i in "${!_widths_ref[@]}"; do
        line+=" $(_tcell "${_vals_ref[$i]:-}" "${_widths_ref[$i]}") │"
    done

    note "${line}"
}

# Prints a decorative info divider line.
function _info_divider() {
    info "$(_repeat_char 72 "─")"
}

# Returns current Unix time in milliseconds.
function _now_ms() {
    local ts

    ts="$(date +%s%3N 2>/dev/null || true)"
    if [[ -n "${ts}" && "${ts}" =~ ^[0-9]+$ ]]; then
        printf '%s\n' "${ts}"
        return 0
    fi

    printf '%s\n' "$(( $(date +%s) * 1000 ))"
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
        succ "Removed      ${found}  (${label})"
    else
        warn "Not found:   ${INSTALL_DIR}/${base}  (${label}, skipped)"
    fi
}

# ============================================================================ #
#                               Commands                                       #
# ============================================================================ #

# Prints full usage information and examples.
function _usage() {
    cat <<EOF

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
    ADMIRAL_BINARY_NAME   Primary binary name          (default: ${ADMIRAL_BINARY_NAME})
    ADMIRAL_SOURCE_DIR    Default source path          (default: ${ADMIRAL_SOURCE_DIR})
    ADMIRAL_BUILD_DIR     Cross-build output directory (default: ${ADMIRAL_BUILD_DIR})
    ADMIRAL_INSTALL_DIR   Destination directory        (default: ${ADMIRAL_INSTALL_DIR})

EOF
}

# Run golangci-lint via `go tool` from module root.
#
# Flags:
#   -s, --src <path>   Source path to resolve module root from.
function cmd_lint() {
    local _fp_name="" _fp_src="" _fp_out="" _fp_dir=""
    local -a _fp_aliases=() _fp_rest=()
    _parse_flags "$@"

    local src="${_fp_src:-${SOURCE_DIR}}"
    src="$(_abs_path "${src:-./}" "${_LAUNCH_DIR}")"

    local module_root
    if ! module_root="$(_find_go_module_root "${src}")"; then
        fail "Go module root not found for '${src}'."
        info "Run lint inside a Go module or pass a path with --src."
        exit 1
    fi

    info "Linting      ${module_root}"
    info "Command      go tool golangci-lint run ./..."

    if ! (
        cd "${module_root}" &&
        go tool golangci-lint run ./...
    ); then
        fail "Lint failed."
        exit 1
    fi

    succ "Lint passed"
}

# Build Go binaries for multiple OS targets into a dedicated output directory.
#
# Flags:
#   -n, --name <name>   Binary name.
#   -s, --src  <path>   Source path.
#   -o, --out  <path>   Build output directory.
function cmd_build() {
    local _fp_name="" _fp_src="" _fp_out="" _fp_dir=""
    local -a _fp_aliases=() _fp_rest=()
    _parse_flags "$@"

    local name="${_fp_name:-${BINARY_NAME}}"
    local src="${_fp_src:-${SOURCE_DIR}}"
    src="$(_abs_path "${src:-./}" "${_LAUNCH_DIR}")"

    local build_context
    if ! build_context="$(_resolve_build_context "${src}" "${name}")"; then
        fail "Build target not found for --src '${src}' and --name '${name}'."
        info "Expected one of: ${src}/main.go or ${src}/cmd/${name}/main.go"
        exit 1
    fi

    local build_target module_root package_dir package_path
    IFS='|' read -r build_target module_root package_dir package_path <<< "${build_context}"

    local output_dir="${_fp_out:-${BUILD_DIR}}"
    output_dir="$(_abs_path "${output_dir:-./build}" "${_LAUNCH_DIR}")"

    mkdir -p "${output_dir}"
    output_dir="$(_abs_dir "${output_dir}")"

    _info_divider
    info "Building     ${name}"
    info "Source       ${src}"
    info "Target       ${build_target}"
    info "Module Root  ${module_root}"
    info "Package Dir  ${package_dir}"
    info "Output Dir   ${output_dir}"
    _info_divider

    local targets=(
        "linux:amd64"
        "darwin:amd64"
        "darwin:arm64"
        "windows:amd64"
    )

    local -a result_targets=()
    local -a result_statuses=()
    local -a result_durations=()
    local -a result_artifacts=()
    local failed=0

    local target
    for target in "${targets[@]}"; do
        local goos goarch
        IFS=':' read -r goos goarch <<< "${target}"

        local artifact="${output_dir}/${name}_${goos}_${goarch}"
        [[ "${goos}" == "windows" ]] && artifact="${artifact}.exe"

        local start_ms end_ms status duration
        start_ms="$(_now_ms)"
        status="OK"

        info "Cross-build  ${goos}/${goarch} -> ${artifact}"

        if ! (
            cd "${module_root}" &&
            GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -o "${artifact}" "${package_path}"
        ); then
            status="FAIL"
            failed=$((failed + 1))
            fail "Build failed for ${goos}/${goarch}"
        else
            succ "Built        ${artifact}"
        fi

        end_ms="$(_now_ms)"
        duration="$((end_ms - start_ms))ms"

        result_targets+=("${goos}/${goarch}")
        result_statuses+=("${status}")
        result_durations+=("${duration}")
        result_artifacts+=("${artifact##*/}")
    done

    local ok=$(( ${#result_statuses[@]} - failed ))
    local skip=0

    local -a widths=(14 8 8 36)

    note ""
    _table_border "┌" "┬" "┐" "${widths[@]}"
    _table_row widths "Target" "Status" "Duration" "Artifact"
    _table_border "├" "┼" "┤" "${widths[@]}"

    local i
    for i in "${!result_targets[@]}"; do
        _table_row widths \
            "${result_targets[$i]}" \
            "${result_statuses[$i]}" \
            "${result_durations[$i]}" \
            "${result_artifacts[$i]}"
    done

    _table_border "└" "┴" "┘" "${widths[@]}"
    note "  Totals: ${ok} built  ${failed} failed  ${skip} skipped"
    note ""

    if (( failed > 0 )); then
        exit 1
    fi
}

# Build a Go binary and install it, creating symlinks/hardlinks for aliases.
#
# Flags:
#   -n, --name  <name>   Binary name.
#   -s, --src   <path>   Source path.
#   -d, --dir   <path>   Install directory.
#   -a, --alias <name>   Alias (repeatable).
function cmd_install() {
    local _fp_name="" _fp_src="" _fp_out="" _fp_dir=""
    local -a _fp_aliases=() _fp_rest=()
    _parse_flags "$@"

    local name="${_fp_name:-${BINARY_NAME}}"
    local src="${_fp_src:-${SOURCE_DIR}}"
    src="$(_abs_path "${src:-./}" "${_LAUNCH_DIR}")"

    local install_dir="${_fp_dir:-${INSTALL_DIR}}"
    install_dir="$(_abs_path "${install_dir}" "${_LAUNCH_DIR}")"

    local bin_name
    bin_name="$(_bin_name "${name}")"

    local build_context
    if ! build_context="$(_resolve_build_context "${src}" "${name}")"; then
        fail "Build target not found for --src '${src}' and --name '${name}'."
        info "Expected one of: ${src}/main.go or ${src}/cmd/${name}/main.go"
        exit 1
    fi

    local build_target module_root package_dir package_path
    IFS='|' read -r build_target module_root package_dir package_path <<< "${build_context}"

    local build_artifact
    build_artifact="$(pwd -P)/${bin_name}"

    _info_divider
    info "Building     ${bin_name}"
    info "Source       ${src}"
    info "Target       ${build_target}"
    info "Module Root  ${module_root}"
    info "Package Dir  ${package_dir}"
    info "Destination  ${install_dir}"
    _info_divider

    if ! (
        cd "${module_root}" &&
        go build -o "${build_artifact}" "${package_path}"
    ); then
        fail "Build failed: ${bin_name}"
        exit 1
    fi
    succ "Built        ${bin_name}"

    mkdir -p "${install_dir}"
    mv "${build_artifact}" "${install_dir}/${bin_name}"
    succ "Installed    ${install_dir}/${bin_name}"

    for alias_name in "${_fp_aliases[@]+"${_fp_aliases[@]}"}"; do
        local alias_bin
        alias_bin="$(_bin_name "${alias_name}")"
        if _is_windows_runtime; then
            cp "${install_dir}/${bin_name}" "${install_dir}/${alias_bin}"
            succ "Copy         ${install_dir}/${alias_bin}  ->  ${bin_name}"
        else
            ln -sf "${bin_name}" "${install_dir}/${alias_bin}"
            succ "Symlink      ${install_dir}/${alias_bin}  ->  ${bin_name}"
        fi
    done

    _ensure_on_path "${install_dir}"
}

# Remove an installed binary and any symlink/copy aliases.
#
# Flags:
#   -n, --name  <name>   Binary name.
#   -d, --dir   <path>   Install directory.
#   -a, --alias <name>   Alias (repeatable).
function cmd_uninstall() {
    local _fp_name="" _fp_src="" _fp_out="" _fp_dir=""
    local -a _fp_aliases=() _fp_rest=()
    _parse_flags "$@"

    local name="${_fp_name:-${BINARY_NAME}}"
    local install_dir="${_fp_dir:-${INSTALL_DIR}}"
    install_dir="$(_abs_path "${install_dir}" "${_LAUNCH_DIR}")"

    # Temporarily override INSTALL_DIR so _remove_one and _remove_from_path
    # operate on the correct directory when --dir was passed.
    INSTALL_DIR="${install_dir}"

    _remove_one "${name}" "binary"

    for alias_name in "${_fp_aliases[@]+"${_fp_aliases[@]}"}"; do
        _remove_one "${alias_name}" "alias"
    done

    _remove_from_path "${install_dir}"
}

# ============================================================================ #
#                                  Entry Point                                 #
# ============================================================================ #

function main() {
    local g_env_file=""
    local g_name="" g_src="" g_out="" g_dir=""

    while (( $# > 0 )); do
        case "$1" in
            -e|--env)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                g_env_file="$2"; shift 2 ;;
            --env=*)
                g_env_file="${1#--env=}"; shift ;;

            -n|--name)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                g_name="$2"; shift 2 ;;
            --name=*)
                g_name="${1#--name=}"; shift ;;

            -s|--src)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                g_src="$2"; shift 2 ;;
            --src=*)
                g_src="${1#--src=}"; shift ;;

            -o|--out)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                g_out="$2"; shift 2 ;;
            --out=*)
                g_out="${1#--out=}"; shift ;;

            -d|--dir)
                [[ $# -ge 2 ]] || { fail "Missing value for $1"; exit 1; }
                g_dir="$2"; shift 2 ;;
            --dir=*)
                g_dir="${1#--dir=}"; shift ;;

            *)
                break ;;
        esac
    done

    _load_env_config "${g_env_file}" "${g_name}" "${g_src}" "${g_out}" "${g_dir}"

    case "${1:-}" in
        install)   shift; cmd_install   "$@" ;;
        build)     shift; cmd_build     "$@" ;;
        lint)      shift; cmd_lint      "$@" ;;
        uninstall) shift; cmd_uninstall "$@" ;;
        help|"")   _usage ;;
        *)         fail "Unknown command: ${1}"; _usage; exit 1 ;;
    esac
}

main "$@"
