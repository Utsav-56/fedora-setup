package lang

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
)

// InstallDart extracts the Flutter/Dart archive into a versioned subdirectory and links target binaries.
func InstallDart(cfg *config.ToolConfig) error {
	if installer.IsToolInstalled(cfg) {
		logger.Warning("Dart & Flutter SDK is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Dart & Flutter SDK...")
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	targetFolder := "flutter_" + cfg.Version
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
