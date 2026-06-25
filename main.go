package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"text/template"
	"time"

	"charm.land/huh/v2"

	"setup/config"
	"setup/downloader"
	"setup/installer"
	"setup/installer/browser"
	"setup/installer/ide"
	"setup/installer/lang"
	"setup/logger"
	"setup/sysutils"
)

//go:embed shell-scripts/path_login.sh
var loginScriptTmpl string

//go:embed shell-scripts/env_login.sh
var envScriptTmpl string

//go:embed oh-my-posh/themes/shell.omp.toml
var poshThemeContent string

var (
	selectedLangs    []string
	selectedBrowsers []string
	selectedIDEs     []string
)

var dl *downloader.Downloader

func init() {
	logger.Info("==========================================")
	logger.Info("            USETUP FEDORA GO              ")
	logger.Info("==========================================")

	// 1. Root Check
	if !sysutils.IsRoot() {
		logger.Error("This setup application must be run as root.")
		os.Exit(1)
	}

	// 2. Ensure DNF Package Manager
	if !sysutils.CommandExists("dnf") {
		logger.Error("This setup is designed specifically for Fedora/DNF environments.")
		os.Exit(1)
	}

	// 3. Bootstrapping Core Package Dependencies
	logger.Info("Bootstrapping core packages (aria2, acl, git, etc.)...")
	corePkgs := config.CorePackages
	if err := installer.Package(corePkgs...); err != nil {
		logger.Warning("failed to install some core packages: %v", err)
	}

	// 4. Start Aria2 RPC Daemon
	logger.Info("Launching Aria2 RPC daemon...")
	_ = exec.Command("pkill", "-f", "aria2c.*67486").Run() // clean up any previous instance
	cmd := exec.Command("aria2c",
		"--enable-rpc",
		fmt.Sprintf("--rpc-listen-port=%d", config.Aria2RpcPort),
		fmt.Sprintf("--rpc-secret=%s", config.Aria2RpcSecret),
		"--rpc-listen-all=false",
		"--daemon=true",
	)
	if err := cmd.Run(); err != nil {
		logger.Warning("starting aria2c: %v", err)
	}

	// Give the daemon a moment to bind
	time.Sleep(500 * time.Millisecond)

	dl = downloader.NewDownloader(downloader.Config{
		Host:   config.Aria2RpcHost,
		Port:   fmt.Sprintf("%d", config.Aria2RpcPort),
		Secret: config.Aria2RpcSecret,
	})
}

// RenderLoginScript renders the login script.
func RenderLoginScript() (string, error) {
	return loginScriptTmpl, nil
}

// RenderEnvScript renders the env_login script template with configuration paths.
func RenderEnvScript() (string, error) {
	tmpl, err := template.New("env").Parse(envScriptTmpl)
	if err != nil {
		return "", err
	}

	data := struct {
		AndroidHome        string
		FlutterPath        string
		GoPath             string
		BunInstall         string
		DenoInstall        string
		FnmDir             string
		PnpmHome           string
		JavaHome           string
		GoCache            string
		GoModCache         string
		PubCache           string
		PnpmStorePath      string
		UvCacheDir         string
		RustupHome         string
		CargoHome          string
		UvPythonInstallDir string
		UvToolDir          string
	}{
		AndroidHome:        config.AndroidHome,
		FlutterPath:        filepath.Join(config.PublicSourceDir, "flutter", "flutter_"+config.Tools["dart"].Version),
		GoPath:             filepath.Join(config.PublicSourceDir, "golang", "go"+config.Tools["go"].Version),
		BunInstall:         filepath.Join(config.PublicSourceDir, "bun"),
		DenoInstall:        config.DenoInstall,
		FnmDir:             config.FnmDir,
		PnpmHome:           config.PnpmHome,
		JavaHome:           config.JavaHome,
		GoCache:            config.GoCache,
		GoModCache:         config.GoModCache,
		PubCache:           config.PubCache,
		PnpmStorePath:      config.PnpmStorePath,
		UvCacheDir:         config.UvCacheDir,
		RustupHome:         filepath.Join(config.PublicSourceDir, "rust", "rustup"),
		CargoHome:          filepath.Join(config.PublicSourceDir, "rust", "cargo"),
		UvPythonInstallDir: config.UvPythonInstallDir,
		UvToolDir:          config.UvToolDir,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func main() {
	// Set SDK and global cache environment variables for the current run to support installer tools
	os.Setenv("ANDROID_HOME", config.AndroidHome)
	os.Setenv("ANDROID_SDK_ROOT", config.AndroidHome)
	os.Setenv("FLUTTER_PATH", filepath.Join(config.PublicSourceDir, "flutter", "flutter_"+config.Tools["dart"].Version))
	os.Setenv("FLUTTER_ROOT", filepath.Join(config.PublicSourceDir, "flutter", "flutter_"+config.Tools["dart"].Version))
	os.Setenv("GO_PATH", filepath.Join(config.PublicSourceDir, "golang", "go"+config.Tools["go"].Version))
	os.Setenv("GOROOT", filepath.Join(config.PublicSourceDir, "golang", "go"+config.Tools["go"].Version))
	os.Setenv("GOPATH", filepath.Join(config.PublicSourceDir, "go"))
	os.Setenv("BUN_INSTALL", filepath.Join(config.PublicSourceDir, "bun"))
	os.Setenv("DENO_INSTALL", config.DenoInstall)
	os.Setenv("FNM_DIR", config.FnmDir)
	os.Setenv("PNPM_HOME", config.PnpmHome)
	os.Setenv("JAVA_HOME", config.JavaHome)

	os.Setenv("GOCACHE", config.GoCache)
	os.Setenv("GOMODCACHE", config.GoModCache)
	os.Setenv("PUB_CACHE", config.PubCache)
	os.Setenv("PNPM_STORE_PATH", config.PnpmStorePath)
	os.Setenv("UV_CACHE_DIR", config.UvCacheDir)

	os.Setenv("RUSTUP_HOME", filepath.Join(config.PublicSourceDir, "rust", "rustup"))
	os.Setenv("CARGO_HOME", filepath.Join(config.PublicSourceDir, "rust", "cargo"))
	os.Setenv("UV_PYTHON_INSTALL_DIR", config.UvPythonInstallDir)
	os.Setenv("UV_TOOL_DIR", config.UvToolDir)

	// Prepend bin dirs to current process PATH so that newly spawned processes can resolve installed tools
	os.Setenv("PATH", config.SrcBinDir+":"+config.AppBinDir+":"+os.Getenv("PATH"))

	// Load FNM env in the current session so that if Node.js is already installed via FNM,
	// we can resolve the node/npm/pnpm tools in the current run.
	_ = sysutils.LoadFnmEnv()

	// Ensure we shut down the aria2c daemon at exit
	defer func() {
		logger.Info("Cleaning up Aria2 RPC daemon...")
		_ = exec.Command("pkill", "-f", "aria2c.*67486").Run()
	}()

	// 1. Setup Workspace Directories and Permissions
	workspaceDirs := []string{
		config.PublicSourceDir,
		config.ToolsDir,
		config.SrcBinDir,
		config.ApplicationsDir,
		config.AppBinDir,
		config.AppXdgDir,
		config.PoshThemeDir,
	}
	logger.Info("Preparing workspace directory structure...")
	if err := sysutils.SetupWorkspaceDirs(config.PublicSourceDir, workspaceDirs, config.DefaultUserGroup); err != nil {
		logger.Error("Configuring workspace directories: %v", err)
		os.Exit(1)
	}

	langOpts := []huh.Option[string]{
		huh.NewOption("Node.js", "Node.js"),
		huh.NewOption("Bun", "Bun"),
		huh.NewOption("Python", "Python"),
		huh.NewOption("Go", "Go"),
		huh.NewOption("Rust", "Rust"),
		huh.NewOption("Dart & Flutter", "Dart"),
		huh.NewOption("Podman & Compose", "Podman"),
		huh.NewOption("Caddy & FrankenPHP", "Caddy"),
	}

	browserOpts := []huh.Option[string]{
		huh.NewOption("Chrome", "Chrome"),
		huh.NewOption("Brave", "Brave"),
		huh.NewOption("Bitwarden", "Bitwarden"),
		huh.NewOption("Proprietary Codecs (VLC/Ffmpeg)", "Codecs"),
	}

	ideOpts := []huh.Option[string]{
		huh.NewOption("Neovim", "Neovim"),
		huh.NewOption("Antigravity IDE", "Antigravity"),
		huh.NewOption("VS Code", "VS Code"),
		huh.NewOption("JetBrains DataGrip", "DataGrip"),
		huh.NewOption("Zed Editor", "Zed"),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Languages & Tools to Install").
				Options(langOpts...).
				Value(&selectedLangs),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Browsers & Tools to Install").
				Options(browserOpts...).
				Value(&selectedBrowsers),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select IDEs & Editors to Install").
				Options(ideOpts...).
				Value(&selectedIDEs),
		),
	)

	if err := form.Run(); err != nil {
		logger.Warning("Selection canceled: %v", err)
		return
	}

	downloadRequests := []downloader.DownloadRequest{}
	tempDir := "/tmp/usetup"
	_ = os.MkdirAll(tempDir, 0777)

	if slices.Contains(selectedLangs, "Go") {
		cfg := config.Tools["go"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedLangs, "Dart") {
		cfg := config.Tools["dart"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedBrowsers, "Chrome") {
		cfg := config.Tools["chrome"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedBrowsers, "Bitwarden") {
		cfg := config.Tools["bitwarden"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedIDEs, "Antigravity") {
		cfg := config.Tools["antigravity"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedIDEs, "VS Code") {
		cfg := config.Tools["vscode"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedIDEs, "DataGrip") {
		cfg := config.Tools["datagrip"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}
	if slices.Contains(selectedIDEs, "Zed") {
		cfg := config.Tools["zed"]
		if !installer.IsToolInstalled(cfg) {
			downloadRequests = append(downloadRequests, downloader.DownloadRequest{URL: cfg.DownloadURL, Out: cfg.FileName})
		}
	}

	// Execute Parallel Downloads
	if len(downloadRequests) > 0 {
		logger.Info("Downloading %d selected package archives...", len(downloadRequests))
		res, err := dl.DownloadBatch(
			tempDir,
			downloader.BatchOptions{
				ConcurrentDownloads: 3,
				ConnectionsPerFile:  16,
			},
			downloadRequests,
		)
		if err != nil {
			logger.Error("Error queuing downloads: %v", err)
			os.Exit(1)
		}

		if err := downloader.ShowBatchProgress(res); err != nil {
			logger.Warning("Download error details: %v", err)
		}
	}

	// 5. Render and Install Shell Configurations
	compiledEnv, err := RenderEnvScript()
	if err != nil {
		logger.Warning("failed to compile env login script template: %v", err)
	} else {
		if err := sysutils.InstallEnvConfig(compiledEnv, config.PublicSourceDir); err != nil {
			logger.Warning("failed to install env login environment script: %v", err)
		}
	}

	compiledScript, err := RenderLoginScript()
	if err != nil {
		logger.Warning("failed to compile login script template: %v", err)
	} else {
		if err := sysutils.InstallShellConfig(compiledScript, config.PublicSourceDir); err != nil {
			logger.Warning("failed to install login environment script: %v", err)
		}
	}

	// 6. Setup ZSH shell features if requested
	exePath, _ := os.Executable()
	scriptDir := filepath.Dir(exePath)
	if !sysutils.CommandExists(filepath.Join(scriptDir, "zsh-init.sh")) {
		scriptDir = "/home/hwakins/setup/shell-scripts"
	}
	if slices.Contains(config.CorePackages, "zsh") {
		logger.Info("Initializing Zsh setup...")
		if err := sysutils.RunCommand("bash", filepath.Join(scriptDir, "zsh-init.sh")); err != nil {
			logger.Warning("ZSH bootstrap failed: %v", err)
		}
	}

	// 7. Bootstrap Oh My Posh Theme
	ensureOhMyPosh()

	// 8. Bootstrap Witr tool
	if !sysutils.CommandExists("witr") {
		logger.Info("Bootstrapping Witr...")
		_ = sysutils.RunCommand("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/pranshuparmar/witr/main/install.sh | bash")
	}

	// 9. Execute Language Installations
	for _, l := range selectedLangs {
		var err error
		switch l {
		case "Node.js":
			err = lang.InstallNode()
		case "Bun":
			err = lang.InstallBun()
		case "Python":
			err = lang.InstallPython()
		case "Go":
			err = lang.InstallGo(config.Tools["go"])
		case "Rust":
			err = lang.InstallRust()
		case "Dart":
			err = lang.InstallDart(config.Tools["dart"])
		case "Podman":
			err = lang.InstallPodman()
		case "Caddy":
			err = lang.InstallCaddyFrankenPHP()
		}
		if err != nil {
			logger.Error("installing %s: %v", l, err)
		}
	}

	// 10. Execute Browser Installations
	for _, b := range selectedBrowsers {
		var err error
		switch b {
		case "Chrome":
			err = browser.InstallChrome(config.Tools["chrome"])
		case "Brave":
			err = browser.InstallBrave()
		case "Bitwarden":
			err = browser.InstallBitwarden(config.Tools["bitwarden"])
		case "Codecs":
			err = browser.InstallCodecs()
		}
		if err != nil {
			logger.Error("installing %s: %v", b, err)
		}
	}

	// 11. Execute IDE Installations
	for _, i := range selectedIDEs {
		var err error
		switch i {
		case "Neovim":
			err = ide.InstallNeovim()
		case "Antigravity":
			err = ide.InstallAntigravity(config.Tools["antigravity"])
		case "VS Code":
			err = ide.InstallVSCode(config.Tools["vscode"])
		case "DataGrip":
			err = ide.InstallDataGrip(config.Tools["datagrip"])
		case "Zed":
			err = ide.InstallZed(config.Tools["zed"])
		}
		if err != nil {
			logger.Error("installing %s: %v", i, err)
		}
	}

	// 12. Final Perms Sweeping
	logger.Info("Performing final ownership and permissions sweep...")
	if err := sysutils.SweepACL(config.PublicSourceDir, config.DefaultUserGroup); err != nil {
		logger.Warning("final permissions sweep failed: %v", err)
	}

	logger.Success("Setup complete. The shared workspace under /src is fully configured.")
}

func ensureOhMyPosh() {
	poshBin := filepath.Join(config.PoshDir, "bin/oh-my-posh")
	if sysutils.CommandExists("oh-my-posh") || sysutils.CommandExists(poshBin) {
		logger.Warning("Oh My Posh already available. Checking themes...")
		ensurePoshThemes()
		return
	}

	logger.Info("Bootstrapping Oh My Posh...")
	poshBinDir := filepath.Join(config.PoshDir, "bin")
	_ = os.MkdirAll(poshBinDir, 0755)

	scriptPath := "/tmp/posh_install.sh"
	if err := sysutils.RunCommand("curl", "-sSL", "https://ohmyposh.dev/install.sh", "-o", scriptPath); err != nil {
		logger.Warning("failed to download Oh My Posh installer: %v", err)
		return
	}
	defer os.Remove(scriptPath)

	if err := sysutils.RunCommand("bash", scriptPath, "-d", poshBinDir); err != nil {
		logger.Warning("failed to execute Oh My Posh installer: %v", err)
		return
	}

	_ = sysutils.LinkFiles(poshBinDir, config.SrcBinDir)
	ensurePoshThemes()
}

func ensurePoshThemes() {
	themePath := filepath.Join(config.PoshThemeDir, "shell.omp.toml")
	_ = os.MkdirAll(config.PoshThemeDir, 0755)
	if err := os.WriteFile(themePath, []byte(poshThemeContent), 0644); err != nil {
		logger.Warning("failed to write embedded Oh My Posh theme: %v", err)
	} else {
		logger.Success("Deployed embedded Oh My Posh theme to %s", themePath)
	}
}
