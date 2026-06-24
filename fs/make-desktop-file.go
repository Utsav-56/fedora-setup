package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DesktopFileConfig holds all XDG-compliant properties for generating a Linux desktop launcher.
type DesktopFileConfig struct {
	Name          string   // Required: Name of the application (e.g., "Android Studio")
	Exec          string   // Required: Path to executable, plus any arguments
	Type          string   // Optional: Default "Application"
	Icon          string   // Optional: Default "system-run"
	Terminal      bool     // Optional: Default false
	Categories    []string // Optional: Default []string{"Utility"}
	Comment       string   // Optional: Default "Launch <Name>"
	GenericName   string   // Optional: Search bar subtext. Default matches Name
	WMClass       string   // Optional: For dock window grouping. Default falls back to binary name
	Keywords      []string // Optional: Extra launcher search tags (e.g., []string{"ide", "code"})
	CategoriesStr string
}

func (config *DesktopFileConfig) sanitize() error {
	if strings.TrimSpace(config.Name) == "" {
		return fmt.Errorf("validation error: 'Name' is a required field")
	}
	if strings.TrimSpace(config.Exec) == "" {
		return fmt.Errorf("validation error: 'Exec' is a required field")
	}

	if config.Type == "" {
		config.Type = "Application"
	}
	if config.Icon == "" {
		config.Icon = "system-run"
	}
	if config.Comment == "" {
		config.Comment = fmt.Sprintf("Launch %s", config.Name)
	}
	if config.GenericName == "" {
		config.GenericName = config.Name
	}

	// Handle Categories string formatting (must end with a semicolon)
	config.CategoriesStr = ""
	if len(config.Categories) == 0 {
		config.CategoriesStr = "Utility;"
	} else {
		config.CategoriesStr = strings.Join(config.Categories, ";") + ";"
	}

	// Handle StartupWMClass fallback logic
	if config.WMClass == "" {
		// Grab the binary string (strip arguments if present)
		binaryPart := strings.Fields(config.Exec)[0]
		// Emulate: basename | tr '[:upper:]' '[:lower:]'
		config.WMClass = strings.ToLower(filepath.Base(binaryPart))
	}

	return nil
}

// MakeDesktopFile generates a compliant .desktop entry and saves it to the specified targetDirectory.
func MakeDesktopFile(config DesktopFileConfig, targetDirectory string) error {

	if err := config.sanitize(); err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s.desktop", strings.ReplaceAll(config.Name, " ", "_"))
	saveFilePath := filepath.Join(targetDirectory, fileName)

	if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDirectory, err)
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=%s
Name=%s
GenericName=%s
Comment=%s
Exec=%s
Icon=%s
Terminal=%t
Categories=%s
StartupWMClass=%s
StartupNotify=true
`,
		config.Type,
		config.Name,
		config.GenericName,
		config.Comment,
		config.Exec,
		config.Icon,
		config.Terminal,
		config.CategoriesStr,
		config.WMClass,
	)

	if len(config.Keywords) > 0 {
		keywordsStr := strings.Join(config.Keywords, ";") + ";"
		content += fmt.Sprintf("Keywords=%s\n", keywordsStr)
	}

	if err := os.WriteFile(saveFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write desktop file: %w", err)
	}

	fmt.Printf("[usetup] Desktop entry for '%s' deployed perfectly to: %s\n", config.Name, saveFilePath)
	return nil
}
