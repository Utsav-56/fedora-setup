package browser

import (
	"fmt"
	"setup/installer"
	"setup/sysutils"
)

// InstallBrave registers the Brave repository and installs brave-browser.
func InstallBrave() error {
	if installer.IsPackageInstalled("brave-browser") {
		return nil
	}

	fmt.Println("[usetup] Installing Brave Browser...")
	if err := installer.Package("dnf-plugins-core"); err != nil {
		return err
	}

	if err := sysutils.RunCommand("dnf", "config-manager", "--add-repo", "https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo"); err != nil {
		return fmt.Errorf("failed to add brave repository: %w", err)
	}

	return installer.Package("brave-browser")
}
