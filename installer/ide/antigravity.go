package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
	fs "setup/utils"
)

// InstallAntigravity extracts and installs Antigravity IDE.
func InstallAntigravity(cfg *config.ToolConfig) error {
	if installer.IsToolInstalled(cfg) {
		logger.Warning("Antigravity IDE is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Antigravity IDE...")
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	if err := installer.Extract(archivePath, cfg.InstallDir, ""); err != nil {
		return err
	}

	execFile := filepath.Join(cfg.InstallDir, "bin/antigravity-ide")
	iconFile := filepath.Join(cfg.InstallDir, "resources/app/resources/linux/code.png")

	if _, err := os.Stat(execFile); os.IsNotExist(err) {
		return fmt.Errorf("could not locate Antigravity executable: %s", execFile)
	}

	// Link executable
	if err := sysutils.AppAdd(execFile); err != nil {
		return err
	}

	// Create desktop file
	deskConfig := fs.DesktopFileConfig{
		Name:       "Antigravity IDE",
		Exec:       execFile,
		Icon:       iconFile,
		Categories: []string{"Development", "IDE"},
	}
	if err := fs.MakeDesktopFile(deskConfig, config.AppXdgDir); err != nil {
		return err
	}

	// Setup extensions symlink
	var userHome string
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		userHome = filepath.Join("/home", sudoUser)
	} else {
		userHome, _ = os.UserHomeDir()
	}

	extStore := filepath.Join(config.ToolsDir, "shared/vscode-extensions")
	antiBaseDir := filepath.Join(userHome, ".antigravity-ide")
	antiExtDir := filepath.Join(antiBaseDir, "extensions")

	if err := os.MkdirAll(antiBaseDir, 0755); err != nil {
		return err
	}

	_ = os.RemoveAll(antiExtDir)
	if err := os.Symlink(extStore, antiExtDir); err != nil {
		return fmt.Errorf("failed to symlink extensions to Antigravity: %w", err)
	}

	return nil
}
