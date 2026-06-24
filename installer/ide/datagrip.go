package ide

import (
	"fmt"
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/sysutils"
	fs "setup/utils"
)

// InstallDataGrip extracts and configures JetBrains DataGrip IDE.
func InstallDataGrip(cfg *config.ToolConfig) error {
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled := installer.IsTarInstalled(archivePath, cfg.InstallDir, "")
	if isInstalled {
		return nil
	}

	fmt.Println("[usetup] Installing JetBrains DataGrip...")
	if err := installer.Extract(archivePath, cfg.InstallDir, ""); err != nil {
		return err
	}

	execFile := filepath.Join(cfg.InstallDir, "bin/datagrip.sh")
	iconFile := filepath.Join(cfg.InstallDir, "bin/datagrip.png")

	// Link binary
	if err := sysutils.LinkFiles(execFile, config.AppBinDir); err != nil {
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
