package main

import (
	"fmt"
	"os"
	"os/exec"
	"setup/downloader"
	"strings"

	"charm.land/huh/v2"
)

const (
	GOLANG_DOWNLOAD_LINK = "https://dl.google.com/go/go1.22.4.linux-amd64.tar.gz"

	DART_DOWNLOAD_LINK                = "https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_3.44.2-stable.tar.xz"
	JETBRAINS_DATA_GRIP_DOWNLOAD_LINK = "https://download-cdn.jetbrains.com/datagrip/datagrip-2026.1.3.tar.gz"

	CHROME_DOWNLOAD_LINK      = "https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm"
	ANTIGRAVITY_DOWNLOAD_LINK = "https://edgedl.me.gvt1.com/edgedl/release2/j0qc3/antigravity/stable/2.0.4-6381998290370560/linux-x64/Antigravity%20IDE.tar.gz"
	VS_CODE_DOWNLOAD_LINK     = "https://vscode.download.prss.microsoft.com/dbazure/download/stable/93cfdd489c3b228840d0f86ec77c3636277c93ea/code-1.125.0-1781601611.el8.x86_64.rpm"
	ZED_DOWNLOAD_LINK         = "https://release-assets.githubusercontent.com/github-production-release-asset/340547520/5371e7e4-8f6c-499b-aed7-e249742fb2f1?sp=r&sv=2018-11-09&sr=b&spr=https&se=2026-06-18T17%3A13%3A48Z&rscd=attachment%3B+filename%3Dzed-linux-x86_64.tar.gz&rsct=application%2Foctet-stream&skoid=96c2d410-5711-43a1-aedd-ab1947aa7ab0&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skt=2026-06-18T16%3A13%3A03Z&ske=2026-06-18T17%3A13%3A48Z&sks=b&skv=2018-11-09&sig=EnTHa8T5nJ62L3PDKG00L9pNexX2DyFMJpoZaZwjMgA%3D&jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmVsZWFzZS1hc3NldHMuZ2l0aHVidXNlcmNvbnRlbnQuY29tIiwia2V5Ijoia2V5MSIsImV4cCI6MTc4MTgwNDMzNywibmJmIjoxNzgxODAwNzM3LCJwYXRoIjoicmVsZWFzZWFzc2V0cHJvZHVjdGlvbi5ibG9iLmNvcmUud2luZG93cy5uZXQifQ.i0P-U3C1fzCU605pDuqPao6ff7dO9LmHr8Hyk87ZSW4&response-content-disposition=attachment%3B%20filename%3Dzed-linux-x86_64.tar.gz&response-content-type=application%2Foctet-stream"
	BITWARDEN_DOWNLOAD_LINK   = "https://release-assets.githubusercontent.com/github-production-release-asset/53538899/e355eb83-eed4-49d4-9846-9cbb9de5c48b?sp=r&sv=2018-11-09&sr=b&spr=https&se=2026-06-18T17%3A40%3A23Z&rscd=attachment%3B+filename%3DBitwarden-2026.5.0-x86_64.rpm&rsct=application%2Foctet-stream&skoid=96c2d410-5711-43a1-aedd-ab1947aa7ab0&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skt=2026-06-18T16%3A39%3A39Z&ske=2026-06-18T17%3A40%3A23Z&sks=b&skv=2018-11-09&sig=rz%2F7543y%2FH5qx4V%2BJpfso8Y4fdJ%2BiJRNftw6yWB8UKk%3D&jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmVsZWFzZS1hc3NldHMuZ2l0aHVidXNlcmNvbnRlbnQuY29tIiwia2V5Ijoia2V5MSIsImV4cCI6MTc4MTgwNTI2OSwibmJmIjoxNzgxODAxNjY5LCJwYXRoIjoicmVsZWFzZWFzc2V0cHJvZHVjdGlvbi5ibG9iLmNvcmUud2luZG93cy5uZXQifQ.izDZptSBRiT8W1hxIRcK2sP6bF3ZVhQ9bdmtZJ3oPcs&response-content-disposition=attachment%3B%20filename%3DBitwarden-2026.5.0-x86_64.rpm&response-content-type=application%2Foctet-stream"
)

var (
	DNF    = "dnf"
	APTD   = "apt-get"
	PACMAN = "pacman"
)

// the possible sources of install also includes dnf
var (
	TAR = ".tar."
	RPM = ".rpm"
)

var availablePackageManager string

var dl = downloader.NewDownloader(downloader.Config{
	Host:   "localhost",
	Port:   "67486",
	Secret: "UsetupSecretKeyValue",
})

// similar alternative to cmd-exists() { command -v "$1" >/dev/null 2>&1; }
// Why this is the right approach
// 1. exec.LookPath searches the directories in $PATH
// 2. It works like command -v or which
// 3. It does not depend on the command supporting -v or any flags
// 4. It is portable across Unix/Linux/macOS/Windows
func cmdExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// package identifier for the package it contains the name and package for each distro package managers
//
// There may also be packages which differs in command and name like facl and setfacl so we should ensure that
type Package struct {
	// the identification name of the package as geenric like aria2c and so on
	Name string

	// the name of the package in the apt manager
	Apt    string
	Dnf    string
	Pacman string

	// the command to see if it exists or not, for e.g package name is aria2 but the command to check is aria2c  so this is optional but better to have field
	CheckCmd string
}

func (p Package) exists() bool {

	if p.CheckCmd == "" {
		return cmdExists(p.Name)
	}

	return cmdExists(p.CheckCmd)
}

func (p Package) resolve(mgr string) string {
	switch mgr {
	case DNF:
		if p.Dnf != "" {
			return p.Dnf
		}
	case APTD:
		if p.Apt != "" {
			return p.Apt
		}
	case PACMAN:
		if p.Pacman != "" {
			return p.Pacman
		}
	}
	return p.Name
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func isRPMInstalled(rpmFile string) (bool, error) {
	// get package name from rpm file
	nameBytes, err := exec.Command(
		"rpm", "-qp", "--qf", "%{NAME}", rpmFile,
	).Output()
	if err != nil {
		return false, err
	}
	name := strings.TrimSpace(string(nameBytes))

	// get NEVRA from rpm file
	fileNevraBytes, err := exec.Command(
		"rpm", "-qp", "--qf", "%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}", rpmFile,
	).Output()
	if err != nil {
		return false, err
	}
	fileNevra := strings.TrimSpace(string(fileNevraBytes))

	// get installed NEVRA (may fail if not installed)
	installedNevraBytes, err := exec.Command(
		"rpm", "-q", "--qf", "%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}", name,
	).Output()

	if err != nil {
		return false, nil // not installed = false, no error
	}

	installedNevra := strings.TrimSpace(string(installedNevraBytes))

	// compare
	return fileNevra == installedNevra, nil
}

func install(pkgs ...Package) error {
	if len(pkgs) == 0 {
		return nil
	}

	mgr := availablePackageManager

	args := make([]string, 0, len(pkgs)+3)

	switch mgr {
	case DNF, APTD:
		args = append(args, "install", "-y")
	case PACMAN:
		args = append(args, "-S", "--noconfirm")
	default:
		return fmt.Errorf("unsupported package manager: %s", mgr)
	}

	for _, p := range pkgs {
		args = append(args, p.resolve(mgr))
	}

	return run(mgr, args...)
}

var neededPackages = []Package{
	// core download & archive tools
	{Name: "aria2", CheckCmd: "aria2c"},
	{Name: "curl"},
	{Name: "wget"},
	{Name: "unzip"},
	{Name: "xz"},
	{Name: "tar"},
	{Name: "pixz"},
	{Name: "pigz"},

	// shell productivity tools
	{Name: "fzf"},
	{Name: "zoxide"},
	{Name: "eza"},

	// interactive / CLI UX tools
	{
		Name:   "oh-my-posh",
		Pacman: "oh-my-posh-bin", // varies in pacman so we explicitly tell to use this
		// check command is same as name so no need
	},
	{Name: "jq"},

	// core shell tools
	{Name: "zsh"},
	{Name: "git"},
}

type DirectLink struct {
	link string
	name string
}

var knownLinks = []DirectLink{
	// Languages
	{"go.tar.gz", GOLANG_DOWNLOAD_LINK},
	{"dart.tar.xz", DART_DOWNLOAD_LINK},

	// IDEs (heavy tarballs)
	{"antigravity.tar.gz", ANTIGRAVITY_DOWNLOAD_LINK},
	{"zed.tar.gz", ZED_DOWNLOAD_LINK},
	{"datagrip.tar.gz", JETBRAINS_DATA_GRIP_DOWNLOAD_LINK},

	// Browsers / apps (RPMs)
	{"chrome.rpm", CHROME_DOWNLOAD_LINK},
	{"vscode.rpm", VS_CODE_DOWNLOAD_LINK},
	{"bitwarden.rpm", BITWARDEN_DOWNLOAD_LINK},
}

func chooseOptions(header string, options []string) ([]string, error) {
	var selected []string

	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(header).
				Options(opts...).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}

func init() {
	if cmdExists(DNF) {
		availablePackageManager = DNF
	} else if cmdExists(APTD) {
		availablePackageManager = APTD
	} else if cmdExists(PACMAN) {
		availablePackageManager = PACMAN
	} else {
		panic("No supported package manager found")
	}

	install(neededPackages...)
}

func main() {

}
