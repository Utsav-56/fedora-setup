package installer

import (
	"fmt"
	"os"
	"os/exec"
	"setup/logger"
	"strings"
)

func IsRPMInstalled(rpmFile string) (bool, error) {
	// get package name from rpm file
	nameBytes, err := exec.Command(
		"rpm", "-qp", "--qf", "%{NAME}", rpmFile,
	).Output()
	if err != nil {
		return false, err
	}
	name := strings.TrimSpace(string(nameBytes))

	// get NEVRA from rpm file
	fileNevraBytes, err := exec.Command(
		"rpm", "-qp", "--qf", "%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}", rpmFile,
	).Output()
	if err != nil {
		return false, err
	}
	fileNevra := strings.TrimSpace(string(fileNevraBytes))

	// get installed NEVRA (may fail if not installed)
	installedNevraBytes, err := exec.Command(
		"rpm", "-q", "--qf", "%{NAME}-%{VERSION}-%{RELEASE}.%{ARCH}", name,
	).Output()

	if err != nil {
		return false, nil // not installed = false, no error
	}

	installedNevra := strings.TrimSpace(string(installedNevraBytes))

	// compare
	return fileNevra == installedNevra, nil
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func InstallRpm(path string) error {
	isInstalled, err := IsRPMInstalled(path)
	if err != nil {
		return err
	}

	if isInstalled {
		logger.Warning("%s is already installed. Skipping...", path)
		return nil
	}

	if err := run("rpm", "-i", path); err != nil {
		return err
	}

	return nil
}
