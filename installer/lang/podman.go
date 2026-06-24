package lang

import (
	"fmt"
	"os"
	"setup/installer"
	"setup/logger"
	"setup/sysutils"
)

// InstallPodman installs podman, podman-compose and configures search registries.
func InstallPodman() error {
	if !sysutils.CommandExists("podman") {
		logger.Info("Installing Podman...")
		if err := installer.Package("podman"); err != nil {
			return err
		}

		// Configure search registries
		regDir := "/etc/containers/registries.conf.d"
		if err := os.MkdirAll(regDir, 0755); err != nil {
			return err
		}

		content := `unqualified-search-registries = [
    "docker.io",
    "quay.io",
    "registry.fedoraproject.org"
]
`
		if err := os.WriteFile(regDir+"/99-dockerio.conf", []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write registries config: %w", err)
		}
	}

	if !installer.IsPackageInstalled("podman-compose") {
		logger.Info("Installing Podman Compose...")
		if err := installer.Package("podman-compose"); err != nil {
			return err
		}
	}

	return nil
}
