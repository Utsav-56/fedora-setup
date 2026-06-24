package installer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"setup/logger"
	"strings"
	"time"

	"github.com/mholt/archives"
)

type FileData struct {
	SHA256     string `json:"sha256"`
	CreateTime string `json:"createTime"`
}

func (fd FileData) write(path string) error {
	data, err := json.Marshal(fd)
	if err != nil {
		return fmt.Errorf("failed to marshal FileData: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write FileData: %w", err)
	}
	return nil
}

func NewFileData(path string) (FileData, error) {
	sha, err := sha256OfFile(path)
	if err != nil {
		return FileData{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return FileData{}, err
	}
	return FileData{SHA256: sha, CreateTime: info.ModTime().Format(time.DateTime)}, nil
}

func ReadFileData(path string) (FileData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileData{}, err
	}
	var fd FileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return FileData{}, err
	}
	return fd, nil
}

func sha256OfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Extract extracts the archive into the outputPath.
// If targetFolderName is provided, and the archive contains a single root folder,
// that root folder is renamed/replaced with targetFolderName during extraction.
func Extract(archivePath, outputPath, targetFolderName string) error {
	actualCheckDir := outputPath
	if targetFolderName != "" {
		actualCheckDir = filepath.Join(outputPath, targetFolderName)
	}
	infoFile := filepath.Join(actualCheckDir, "usetup.install-info")

	isInstalled := IsTarInstalled(archivePath, outputPath, targetFolderName)
	if isInstalled {
		logger.Warning("the file %s is already installed → skipping extraction", archivePath)
		return nil
	}

	ctx := context.Background()

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	format, stream, err := archives.Identify(ctx, archivePath, f)
	if err != nil {
		return fmt.Errorf("failed to identify archive format: %w", err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return fmt.Errorf("format %s does not support extraction", format.Extension())
	}

	logger.Info("Extracting: %s", archivePath)

	// Variables to track single root folder behavior dynamically
	var detectedRoot string
	var hasSingleRoot = targetFolderName != ""

	// 6. Walk and extract files
	err = extractor.Extract(ctx, stream, func(ctx context.Context, file archives.FileInfo) error {
		// Standardize separators to slash for archive reading consistency
		archiveName := filepath.ToSlash(file.NameInArchive)
		parts := strings.Split(strings.Trim(archiveName, "/"), "/")

		if len(parts) == 0 || parts[0] == "" {
			return nil // skip unexpected empty entries
		}

		// Initialize root detection on the very first file entry
		if hasSingleRoot && detectedRoot == "" {
			detectedRoot = parts[0]
		}

		// If we encounter any file that does NOT belong to the initial root directory,
		// then it means there isn't a single root directory inside this archive.
		if hasSingleRoot && parts[0] != detectedRoot {
			hasSingleRoot = false
		}

		// Compute target file path
		var targetPath string
		if hasSingleRoot && targetFolderName != "" {
			// Replace the original root folder (parts[0]) with the custom folder name
			remainingPath := filepath.Join(parts[1:]...)
			targetPath = filepath.Join(outputPath, targetFolderName, remainingPath)
		} else {
			targetPath = filepath.Join(outputPath, file.NameInArchive)
		}

		// Handle Directory creation
		if file.IsDir() {
			return os.MkdirAll(targetPath, file.Mode())
		}

		// Ensure parent directory exists for files
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		// Extract file contents
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer out.Close()

		in, err := file.Open()
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(out, in)
		return err
	})

	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	fd, err := NewFileData(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create file metadata: %w", err)
	}

	if err := fd.write(infoFile); err != nil {
		return fmt.Errorf("failed to write install-info: %w", err)
	}

	logger.Success("Extraction complete")
	return nil
}

// checks if the path exists if not then its false
// then match the file stored sum if no file its false
// if no match of sum then also its false
// IsTarInstalled checks if the archive at archivePath has already been correctly
// extracted and matches the target installation state.
func IsTarInstalled(archivePath, outputPath, targetFolderName string) bool {
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return false
	}

	actualCheckDir := outputPath
	if targetFolderName != "" {
		actualCheckDir = filepath.Join(outputPath, targetFolderName)
		if _, err := os.Stat(actualCheckDir); os.IsNotExist(err) {
			return false
		}
	}

	infoFile := filepath.Join(actualCheckDir, "usetup.install-info")

	storedData, err := ReadFileData(infoFile)
	if err != nil {
		// Missing file or corrupted JSON -> needs installation
		return false
	}

	currentArchiveData, err := NewFileData(archivePath)
	if err != nil {
		return false
	}

	return storedData.SHA256 == currentArchiveData.SHA256
}
