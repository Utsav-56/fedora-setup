package ide

import (
	"fmt"
	"path/filepath"
	"setup/config"
	"setup/installer"
	"setup/sysutils"
	fs "setup/utils"
)

// InstallZed extracts and installs the Zed Editor.
func InstallZed(cfg *config.ToolConfig) error {
	archivePath := filepath.Join("/tmp/usetup", cfg.FileName)
	isInstalled := installer.IsTarInstalled(archivePath, cfg.InstallDir, "")
	if isInstalled {
		return nil
	}

	fmt.Println("[usetup] Installing Zed Editor...")
	if err := installer.Extract(archivePath, cfg.InstallDir, ""); err != nil {
		return err
	}

	execFile := filepath.Join(cfg.InstallDir, "bin/zed")
	iconFile := filepath.Join(cfg.InstallDir, "share/icons/hicolor/512x512/apps/zed.png")

	// Link binary
	if err := sysutils.LinkFiles(execFile, config.AppBinDir); err != nil {
		return err
	}

	// Create desktop file
	deskConfig := fs.DesktopFileConfig{
		Name:       "Zed Editor",
		Exec:       execFile,
		Icon:       iconFile,
		Categories: []string{"Development", "IDE"},
		WMClass:    "zed",
	}

	return fs.MakeDesktopFile(deskConfig, config.AppXdgDir)
}
