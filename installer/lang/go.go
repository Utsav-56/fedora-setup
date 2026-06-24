package lang

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
)

// InstallGo extracts the Go archive and links go/gofmt binaries.
func InstallGo(cfg *config.ToolConfig) error {
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled := installer.IsTarInstalled(archivePath, cfg.InstallDir, "go")
	if isInstalled {
		return nil
	}

	logger.Info("Installing Go...")
	if err := installer.Extract(archivePath, cfg.InstallDir, "go"); err != nil {
		return err
	}

	for _, bin := range cfg.BinPaths {
		target := filepath.Join(cfg.InstallDir, bin)
		if err := sysutils.LinkFiles(target, config.SrcBinDir); err != nil {
			return err
		}
	}
	return nil
}
