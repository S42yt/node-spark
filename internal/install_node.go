package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/s42yt/node-spark/pkg"
)

// InstallVersion is the main entry point for node installation
func InstallVersion(version string, config *pkg.Config) error {
	return InstallNodeVersion(version, config)
}

// InstallNodeVersion properly installs a Node.js version
func InstallNodeVersion(version string, config *pkg.Config) error {
	fmt.Printf("Installing Node.js version %s...\n", version)

	// First, check if this version is already installed
	cleanVersion := strings.TrimPrefix(version, "v")
	for _, v := range config.InstalledVersions {
		if v == cleanVersion {
			fmt.Printf("Node.js version %s is already installed. Use 'nsk use %s' to switch to it.\n", cleanVersion, cleanVersion)
			return nil
		}
	}

	installPath := pkg.GetInstallPath(config)
	versionDir := filepath.Join(installPath, version)

	// 1. Ensure the base installation directory exists
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory %s: %w", installPath, err)
	}

	// 2. Ensure the specific version directory exists
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", versionDir, err)
	}

	// 3. Determine download URL and filename with improved architecture detection
	// First, check if we need to add v prefix
	versionStr := version
	if !strings.HasPrefix(versionStr, "v") {
		versionStr = "v" + version
	}

	// Improved architecture detection
	nodeArch, nodeOS, ext, err := detectSystemInfo()
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("node-%s-%s-%s.%s", versionStr, nodeOS, nodeArch, ext)
	downloadURL := fmt.Sprintf("https://nodejs.org/dist/%s/%s", versionStr, filename)

	// 4. Download the Node.js archive
	archivePath := filepath.Join(os.TempDir(), filename) // Download to temp dir first
	fmt.Printf("Downloading %s from %s...\n", filename, downloadURL)
	err = DownloadFile(archivePath, downloadURL)
	if err != nil {
		// If download fails, try alternative architectures (Windows may need x86 instead of x64)
		if runtime.GOOS == "windows" && nodeArch == "x64" {
			fmt.Println("x64 download failed, trying x86 version instead...")
			nodeArch = "x86"
			filename = fmt.Sprintf("node-%s-%s-%s.%s", versionStr, nodeOS, nodeArch, ext)
			downloadURL = fmt.Sprintf("https://nodejs.org/dist/%s/%s", versionStr, filename)
			archivePath = filepath.Join(os.TempDir(), filename)
			err = DownloadFile(archivePath, downloadURL)
		}

		if err != nil {
			return fmt.Errorf("failed to download Node.js archive: %w", err)
		}
	}
	defer os.Remove(archivePath) // Clean up downloaded archive
	fmt.Println("Download complete.")

	// 5. Extract the downloaded archive
	fmt.Printf("Extracting %s to %s...\n", filename, versionDir)
	err = ExtractArchive(archivePath, versionDir)
	if err != nil {
		return fmt.Errorf("failed to extract Node.js archive: %w", err)
	}
	fmt.Println("Extraction complete.")

	// 6. Post-installation verification and setup
	if runtime.GOOS == "windows" {
		// Verify the node executable exists and is valid
		nodePath := filepath.Join(versionDir, "node.exe")
		if _, err := os.Stat(nodePath); os.IsNotExist(err) {
			// Look in the bin subdirectory
			nodePath = filepath.Join(versionDir, "bin", "node.exe")
			if _, err := os.Stat(nodePath); os.IsNotExist(err) {
				return fmt.Errorf("node executable not found after extraction")
			}
		}

		// Run a simple version check to verify the binary is valid
		cmd := exec.Command(nodePath, "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("installed node binary is invalid: %w\nOutput: %s", err, string(output))
		}
		fmt.Printf("Verified installation with output: %s\n", strings.TrimSpace(string(output)))
	}

	// 7. Update config
	found := false
	for _, v := range config.InstalledVersions {
		if v == cleanVersion {
			found = true
			break
		}
	}
	if !found {
		config.InstalledVersions = append(config.InstalledVersions, cleanVersion)
	}

	fmt.Printf("Successfully installed Node.js version %s.\n", version)

	// Offer to use this version if no version is currently active
	if config.ActiveVersion == "" {
		fmt.Println("No Node.js version is currently active.")
		fmt.Printf("Would you like to use Node.js %s now? [Y/n] ", version)

		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(response)

		if response == "" || response == "y" || response == "yes" {
			if err := SetActiveVersion(version, config); err != nil {
				fmt.Printf("Warning: Failed to set active version: %v\n", err)
			} else {
				fmt.Printf("Node.js %s is now active.\n", version)
			}
		}
	}

	return nil
}

// detectSystemInfo determines the system information needed for Node.js installation
func detectSystemInfo() (nodeArch, nodeOS, ext string, err error) {
	osName := runtime.GOOS
	ext = "tar.gz" // Default extension

	// Map Go OS names to Node.js OS names and determine extension
	switch osName {
	case "linux":
		nodeOS = "linux"
	case "darwin":
		nodeOS = "darwin"
	case "windows":
		nodeOS = "win"
		ext = "zip"
	default:
		return "", "", "", fmt.Errorf("unsupported operating system: %s", osName)
	}

	// For Windows, perform more accurate architecture detection
	if osName == "windows" {
		// First, try to determine if the OS is 32-bit or 64-bit
		// On Windows, GOARCH might not accurately reflect the OS architecture capability
		var is64BitOS bool

		// Check if the system is 64-bit capable
		cmd := exec.Command("powershell", "-Command", "[Environment]::Is64BitOperatingSystem")
		output, err := cmd.Output()
		if err == nil {
			is64BitOS = strings.TrimSpace(string(output)) == "True"
		} else {
			// Fallback to GOARCH if PowerShell command fails
			is64BitOS = runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
		}

		// Check if we're running in a 32-bit process on a 64-bit OS
		var is32BitProcess bool
		cmd = exec.Command("powershell", "-Command", "![Environment]::Is64BitProcess")
		output, err = cmd.Output()
		if err == nil {
			is32BitProcess = strings.TrimSpace(string(output)) == "True"
		} else {
			is32BitProcess = runtime.GOARCH == "386"
		}

		// Check for ARM architecture
		var isARM bool
		cmd = exec.Command("powershell", "-Command",
			"(Get-WmiObject -Class Win32_Processor | Select-Object -First 1).Architecture -in @(5, 12)")
		output, err = cmd.Output()
		if err == nil {
			isARM = strings.TrimSpace(string(output)) == "True"
		}

		fmt.Println("System architecture detection:")
		fmt.Printf("- 64-bit OS: %v\n", is64BitOS)
		fmt.Printf("- 32-bit process: %v\n", is32BitProcess)
		fmt.Printf("- ARM processor: %v\n", isARM)

		// Always use 32-bit (x86) Node.js for better compatibility
		// This ensures the binaries will work on both 32-bit and 64-bit Windows
		nodeArch = "x86"
		fmt.Println("Using x86 architecture for better compatibility")

	} else {
		// For macOS and Linux, use standard architecture mapping
		switch runtime.GOARCH {
		case "amd64":
			nodeArch = "x64"
		case "386":
			nodeArch = "x86"
		case "arm64":
			nodeArch = "arm64"
		case "arm":
			nodeArch = "armv7l"
		default:
			return "", "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
		}
	}

	return nodeArch, nodeOS, ext, nil
}
