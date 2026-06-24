#!/bin/bash
# /src/login.sh -> Linked to /etc/profile.d/src-workspace-env.sh

PUBLIC_SOURCE_DIR="/src"
TOOLS_DIR="$PUBLIC_SOURCE_DIR/Tools"
SRC_BIN_DIR="$TOOLS_DIR/bin"
APPLICATIONS_DIR="$PUBLIC_SOURCE_DIR/Applications"
APP_BIN_DIR="$APPLICATIONS_DIR/bin"
APP_XDG_DIR="$APPLICATIONS_DIR/applications"
DEFAULT_USER_GROUPS="shared"

# 1. Unified Binary Shell Prepend Path Maps (POSIX-safe checks)
case ":$PATH:" in
    *:"$SRC_BIN_DIR":*) ;;
    *) export PATH="$SRC_BIN_DIR:$PATH" ;;
esac

case ":$PATH:" in
    *:"$APP_BIN_DIR":*) ;;
    *) export PATH="$APP_BIN_DIR:$PATH" ;;
esac

# 2. XDG System Desktop Integration Menu Paths (POSIX-safe checks)
if [ -z "$XDG_DATA_DIRS" ]; then
    export XDG_DATA_DIRS="$APPLICATIONS_DIR:/usr/local/share:/usr/share"
else
    case ":$XDG_DATA_DIRS:" in
        *:"$APPLICATIONS_DIR":*) ;;
        *) export XDG_DATA_DIRS="$APPLICATIONS_DIR:$XDG_DATA_DIRS" ;;
    esac
fi

# 3. Prevent permissions drift for shared group developers
if id -nG 2>/dev/null | grep -qw "$DEFAULT_USER_GROUPS"; then
    umask 002
fi

# 4. Interactive Shell Tools (Only loaded in interactive bash/zsh sessions)
if [ -n "$BASH_VERSION" ] || [ -n "$ZSH_VERSION" ]; then

    # Colorized Logging Utilities
    _workspace_log() {
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
    _workspace_error()   { _workspace_log error "$@"; }
    _workspace_success() { _workspace_log success "$@"; }
    _workspace_warning() { _workspace_log warning "$@"; }
    _workspace_info()    { _workspace_log info "$@"; }

    # Core Workspace Linking Engine
    _workspace_linker() {
        local mode="$1"         # "link" or "copy"
        local dest_dir="$2"     # Target folder destination
        shift 2                 # Remaining arguments are file targets
        
        # Ensure destination directory exists
        if [ ! -d "$dest_dir" ]; then
            mkdir -p "$dest_dir" 2>/dev/null
        fi
        
        for target in "$@"; do
            local abs_path
            abs_path="$(realpath "$target" 2>/dev/null)"
            
            if [ -z "$abs_path" ] || [ ! -e "$abs_path" ]; then
                _workspace_warning "'$target' is not a valid accessible path. Skipping."
                continue
            fi
            
            # Security Guardrail: Only enforce inside /src for "link" mode
            # Symlinks must stay within the shared boundary to remain self-contained/portable
            if [ "$mode" = "link" ] && [ "${abs_path#$PUBLIC_SOURCE_DIR/}" = "$abs_path" ]; then
                _workspace_error "Security boundary violation: Symlink target '$target' is outside $PUBLIC_SOURCE_DIR. Denied."
                continue
            fi
            
            if [ "$mode" = "copy" ]; then
                cp -f "$abs_path" "$dest_dir/"
                _workspace_success "Copied: $(basename "$abs_path") -> $dest_dir"
            elif [ -d "$abs_path" ]; then
                # Directory processing: find nested files and link them
                local files_linked=0
                for file in "$abs_path"/*; do
                    if [ -f "$file" ]; then
                        ln -sf "$file" "$dest_dir/$(basename "$file")"
                        files_linked=$((files_linked + 1))
                    fi
                done
                _workspace_success "Linked $files_linked binaries from directory: $(basename "$abs_path") -> $dest_dir"
            else
                ln -sf "$abs_path" "$dest_dir/$(basename "$abs_path")"
                _workspace_success "Linked binary: $(basename "$abs_path") -> $dest_dir"
            fi
        done
    }

    # Lightweight Wrapper Commands
    src-add() {
        case "$1" in
            rm)         shift; for t in "$@"; do rm -f "$SRC_BIN_DIR/$(basename "$t")"; done ;;
            autoremove) find "$SRC_BIN_DIR" -maxdepth 1 -xtype l -delete 2>/dev/null ;;
            *)          _workspace_linker "link" "$SRC_BIN_DIR" "$@" ;;
        esac
    }

    app-add() {
        case "$1" in
            rm)         shift; for t in "$@"; do rm -f "$APP_BIN_DIR/$(basename "$t")"; done ;;
            autoremove) find "$APP_BIN_DIR" -maxdepth 1 -xtype l -delete 2>/dev/null ;;
            *)          _workspace_linker "link" "$APP_BIN_DIR" "$@" ;;
        esac
    }

    app-icon-add() {
        case "$1" in
            rm)         shift; for t in "$@"; do rm -f "$APP_XDG_DIR/$(basename "$t")"; done ;;
            autoremove) find "$APP_XDG_DIR" -maxdepth 1 -xtype l -delete 2>/dev/null ;;
            *)          _workspace_linker "copy" "$APP_XDG_DIR" "$@" ;;
        esac
    }

fi