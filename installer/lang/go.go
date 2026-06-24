package lang

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
)

// InstallGo extracts the Go archive into a versioned subdirectory and links target binaries.
func InstallGo(cfg *config.ToolConfig) error {
	if installer.IsToolInstalled(cfg) {
		logger.Warning("Go is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Go...")
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	targetFolder := "go" + cfg.Version
	if err := installer.Extract(archivePath, cfg.InstallDir, targetFolder); err != nil {
		return err
	}

	for _, bin := range cfg.BinPaths {
		target := filepath.Join(cfg.InstallDir, bin)
		if err := sysutils.SrcAdd(target); err != nil {
			return err
		}
	}
	return nil
}
