#!/usr/bin/env bash



    # zsh is installed so it will always be available
if ! grep -q "BEGIN SRC ZINIT BOOTSTRAP" /etc/zshrc 2>/dev/null; then

cat << 'EOF' >> /etc/zshrc

# BEGIN SRC ZINIT BOOTSTRAP
ZINIT_HOME="${XDG_DATA_HOME:-${HOME}/.local/share}/zinit/zinit.git"
if [ ! -d "$ZINIT_HOME" ]; then
    mkdir -p "$(dirname $ZINIT_HOME)"
    git clone https://github.com/zdharma-continuum/zinit.git "$ZINIT_HOME"
fi
source "${ZINIT_HOME}/zinit.zsh"

# History Opts
HISTSIZE=50000
SAVEHIST=50000
HISTFILE=~/.zsh_history
setopt INC_APPEND_HISTORY SHARE_HISTORY HIST_IGNORE_ALL_DUPS HIST_REDUCE_BLANKS

# Zinit Plugins
zinit snippet OMZP::git
zinit snippet OMZP::sudo
zinit snippet OMZP::extract

zinit light wfxr/forgit
zinit light MichaelAquilina/zsh-you-should-use
zinit light zsh-users/zsh-history-substring-search

zinit wait lucid for \
    atinit"zicompinit; zicdreplay" \
    zsh-users/zsh-completions

zinit light Aloxaf/fzf-tab
zinit light zsh-users/zsh-autosuggestions
zinit light zdharma-continuum/fast-syntax-highlighting

# END SRC ZINIT BOOTSTRAP

EOF

fi