package browser

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
)

// InstallBitwarden installs Bitwarden from the downloaded RPM.
func InstallBitwarden(cfg *config.ToolConfig) error {
	if installer.IsToolInstalled(cfg) {
		logger.Warning("Bitwarden is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Bitwarden...")
	rpmPath := filepath.Join("/tmp/usetup", cfg.FileName)
	return installer.InstallRpm(rpmPath)
}
