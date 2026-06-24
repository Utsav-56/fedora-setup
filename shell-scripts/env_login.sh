#!/bin/sh
# /src/env_login.sh -> Contains all SDK home and cache environment variables

# --- 2. DEVELOPMENT SDK HOME DIRECTORIES ---
export ANDROID_HOME="{{.AndroidHome}}"
export FLUTTER_PATH="{{.FlutterPath}}"
export GO_PATH="{{.GoPath}}"
export BUN_INSTALL="{{.BunInstall}}"
export DENO_INSTALL="{{.DenoInstall}}"
export FNM_DIR="{{.FnmDir}}"
export PNPM_HOME="{{.PnpmHome}}"
export JAVA_HOME="{{.JavaHome}}"

# --- 3. GLOBAL CACHE SETTINGS ---
export GOCACHE="{{.GoCache}}"
export GOMODCACHE="{{.GoModCache}}"
export PUB_CACHE="{{.PubCache}}"
export PNPM_STORE_PATH="{{.PnpmStorePath}}"
export UV_CACHE_DIR="{{.UvCacheDir}}"

# --- 4. RUST & PYTHON (uv) ---
export RUSTUP_HOME="{{.RustupHome}}"
export CARGO_HOME="{{.CargoHome}}"
export UV_PYTHON_INSTALL_DIR="{{.UvPythonInstallDir}}"
export UV_TOOL_DIR="{{.UvToolDir}}"
