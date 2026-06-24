package lang

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
)

// InstallDart extracts the Flutter/Dart archive and links target binaries.
func InstallDart(cfg *config.ToolConfig) error {
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled := installer.IsTarInstalled(archivePath, cfg.InstallDir, "flutter")
	if isInstalled {
		return nil
	}

	logger.Info("Installing Dart & Flutter SDK...")
	if err := installer.Extract(archivePath, cfg.InstallDir, "flutter"); err != nil {
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
