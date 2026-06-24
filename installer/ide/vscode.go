package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/sysutils"
)

// InstallVSCode installs VS Code, sets up shared extension path, and installs plugins.
func InstallVSCode(cfg *config.ToolConfig) error {
	rpmPath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled, err := installer.IsRPMInstalled(rpmPath)
	if err == nil && isInstalled {
		fmt.Println("[usetup] VS Code is already installed. Skipping...")
		return nil
	}

	fmt.Println("[usetup] Installing VS Code...")
	if err := installer.InstallRpm(rpmPath); err != nil {
		return err
	}

	// Prepare user home VS Code path
	var userHome string
	currUser := sysutils.GetCurrentUser()
	if currUser != "root" {
		userHome = filepath.Join("/home", currUser)
	} else {
		userHome, _ = os.UserHomeDir()
	}

	vscodeDir := filepath.Join(userHome, ".vscode")
	extStore := filepath.Join(config.ToolsDir, "shared/vscode-extensions")
	userExtDir := filepath.Join(vscodeDir, "extensions")

	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return err
	}

	// Safe handling of directory migration to store
	if info, err := os.Lstat(userExtDir); err == nil {
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			_ = os.MkdirAll(filepath.Dir(extStore), 0755)
			_ = os.Rename(userExtDir, extStore)
		} else {
			_ = os.RemoveAll(userExtDir)
		}
	} else {
		_ = os.MkdirAll(extStore, 0755)
	}

	_ = os.Remove(userExtDir)
	if err := os.Symlink(extStore, userExtDir); err != nil {
		return fmt.Errorf("failed to link VS Code extensions to shared store: %w", err)
	}

	// Run chown on userHome/.vscode to ensure the user owns their directory
	if currUser != "root" {
		_ = sysutils.RunCommand("chown", "-R", currUser+":", vscodeDir)
	}

	// Install extensions
	fmt.Println("[usetup] Installing VS Code extensions...")
	for _, ext := range config.VSCodeExtensions {
		if currUser != "root" {
			// Run as the actual user to avoid permissions issues
			_ = sysutils.RunCommand("sudo", "-u", currUser, "code", "--install-extension", ext, "--force")
		} else {
			_ = sysutils.RunCommand("code", "--install-extension", ext, "--force")
		}
	}

	return nil
}
