package browser

import (
	"fmt"
	"path/filepath"
	"setup/config"
	"setup/installer"
)

// InstallBitwarden installs Bitwarden from the downloaded RPM.
func InstallBitwarden(cfg *config.ToolConfig) error {
	rpmPath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled, err := installer.IsRPMInstalled(rpmPath)
	if err == nil && isInstalled {
		fmt.Println("[usetup] Bitwarden is already installed. Skipping...")
		return nil
	}

	fmt.Println("[usetup] Installing Bitwarden...")
	return installer.InstallRpm(rpmPath)
}
