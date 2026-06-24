package installer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
)

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
	infoFile := filepath.Join(outputPath, "usetup.install-info")

	// 1. Compute current SHA256 of the archive file
	currentSHA, err := sha256OfFile(archivePath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	// 2. Cache Check
	if _, err := os.Stat(outputPath); err == nil {
		if data, err := os.ReadFile(infoFile); err == nil {
			if string(data) == currentSHA {
				fmt.Println("[usetup] cache hit → skipping extraction")
				return nil
			}
		}
	}

	// 3. Ensure the base output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 4. Open the archive file
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	ctx := context.Background()

	// 5. Identify format
	format, stream, err := archives.Identify(ctx, archivePath, f)
	if err != nil {
		return fmt.Errorf("failed to identify archive format: %w", err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return fmt.Errorf("format %s does not support extraction", format.Extension())
	}

	fmt.Println("[usetup] extracting:", archivePath)

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

	// 7. Write the successful shasum to the info file
	err = os.WriteFile(infoFile, []byte(currentSHA), 0644)
	if err != nil {
		return fmt.Errorf("failed to write install-info: %w", err)
	}

	fmt.Println("[usetup] extraction complete")
	return nil
}
