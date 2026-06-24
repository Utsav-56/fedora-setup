# Fedora Developer Environment Bootstrapper (`usetup`)

A simple, fast, and automated tool written in Go to set up your Fedora Workstation for development. With a single command, it configures a central workspace under `/src`, optimizing permissions for seamless coding and installing your favorite runtimes, IDEs, and tools with multi-threaded download speeds.

---

## How to Use It

Open your terminal and paste the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/Utsav-56/fedora-setup/main/run.sh | bash

```

> [!NOTE]
> This tool sets up core system paths and layout configurations, so it will prompt you for your `sudo` password to execute.

---

## What You Get

An interactive checklist will let you select exactly what you want to install.

### 1. Runtimes & Languages

* **Go & Rust:** Complete compiler toolchains configured globally.
* **Dart & Flutter:** Mobile and web development SDKs.
* **Node.js & Bun:** Modern JavaScript environments using fast package managers (`fnm` and `bun`).
* **Python:** Isolated version management powered by `uv`.
* **Docker/Podman:** Native containerization engines.
* **Caddy & FrankenPHP:** Modern web server stack ready for local testing.

### 2. IDEs & Browsers

* **Popular Editors:** JetBrains DataGrip, VS Code, Zed Editor, Neovim, and Antigravity IDE.
* **Browsers:** Google Chrome and Brave Browser.
* **Security & Media:** Bitwarden credential client and full multi-media codecs (VLC/FFmpeg) via RPM Fusion.

### 3. Shell Upgrades

* Pre-configured **Zsh** paired with **Oh My Posh**, **fzf** (fuzzy finder), **zoxide** (smart directory jumping), and **eza** (modern `ls` replacement).

---

## The `/src` Workspace Layout

Instead of cluttering your hidden home directories, everything is neatly organized inside a dedicated, shared `/src` partition:

* `/src/Applications/` — Graphical apps like Zed, DataGrip, and your desktop launcher icons.
* `/src/Tools/` — Command-line tools and language compilers (Go, Rust, Flutter).
* `/src/Tools/shared/` — Shared global storage (e.g., sharing your VS Code extensions across the system to save space).
* `/src/login.sh` — An automated script linked to `/etc/profile.d/` that injects all software paths dynamically whenever you log in.