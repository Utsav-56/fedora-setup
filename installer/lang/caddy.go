package lang

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
	"strings"
)

// IsCaddyFrankenPHPInstalled checks if Caddy is already swapped with FrankenPHP.
func IsCaddyFrankenPHPInstalled() bool {
	var out bytes.Buffer
	cmdVer := exec.Command("caddy", "version")
	cmdVer.Stdout = &out
	_ = cmdVer.Run()
	return strings.Contains(out.String(), "FrankenPHP")
}

// InstallCaddyFrankenPHP installs Caddy, downloads FrankenPHP, swaps caddy with frankenphp, and enables the systemd service.
func InstallCaddyFrankenPHP() error {
	if IsCaddyFrankenPHPInstalled() {
		logger.Warning("Caddy with FrankenPHP is already installed. Skipping...")
		return nil
	}

	logger.Info("Installing Caddy...")

	// Enable COPR repo
	if err := sysutils.RunCommand("dnf", "install", "-y", "dnf5-plugins"); err != nil {
		logger.Warning("dnf5-plugins install warning: %v", err)
	}
	if err := sysutils.RunCommand("dnf", "copr", "enable", "-y", "@caddy/caddy"); err != nil {
		return fmt.Errorf("failed to enable caddy COPR repository: %w", err)
	}
	if err := installer.Package("caddy"); err != nil {
		return err
	}

	logger.Info("Installing FrankenPHP...")
	scriptPath := "/tmp/frankenphp_install.sh"
	if err := sysutils.RunCommand("curl", "-fsSL", "https://frankenphp.dev/install.sh", "-o", scriptPath); err != nil {
		return fmt.Errorf("failed to download frankenphp installer: %w", err)
	}
	defer os.Remove(scriptPath)

	// Run frankenphp installer (extracts binary to current directory or standard path)
	// Usually install.sh downloads frankenphp to the current working directory.
	// We can execute it in /usr/local/bin or move it there.
	// Let's run it in /tmp and get the binary.
	c := exec.Command("sh", scriptPath)
	c.Dir = "/tmp"
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to run frankenphp installer: %w", err)
	}

	frankenphpTempPath := "/tmp/frankenphp"
	frankenphpDestPath := "/usr/local/bin/frankenphp"
	if _, err := os.Stat(frankenphpTempPath); err == nil {
		_ = os.Rename(frankenphpTempPath, frankenphpDestPath)
		_ = os.Chmod(frankenphpDestPath, 0755)
	}

	// Check if already swapped
	var out bytes.Buffer
	cmdVer := exec.Command("caddy", "version")
	cmdVer.Stdout = &out
	_ = cmdVer.Run()
	if strings.Contains(out.String(), "FrankenPHP") {
		logger.Warning("Caddy is already FrankenPHP. Skipping swap.")
		return nil
	}

	caddyPath, err := exec.LookPath("caddy")
	if err != nil {
		return fmt.Errorf("could not locate caddy path: %w", err)
	}

	// Double check if frankenphp was placed in /usr/local/bin/frankenphp
	if _, err := os.Stat(frankenphpDestPath); os.IsNotExist(err) {
		// Try to look up in $PATH
		if path, err := exec.LookPath("frankenphp"); err == nil {
			frankenphpDestPath = path
		} else {
			return fmt.Errorf("could not locate frankenphp binary")
		}
	}

	logger.Info("Swapping Caddy with FrankenPHP...")
	caddyBackup := caddyPath + ".bak"
	_ = os.Remove(caddyBackup)
	if err := os.Rename(caddyPath, caddyBackup); err != nil {
		return fmt.Errorf("failed to backup caddy: %w", err)
	}

	// Copy frankenphp to caddyPath
	if err := sysutils.CopyFile(frankenphpDestPath, caddyPath, 0755); err != nil {
		// Restore backup on failure
		_ = os.Rename(caddyBackup, caddyPath)
		return fmt.Errorf("failed to swap caddy with frankenphp: %w", err)
	}

	if err := sysutils.RunCommand("systemctl", "enable", "caddy"); err != nil {
		logger.Warning("systemctl enable caddy warning: %v", err)
	}

	return nil
}
