#!/bin/bash
# /src/login.sh -> Linked to /etc/profile.d/src-workspace-env.sh

if [ -f "/src/env_login.sh" ]; then
    source "/src/env_login.sh"
fi

PUBLIC_SOURCE_DIR="/src"
TOOLS_DIR="$PUBLIC_SOURCE_DIR/Tools"
SRC_BIN_DIR="$TOOLS_DIR/bin"
APPLICATIONS_DIR="$PUBLIC_SOURCE_DIR/Applications"
APP_BIN_DIR="$APPLICATIONS_DIR/bin"
APP_XDG_DIR="$APPLICATIONS_DIR/applications"
DEFAULT_USER_GROUPS="shared"

# --- 5. Prepend PATH Maps (POSIX-safe checks) ---
path_prepend() {
    local p="$1"
    if [ -d "$p" ]; then
        case ":$PATH:" in
            *:"$p":*) ;;
            *) export PATH="$p:$PATH" ;;
        esac
    fi
}

setup_src_bin() {
    mkdir -p "$SRC_BIN_DIR"
    mkdir -p "$APP_BIN_DIR"
    path_prepend "$SRC_BIN_DIR"
    path_prepend "$APP_BIN_DIR"
}
setup_src_bin

# --- 6. XDG System Desktop Integration Menu Paths ---
if [ -z "$XDG_DATA_DIRS" ]; then
    export XDG_DATA_DIRS="$APPLICATIONS_DIR:/usr/local/share:/usr/share"
else
    case ":$XDG_DATA_DIRS:" in
        *:"$APPLICATIONS_DIR":*) ;;
        *) export XDG_DATA_DIRS="$APPLICATIONS_DIR:$XDG_DATA_DIRS" ;;
    esac
fi

# --- 7. Prevent permissions drift for shared group developers ---
if id -nG 2>/dev/null | grep -qw "$DEFAULT_USER_GROUPS"; then
    umask 002
fi

# --- 8. Interactive Shell Tools (Only loaded in interactive bash/zsh sessions) ---
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
            
            if [ "$mode" = "link" ] && [ "${abs_path#$PUBLIC_SOURCE_DIR/}" = "$abs_path" ]; then
                _workspace_error "Security boundary violation: Symlink target '$target' is outside $PUBLIC_SOURCE_DIR. Denied."
                continue
            fi
            
            if [ "$mode" = "copy" ]; then
                cp -f "$abs_path" "$dest_dir/"
                _workspace_success "Copied: $(basename "$abs_path") -> $dest_dir"
            elif [ -d "$abs_path" ]; then
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

    # Setup FNM if available
    if command -v fnm >/dev/null 2>&1; then
        eval "$(fnm env --use-on-cd)"
    fi

    # Setup Oh My Posh if available
    if command -v oh-my-posh >/dev/null 2>&1; then
        if [ -n "$BASH_VERSION" ]; then
            eval "$(oh-my-posh init bash --config /src/oh-my-posh/themes/shell.omp.toml)"
        elif [ -n "$ZSH_VERSION" ]; then
            eval "$(oh-my-posh init zsh --config /src/oh-my-posh/themes/shell.omp.toml)"
        fi
    fi

fi