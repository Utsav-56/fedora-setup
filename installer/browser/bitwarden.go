package browser

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
)

// InstallBitwarden installs Bitwarden from the downloaded RPM.
func InstallBitwarden(cfg *config.ToolConfig) error {
	rpmPath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled, err := installer.IsRPMInstalled(rpmPath)
	if err == nil && isInstalled {
		logger.Warning("Bitwarden is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Bitwarden...")
	return installer.InstallRpm(rpmPath)
}
