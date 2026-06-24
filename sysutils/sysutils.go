package sysutils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRoot checks if the program is running with root permissions.
func IsRoot() bool {
	return os.Geteuid() == 0
}

// RunCommand executes a shell command, routing its output to stdout/stderr.
func RunCommand(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// CommandExists checks if a executable is found in $PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GroupExists checks if a group exists in the system.
func GroupExists(group string) bool {
	err := exec.Command("getent", "group", group).Run()
	return err == nil
}

// CreateGroup creates a new group.
func CreateGroup(group string) error {
	return RunCommand("groupadd", group)
}

// UserInGroup checks if a user is a member of the group.
func UserInGroup(user, group string) bool {
	out, err := exec.Command("id", "-nG", user).Output()
	if err != nil {
		return false
	}
	groups := strings.Fields(string(out))
	for _, g := range groups {
		if g == group {
			return true
		}
	}
	return false
}

// AddUserToGroup appends a user to a group.
func AddUserToGroup(user, group string) error {
	return RunCommand("usermod", "-aG", group, user)
}

// GetCurrentUser returns the current non-root user via env, falling back to root.
func GetCurrentUser() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return sudoUser
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "root"
}

// SweepACL configures group/owner permissions on the specified workspace path.
func SweepACL(publicSourceDir, defaultUserGroup string) error {
	if !CommandExists("setfacl") {
		fmt.Println("setfacl not found. Attempting to install acl...")
		_ = RunCommand("dnf", "install", "-y", "acl")
	}

	fmt.Printf("[usetup] Configuring permissions on %s...\n", publicSourceDir)

	// 1. Ownership: root:shared
	if err := RunCommand("chown", "-R", "root:"+defaultUserGroup, publicSourceDir); err != nil {
		return fmt.Errorf("chown failed: %w", err)
	}

	// 2. SetGID on directories: forces new creations to inherit group 'shared'
	if err := RunCommand("find", publicSourceDir, "-type", "d", "-exec", "chmod", "2775", "{}", "+"); err != nil {
		return fmt.Errorf("setgid chmod failed: %w", err)
	}

	// 3. ACL Rules
	if CommandExists("setfacl") {
		// Modify existing
		err1 := RunCommand("setfacl", "-R", "-m", "u::rwX", "-m", "g:"+defaultUserGroup+":rwX", "-m", "o::rX", publicSourceDir)
		// Default for future files
		err2 := RunCommand("setfacl", "-R", "-d", "-m", "u::rwx", "-m", "g:"+defaultUserGroup+":rwx", "-m", "o::rx", publicSourceDir)
		if err1 != nil || err2 != nil {
			return fmt.Errorf("setfacl failed: err1=%v, err2=%v", err1, err2)
		}
		fmt.Println("[usetup] Access Control Lists (ACLs) configured successfully.")
	} else {
		fmt.Println("[usetup] WARNING: ACL support unavailable. Group inheritance relies solely on SetGID.")
	}
	return nil
}

// SetupWorkspaceDirs prepares the workspace layout, ensuring group/user are setup.
func SetupWorkspaceDirs(publicSourceDir string, dirs []string, defaultUserGroup string) error {
	// Create all paths
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0775); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create shared group if it doesn't exist
	if !GroupExists(defaultUserGroup) {
		fmt.Printf("[usetup] Creating group '%s'...\n", defaultUserGroup)
		if err := CreateGroup(defaultUserGroup); err != nil {
			return err
		}
	}

	// Add current user to group
	currUser := GetCurrentUser()
	if currUser != "root" {
		if !UserInGroup(currUser, defaultUserGroup) {
			fmt.Printf("[usetup] Adding user '%s' to group '%s'...\n", currUser, defaultUserGroup)
			if err := AddUserToGroup(currUser, defaultUserGroup); err != nil {
				return err
			}
			fmt.Println("[usetup] NOTE: User must re-login for group membership to become active.")
		}
	}

	return SweepACL(publicSourceDir, defaultUserGroup)
}

// LinkFiles links files individually from a target folder, or links a single file target.
func LinkFiles(targetPath, destDir string) error {
	if err := os.MkdirAll(destDir, 0775); err != nil {
		return err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("link target not accessible: %w", err)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			// Link only files
			if entry.Type().IsRegular() {
				src := filepath.Join(targetPath, entry.Name())
				dst := filepath.Join(destDir, entry.Name())
				_ = os.Remove(dst) // remove old link if exists
				if err := os.Symlink(src, dst); err != nil {
					return fmt.Errorf("failed to link %s: %w", entry.Name(), err)
				}
			}
		}
		fmt.Printf("[usetup] Linked files from folder '%s' -> %s\n", filepath.Base(targetPath), destDir)
	} else {
		dst := filepath.Join(destDir, filepath.Base(targetPath))
		_ = os.Remove(dst)
		if err := os.Symlink(targetPath, dst); err != nil {
			return fmt.Errorf("failed to link file %s: %w", filepath.Base(targetPath), err)
		}
		fmt.Printf("[usetup] Linked file '%s' -> %s\n", filepath.Base(targetPath), destDir)
	}
	return nil
}

// UnlinkFiles removes links pointing to targetPath or inside targetPath from destDir.
func UnlinkFiles(targetPath, destDir string) error {
	absTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		absTarget, err = filepath.Abs(targetPath)
		if err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		itemPath := filepath.Join(destDir, entry.Name())
		lst, err := os.Lstat(itemPath)
		if err != nil {
			continue
		}
		// Process only symlinks
		if lst.Mode()&os.ModeSymlink == 0 {
			continue
		}

		resolvedPath, err := filepath.EvalSymlinks(itemPath)
		if err != nil {
			// Try reading raw link target for dead links
			linkVal, err := os.Readlink(itemPath)
			if err == nil {
				// Make absolute
				if !filepath.IsAbs(linkVal) {
					linkVal = filepath.Join(destDir, linkVal)
				}
				absLinkVal, err := filepath.Abs(linkVal)
				if err == nil && (absLinkVal == absTarget || strings.HasPrefix(absLinkVal, absTarget+string(filepath.Separator))) {
					_ = os.Remove(itemPath)
					fmt.Printf("[usetup] Removed dead link: %s\n", entry.Name())
				}
			}
			continue
		}

		if resolvedPath == absTarget {
			_ = os.Remove(itemPath)
			fmt.Printf("[usetup] Removed link: %s\n", entry.Name())
		} else if strings.HasPrefix(resolvedPath, absTarget+string(filepath.Separator)) {
			_ = os.Remove(itemPath)
			fmt.Printf("[usetup] Removed internal link: %s\n", entry.Name())
		}
	}
	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// InstallShellConfig copies path_login.sh and registers the environment globally.
func InstallShellConfig(scriptDir, publicSourceDir string) error {
	srcLogin := filepath.Join(scriptDir, "path_login.sh")
	destLogin := filepath.Join(publicSourceDir, "login.sh")
	profileDLink := "/etc/profile.d/src-workspace-env.sh"

	if _, err := os.Stat(srcLogin); err == nil {
		if err := CopyFile(srcLogin, destLogin, 0755); err != nil {
			return fmt.Errorf("failed to copy login.sh: %w", err)
		}
		fmt.Printf("[usetup] Copied login.sh to %s\n", destLogin)
	} else {
		// Fallback: create empty login.sh if sibling file missing
		fmt.Println("[usetup] WARNING: path_login.sh not found in script directory. Creating empty fallback.")
		if err := os.WriteFile(destLogin, []byte("# empty fallback"), 0755); err != nil {
			return err
		}
	}

	_ = os.Remove(profileDLink)
	if err := os.Symlink(destLogin, profileDLink); err != nil {
		return fmt.Errorf("failed to symlink profile.d loader: %w", err)
	}
	fmt.Printf("[usetup] Linked profile loader to %s\n", profileDLink)
	return nil
}
