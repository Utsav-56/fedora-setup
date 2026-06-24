package browser

import (
	"fmt"
	"os/exec"
	"setup/installer"
	"setup/sysutils"
	"strings"
)

// InstallCodecs installs proprietary media codecs, RPMFusion repositories, swaps ffmpeg-free, and installs VLC.
func InstallCodecs() error {
	fmt.Println("[usetup] Installing proprietary media codecs...")

	fedoraVerBytes, err := exec.Command("rpm", "-E", "%fedora").Output()
	if err != nil {
		return fmt.Errorf("failed to determine fedora version: %w", err)
	}
	fedoraVer := strings.TrimSpace(string(fedoraVerBytes))

	freeURL := fmt.Sprintf("https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-%s.noarch.rpm", fedoraVer)
	nonFreeURL := fmt.Sprintf("https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-%s.noarch.rpm", fedoraVer)

	if err := sysutils.RunCommand("dnf", "install", "-y", freeURL, nonFreeURL); err != nil {
		return fmt.Errorf("failed to install RPMFusion repositories: %w", err)
	}

	if err := sysutils.RunCommand("dnf", "swap", "-y", "ffmpeg-free", "ffmpeg", "--allowerasing"); err != nil {
		fmt.Printf("ffmpeg swap warning: %v\n", err)
	}

	return installer.Package("vlc-plugins-freeworld", "vlc")
}
