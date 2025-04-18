package internal

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ExtractArchive extracts a downloaded archive (.tar.gz or .zip) to a destination directory.
// It handles stripping the top-level directory often found in Node archives.
func ExtractArchive(archivePath string, destPath string) error {
	ext := filepath.Ext(archivePath)

	if ext == ".gz" && strings.HasSuffix(strings.TrimSuffix(archivePath, ext), ".tar") {
		return extractTarGz(archivePath, destPath)
	} else if ext == ".zip" {
		return extractZip(archivePath, destPath)
	} else {
		return fmt.Errorf("unsupported archive format: %s", ext)
	}
}

// extractTarGz extracts a .tar.gz file
func extractTarGz(archivePath string, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		// Determine the target path, stripping the top-level directory
		// e.g., node-v18.17.0-linux-x64/bin/node -> bin/node
		parts := strings.SplitN(header.Name, string(filepath.Separator), 2)
		if len(parts) < 2 {
			continue // Skip top-level directory entry itself or empty names
		}
		target := filepath.Join(destPath, parts[1])

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			// Note: Creating symlinks might require special permissions on Windows.
			// For simplicity, we might skip them or handle them differently based on OS.
			if runtime.GOOS != "windows" {
				if err := os.Symlink(header.Linkname, target); err != nil {
					// Log warning instead of failing? Symlinks might not be critical.
					fmt.Printf("Warning: Failed to create symlink %s -> %s: %v\n", target, header.Linkname, err)
				}
			} else {
				fmt.Printf("Skipping symlink creation on Windows: %s -> %s\n", target, header.Linkname)
			}

		default:
			fmt.Printf("Unsupported tar entry type %c for %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}

// extractZip extracts a .zip file
func extractZip(archivePath string, destPath string) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	// First, determine the top-level directory name in the zip
	var topLevelDir string
	for _, file := range zipReader.File {
		// Normalize path separators (some zips might use / even on Windows)
		normalizedPath := filepath.FromSlash(file.Name)
		parts := strings.Split(normalizedPath, string(filepath.Separator))
		if len(parts) > 0 {
			topLevelDir = parts[0]
			break
		}
	}

	// If no files were found, return an error
	if topLevelDir == "" {
		return fmt.Errorf("no files found in archive")
	}

	fmt.Printf("Detected top-level directory: %s\n", topLevelDir)

	for _, file := range zipReader.File {
		// Normalize path separators (some zips might use / even on Windows)
		normalizedPath := filepath.FromSlash(file.Name)

		// Skip the pure top-level directory entry
		if normalizedPath == topLevelDir+"/" || normalizedPath == topLevelDir {
			continue
		}

		// Determine target path by removing top-level directory
		var targetPath string
		if strings.HasPrefix(normalizedPath, topLevelDir) {
			// Remove top-level directory and any leading separator
			relativePath := strings.TrimPrefix(normalizedPath, topLevelDir)
			relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))
			// Handle cases where the separator might be forward slash
			relativePath = strings.TrimPrefix(relativePath, "/")
			targetPath = filepath.Join(destPath, relativePath)
		} else {
			// In case there's no top-level directory (unusual for Node.js archives)
			targetPath = filepath.Join(destPath, normalizedPath)
		}

		// Fix for illegal file path - ensure path is clean and relative
		cleanedTarget := filepath.Clean(targetPath)

		// Ensure we're not extracting outside the destination directory (prevent Zip Slip)
		if !strings.HasPrefix(cleanedTarget, filepath.Clean(destPath)) {
			fmt.Printf("Warning: Skipping potentially unsafe path: %s\n", file.Name)
			continue
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanedTarget, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(cleanedTarget), 0755); err != nil {
			return err
		}

		// Skip if somehow the file path still ends up being a directory
		if strings.HasSuffix(cleanedTarget, string(os.PathSeparator)) {
			continue
		}

		outFile, err := os.OpenFile(cleanedTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	// Verify extraction was successful - for Windows, check if node.exe exists
	if runtime.GOOS == "windows" {
		nodePath := filepath.Join(destPath, "node.exe")
		if _, err := os.Stat(nodePath); os.IsNotExist(err) {
			// Try to find node.exe recursively
			var nodeExePath string
			filepath.Walk(destPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip errors
				}
				if !info.IsDir() && info.Name() == "node.exe" {
					nodeExePath = path
					return filepath.SkipAll // Stop walking, we found it
				}
				return nil
			})

			if nodeExePath != "" {
				fmt.Printf("Found node.exe at %s, copying to root directory\n", nodeExePath)
				// If we found node.exe in a subdirectory, copy it to the root
				data, err := os.ReadFile(nodeExePath)
				if err == nil {
					err = os.WriteFile(nodePath, data, 0755)
					if err != nil {
						fmt.Printf("Warning: Failed to copy node.exe to root: %v\n", err)
					}
				}
			} else {
				fmt.Println("Warning: node.exe not found in the extracted files")
			}
		}
	}

	return nil
}
