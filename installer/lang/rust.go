package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"setup/config"
	"setup/logger"
	"setup/sysutils"
)

// InstallRust installs the Rust toolchain via rustup.
func InstallRust() error {
	if sysutils.CommandExists("rustc") {
		logger.Warning("Rust is already available. Skipping...")
		return nil
	}

	logger.Info("Installing Rust...")

	rustupHome := filepath.Join(config.PublicSourceDir, "rust", "rustup")
	cargoHome := filepath.Join(config.PublicSourceDir, "rust", "cargo")

	os.Setenv("RUSTUP_HOME", rustupHome)
	os.Setenv("CARGO_HOME", cargoHome)

	scriptPath := "/tmp/rustup.sh"
	if err := sysutils.RunCommand("curl", "--proto", "=https", "--tlsv1.2", "-sSf", "https://sh.rustup.rs", "-o", scriptPath); err != nil {
		return fmt.Errorf("failed to download rustup script: %w", err)
	}
	defer os.Remove(scriptPath)

	if err := sysutils.RunCommand("sh", scriptPath, "--default-toolchain", "stable", "-y", "--no-modify-path"); err != nil {
		return fmt.Errorf("failed to run rustup script: %w", err)
	}

	cargoBinDir := filepath.Join(cargoHome, "bin")
	return sysutils.LinkFiles(cargoBinDir, config.SrcBinDir)
}
