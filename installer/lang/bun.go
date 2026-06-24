package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"setup/config"
	"setup/logger"
	"setup/sysutils"
)

// InstallBun installs Bun JavaScript runtime.
func InstallBun() error {
	if sysutils.CommandExists("bun") {
		logger.Warning("Bun is already available. Skipping...")
		return nil
	}

	logger.Info("Installing Bun...")

	bunInstall := filepath.Join(config.PublicSourceDir, "bun")
	os.Setenv("BUN_INSTALL", bunInstall)

	scriptPath := "/tmp/bun_install.sh"
	if err := sysutils.RunCommand("curl", "-fsSL", "https://bun.sh/install", "-o", scriptPath); err != nil {
		return fmt.Errorf("failed to download bun installer: %w", err)
	}
	defer os.Remove(scriptPath)

	if err := sysutils.RunCommand("bash", scriptPath); err != nil {
		return fmt.Errorf("failed to execute bun installer: %w", err)
	}

	bunBinDir := filepath.Join(bunInstall, "bin")
	return sysutils.LinkFiles(bunBinDir, config.SrcBinDir)
}
