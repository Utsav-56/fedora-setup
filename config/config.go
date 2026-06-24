package config

import (
	"path/filepath"
)

var (
	PublicSourceDir  = "/src"
	ToolsDir         = filepath.Join(PublicSourceDir, "Tools")
	SrcBinDir        = filepath.Join(ToolsDir, "bin")
	ApplicationsDir  = filepath.Join(PublicSourceDir, "Applications")
	AppBinDir        = filepath.Join(ApplicationsDir, "bin")
	AppXdgDir        = filepath.Join(ApplicationsDir, "applications")
	PoshDir          = filepath.Join(PublicSourceDir, "oh-my-posh")
	PoshThemeDir     = filepath.Join(PoshDir, "themes")
	DefaultUserGroup = "shared"

	// --- 2. DEVELOPMENT SDK HOME DIRECTORIES ---
	AndroidHome = filepath.Join(PublicSourceDir, "android-sdk")
	DenoInstall = filepath.Join(PublicSourceDir, "deno")
	FnmDir      = filepath.Join(PublicSourceDir, "nodejs")
	PnpmHome    = filepath.Join(PublicSourceDir, "pnpm")
	JavaHome    = "/opt/android-studio/jbr"

	// --- 3. GLOBAL CACHE SETTINGS ---
	GoCache       = "/cache/go/build"
	GoModCache    = "/cache/go/mod"
	PubCache      = "/cache/pub"
	PnpmStorePath = "/cache/pnpm/store"
	UvCacheDir    = "/cache/uv"

	// --- 4. RUST & PYTHON (uv) ---
	UvPythonInstallDir = filepath.Join(PublicSourceDir, "python", "versions")
	UvToolDir          = filepath.Join(PublicSourceDir, "python", "tools")
)

type ToolConfig struct {
	ID          string   // unique key e.g. "go", "dart"
	Name        string   // display name e.g. "Go", "Dart & Flutter"
	DownloadURL string   // URL to download from
	FileName    string   // local filename in /tmp/usetup
	InstallDir  string   // parent directory to extract to
	Version     string   // version of the tool
	Type        string   // "tar", "rpm", "dnf", "custom"
	BinPaths    []string // path(s) relative to InstallDir to symlink into SrcBinDir or AppBinDir
}

// Global registry of all downloadable/installable tools
var Tools = map[string]*ToolConfig{
	"go": {
		ID:          "go",
		Name:        "Go",
		DownloadURL: "https://dl.google.com/go/go1.22.4.linux-amd64.tar.gz",
		FileName:    "go.tar.gz",
		InstallDir:  filepath.Join(PublicSourceDir, "golang"),
		Version:     "1.22.4",
		Type:        "tar",
		BinPaths:    []string{"go1.22.4/bin/go", "go1.22.4/bin/gofmt"},
	},
	"dart": {
		ID:          "dart",
		Name:        "Dart & Flutter",
		DownloadURL: "https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_3.44.2-stable.tar.xz",
		FileName:    "dart.tar.xz",
		InstallDir:  filepath.Join(PublicSourceDir, "flutter"),
		Version:     "3.44.2",
		Type:        "tar",
		BinPaths:    []string{"flutter_3.44.2/bin/flutter", "flutter_3.44.2/bin/dart"},
	},
	"antigravity": {
		ID:          "antigravity",
		Name:        "Antigravity IDE",
		DownloadURL: "https://edgedl.me.gvt1.com/edgedl/release2/j0qc3/antigravity/stable/2.0.4-6381998290370560/linux-x64/Antigravity%20IDE.tar.gz",
		FileName:    "antigravity.tar.gz",
		InstallDir:  filepath.Join(ApplicationsDir, "antigravity-ide"),
		Version:     "2.0.4",
		Type:        "tar",
		BinPaths:    []string{"bin/antigravity-ide"},
	},
	"zed": {
		ID:          "zed",
		Name:        "Zed Editor",
		DownloadURL: "https://release-assets.githubusercontent.com/github-production-release-asset/340547520/5371e7e4-8f6c-499b-aed7-e249742fb2f1?sp=r&sv=2018-11-09&sr=b&spr=https&se=2026-06-18T17%3A13%3A48Z&rscd=attachment%3B+filename%3Dzed-linux-x86_64.tar.gz&rsct=application%2Foctet-stream&skoid=96c2d410-5711-43a1-aedd-ab1947aa7ab0&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skt=2026-06-18T16%3A13%3A03Z&ske=2026-06-18T17%3A13%3A48Z&sks=b&skv=2018-11-09&sig=EnTHa8T5nJ62L3PDKG00L9pNexX2DyFMJpoZaZwjMgA%3D&jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmVsZWFzZS1hc3NldHMuZ2l0aHVidXNlcmNvbnRlbnQuY29tIiwia2V5Ijoia2V5MSIsImV4cCI6MTc4MTgwNDMzNywibmJmIjoxNzgxODAwNzM3LCJwYXRoIjoicmVsZWFzZWFzc2V0cHJvZHVjdGlvbi5ibG9iLmNvcmUud2luZG93cy5uZXQifQ.i0P-U3C1fzCU605pDuqPao6ff7dO9LmHr8Hyk87ZSW4&response-content-disposition=attachment%3B%20filename%3Dzed-linux-x86_64.tar.gz&response-content-type=application%2Foctet-stream",
		FileName:    "zed.tar.gz",
		InstallDir:  filepath.Join(ApplicationsDir, "zed"),
		Version:     "latest",
		Type:        "tar",
		BinPaths:    []string{"bin/zed"},
	},
	"datagrip": {
		ID:          "datagrip",
		Name:        "JetBrains DataGrip",
		DownloadURL: "https://download-cdn.jetbrains.com/datagrip/datagrip-2026.1.3.tar.gz",
		FileName:    "datagrip.tar.gz",
		InstallDir:  filepath.Join(ApplicationsDir, "datagrip"),
		Version:     "2026.1.3",
		Type:        "tar",
		BinPaths:    []string{"bin/datagrip"},
	},
	"chrome": {
		ID:          "chrome",
		Name:        "Chrome",
		DownloadURL: "https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm",
		FileName:    "chrome_browser.rpm",
		Type:        "rpm",
		Version:     "latest",
	},
	"vscode": {
		ID:          "vscode",
		Name:        "VS Code",
		DownloadURL: "https://vscode.download.prss.microsoft.com/dbazure/download/stable/93cfdd489c3b228840d0f86ec77c3636277c93ea/code-1.125.0-1781601611.el8.x86_64.rpm",
		FileName:    "vscode.rpm",
		Type:        "rpm",
		Version:     "1.125.0",
	},
	"bitwarden": {
		ID:          "bitwarden",
		Name:        "Bitwarden",
		DownloadURL: "https://release-assets.githubusercontent.com/github-production-release-asset/53538899/e355eb83-eed4-49d4-9846-9cbb9de5c48b?sp=r&sv=2018-11-09&sr=b&spr=https&se=2026-06-18T17%3A40%3A23Z&rscd=attachment%3B+filename%3DBitwarden-2026.5.0-x86_64.rpm&rsct=application%2Foctet-stream&skoid=96c2d410-5711-43a1-aedd-ab1947aa7ab0&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skt=2026-06-18T16%3A39%3A39Z&ske=2026-06-18T17%3A40%3A23Z&sks=b&skv=2018-11-09&sig=rz%2F7543y%2FH5qx4V%2BJpfso8Y4fdJ%2BiJRNftw6yWB8UKk%3D&jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmVsZWFzZS1hc3NldHMuZ2l0aHVidXNlcmNvbnRlbnQuY29tIiwia2V5Ijoia2V5MSIsImV4cCI6MTc4MTgwNTI2OSwibmJmIjoxNzgxODAxNjY5LCJwYXRoIjoicmVsZWFzZWFzc2V0cHJvZHVjdGlvbi5ibG9iLmNvcmUud2luZG93cy5uZXQifQ.izDZptSBRiT8W1hxIRcK2sP6bF3ZVhQ9bdmtZJ3oPcs&response-content-disposition=attachment%3B%20filename%3DBitwarden-2026.5.0-x86_64.rpm&response-content-type=application%2Foctet-stream",
		FileName:    "bitwarden.rpm",
		Type:        "rpm",
		Version:     "2026.5.0",
	},
}

// Core DNF Packages to install during boot phase
var CorePackages = []string{
	"aria2", "curl", "wget", "unzip", "xz", "tar", "pixz", "pigz",
	"fzf", "zoxide", "eza", "oh-my-posh", "gum", "jq", "zsh", "git", "acl",
}

// Extensions for VS Code / Antigravity IDE
var VSCodeExtensions = []string{
	"golang.go",
	"dart-code.dart-code",
	"dart-code.flutter",
	"vscode-icons-team.vscode-icons",
	"yzhang.markdown-all-in-one",
	"ms-azuretools.vscode-docker",
	"esbenp.prettier-vscode",
	"dsznajder.es7-react-js-snippets",
	"dbaeumer.vscode-eslint",
	"jeanp413.open-remote-wsl",
	"oven.bun-vscode",
	"bradlc.vscode-tailwindcss",
	"tamasfe.even-better-toml",
}
