#!/usr/bin/env bash

{

log() { 
    local type="$1" gray="\033[38;2;134;134;134m" reset="\033[0m"
    shift
    case "$type" in
        error)   printf "${gray}[Error]${reset} \033[38;2;255;85;85m%s${reset}\n" "$*" ;;
        success) printf "${gray}[Success]${reset} \033[38;2;80;250;123m%s${reset}\n" "$*" ;;
        warning) printf "${gray}[Warning]${reset} \033[38;2;235;208;9m%s${reset}\n" "$*" ;;
        info)    printf "\033[38;2;139;233;253m%s${reset}\n" "$*" ;;
        *)       echo "$type" "$@" ;;
    esac
}

error()   { log error "$@"; }
success() { log success "$@"; }
warning() { log warning "$@"; }
info()    { log info "$@"; }

cmd-exists() { command -v "$1" >/dev/null 2>&1; }

GITHUB_URL="https://github.com/Utsav-56/fedora-setup/releases/latest/download/usetup"
DL_DIR="/tmp/usetup"
BIN_PATH="$DL_DIR/usetup"

SUDO_CMD=""
if [[ $EUID -ne 0 ]]; then
    warning "This setup requires root privileges. You may be prompted for your password."
    
    if ! cmd-exists sudo; then
        error "You are not root and 'sudo' is not installed. Aborting."
        exit 1
    fi
    SUDO_CMD="sudo"
    
    $SUDO_CMD -v || { error "Sudo authentication failed."; exit 1; }
fi


if ! cmd-exists aria2c; then
    warning "aria2c is not installed. Attempting to install for faster downloads..."
    if cmd-exists dnf; then
        $SUDO_CMD dnf install -y aria2
    else
        warning "dnf package manager not found. Skipping aria2c installation."
    fi
fi


$SUDO_CMD mkdir -p "$DL_DIR"

if cmd-exists aria2c; then
    info "Downloading via aria2c (8 threads)..."
    $SUDO_CMD aria2c --console-log-level=warn -x 8 -k 2M -d "$DL_DIR" -o "usetup" "$GITHUB_URL"
elif cmd-exists curl; then
    warning "Falling back to curl..."
    $SUDO_CMD curl -L --progress-bar -o "$BIN_PATH" "$GITHUB_URL"
elif cmd-exists wget; then
    warning "Falling back to wget..."
    $SUDO_CMD wget -q --show-progress -O "$BIN_PATH" "$GITHUB_URL"
else
    error "No suitable downloaders found (aria2c, curl, wget). Aborting."
    exit 1
fi


if [[ -f "$BIN_PATH" ]]; then
    info "Setting executable permissions..."
    $SUDO_CMD chmod +x "$BIN_PATH"
    
    success "Bootstrapping complete. Launching usetup..."
    $SUDO_CMD "$BIN_PATH"
else
    error "Download failed. Binary not found at $BIN_PATH."
    exit 1
fi

}