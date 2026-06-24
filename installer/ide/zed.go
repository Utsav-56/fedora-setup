package ide

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
	fs "setup/utils"
)

// InstallZed extracts and installs the Zed Editor.
func InstallZed(cfg *config.ToolConfig) error {
	if installer.IsToolInstalled(cfg) {
		logger.Warning("Zed Editor is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Zed Editor...")
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	if err := installer.Extract(archivePath, cfg.InstallDir, ""); err != nil {
		return err
	}

	execFile := filepath.Join(cfg.InstallDir, "bin/zed")
	iconFile := filepath.Join(cfg.InstallDir, "share/icons/hicolor/512x512/apps/zed.png")

	// Link binary
	if err := sysutils.AppAdd(execFile); err != nil {
		return err
	}

	// Create desktop file
	deskConfig := fs.DesktopFileConfig{
		Name:       "Zed Editor",
		Exec:       execFile,
		Icon:       iconFile,
		Categories: []string{"Development", "IDE"},
		WMClass:    "zed",
	}

	return fs.MakeDesktopFile(deskConfig, config.AppXdgDir)
}
