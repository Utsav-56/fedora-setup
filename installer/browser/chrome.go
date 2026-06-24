package browser

import (
	"fmt"
	"path/filepath"
	"setup/config"
	"setup/installer"
)

// InstallChrome installs Google Chrome from the downloaded RPM.
func InstallChrome(cfg *config.ToolConfig) error {
	rpmPath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled, err := installer.IsRPMInstalled(rpmPath)
	if err == nil && isInstalled {
		fmt.Println("[usetup] Google Chrome is already installed. Skipping...")
		return nil
	}

	fmt.Println("[usetup] Installing Google Chrome...")
	return installer.InstallRpm(rpmPath)
}
