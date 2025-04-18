// internal/windows.go
package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateWindowsShims creates the necessary shim files for Node.js on Windows
func CreateWindowsShims(nodeBinPath, shimDir string) error {
	// Ensure the shim directory exists
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("failed to create shim directory: %w", err)
	}

	// Create proper shims for all relevant executables
	binaries := map[string]string{
		"node.exe": "@echo off\r\n\"%s\" %%*",
		"npm.cmd":  "@echo off\r\n\"%s\" %%*",
		"npx.cmd":  "@echo off\r\n\"%s\" %%*",
	}

	// First, try to find node.exe if not in the expected location
	nodePath := filepath.Join(nodeBinPath, "node.exe")
	if _, err := os.Stat(nodePath); os.IsNotExist(err) {
		fmt.Printf("Node.exe not found at expected path: %s\n", nodePath)

		// Check for the bin subdirectory first
		binDir := filepath.Join(nodeBinPath, "bin")
		binNodePath := filepath.Join(binDir, "node.exe")
		if _, err := os.Stat(binNodePath); err == nil {
			fmt.Printf("Found node.exe in bin subdirectory: %s\n", binNodePath)
			nodeBinPath = binDir
		} else {
			// Search recursively for node.exe
			var nodeExePath string
			filepath.Walk(nodeBinPath, func(path string, info os.FileInfo, err error) error {
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
				fmt.Printf("Found node.exe at %s\n", nodeExePath)
				// Use the directory containing node.exe
				nodeBinPath = filepath.Dir(nodeExePath)
			}
		}
	}

	// Create shim files
	for binary, template := range binaries {
		sourcePath := filepath.Join(nodeBinPath, binary)

		// Skip if the source binary doesn't exist
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			fmt.Printf("Warning: %s not found in %s, skipping shim creation\n", binary, nodeBinPath)

			// For npm and npx, if .cmd versions don't exist, try to find them
			if binary == "npm.cmd" || binary == "npx.cmd" {
				baseName := strings.TrimSuffix(binary, ".cmd")
				var foundPath string

				// Look for alternatives (.cmd, .bat, or no extension)
				alternatives := []string{baseName, baseName + ".bat", baseName}
				for _, alt := range alternatives {
					altPath := filepath.Join(nodeBinPath, alt)
					if _, err := os.Stat(altPath); err == nil {
						foundPath = altPath
						fmt.Printf("Found alternative for %s: %s\n", binary, foundPath)
						break
					}
				}

				// If we found an alternative, create a shim for it
				if foundPath != "" {
					shimPath := filepath.Join(shimDir, binary)
					shimContent := fmt.Sprintf(template, foundPath)
					if err := os.WriteFile(shimPath, []byte(shimContent), 0755); err != nil {
						return fmt.Errorf("failed to create shim for %s: %w", binary, err)
					}
				}
			}

			// Continue with the next binary
			continue
		}

		shimPath := filepath.Join(shimDir, binary)
		shimContent := fmt.Sprintf(template, sourcePath)

		if err := os.WriteFile(shimPath, []byte(shimContent), 0755); err != nil {
			return fmt.Errorf("failed to create shim for %s: %w", binary, err)
		}
	}

	// Special handling for node.exe - if we still couldn't find it, create a special warning shim
	nodeShimPath := filepath.Join(shimDir, "node.exe")
	if _, err := os.Stat(nodeShimPath); os.IsNotExist(err) {
		// Create an error-reporting batch file as fallback
		errorShim := "@echo off\r\necho Node.js executable could not be found. Please reinstall this version.\r\nexit /b 1"
		if err := os.WriteFile(nodeShimPath, []byte(errorShim), 0755); err != nil {
			return fmt.Errorf("failed to create error shim for node.exe: %w", err)
		}

		// Let the user know about the problem but don't fail completely
		fmt.Printf("WARNING: node.exe could not be found in the installed files.\n")
		fmt.Printf("A placeholder shim has been created. Consider reinstalling this version.\n")
	}

	return nil
}

// UpdateWindowsPath updates the PATH environment variable on Windows
func UpdateWindowsPath(shimDir string) error {
	// Update both user and process PATH
	updatePaths := []struct {
		name string
		cmd  string
	}{
		// Update the User's PATH environment variable (for future sessions)
		{
			name: "User",
			cmd: fmt.Sprintf(`$path = [Environment]::GetEnvironmentVariable('Path', 'User'); 
			if (-not $path.Contains('%s')) { 
				[Environment]::SetEnvironmentVariable('Path', "$path;%s", 'User')
			}`, shimDir, shimDir),
		},
		// Update the current process PATH (for immediate use)
		{
			name: "Process",
			cmd:  fmt.Sprintf(`$env:Path = $env:Path + ";%s"`, shimDir),
		},
	}

	for _, update := range updatePaths {
		cmd := exec.Command("powershell", "-Command", update.cmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Could not update %s PATH: %v\n", update.name, err)
			// Continue anyway, don't fail the entire operation
		}
	}

	return nil
}

// CreateWindowsActivationScript creates a PowerShell script that can be dot-sourced
// to update the PATH in the current terminal session
func CreateWindowsActivationScript(shimDir, version string) (string, error) {
	activateScript := filepath.Join(shimDir, "activate.ps1")
	scriptContent := fmt.Sprintf(`# Node.js activation script for Windows
# This script updates your current terminal's PATH to use Node.js %s

$ErrorActionPreference = "Stop"
$shimDir = "%s"

if (-not $env:Path.Contains($shimDir)) {
    $env:Path = "$shimDir;$env:Path"
}

# Verify node is accessible with proper error handling
try {
    # First check if the executable exists
    $nodePath = Join-Path $shimDir "node.exe"
    if (-not (Test-Path $nodePath)) {
        throw "Node.js executable not found at $nodePath"
    }
    
    # Try running node with safe approach using ProcessStartInfo to properly handle failures
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $nodePath
    $psi.Arguments = "--version"
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $psi
    
    try {
        [void]$process.Start()
        $stdout = $process.StandardOutput.ReadToEnd()
        $stderr = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        
        if ($process.ExitCode -eq 0) {
            Write-Host "Node.js $stdout is now active in this terminal session."
        } else {
            throw "Node.js failed to execute properly (Exit code: $($process.ExitCode)). Error: $stderr"
        }
    } catch {
        throw "Failed to run Node.js: $_"
    }
} catch {
    Write-Host "Warning: Node.js activation failed." -ForegroundColor Yellow
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "This is likely due to an architecture mismatch." -ForegroundColor Yellow
    Write-Host "Please reinstall Node.js with the x86 (32-bit) version:" -ForegroundColor Yellow
    Write-Host "  nsk uninstall %s" -ForegroundColor Cyan
    Write-Host "  nsk install %s" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "After reinstalling, run the following to try again:" -ForegroundColor Yellow
    Write-Host "  nsk use %s" -ForegroundColor Cyan
}
`, version, shimDir, version, version, version)

	if err := os.WriteFile(activateScript, []byte(scriptContent), 0755); err != nil {
		return "", fmt.Errorf("could not create activation script: %w", err)
	}

	return activateScript, nil
}

// IsProperArchForSystem checks if the given Node.js binary is compatible with the system
func IsProperArchForSystem(nodePath string) bool {
	cmd := exec.Command(nodePath, "--version")
	if err := cmd.Run(); err != nil {
		// The binary is not compatible or has other issues
		return false
	}
	return true
}
