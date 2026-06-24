package ide

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
	fs "setup/utils"
)

// InstallDataGrip extracts and configures JetBrains DataGrip IDE.
func InstallDataGrip(cfg *config.ToolConfig) error {
	if installer.IsToolInstalled(cfg) {
		logger.Warning("JetBrains DataGrip is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing JetBrains DataGrip...")
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	if err := installer.Extract(archivePath, cfg.InstallDir, ""); err != nil {
		return err
	}

	execFile := filepath.Join(cfg.InstallDir, "bin/datagrip.sh")
	iconFile := filepath.Join(cfg.InstallDir, "bin/datagrip.png")

	// Link binary
	if err := sysutils.AppAdd(execFile); err != nil {
		return err
	}

	// Create desktop file
	deskConfig := fs.DesktopFileConfig{
		Name:       "JetBrains DataGrip",
		Exec:       execFile,
		Icon:       iconFile,
		Categories: []string{"Development", "IDE"},
		WMClass:    "jetbrains-datagrip",
	}

	return fs.MakeDesktopFile(deskConfig, config.AppXdgDir)
}
