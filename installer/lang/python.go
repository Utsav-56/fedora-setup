package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"setup/config"
	"setup/sysutils"
)

// InstallPython installs UV python manager and uses it to install Python.
func InstallPython() error {
	if sysutils.CommandExists("python") || sysutils.CommandExists("python3") {
		fmt.Println("[usetup] Python is already available. Skipping...")
		return nil
	}

	fmt.Println("[usetup] Installing Python via UV...")

	uvInstallDir := filepath.Join(config.PublicSourceDir, "uv")
	os.Setenv("UV_INSTALL_DIR", uvInstallDir)

	scriptPath := "/tmp/uv_install.sh"
	if err := sysutils.RunCommand("curl", "-LsSf", "https://astral.sh/uv/install.sh", "-o", scriptPath); err != nil {
		return fmt.Errorf("failed to download uv installer: %w", err)
	}
	defer os.Remove(scriptPath)

	if err := sysutils.RunCommand("sh", scriptPath); err != nil {
		return fmt.Errorf("failed to run uv installer: %w", err)
	}

	if err := sysutils.LinkFiles(uvInstallDir, config.SrcBinDir); err != nil {
		return err
	}

	uvExec := filepath.Join(config.SrcBinDir, "uv")
	if err := sysutils.RunCommand(uvExec, "python", "install"); err != nil {
		return fmt.Errorf("failed to install python via uv: %w", err)
	}

	return nil
}
