package installer

import (
	"fmt"
	"os/exec"
)

func Package(pkgs ...string) error {
	if len(pkgs) == 0 {
		return nil
	}

	args := make([]string, 0, len(pkgs)+1)
	args = append(args, "install", "-y")
	args = append(args, pkgs...)

	return run("dnf", args...)
}

func checkRPMInstalled(target string) (bool, error) {
	cmd := exec.Command("rpm", "-q", target)
	err := cmd.Run()

	// If no error → exit code 0 → installed
	if err == nil {
		fmt.Printf("%s is already installed. Skipping...", target)
		return true, nil
	}

	// Check if it's a real exit error (rpm ran but returned non-zero)
	if exitErr, ok := err.(*exec.ExitError); ok {
		// rpm exit code != 0 means not installed
		_ = exitErr
		return false, nil
	}

	// Here: actual system error (rpm not found, permission issue, etc.)
	return false, fmt.Errorf("failed to run rpm command: %w", err)
}

func IsPackageInstalled(packageName string) bool {
	isInstalled, err := checkRPMInstalled(packageName)
	if err != nil {
		return false
	}
	return isInstalled
}
