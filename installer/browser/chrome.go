package browser

import (
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/logger"
)

// InstallChrome installs Google Chrome from the downloaded RPM.
func InstallChrome(cfg *config.ToolConfig) error {
	rpmPath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled, err := installer.IsRPMInstalled(rpmPath)
	if err == nil && isInstalled {
		logger.Warning("Google Chrome is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Google Chrome...")
	return installer.InstallRpm(rpmPath)
}
