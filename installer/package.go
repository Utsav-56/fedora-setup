package installer

import (
	"fmt"
	"setup/sysutils"
)

// Package installs packages via the native package manager (Fedora DNF).
func Package(pkgs ...string) error {
	if len(pkgs) == 0 {
		return nil
	}

	args := make([]string, 0, len(pkgs)+2)
	args = append(args, "install", "-y")
	args = append(args, pkgs...)

	return sysutils.RunCommand("dnf", args...)
}

// IsPackageInstalled checks if the package is installed via system rpm query.
func IsPackageInstalled(packageName string) bool {
	err := sysutils.RunCommand("rpm", "-q", packageName)
	if err == nil {
		fmt.Printf("[usetup] Package %s is already installed. Skipping...\n", packageName)
		return true
	}
	return false
}
