package lang

import (
	"fmt"
	"os"
	"setup/config"
	"setup/logger"
	"setup/sysutils"
)

// InstallNode downloads FNM, installs Node.js, and activates corepack pnpm.
func InstallNode() error {
	if sysutils.CommandExists("node") {
		logger.Warning("Node.js is already available. Skipping...")
		return nil
	}

	logger.Info("Installing Node.js via FNM...")

	fnmDir := config.FnmDir
	if err := os.MkdirAll(fnmDir, 0755); err != nil {
		return err
	}

	// Install FNM
	scriptPath := "/tmp/fnm_install.sh"
	if err := sysutils.RunCommand("curl", "-fsSL", "https://fnm.vercel.app/install", "-o", scriptPath); err != nil {
		return fmt.Errorf("failed to download fnm installer: %w", err)
	}
	defer os.Remove(scriptPath)

	if err := sysutils.RunCommand("bash", scriptPath, "--install-dir", fnmDir, "--skip-shell"); err != nil {
		return fmt.Errorf("failed to run fnm installer: %w", err)
	}

	// Symlink FNM binary using SrcAdd
	if err := sysutils.SrcAdd(fnmDir); err != nil {
		return err
	}

	// Run node installation via fnm under bash (to source env)
	shellCmd := fmt.Sprintf(`export PATH="%s:$PATH" && eval "$(fnm env --use-on-cd)" && fnm install --latest && fnm use latest && fnm default latest && corepack enable && corepack prepare pnpm@latest --activate`, fnmDir)
	if err := sysutils.RunCommand("bash", "-c", shellCmd); err != nil {
		return fmt.Errorf("failed to configure node via fnm: %w", err)
	}

	// Load FNM env in the current session so that subsequent tool installs can resolve node/pnpm
	if err := sysutils.LoadFnmEnv(); err != nil {
		logger.Warning("Failed to load FNM environment into current run: %v", err)
	}

	return nil
}
