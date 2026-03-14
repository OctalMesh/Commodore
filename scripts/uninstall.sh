#!/usr/bin/env bash

# ============================================================================ #
# uninstall.sh - uninstall command for the Commodore CLI tool.                 #
#                                                                              #
# Provides:                                                                    #
#   cmd_uninstall   Remove an installed binary and any associated aliases.     #
#                                                                              #
# Required variables (set by the root script):                                 #
#   INSTALL_DIR   Directory where the binary was installed.                    #
# ============================================================================ #

# Guard against double-sourcing
[[ -n "${_COMMODORE_UNINSTALL_LOADED:-}" ]] && return 0
readonly _COMMODORE_UNINSTALL_LOADED=1

# ============================================================================ #
#                                  Functions                                   #
# ============================================================================ #

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

    # Remove binary
    _remove_one "${name}" "binary"

    # Remove aliases
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

    # Try bare name first, then .exe variant (covers cross-platform edge cases).
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
