#!/usr/bin/env bash

[[ $EUID -ne 0 ]] && { error "This script must be run as root."; return 1; }

PUBLIC_SOURCE_DIR="/src"
POSH_DIR="$PUBLIC_SOURCE_DIR/oh-my-posh"
POSH_THEME_DIR="$POSH_DIR/themes"
BASH_THEME_PATH="$POSH_THEME_DIR/bash.omp.json"
ZSH_THEME_PATH="$POSH_THEME_DIR/zsh.omp.json"

DEFAULT_USER_GROUPS="shared"
CURRENT_USER="${SUDO_USER:-${USER:-root}}"

TOOLS_DIR="$PUBLIC_SOURCE_DIR/Tools"
SRC_BIN_DIR="$TOOLS_DIR/bin"
APPLICATIONS_DIR="$PUBLIC_SOURCE_DIR/Applications"
APP_BIN_DIR="$APPLICATIONS_DIR/bin"
APP_XDG_DIR="$APPLICATIONS_DIR/applications"


locate_sibling_file(){
    local filename=$1

    SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "$SCRIPT_DIR/$filename" ]]; then
        echo "$SCRIPT_DIR/$filename"
    else
        error "File '$filename' not found in '$SCRIPT_DIR'"
        return 1
    fi
}

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

# global helpers
cmd-exists() { command -v "$1" >/dev/null 2>&1; }

# Optimized Bulk Installer
install_if_missing() {
    if cmd-exists "dnf"; then
        # dnf handles simultaneous downloads natively when configured in dnf.conf
        dnf install -y "${to_install[@]}"
    elif cmd-exists "apt-get"; then
        apt-get update -y && apt-get install -y "${to_install[@]}"
    elif cmd-exists "pacman"; then
        pacman -S --noconfirm "${to_install[@]}"
    else
        error "No supported native system package manager found."
        exit 1
    fi
}

is_rpm_installed() {
    local rpm_file="$1"

    local pkg_nevra installed_nevra pkg_name

    pkg_name=$(rpm -qp --qf '%{NAME}' "$rpm_file") || return 1

    pkg_nevra=$(rpm -qp --qf '%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}' "$rpm_file")
    installed_nevra=$(rpm -q --qf '%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}' "$pkg_name" 2>/dev/null)

    [[ "$pkg_nevra" == "$installed_nevra" ]]
}

is_installed() {
    local type="$1"
    local target="$2"
    local tar_file="$3"

    case "$type" in
        rpm)
            if [[ ! -f "$target" ]]; then
                return 1
            fi
            local pkg_name
            pkg_name=$(rpm -qp --qf '%{NAME}' "$target" 2>/dev/null)
            if is_rpm_installed "$target"; then
                warning "$pkg_name is already installed with matching version. Skipping..."
                return 0
            fi
            return 1
            ;;
        tar)
            if [[ ! -d "$target" ]]; then
                return 1
            fi
            local info_file="$target/usetup.install-info"
            if [[ ! -f "$info_file" ]]; then
                return 1
            fi
            local installed_sha
            installed_sha=$(grep "^sha256=" "$info_file" | cut -d= -f2-)
            if [[ -z "$installed_sha" ]]; then
                return 1
            fi
            if [[ -z "$tar_file" ]]; then
                local folder_name
                folder_name=$(basename "$target")
                case "$folder_name" in
                    zed) tar_file="/tmp/usetup/zed.tar.gz" ;;
                    antigravity-ide) tar_file="/tmp/usetup/antigravity.tar.gz" ;;
                    golang) tar_file="/tmp/usetup/go.tar.gz" ;;
                    flutter) tar_file="/tmp/usetup/dart.tar.xz" ;;
                esac
            fi
            if [[ -f "$tar_file" ]]; then
                local current_sha
                current_sha=$(sha256sum "$tar_file" | cut -d' ' -f1)
                if [[ "$installed_sha" == "$current_sha" ]]; then
                    local app_name
                    app_name=$(grep "^name=" "$info_file" | cut -d= -f2-)
                    [[ -z "$app_name" ]] && app_name=$(basename "$target")
                    warning "$app_name is already installed and matches the current version hash. Skipping..."
                    return 0
                fi
            fi
            return 1
            ;;
        dnf)
            if rpm -q "$target" &>/dev/null; then
                warning "$target is already installed. Skipping..."
                return 0
            fi
            return 1
            ;;
        cmd)
            if command -v "$target" &>/dev/null; then
                warning "$target is already available in $(command -v "$target"). Skipping..."
                return 0
            fi
            return 1
            ;;
    esac
}

group_exists() {
    getent group "$1" >/dev/null 2>&1
}

user_in_group() {
    id -nG "$1" 2>/dev/null | tr ' ' '\n' | grep -qx "$2"
}


do_src_acl_sweep(){
    
    if ! command -v setfacl >/dev/null 2>&1; then
        install_if_missing acl
    fi

    info "Configuring rock-solid shared permissions..."
    
    # 1. Ownership & SetGID
    # SetGID (the '2' in 2775 or 's' flag) forces all NEW files/folders created inside 
    # to inherit the 'shared' group automatically, no matter who creates them.
    chown -R root:"$DEFAULT_USER_GROUPS" "$PUBLIC_SOURCE_DIR"
    find "$PUBLIC_SOURCE_DIR" -type d -exec chmod 2775 {} \;

    # 2. Access Control Lists (ACLs)
    if command -v setfacl >/dev/null 2>&1; then
        # Modify existing files (-m): 
        # Group gets read/write, and Capital X ensures executables KEEP their execute bit.
        setfacl -R \
            -m u::rwX \
            -m g:${DEFAULT_USER_GROUPS}:rwX \
            -m o::rX \
            "$PUBLIC_SOURCE_DIR"

        # Default for future files (-d):
        # Even if a file is moved (mv) or downloaded (curl), the directory forces 
        # these permissions onto the new file.
        setfacl -R -d \
            -m u::rwx \
            -m g:${DEFAULT_USER_GROUPS}:rwx \
            -m o::rx \
            "$PUBLIC_SOURCE_DIR"

        echo "Default ACLs configured successfully."
    else
        echo "WARNING: ACL support unavailable. Group inheritance relies solely on SetGID."
    fi
}

mkdirall(){
}










# 1. THE MAIN FLOW: DIR & PERMISSION SETUP
setup_src_dir(){
    info "Configuring /src shared workspace layout..."
    
    # Structural Generation
    mkdir -p "$PUBLIC_SOURCE_DIR"
    mkdir -p "$SRC_BIN_DIR"
    mkdir -p "$APP_BIN_DIR"
    mkdir -p "$APP_XDG_DIR"
    mkdir -p "$POSH_THEME_DIR"

    if ! group_exists "$DEFAULT_USER_GROUPS"; then
        info "Creating group '$DEFAULT_USER_GROUPS'..."
        groupadd "$DEFAULT_USER_GROUPS"
    fi

    if id "$CURRENT_USER" >/dev/null 2>&1; then
        if ! user_in_group "$CURRENT_USER" "$DEFAULT_USER_GROUPS"; then
            info "Adding '$CURRENT_USER' to '$DEFAULT_USER_GROUPS'..."
            usermod -aG "$DEFAULT_USER_GROUPS" "$CURRENT_USER"
            info "NOTE: User must re-login for group membership to become active."
        fi
    fi

    # Trigger a single master sweep over the entire structure
    do_src_acl_sweep
    
    # Copy script folder's login.sh to /src/login.sh and symlink to profile.d
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [ -f "$script_dir/login.sh" ]; then
        cp -f "$script_dir/login.sh" "$PUBLIC_SOURCE_DIR/login.sh"
        chmod 755 "$PUBLIC_SOURCE_DIR/login.sh"
        success "Copied login.sh to $PUBLIC_SOURCE_DIR/login.sh"
    else
        warning "login.sh not found in setup script directory. Creating an empty fallback."
        touch "$PUBLIC_SOURCE_DIR/login.sh"
    fi

    ln -sf "$PUBLIC_SOURCE_DIR/login.sh" "/etc/profile.d/src-workspace-env.sh"
    success "Linked global profile loader to /etc/profile.d/"

    # Source the profile loader to make all shell commands & variables available to setup.sh
    if [ -f "$PUBLIC_SOURCE_DIR/login.sh" ]; then
        source "$PUBLIC_SOURCE_DIR/login.sh"
    fi
}
setup_src_dir

path_prepend() {
    case ":$PATH:" in
        *:"$1":*) ;;
        *) export PATH="$1:$PATH" ;;
    esac
}

setup_src_bin() {
    mkdir -p "$SRC_BIN_DIR"
    mkdir -p "$APP_BIN_DIR"
    path_prepend "$SRC_BIN_DIR"
    path_prepend "$APP_BIN_DIR"
}
setup_src_bin


# an unified workspace linker function
# takes an associative array and decides the logic
_link_files(){
    local -n config="$1"

    local type="${config[type]:-bin}"
    local target_path="${config[target]:? "Error: target path is required"}"
    local dest_dir=""

    case "$type" in
        bin)     dest_dir="$SRC_BIN_DIR" ;;
        app)     dest_dir="$APP_BIN_DIR" ;;
        desktop) dest_dir="$APP_XDG_DIR" ;;
        *)       error "Unknown linking type: $type"; return 1 ;;
    esac

    [[ ! -e "$target_path" ]] && { warning "$target_path is not accessible."; return 1; }

    if [[ -d "$target_path" ]]; then
        # Loop through files in the directory and link them individually
        for f in "$target_path"/*; do 
            [[ -f "$f" ]] && ln -sf "$f" "$dest_dir/"
        done
        success "Linked binaries from folder '$(basename "$target_path")' -> $dest_dir"
    else
        ln -sf "$target_path" "$dest_dir/"
        success "Linked file '$(basename "$target_path")' -> $dest_dir"
    fi
}

# Unified Workspace Unlinker Engine
_unlink_files(){
    local -n config="$1" # FIX: Added reference assignment

    local type="${config[type]:-bin}"
    local target_path="${config[target]:? "Error: target path is required"}"
    local dest_dir=""

    case "$type" in
        bin)     dest_dir="$SRC_BIN_DIR" ;;
        app)     dest_dir="$APP_BIN_DIR" ;;
        desktop) dest_dir="$APP_XDG_DIR" ;;
        *)       error "Unknown unlinking type: $type"; return 1 ;;
    esac

    [[ ! -d "$dest_dir" ]] && return 0

    # Get the absolute canonical path of the target to ensure matching matches perfectly
    local abs_target
    abs_target="$(readlink -f "$target_path")"

    # Iterate through the destination directory to find matching links
    for item in "$dest_dir"/*; do
        [[ ! -e "$item" && ! -L "$item" ]] && continue # Catch dead symlinks too
        
        # Resolve where the current link points to
        local resolved_path
        resolved_path="$(readlink -f "$item")"

        if [[ "$resolved_path" == "$abs_target" ]]; then
            rm -f "$item"
            success "Removed link: $(basename "$item") from $dest_dir"
        
        # Special case: If the target was a directory, find links pointing INSIDE that directory
        elif [[ -d "$abs_target" && "$resolved_path" == "$abs_target"/* ]]; then
            rm -f "$item"
            success "Removed internal link: $(basename "$item") from $dest_dir"
        fi
    done
}

src-add() {
    declare -A config=( [type]="bin" )
    if [ "$1" = "rm" ]; then
        shift
        for target in "$@"; do config[target]="$target"; _unlink_files config; done
    else
        for target in "$@"; do config[target]="$target"; _link_files config; done
    fi
}

app-add() {
    declare -A config=( [type]="app" )
    if [ "$1" = "rm" ]; then
        shift
        for target in "$@"; do config[target]="$target"; _unlink_files config; done
    else
        for target in "$@"; do config[target]="$target"; _link_files config; done
    fi
}

app-desktop-add() {
    declare -A config=( [type]="desktop" )
    if [ "$1" = "rm" ]; then
        shift
        for target in "$@"; do config[target]="$target"; _unlink_files config; done
    else
        for target in "$@"; do config[target]="$target"; _link_files config; done
    fi
}

# we have 2 dirs as shared, 
# 1. src
# 2. applications
# the src contains the binary and the coding tools i use
# where the applications is like the opt dir where the GUI tools like browsers and other things like that are stored
# /src/
# ├── Applications/        <-- Central app binaries go here (VS Code, Brave, JetBrains)
# │   ├── bin/             <-- App execution links (added by app-add)
# │   └── applications/    <-- Native Linux Desktop Entries (added by app-icon-add)
# ├── Tools/               <-- The main cli tools and their concrete folder lives inside this
# │   └── bin/             <-- The bin of each tool, symlinked (added by src-add)
# └── login.sh             <-- The shared script that lives in the /etc/profile.d as a symlink to autoload the paths and XDG dir

make_desktop_file() {
    # Expects the name of an associative array passed by reference
    local -n config="$1"
    
    # 1. Core Required Fields (Will halt script if missing)
    local name="${config[name]:? "Error: 'name' is required"}"
    local exec_path="${config[exec]:? "Error: 'exec' is required"}"
    
    # 2. XDG Compliant Defaults
    local type="${config[type]:-Application}"
    local icon="${config[icon]:-system-run}"
    local terminal="${config[terminal]:-false}"
    local categories="${config[categories]:-Utility;}"
    local comment="${config[comment]:-Launch $name}"
    
    # 3. Desktop Environment & Window Manager Optimizations
    # GenericName: Shows up in search bars next to or below the app name (e.g., "Web Browser")
    local generic="${config[generic]:-$name}"
    
    # StartupWMClass: Crucial for GNOME/KDE docks to group windows under this icon.
    # Falls back to a lowercase, hyphenated string matching the binary name.
    local wm_class="${config[wm_class]:-$(basename "${exec_path%% *}" | tr '[:upper:]' '[:lower:]')}"
    
    # Keywords: Extra search tags for launchers (Rofi, Wofi, GNOME Activities)
    local keywords="${config[keywords]:-}"

    # Define and clean target path
    local file_name="${name// /_}.desktop"
    local save_file_path="$APPLICATIONS_DIR/applications/$file_name"
    
    # Ensure directory exists before writing
    mkdir -p "$(dirname "$save_file_path")"

    # 4. Generate the Compliant File
    cat > "$save_file_path" <<EOF
[Desktop Entry]
Type=$type
Name=$name
GenericName=$generic
Comment=$comment
Exec=$exec_path
Icon=$icon
Terminal=$terminal
Categories=$categories
StartupWMClass=$wm_class
StartupNotify=true
EOF

    # Dynamically inject Keywords only if they are provided (keeps file clean)
    if [[ -n "$keywords" ]]; then
        echo "Keywords=$keywords" >> "$save_file_path"
    fi

    success "Desktop entry for '$name' deployed perfectly to: $save_file_path"
}

install_tar(){
    local name="${1:? "Error: 'name' argument is required for extraction."}"
    local tar_path="${2:? "Error: 'tar_path' argument is required for extraction."}"
    local type="${3:-application}"
    local custom_dest="${4:-}"
    
    if [[ ! -f "$tar_path" ]]; then
        error "Archive not found at path: $tar_path"
        return 1
    fi

    local dir_name="${name//-/_}"
    local dest_path=""

    if [[ -n "$custom_dest" ]]; then
        dest_path="$custom_dest"
    else
        case "$type" in
            application) dest_path="$APPLICATIONS_DIR/$dir_name" ;;
            tool)        dest_path="$TOOLS_DIR/$dir_name" ;;
            *)           dest_path="/opt/$dir_name" ;;
        esac
    fi

    local sha256_val=""
    if [[ -f "$tar_path" ]]; then
        sha256_val=$(sha256sum "$tar_path" | cut -d' ' -f1)
    fi

    mkdir -p "$dest_path"
    export XZ_OPT="-T0"
    
    local compress_flag=""
    if [[ "$tar_path" =~ \.gz$ ]] && cmd-exists pigz; then
        compress_flag="--use-compress-program=pigz"
    elif [[ "$tar_path" =~ \.xz$ ]] && cmd-exists pixz; then
        compress_flag="--use-compress-program=pixz"
    fi

    info "Extracting '$name' to $dest_path..."

    # Extract stripping top-level directory safely
    tar $compress_flag -xf "$tar_path" -C "$dest_path" --strip-components=1 --overwrite-dir
    
    if [ $? -ne 0 ]; then
        error "Failed to extract archive: $tar_path"
        return 1
    fi
    
    success "Successfully extracted $name to $dest_path"

    local version="${5:-}"
    if [[ -z "$version" ]]; then
        case "$name" in
            go) version="1.22.4" ;;
            flutter|dart) version="3.44.2" ;;
            antigravity|antigravity-ide) version="2.0.4" ;;
            zed) version="latest" ;;
            *) version="1.0.0" ;;
        esac
    fi

    local info_file="$dest_path/usetup.install-info"
    cat > "$info_file" <<EOF
name=$name
version=$version
sha256=$sha256_val
installed_at=$(date +%Y-%m-%d)
EOF

    rm -f "$tar_path"
}

ensure_gum(){
    if ! cmd-exists "gum"; then
    info "Gum is not installed. Installing..."
    if cmd-exists "dnf" ; then
            echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | tee /etc/yum.repos.d/charm.repo

        rpm --import https://repo.charm.sh/yum/gpg.key
        dnf install -y gum
        elif cmd-exists "apt-get"; then
            mkdir -p /etc/apt/keyrings
            curl -fsSL https://repo.charm.sh/apt/gpg.key | gpg --dearmor -o /etc/apt/keyrings/charm.gpg
            echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | tee /etc/apt/sources.list.d/charm.list
            apt update && apt install -y gum
        elif cmd-exists "pacman"; then
            pacman -S --noconfirm gum
        fi
    fi

    if ! cmd-exists "gum"; then
        echo "Gum installation failed. Please install it manually. (This script requires gum for interactive prompts.)"
        exit 1
    fi

}

ensure_oh_my_posh() {
    echo "Bootstrapping Oh My Posh..."
    if command -v oh-my-posh >/dev/null 2>&1 || \
    [ -x "$POSH_DIR/bin/oh-my-posh" ]; then
        echo "Oh My Posh already installed. Skipping bootstrap."
        ensure_posh_themes
        return 0
    fi
    
    mkdir -p "$POSH_DIR/bin"
    curl -s https://ohmyposh.dev/install.sh | bash -s -- -d "$POSH_DIR/bin"
    src-add "$POSH_DIR/bin"
    setup_src_bin
    ensure_posh_themes
}

ensure_posh_themes(){
    BASH_THEME_URL="https://raw.githubusercontent.com/JanDeDobbeleer/oh-my-posh/refs/heads/main/themes/clean-detailed.omp.json"
    ZSH_THEME_URL="https://raw.githubusercontent.com/JanDeDobbeleer/oh-my-posh/refs/heads/main/themes/atomic.omp.json"  

    [ ! -f "$BASH_THEME_PATH" ] && curl -sL -o "$BASH_THEME_PATH" "$BASH_THEME_URL"
    [ ! -f "$ZSH_THEME_PATH" ] && curl -sL -o "$ZSH_THEME_PATH" "$ZSH_THEME_URL"
}

setup_zsh(){
    local_zsh_init_script=$(locate_sibling_file "zsh-init.sh")

    # now run the script
    source "$local_zsh_init_script"
    success "Zsh init script installed to /etc/zshrc"
}

echo "Installing core packages..."
# downloaders packages and its helpers
install_if_missing aria2 curl wget unzip xz tar pixz pigz
 
# shell tools like fzf, zoxide, eza and oh-my-posh and gum for interactive scripts
install_if_missing fzf zoxide eza oh-my-posh gum jq

# the main git and shell tools
install_if_missing zsh git
setup_zsh

echo "Bootstrapping Witr..."
if ! cmd-exists "witr" && [ ! -f "/usr/local/bin/witr" ]; then
    curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash
fi

LANGUAGES_TOOLS=$(gum choose --no-limit "Node.js" "Bun" "Python" "Go" "Rust" "Dart" "Podman & Compose" "Caddy" --header "Select languages/tools to install:")

# its not jsut browsers but browser supporting tools too
BROWSERS_AND_TOOLS=$(gum choose --no-limit "Chrome" "Brave" "Bitwarden" "Propiretary codecs (Ffmpeg freeworld)"  --header "Select browsers/tools to install:")
IDE_AND_TOOLS=$(gum choose --no-limit "Neovim" "Antigravity (ide)" "VS Code" "Jet brains Data Grip" "Zed" --header "Select IDE/Tools to install:")

GOLANG_DOWNLOAD_LINK="https://dl.google.com/go/go1.22.4.linux-amd64.tar.gz"

# DART IS HEAVY SO BETTER TO INSTALL WITH MULTI THEREADED  aria2c lib then standard curl or wget
DART_DOWNLOAD_LINK="https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_3.44.2-stable.tar.xz"
JETBRAINS_DATA_GRIP_DOWNLOAD_LINK="https://download-cdn.jetbrains.com/datagrip/datagrip-2026.1.3.tar.gz"

CHROME_DOWNLOAD_LINK="https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm"
ANTIGRAVITY_DOWNLOAD_LINK="https://edgedl.me.gvt1.com/edgedl/release2/j0qc3/antigravity/stable/2.0.4-6381998290370560/linux-x64/Antigravity%20IDE.tar.gz"
VS_CODE_DOWNLOAD_LINK="https://vscode.download.prss.microsoft.com/dbazure/download/stable/93cfdd489c3b228840d0f86ec77c3636277c93ea/code-1.125.0-1781601611.el8.x86_64.rpm"
ZED_DOWNLOAD_LINK="https://release-assets.githubusercontent.com/github-production-release-asset/340547520/5371e7e4-8f6c-499b-aed7-e249742fb2f1?sp=r&sv=2018-11-09&sr=b&spr=https&se=2026-06-18T17%3A13%3A48Z&rscd=attachment%3B+filename%3Dzed-linux-x86_64.tar.gz&rsct=application%2Foctet-stream&skoid=96c2d410-5711-43a1-aedd-ab1947aa7ab0&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skt=2026-06-18T16%3A13%3A03Z&ske=2026-06-18T17%3A13%3A48Z&sks=b&skv=2018-11-09&sig=EnTHa8T5nJ62L3PDKG00L9pNexX2DyFMJpoZaZwjMgA%3D&jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmVsZWFzZS1hc3NldHMuZ2l0aHVidXNlcmNvbnRlbnQuY29tIiwia2V5Ijoia2V5MSIsImV4cCI6MTc4MTgwNDMzNywibmJmIjoxNzgxODAwNzM3LCJwYXRoIjoicmVsZWFzZWFzc2V0cHJvZHVjdGlvbi5ibG9iLmNvcmUud2luZG93cy5uZXQifQ.i0P-U3C1fzCU605pDuqPao6ff7dO9LmHr8Hyk87ZSW4&response-content-disposition=attachment%3B%20filename%3Dzed-linux-x86_64.tar.gz&response-content-type=application%2Foctet-stream"
BITWARDEN_DOWNLOAD_LINK="https://release-assets.githubusercontent.com/github-production-release-asset/53538899/e355eb83-eed4-49d4-9846-9cbb9de5c48b?sp=r&sv=2018-11-09&sr=b&spr=https&se=2026-06-18T17%3A40%3A23Z&rscd=attachment%3B+filename%3DBitwarden-2026.5.0-x86_64.rpm&rsct=application%2Foctet-stream&skoid=96c2d410-5711-43a1-aedd-ab1947aa7ab0&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skt=2026-06-18T16%3A39%3A39Z&ske=2026-06-18T17%3A40%3A23Z&sks=b&skv=2018-11-09&sig=rz%2F7543y%2FH5qx4V%2BJpfso8Y4fdJ%2BiJRNftw6yWB8UKk%3D&jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmVsZWFzZS1hc3NldHMuZ2l0aHVidXNlcmNvbnRlbnQuY29tIiwia2V5Ijoia2V5MSIsImV4cCI6MTc4MTgwNTI2OSwibmJmIjoxNzgxODAxNjY5LCJwYXRoIjoicmVsZWFzZWFzc2V0cHJvZHVjdGlvbi5ibG9iLmNvcmUud2luZG93cy5uZXQifQ.izDZptSBRiT8W1hxIRcK2sP6bF3ZVhQ9bdmtZJ3oPcs&response-content-disposition=attachment%3B%20filename%3DBitwarden-2026.5.0-x86_64.rpm&response-content-type=application%2Foctet-stream"

has_opt() {
    # Returns 0 (true) if $2 is found inside the selection list $1
    [[ "$1" == *"$2"* ]]
}

has_browser(){
    # proxies the has opt and the arg is just the target option to choose
    has_opt "$BROWSERS_AND_TOOLS" "$1"
}

has_ide(){
    # proxies the has opt and the arg is just the target option to choose
    has_opt "$IDE_AND_TOOLS" "$1"
}

has_language(){
    # proxies the has opt and the arg is just the target option to choose
    has_opt "$LANGUAGES_TOOLS" "$1"
}

# downloads the needed rpm or tar files and puts them inside the
# /tmp/usetup/ dir
# it uses curl parallel downloading for RPMs, and aria2c for heavy TAR files.
setup_artifacts(){
    mkdir -p /tmp/usetup
    
    # 1. Setup Curl Arguments (Strictly for RPM files)
    local curl_args=()
    has_browser "Chrome" && curl_args+=("$CHROME_DOWNLOAD_LINK" "-o" "/tmp/usetup/chrome_browser.rpm")
    has_browser "Bitwarden" && curl_args+=("$BITWARDEN_DOWNLOAD_LINK" "-o" "/tmp/usetup/bitwarden.rpm")
    has_ide "VS Code" && curl_args+=("$VS_CODE_DOWNLOAD_LINK" "-o" "/tmp/usetup/vscode.rpm")

    if [ ${#curl_args[@]} -gt 0 ]; then
        info "Downloading selected RPM packages in parallel via curl..."
        curl -L -Z --parallel-max 5 "${curl_args[@]}"
    fi
    
    # 2. Setup Aria2c Arguments (For heavy TAR archives to maximize speed)
    local aria2_args=()
    
    # Heavy IDEs (Tarballs)
    has_ide "Antigravity" && aria2_args+=("$ANTIGRAVITY_DOWNLOAD_LINK" "--out=antigravity.tar.gz")
    has_ide "Zed" && aria2_args+=("$ZED_DOWNLOAD_LINK" "--out=zed.tar.gz")
    has_ide "Jet brains tool box" && aria2_args+=("$JETBRAINS_TOOLBOX_DOWNLOAD_LINK" "--out=jetbrains-toolbox.tar.gz")
    
    # Languages (Tarballs)
    has_language "Dart" && aria2_args+=("$DART_DOWNLOAD_LINK" "--out=dart.tar.xz")
    has_language "Go" && aria2_args+=("$GOLANG_DOWNLOAD_LINK" "--out=go.tar.gz")

    if [ ${#aria2_args[@]} -gt 0 ]; then
        info "Downloading heavy archives in parallel via aria2c..."
        # --dir: Global destination directory
        # -j 3: Download up to 3 files simultaneously
        # -x 16 / -s 16: Split each individual file into 16 parallel chunks for maximum speed
        # --show-console-readout=false / --summary-interval=5: Keeps the terminal output clean
        aria2c --dir=/tmp/usetup \
               -j 3 \
               -x 16 \
               -s 16 \
               --show-console-readout=false \
               --summary-interval=5 \
               "${aria2_args[@]}"
    fi
}

# The must essential tool for the laptop 
install_damx(){
    curl -fsSL https://raw.githubusercontent.com/PXDiv/Div-Acer-Manager-Max/refs/heads/main/scripts/remoteSetup.sh -o /tmp/setup.sh && sudo bash /tmp/setup.sh
}

install_codecs(){
    sudo dnf install https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm
    sudo dnf swap ffmpeg-free ffmpeg --allowerasing
    sudo dnf install vlc-plugins-freeworld
    sudo dnf install vlc
}

install_nvim(){
    sudo dnf install -y neovim
    git clone https://github.com/LazyVim/starter ~/.config/nvim
    rm -rf ~/.config/nvim/.git
}

install_chrome() {
    is_installed "rpm" "/tmp/usetup/chrome_browser.rpm" && return 0
    info "Installing Google Chrome..."
    sudo dnf install -y /tmp/usetup/chrome_browser.rpm
}

install_brave() {
    is_installed "dnf" "brave-browser" && return 0
    info "Installing Brave Browser..."
    sudo dnf install -y dnf-plugins-core
    sudo dnf config-manager --add-repo https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo
    sudo dnf install -y brave-browser
}

install_bitwarden() {
    is_installed "rpm" "/tmp/usetup/bitwarden.rpm" && return 0
    info "Installing Bitwarden..."
    sudo dnf install -y /tmp/usetup/bitwarden.rpm
}

install_zed() {
    is_installed "tar" "$APPLICATIONS_DIR/zed" "/tmp/usetup/zed.tar.gz" && return 0
    info "Installing Zed Editor..."
    
    local dest_dir="$APPLICATIONS_DIR/zed"
    install_tar "zed" "/tmp/usetup/zed.tar.gz" "application" "$dest_dir"
    
    app-add "$dest_dir/bin/zed"
    
    declare -A desk_config=(
        [name]="Zed Editor"
        [exec]="$dest_dir/bin/zed"
        [icon]="$dest_dir/share/icons/hicolor/512x512/apps/zed.png"
        [categories]="Development;IDE;"
        [wm_class]="zed"
    )
    make_desktop_file desk_config
    success "Zed Editor installed successfully."
}

install_vscode() {
    is_installed "rpm" "/tmp/usetup/vscode.rpm" && return 0
    info "Installing VS Code..."
    sudo dnf install -y /tmp/usetup/vscode.rpm

    local extensions_to_install=("golang.go" "dart-code.dart-code" "dart-code.flutter" "vscode-icons-team.vscode-icons" "yzhang.markdown-all-in-one" "ms-azuretools.vscode-docker" "esbenp.prettier-vscode" "dsznajder.es7-react-js-snippets" "dbaeumer.vscode-eslint" "jeanp413.open-remote-wsl" "oven.bun-vscode" "bradlc.vscode-tailwindcss" "tamasfe.even-better-toml")
    
    local EXTENSIONS_STORE="$TOOLS_DIR/shared/vscode-extensions"
    
    # 1. Prepare the parent profile directory
    mkdir -p "$HOME/.vscode"

    # 2. Safely handle the shared store layout
    if [ -d "$HOME/.vscode/extensions" ] && [ ! -L "$HOME/.vscode/extensions" ]; then
        # If a real directory exists, move its text/contents cleanly to the shared store location
        mkdir -p "$(dirname "$EXTENSIONS_STORE")"
        mv "$HOME/.vscode/extensions" "$EXTENSIONS_STORE"
    else
        # Ensure the shared store folder exists directly
        mkdir -p "$EXTENSIONS_STORE"
        rm -f "$HOME/.vscode/extensions" # Clean up if it was a broken symlink
    fi

    # 3. Establish the symlink cleanly
    ln -s "$EXTENSIONS_STORE" "$HOME/.vscode/extensions"

    # 4. Install all extensions directly into the shared store
    info "Installing VS Code extensions..."
    for ext in "${extensions_to_install[@]}"; do
        code --install-extension "$ext" --force
    done

    success "VS Code installed successfully."
}

install_antigravity() {
    is_installed "tar" "$APPLICATIONS_DIR/antigravity-ide" "/tmp/usetup/antigravity.tar.gz" && return 0
    info "Installing Antigravity IDE..."
    
    local dest_dir="$APPLICATIONS_DIR/antigravity-ide"
    install_tar "antigravity-ide" "/tmp/usetup/antigravity.tar.gz" "application" "$dest_dir"

    local exec_file="$dest_dir/bin/antigravity-ide"
    local icon_file="$dest_dir/resources/app/resources/linux/code.png"
 
    if [ -n "$exec_file" ] ; then
        app-add "$exec_file"
        
        declare -A desk_config=(
            [name]="Antigravity IDE"
            [exec]="$exec_file"
            [icon]="$icon_file"
            [categories]="Development;IDE;"
        )
        make_desktop_file desk_config
    else
        error "Could not locate Antigravity executable in extracted files."
        return 1
    fi

    # Clean path management for shared extensions
    local EXTENSIONS_STORE="$TOOLS_DIR/shared/vscode-extensions"
    local ANTIGRAVITY_BASE_DIR="$HOME/.antigravity-ide"
    local ANTIGRAVITY_EXT_DIR="$ANTIGRAVITY_BASE_DIR/extensions"
    
    # 1. Create the parent config directory directly (No need to bootstrap with the binary)
    mkdir -p "$ANTIGRAVITY_BASE_DIR"

    # 2. Clean up any existing local extensions folder or broken symlink
    rm -rf "$ANTIGRAVITY_EXT_DIR"

    # 3. Link directly to the exact same shared store
    ln -s "$EXTENSIONS_STORE" "$ANTIGRAVITY_EXT_DIR"

    success "Antigravity IDE installed successfully with shared extensions."
}

install_node() {
    is_installed "cmd" "node" && return 0

    echo "Installing Node.js via FNM..."
    curl -fsSL https://fnm.vercel.app/install | bash -s -- --install-dir "$PUBLIC_SOURCE_DIR/fnm" --skip-shell
    
    src-add "$PUBLIC_SOURCE_DIR/fnm"
    setup_src_bin

    export PATH="$PUBLIC_SOURCE_DIR/fnm:$PATH"

    eval "$(fnm env --use-on-cd)"

    fnm install --latest
    fnm use latest
    fnm default latest

    eval "$(fnm env --use-on-cd)"

    echo "Enabling pnpm via Corepack..."
    corepack enable
    corepack prepare pnpm@latest --activate
}

install_bun(){
    is_installed "cmd" "bun" && return 0

    echo "Installing Bun..."
    export BUN_INSTALL="$PUBLIC_SOURCE_DIR/bun"
    curl -fsSL https://bun.sh/install | bash
    src-add "$BUN_INSTALL/bin"
    setup_src_bin
}

install_python() {
    is_installed "cmd" "python" && return 0

    echo "Installing Python via UV..."
    export UV_INSTALL_DIR="$PUBLIC_SOURCE_DIR/uv"
    curl -LsSf https://astral.sh/uv/install.sh | sh
    
    src-add "$PUBLIC_SOURCE_DIR/uv"
    setup_src_bin
    "$SRC_BIN_DIR/uv" python install
}

install_go() {
    is_installed "tar" "$PUBLIC_SOURCE_DIR/golang" "/tmp/usetup/go.tar.gz" && return 0
    info "Installing Go..."
    install_tar "go" "/tmp/usetup/go.tar.gz" "tool" "$PUBLIC_SOURCE_DIR/golang"
    src-add "$PUBLIC_SOURCE_DIR/golang/go/bin"
    setup_src_bin
}

install_rust() {
    is_installed "cmd" "rustc" && return 0

    echo "Installing Rust..."
    export RUSTUP_HOME="$PUBLIC_SOURCE_DIR/rust/rustup"
    export CARGO_HOME="$PUBLIC_SOURCE_DIR/rust/cargo"
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- --default-toolchain stable -y --no-modify-path
    src-add "$PUBLIC_SOURCE_DIR/rust/cargo/bin"
    setup_src_bin
}

install_dart() {
    is_installed "tar" "$PUBLIC_SOURCE_DIR/flutter" "/tmp/usetup/dart.tar.xz" && return 0

    info "Installing Dart & Flutter..."
    install_tar "flutter" "/tmp/usetup/dart.tar.xz" "tool" "$PUBLIC_SOURCE_DIR/flutter"

    src-add "$PUBLIC_SOURCE_DIR/flutter/bin"
    setup_src_bin
    success "Dart & Flutter SDK successfully installed!"
}

install_podman(){
    is_installed "dnf" "podman" && return 0

    echo "Installing Podman..."
    install_if_missing podman
        mkdir -p /etc/containers/registries.conf.d

cat > /etc/containers/registries.conf.d/99-dockerio.conf <<'EOF'
unqualified-search-registries = [
    "docker.io",
    "quay.io",
    "registry.fedoraproject.org"
]
EOF
}

install_podman_compose() {
    if ! command -v podman >/dev/null 2>&1; then
        echo "Podman is not installed. Installing Podman first..."
        install_podman
    fi

    is_installed "dnf" "podman-compose" && return 0
    install_if_missing podman-compose   
}

# installs the caddy and frankenphp 
# then it swpas the caddy with frankenphp, so that the caddy command is actually frankenphp, but with caddy's features
# this is the best way to ensure the php feature builtin to caddy with standard 
# systemctl reload caddy no need to remember anything
install_caddy_frankenphp() {
    echo "Installing Caddy..."

    dnf install dnf5-plugins
    dnf copr enable @caddy/caddy
    dnf install caddy
    
    echo "Installing FrankenPHP..."
    curl https://frankenphp.dev/install.sh | sh

    echo "Swapping Caddy with FrankenPHP..."
    
    CADDY_PATH=$(which caddy)
    FRANKENPHP_PATH=$(which frankenphp)

    if caddy version 2>&1 | grep -q "FrankenPHP"; then
        echo "Caddy is already FrankenPHP. Skipping swap."
    else
        mv "$CADDY_PATH" "$CADDY_PATH.bak"
        mv "$FRANKENPHP_PATH" "$CADDY_PATH"
    fi

    # ENABLE the caddy service, which is now actually frankenphp, so that it can be managed with systemctl
    systemctl enable caddy
}
 
# Trigger artifact setup for selected browsers/IDEs
setup_artifacts

# Install chosen languages and tools
has_language "Node.js" && install_node
has_language "Bun" && install_bun
has_language "Python" && install_python
has_language "Go" && install_go
has_language "Rust" && install_rust
has_language "Dart" && install_dart
has_language "Podman & Compose" && install_podman_compose
has_language "Caddy" && install_caddy_frankenphp

# Install chosen browsers and tools
has_browser "Chrome" && install_chrome
has_browser "Brave" && install_brave
has_browser "Bitwarden" && install_bitwarden
has_browser "Propiretary codecs" && install_codecs

# Install chosen IDEs and tools
has_ide "Neovim" && install_nvim
has_ide "Antigravity" && install_antigravity
has_ide "VS Code" && install_vscode
has_ide "Jet brains tool box" && install_jetbrains_toolbox
has_ide "Zed" && install_zed

chown -R root:"$DEFAULT_USER_GROUPS" "$PUBLIC_SOURCE_DIR"

do_src_acl_sweep

echo "Setup complete. /src is ready."
