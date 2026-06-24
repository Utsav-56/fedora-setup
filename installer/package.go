package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"setup/config"
	"setup/logger"
	"setup/sysutils"
)

// IsToolInstalled checks if a tool is already present on the system.
func IsToolInstalled(cfg *config.ToolConfig) bool {
	if cfg.Type == "tar" {
		for _, bin := range cfg.BinPaths {
			fullPath := filepath.Join(cfg.InstallDir, bin)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				return false
			}
		}
		return len(cfg.BinPaths) > 0
	}
	if cfg.Type == "rpm" {
		packageName := cfg.ID
		if cfg.ID == "chrome" {
			packageName = "google-chrome-stable"
		} else if cfg.ID == "vscode" {
			packageName = "code"
		}
		// Query rpm database silently to check if package is installed
		err := exec.Command("rpm", "-q", packageName).Run()
		return err == nil
	}
	return false
}

// Package installs packages via the native package manager (Fedora DNF).
func Package(pkgs ...string) error {
	if len(pkgs) == 0 {
		return nil
	}

	args := make([]string, 0, len(pkgs)+2)
	args = append(args, "install", "-y")
	args = append(args, pkgs...)

	return sysutils.RunCommand("dnf", args...)
}

// IsPackageInstalled checks if the package is installed via system rpm query.
func IsPackageInstalled(packageName string) bool {
	err := sysutils.RunCommand("rpm", "-q", packageName)
	if err == nil {
		logger.Warning("%s is already installed. Skipping...", packageName)
		return true
	}
	return false
}
