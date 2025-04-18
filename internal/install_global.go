package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// IsInstalledGlobally checks if node-spark is already installed globally
func IsInstalledGlobally() bool {
	switch runtime.GOOS {
	case "windows":
		userProfile := os.Getenv("USERPROFILE")
		if userProfile == "" {
			return false
		}
		destPath := filepath.Join(userProfile, "AppData", "Local", "Programs", "node-spark", "nsk.exe")
		_, err := os.Stat(destPath)
		return err == nil
	case "darwin", "linux":
		possibleLocations := []string{
			"/usr/local/bin/nsk",
			"/usr/bin/nsk",
		}

		homeDir, err := os.UserHomeDir()
		if err == nil {
			possibleLocations = append(possibleLocations,
				filepath.Join(homeDir, "bin", "nsk"),
				filepath.Join(homeDir, ".local", "bin", "nsk"))
		}

		for _, location := range possibleLocations {
			if _, err := os.Stat(location); err == nil {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// InstallGlobalSilently installs node-spark globally without user interaction
func InstallGlobalSilently() error {
	if IsInstalledGlobally() {
		return nil // Already installed, nothing to do
	}

	var err error
	switch runtime.GOOS {
	case "windows":
		err = installWindowsGlobalSilently()
	case "darwin", "linux":
		err = installUnixGlobalSilently()
	default:
		err = fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err == nil {
		fmt.Println("Node Spark has been installed globally.")
	}

	return err
}

// installWindowsGlobalSilently installs node-spark globally on Windows silently
func installWindowsGlobalSilently() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return fmt.Errorf("USERPROFILE not set")
	}

	destDir := filepath.Join(userProfile, "AppData", "Local", "Programs", "node-spark")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destPath := filepath.Join(destDir, "nsk.exe")

	// Use CopyFile instead of ReadFile/WriteFile for better handling of executable files
	if err := copyFile(exePath, destPath); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}

	// Update PATH with PowerShell to ensure proper escaping and handling
	pathCmd := fmt.Sprintf(`
		$destDir = '%s'
		$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
		if ($currentPath -notlike "*$destDir*") {
			[Environment]::SetEnvironmentVariable('Path', "$currentPath;$destDir", 'User')
			Write-Host "Added to PATH: $destDir"
		} else {
			Write-Host "Path already contains: $destDir"
		}
	`, destDir)

	cmd := exec.Command("powershell", "-Command", pathCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update PATH: %w", err)
	}

	// Also update the current process PATH so it's available immediately
	os.Setenv("PATH", os.Getenv("PATH")+";"+destDir)

	return nil
}

// installUnixGlobalSilently installs node-spark globally on Unix-like systems silently
func installUnixGlobalSilently() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	possibleDirs := []string{
		"/usr/local/bin",
		filepath.Join(homeDir, ".local", "bin"),
		filepath.Join(homeDir, "bin"),
	}

	var destDir string
	for _, dir := range possibleDirs {
		dirExists := true
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				continue
			}
			dirExists = false
		}

		testFile := filepath.Join(dir, ".node-spark_write_test")
		if err := os.WriteFile(testFile, []byte{}, 0644); err == nil {
			os.Remove(testFile)
			destDir = dir
			break
		}

		if !dirExists {
			os.Remove(dir)
		}
	}

	if destDir == "" {
		destDir = filepath.Join(homeDir, ".local", "bin")
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("couldn't create installation directory: %w", err)
		}
	}

	destPath := filepath.Join(destDir, "nsk")
	input, err := os.ReadFile(exePath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(destPath, input, 0755); err != nil {
		return err
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return err
	}

	path := os.Getenv("PATH")
	if !strings.Contains(path, destDir) {
		profiles := []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".profile"),
		}

		for _, profile := range profiles {
			if _, err := os.Stat(profile); err == nil {
				appendCmd := fmt.Sprintf("\n# Added by node-spark\nexport PATH=\"%s:$PATH\"\n", destDir)
				profileContent, err := os.ReadFile(profile)
				if err == nil && !strings.Contains(string(profileContent), destDir) {
					os.WriteFile(profile, append(profileContent, []byte(appendCmd)...), 0644)
				}
			}
		}

		os.Setenv("PATH", destDir+":"+os.Getenv("PATH"))
	}

	if destDir != "/usr/local/bin" {
		lnCmd := exec.Command("sudo", "ln", "-sf", destPath, "/usr/local/bin/nsk")
		if err := lnCmd.Run(); err != nil {
			exec.Command("ln", "-sf", destPath, "/usr/local/bin/nsk").Run()
		}
	}

	return nil
}

// InstallGlobal installs node-spark globally with user interaction
func InstallGlobal() error {
	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║ node-spark Global Installation                     ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
	fmt.Println("Installing node-spark globally on your system...")

	progressDone := make(chan bool)
	errorChan := make(chan error)

	go func() {
		var err error
		switch runtime.GOOS {
		case "windows":
			err = installWindowsGlobal()
		case "darwin", "linux":
			err = installUnixGlobal()
		default:
			err = fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}

		if err != nil {
			errorChan <- err
		} else {
			progressDone <- true
		}
	}()

	doneChannel := make(chan bool)
	go PrintIndeterminateProgress("Installing node-spark globally", doneChannel)

	select {
	case <-progressDone:
		doneChannel <- true
		time.Sleep(500 * time.Millisecond)
		fmt.Println("\n✅ node-spark installation complete!")
		fmt.Println("You can now run 'nsk' from any directory.")
		return nil
	case err := <-errorChan:
		doneChannel <- true
		fmt.Println("\n❌ Installation failed.")
		return err
	}
}

// installWindowsGlobal installs node-spark globally on Windows with detailed error handling
func installWindowsGlobal() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return fmt.Errorf("USERPROFILE environment variable not set")
	}

	destDir := filepath.Join(userProfile, "AppData", "Local", "Programs", "node-spark")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	destPath := filepath.Join(destDir, "nsk.exe")

	// Use CopyFile for better copying of executables
	if err := copyFile(exePath, destPath); err != nil {
		return fmt.Errorf("failed to install executable: %w", err)
	}

	// Update PATH with PowerShell for better handling
	pathCmd := fmt.Sprintf(`
		$destDir = '%s'
		$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
		if ($currentPath -notlike "*$destDir*") {
			[Environment]::SetEnvironmentVariable('Path', "$currentPath;$destDir", 'User')
			Write-Output "Added to PATH: $destDir"
		} else {
			Write-Output "Path already contains: $destDir"
		}
	`, destDir)

	cmd := exec.Command("powershell", "-Command", pathCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update PATH: %w (output: %s)", err, string(output))
	}

	fmt.Println(string(output))
	fmt.Println("Global installation complete. You may need to restart your terminal or computer for the PATH changes to take effect.")
	return nil
}

// installUnixGlobal installs node-spark globally on Unix-like systems with detailed error handling
func installUnixGlobal() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	var destDir string

	if os.Getuid() == 0 {
		destDir = "/usr/local/bin"
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		binDir := filepath.Join(homeDir, "bin")
		localBinDir := filepath.Join(homeDir, ".local", "bin")

		path := os.Getenv("PATH")
		if strings.Contains(path, localBinDir) {
			destDir = localBinDir
		} else if strings.Contains(path, binDir) {
			destDir = binDir
		} else {
			destDir = localBinDir
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create bin directory: %w", err)
		}
	}

	destPath := filepath.Join(destDir, "nsk")

	input, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("failed to read executable: %w", err)
	}

	if err := os.WriteFile(destPath, input, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %w", err)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	path := os.Getenv("PATH")
	if !strings.Contains(path, destDir) {
		os.Setenv("PATH", destDir+":"+os.Getenv("PATH"))

		homeDir, _ := os.UserHomeDir()
		profiles := []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".profile"),
		}

		for _, profile := range profiles {
			if _, err := os.Stat(profile); err == nil {
				appendCmd := fmt.Sprintf("\n# Added by node-spark\nexport PATH=\"%s:$PATH\"\n", destDir)
				profileContent, err := os.ReadFile(profile)
				if err == nil && !strings.Contains(string(profileContent), destDir) {
					os.WriteFile(profile, append(profileContent, []byte(appendCmd)...), 0644)
				}
			}
		}
	}

	if destDir != "/usr/local/bin" {
		lnCmd := exec.Command("sudo", "ln", "-sf", destPath, "/usr/local/bin/nsk")
		if err := lnCmd.Run(); err != nil {
			exec.Command("ln", "-sf", destPath, "/usr/local/bin/nsk").Run()
		}
	}

	return nil
}

// UninstallGlobal uninstalls node-spark with user confirmation
func UninstallGlobal() error {
	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║ node-spark Uninstallation                          ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
	fmt.Print("Are you sure you want to uninstall node-spark? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(response)

	if response != "y" && response != "yes" {
		fmt.Println("Uninstallation cancelled.")
		return nil
	}

	fmt.Println("Uninstalling node-spark...")

	progressDone := make(chan bool)
	errorChan := make(chan error)

	go func() {
		var err error
		switch runtime.GOOS {
		case "windows":
			err = uninstallWindowsGlobal()
		case "darwin", "linux":
			err = uninstallUnixGlobal()
		default:
			err = fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}

		if err != nil {
			errorChan <- err
		} else {
			progressDone <- true
		}
	}()

	doneChannel := make(chan bool)
	go PrintIndeterminateProgress("Uninstalling node-spark", doneChannel)

	select {
	case <-progressDone:
		doneChannel <- true
		time.Sleep(500 * time.Millisecond)
		fmt.Println("\n✅ node-spark has been uninstalled successfully.")
		fmt.Println("   You may need to restart your terminal for PATH changes to take effect.")
		return nil
	case err := <-errorChan:
		doneChannel <- true
		fmt.Println("\n❌ Uninstallation failed.")
		return err
	}
}

// uninstallWindowsGlobal uninstalls node-spark globally on Windows
func uninstallWindowsGlobal() error {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return fmt.Errorf("USERPROFILE environment variable not set")
	}

	destDir := filepath.Join(userProfile, "AppData", "Local", "Programs", "node-spark")
	destPath := filepath.Join(destDir, "nsk.exe")

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("node-spark is not installed globally")
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove installation directory: %w", err)
	}

	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf(`[Environment]::SetEnvironmentVariable("PATH", ($env:PATH -replace [regex]::Escape(";%s"), ""), [EnvironmentVariableTarget]::User)`, destDir))
	_ = cmd.Run()

	return nil
}

// uninstallUnixGlobal uninstalls node-spark globally on Unix-like systems
func uninstallUnixGlobal() error {
	possibleLocations := []string{
		"/usr/local/bin/nsk",
		"/usr/bin/nsk",
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		possibleLocations = append(possibleLocations,
			filepath.Join(homeDir, "bin", "nsk"),
			filepath.Join(homeDir, ".local", "bin", "nsk"))
	}

	uninstalled := false
	for _, location := range possibleLocations {
		if _, err := os.Stat(location); err == nil {
			if err := os.Remove(location); err != nil {
				fmt.Printf("Failed to remove %s: %v\n", location, err)
			} else {
				uninstalled = true
			}
		}
	}

	if !uninstalled {
		return fmt.Errorf("node-spark is not installed globally or couldn't be found")
	}

	profiles := []string{
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".bash_profile"),
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".profile"),
	}

	for _, profile := range profiles {
		if _, err := os.Stat(profile); err == nil {
			content, err := os.ReadFile(profile)
			if err == nil {
				lines := strings.Split(string(content), "\n")
				var newLines []string

				for _, line := range lines {
					if strings.Contains(line, "# Added by node-spark") ||
						(strings.Contains(line, "export PATH=") &&
							(strings.Contains(line, "/bin/nsk") ||
								strings.Contains(line, "node-spark"))) {
						continue
					}
					newLines = append(newLines, line)
				}

				os.WriteFile(profile, []byte(strings.Join(newLines, "\n")), 0644)
			}
		}
	}

	return nil
}

// IsFirstRun checks if this is the first run of node-spark
func IsFirstRun() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return true // If we can't get home dir, assume first run
	}

	nodesparkDir := filepath.Join(homeDir, ".node-spark")
	firstRunMarker := filepath.Join(nodesparkDir, ".first_run_complete")

	if _, err := os.Stat(firstRunMarker); os.IsNotExist(err) {
		// Create the marker file
		os.MkdirAll(nodesparkDir, 0755)
		os.WriteFile(firstRunMarker, []byte{}, 0644)
		return true
	}

	return false
}

// PrintIndeterminateProgress prints a progress animation in the terminal
func PrintIndeterminateProgress(message string, done chan bool) {
	spinners := []string{"|", "/", "-", "\\"}
	i := 0

	for {
		select {
		case <-done:
			return
		default:
			fmt.Printf("\r%s %s", message, spinners[i])
			i = (i + 1) % len(spinners)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// copyFile is a helper function to properly copy executable files
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file with the same permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// Remove destination if it already exists
	if _, err := os.Stat(dst); err == nil {
		if err = os.Remove(dst); err != nil {
			return err
		}
	}

	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
