#!/usr/bin/env bash

set -e

main() {
    PLATFORM="$(uname | tr '[:upper:]' '[:lower:]')"
    if [ "$PLATFORM" = "mingw32_nt" ] || [ "$PLATFORM" = "mingw64_nt" ]; then
        PLATFORM="windows"
    fi

    BIN_DIR=${BIN_DIR:-$HOME/.local/bin}

    case $SHELL in
        */zsh)
            PROFILE=$HOME/.zshrc
            ;;
        */bash)
            PROFILE=$HOME/.bashrc
            ;;
        */fish)
            PROFILE=$HOME/.config/fish/config.fish
            ;;
        */ash)
            PROFILE=$HOME/.profile
            ;;
        *)
            echo "could not detect shell, manually add ${BIN_DIR} to your PATH."
            exit 1
    esac

    if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
        echo >> "$PROFILE" && echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$PROFILE"
    fi

    ARCHITECTURE="$(uname -m)"
    if [ "${ARCHITECTURE}" = "x86_64" ]; then
        # Redirect stderr to /dev/null to avoid printing errors if non Rosetta.
        if [ "$(sysctl -n sysctl.proc_translated 2>/dev/null)" = "1" ]; then
            ARCHITECTURE="arm64" # Rosetta.
        else
            ARCHITECTURE="amd64" # Intel.
        fi
    elif [ "${ARCHITECTURE}" = "arm64" ] || [ "${ARCHITECTURE}" = "aarch64" ]; then
        ARCHITECTURE="arm64" # Arm.
    else
        ARCHITECTURE="amd64" # Amd.
    fi

    if [[ "$PLATFORM" == "windows" ]]; then
        EXTENSION=".exe"
    else
        EXTENSION=""
    fi

    BINARY_URL="https://github.com/stmg-ai/claude-squad/releases/latest/download/claude-squad-${PLATFORM}-${ARCHITECTURE}/claude-squad${EXTENSION}"

    if [ ! -d "$BIN_DIR" ]; then
        mkdir -p "$BIN_DIR"
    fi

    echo "Downloading latest binary from $BINARY_URL to $BIN_DIR"
    ensure curl -L "$BINARY_URL" -o "$BIN_DIR/claude-squad${EXTENSION}"

    if [ ! -f "$BIN_DIR/claude-squad${EXTENSION}" ]; then
        echo "Download failed, could not find $BIN_DIR/claude-squad${EXTENSION}"
        exit 1
    fi

    chmod +x "$BIN_DIR/claude-squad${EXTENSION}"
    echo "installed - $("$BIN_DIR/claude-squad${EXTENSION}" --version)"
}

# Run a command that should never fail. If the command fails execution
# will immediately terminate with an error showing the failing
# command.
ensure() {
    if ! "$@"; then err "command failed: $*"; fi
}

err() {
    echo "$1" >&2
    exit 1
}

main "$@" || exit 1
 
