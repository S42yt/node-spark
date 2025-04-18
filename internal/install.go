// internal/install.go

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallNode installs the specified version of Node.js to the user's home directory.
// TODO: Implement the logic to download the specified Node.js version.
func InstallNode(version string) error {
	// Placeholder for download URL
	downloadURL := fmt.Sprintf("https://nodejs.org/dist/v%s/node-v%s.tar.gz", version, version)
	_ = downloadURL // Use downloadURL to avoid unused variable error for now

	// TODO: Create the .node-spark directory in the user's home directory.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	nodeSparkDir := filepath.Join(homeDir, ".node-spark")

	// TODO: Ensure the directory exists.
	if err := os.MkdirAll(nodeSparkDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", nodeSparkDir, err)
	}

	// TODO: Download the Node.js archive from the download URL.
	// Placeholder for download logic

	// TODO: Extract the downloaded archive to the .node-spark directory.
	// Placeholder for extraction logic

	// TODO: Set up symlink or update PATH to use the installed Node.js version.
	// Placeholder for symlink logic

	return nil
}

// getNodeDownloadURL constructs the download URL for a given Node.js version and OS/Arch.
// This function is kept for backward compatibility but is no longer used directly
func getNodeDownloadURL(version string) (url string, filename string, err error) {
	arch := runtime.GOARCH
	osName := runtime.GOOS
	ext := "tar.gz" // Default extension

	// Map Go architecture names to Node.js architecture names
	nodeArch := ""
	switch arch {
	case "amd64":
		nodeArch = "x64"
	case "386":
		nodeArch = "x86"
	case "arm64":
		nodeArch = "arm64"
	case "arm":
		nodeArch = "armv7l"
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Map Go OS names to Node.js OS names and determine extension
	nodeOS := ""
	switch osName {
	case "linux":
		nodeOS = "linux"
	case "darwin":
		nodeOS = "darwin"
	case "windows":
		nodeOS = "win"
		ext = "zip"
	default:
		return "", "", fmt.Errorf("unsupported operating system: %s", osName)
	}

	// Node.js versions usually have a "v" prefix in the URL but not always in the user input
	versionPrefix := version
	if !strings.HasPrefix(versionPrefix, "v") {
		versionPrefix = "v" + version
	}

	filename = fmt.Sprintf("node-%s-%s-%s.%s", versionPrefix, nodeOS, nodeArch, ext)
	url = fmt.Sprintf("https://nodejs.org/dist/%s/%s", versionPrefix, filename)
	return url, filename, nil
}
