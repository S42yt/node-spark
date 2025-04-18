package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WipeAllData removes all node-spark data including configuration, installed Node.js versions, and shims
func WipeAllData() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Directory where node-spark stores its data (.node-spark)
	nodesparkDir := filepath.Join(homeDir, ".node-spark")

	fmt.Println("🧹 Wiping all node-spark data...")

	// Check if directory exists first
	if _, err := os.Stat(nodesparkDir); os.IsNotExist(err) {
		fmt.Println("No node-spark data found to wipe.")
		return nil
	}

	// Remove all the node versions
	versionsDir := filepath.Join(nodesparkDir, "versions")
	if _, err := os.Stat(versionsDir); err == nil {
		fmt.Println("- Removing installed Node.js versions...")
		if err := os.RemoveAll(versionsDir); err != nil {
			return fmt.Errorf("failed to remove versions directory: %w", err)
		}
	}

	// Remove shims directory
	shimsDir := filepath.Join(nodesparkDir, "shims")
	if _, err := os.Stat(shimsDir); err == nil {
		fmt.Println("- Removing Node.js shims...")
		if err := os.RemoveAll(shimsDir); err != nil {
			return fmt.Errorf("failed to remove shims directory: %w", err)
		}
	}

	// Remove current symlink directory
	currentDir := filepath.Join(nodesparkDir, "current")
	if _, err := os.Stat(currentDir); err == nil {
		fmt.Println("- Removing current Node.js symlink...")
		if err := os.RemoveAll(currentDir); err != nil {
			return fmt.Errorf("failed to remove current symlink: %w", err)
		}
	}

	// Remove config file
	configFile := filepath.Join(nodesparkDir, "config.json")
	if _, err := os.Stat(configFile); err == nil {
		fmt.Println("- Removing configuration...")
		if err := os.Remove(configFile); err != nil {
			return fmt.Errorf("failed to remove config file: %w", err)
		}
	}

	// Remove first run marker
	firstRunMarker := filepath.Join(nodesparkDir, ".first_run_complete")
	if _, err := os.Stat(firstRunMarker); err == nil {
		if err := os.Remove(firstRunMarker); err != nil {
			// Non-critical error, just log it
			fmt.Printf("Warning: Failed to remove first run marker: %v\n", err)
		}
	}

	// Finally, remove the entire .node-spark directory
	fmt.Println("- Removing node-spark directory...")
	if err := os.RemoveAll(nodesparkDir); err != nil {
		return fmt.Errorf("failed to remove node-spark directory: %w", err)
	}

	return nil
}

// ProgressReader is a custom io.Reader that reports download progress
type ProgressReader struct {
	Reader        io.Reader
	ContentLength int64
	TotalRead     int64
	OnProgress    func(bytesRead, totalBytes int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.TotalRead += int64(n)

	if pr.OnProgress != nil {
		pr.OnProgress(pr.TotalRead, pr.ContentLength)
	}

	return n, err
}

// PrintProgressBar prints a progress bar for downloads
func PrintProgressBar(progress, total int64) {
	if total <= 0 {
		// If total is unknown, show indeterminate progress
		spinners := []string{"|", "/", "-", "\\"}
		fmt.Printf("\r%s Downloading... ", spinners[int(progress/1024)%4])
		return
	}

	const width = 40
	percentage := float64(progress) / float64(total)
	completed := int(percentage * float64(width))

	// Format size in MB with one decimal place
	progressMB := float64(progress) / 1024 / 1024
	totalMB := float64(total) / 1024 / 1024

	fmt.Printf("\r[")
	for i := 0; i < width; i++ {
		if i < completed {
			fmt.Print("=")
		} else if i == completed {
			fmt.Print(">")
		} else {
			fmt.Print(" ")
		}
	}
	fmt.Printf("] %.1f/%.1fMB (%.1f%%)", progressMB, totalMB, percentage*100)
}
