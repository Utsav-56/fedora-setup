package sysutils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"setup/config"
	"setup/logger"
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
		logger.Info("setfacl not found. Attempting to install acl...")
		_ = RunCommand("dnf", "install", "-y", "acl")
	}

	logger.Info("Configuring permissions on %s...", publicSourceDir)

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
		logger.Success("Access Control Lists (ACLs) configured successfully.")
	} else {
		logger.Warning("ACL support unavailable. Group inheritance relies solely on SetGID.")
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
		logger.Info("Creating group '%s'...", defaultUserGroup)
		if err := CreateGroup(defaultUserGroup); err != nil {
			return err
		}
	}

	// Add current user to group
	currUser := GetCurrentUser()
	if currUser != "root" {
		if !UserInGroup(currUser, defaultUserGroup) {
			logger.Info("Adding user '%s' to group '%s'...", currUser, defaultUserGroup)
			if err := AddUserToGroup(currUser, defaultUserGroup); err != nil {
				return err
			}
			logger.Info("NOTE: User must re-login for group membership to become active.")
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
		logger.Success("Linked files from folder '%s' -> %s", filepath.Base(targetPath), destDir)
	} else {
		dst := filepath.Join(destDir, filepath.Base(targetPath))
		_ = os.Remove(dst)
		if err := os.Symlink(targetPath, dst); err != nil {
			return fmt.Errorf("failed to link file %s: %w", filepath.Base(targetPath), err)
		}
		logger.Success("Linked file '%s' -> %s", filepath.Base(targetPath), destDir)
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
					logger.Success("Removed dead link: %s", entry.Name())
				}
			}
			continue
		}

		if resolvedPath == absTarget {
			_ = os.Remove(itemPath)
			logger.Success("Removed link: %s", entry.Name())
		} else if strings.HasPrefix(resolvedPath, absTarget+string(filepath.Separator)) {
			_ = os.Remove(itemPath)
			logger.Success("Removed internal link: %s", entry.Name())
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

// InstallShellConfig writes the pre-compiled login.sh script to publicSourceDir and registers the environment globally.
func InstallShellConfig(compiledScript string, publicSourceDir string) error {
	destLogin := filepath.Join(publicSourceDir, "login.sh")
	profileDLink := "/etc/profile.d/src-workspace-env.sh"

	if err := os.WriteFile(destLogin, []byte(compiledScript), 0755); err != nil {
		return fmt.Errorf("failed to write login.sh: %w", err)
	}
	logger.Success("Copied login.sh to %s", destLogin)

	_ = os.Remove(profileDLink)
	if err := os.Symlink(destLogin, profileDLink); err != nil {
		return fmt.Errorf("failed to symlink profile.d loader: %w", err)
	}
	logger.Success("Linked profile loader to %s", profileDLink)
	return nil
}

// InstallEnvConfig writes the pre-compiled env_login.sh script to publicSourceDir.
func InstallEnvConfig(compiledScript string, publicSourceDir string) error {
	destEnv := filepath.Join(publicSourceDir, "env_login.sh")
	if err := os.WriteFile(destEnv, []byte(compiledScript), 0755); err != nil {
		return fmt.Errorf("failed to write env_login.sh: %w", err)
	}
	logger.Success("Copied env_login.sh to %s", destEnv)
	return nil
}

// SrcAdd maps a tool directory or binary file into the public source bin directory (/src/Tools/bin).
func SrcAdd(targetPath string) error {
	return LinkFiles(targetPath, config.SrcBinDir)
}

// AppAdd maps an application directory or binary file into the application bin directory (/src/Applications/bin).
func AppAdd(targetPath string) error {
	return LinkFiles(targetPath, config.AppBinDir)
}

// LoadFnmEnv loads environment variables dynamically from FNM command into the current Go execution session.
func LoadFnmEnv() error {
	if !CommandExists("fnm") {
		return nil
	}
	cmd := exec.Command("fnm", "env", "--shell=bash")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "export ") {
			continue
		}
		parts := strings.SplitN(line[7:], "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		} else if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
			val = val[1 : len(val)-1]
		}

		if key == "PATH" {
			pathParts := strings.Split(val, ":")
			for _, p := range pathParts {
				if p == "$PATH" || p == "" {
					continue
				}
				os.Setenv("PATH", p+":"+os.Getenv("PATH"))
			}
		} else {
			os.Setenv(key, val)
		}
	}
	return nil
}

