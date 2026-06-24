package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
)

// InstallNeovim installs neovim and clones LazyVim configuration template.
func InstallNeovim() error {
	if !installer.IsPackageInstalled("neovim") {
		logger.Info("Installing Neovim...")
		if err := installer.Package("neovim"); err != nil {
			return err
		}
	}

	var userHome string
	currUser := sysutils.GetCurrentUser()
	if currUser != "root" {
		userHome = filepath.Join("/home", currUser)
	} else {
		userHome, _ = os.UserHomeDir()
	}

	nvimConfigDir := filepath.Join(userHome, ".config/nvim")
	if _, err := os.Stat(nvimConfigDir); os.IsNotExist(err) {
		logger.Info("Bootstrapping LazyVim config...")
		if err := sysutils.RunCommand("git", "clone", "https://github.com/LazyVim/starter", nvimConfigDir); err != nil {
			return fmt.Errorf("failed to clone LazyVim starter: %w", err)
		}
		_ = os.RemoveAll(filepath.Join(nvimConfigDir, ".git"))

		// Set ownership
		if currUser != "root" {
			_ = sysutils.RunCommand("chown", "-R", currUser+":", nvimConfigDir)
		}
	} else {
		logger.Warning("Neovim config already exists. Skipping config bootstrap.")
	}

	return nil
}
