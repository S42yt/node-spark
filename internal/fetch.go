package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// NodeVersion represents a version entry from the Node.js index.
type NodeVersion struct {
	Version string      `json:"version"`
	Date    string      `json:"date"`
	Files   []string    `json:"files"`
	LTS     interface{} `json:"lts"`     // Can be boolean false or string like "Hydrogen"
	Modules interface{} `json:"modules"` // Can be int or string
	NPM     string      `json:"npm,omitempty"`
	V8      string      `json:"v8"`
}

// FetchAvailableVersions fetches the list of available Node.js versions.
func FetchAvailableVersions() ([]NodeVersion, error) {
	fmt.Println("Fetching Node.js versions from https://nodejs.org/dist/index.json...")

	// Create a client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://nodejs.org/dist/index.json", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add user agent to avoid being blocked
	req.Header.Set("User-Agent", "node-spark/1.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching version index:", err)
		return nil, fmt.Errorf("failed to fetch version index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Bad status code:", resp.Status)
		return nil, fmt.Errorf("failed to fetch version index: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, fmt.Errorf("failed to read version index body: %w", err)
	}

	// Try to fix common JSON issues before parsing
	bodyStr := string(body)
	if strings.TrimSpace(bodyStr) == "" {
		fmt.Println("Received empty response body")
		return nil, fmt.Errorf("received empty response body")
	}

	// Try parsing the JSON
	var versions []NodeVersion
	err = json.Unmarshal(body, &versions)
	if err != nil {
		fmt.Println("JSON parse error:", err)

		// Try using a more flexible approach
		var rawData []map[string]interface{}
		err2 := json.Unmarshal(body, &rawData)
		if err2 != nil {
			fmt.Println("Even flexible parsing failed:", err2)

			// Save response for debugging
			debugFile := filepath.Join(os.TempDir(), "node_versions_response.json")
			_ = os.WriteFile(debugFile, body, 0644)
			fmt.Printf("Saved failed response to %s for debugging\n", debugFile)

			return nil, fmt.Errorf("failed to parse version index: %w", err)
		}

		// Convert manually
		versions = make([]NodeVersion, len(rawData))
		for i, item := range rawData {
			var version NodeVersion
			if v, ok := item["version"].(string); ok {
				version.Version = v
			}
			if d, ok := item["date"].(string); ok {
				version.Date = d
			}
			if f, ok := item["files"].([]interface{}); ok {
				files := make([]string, len(f))
				for j, file := range f {
					if s, ok := file.(string); ok {
						files[j] = s
					}
				}
				version.Files = files
			}
			version.LTS = item["lts"]
			version.Modules = item["modules"]
			if n, ok := item["npm"].(string); ok {
				version.NPM = n
			}
			if v, ok := item["v8"].(string); ok {
				version.V8 = v
			}
			versions[i] = version
		}
		fmt.Println("Manual parsing succeeded with", len(versions), "versions")
	} else {
		fmt.Println("Successfully parsed", len(versions), "Node.js versions")
	}

	// Sort versions (optional, but good for display)
	sort.Slice(versions, func(i, j int) bool {
		// Extract numeric version parts for proper comparison
		vNumI := extractVersionNumbers(versions[i].Version)
		vNumJ := extractVersionNumbers(versions[j].Version)

		// Compare each part of the version
		for k := 0; k < len(vNumI) && k < len(vNumJ); k++ {
			if vNumI[k] != vNumJ[k] {
				return vNumI[k] > vNumJ[k] // Sort in descending order (newer first)
			}
		}

		// If all parts are equal so far, the longer version is considered greater
		return len(vNumI) > len(vNumJ)
	})

	return versions, nil
}

// Helper function to extract version numbers for comparison
func extractVersionNumbers(version string) []int {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Split by dots
	parts := strings.Split(version, ".")

	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		// Handle cases like "1.2.3-rc.1" by taking only the number part
		numPart := strings.Split(part, "-")[0]
		num, err := strconv.Atoi(numPart)
		if err == nil {
			numbers = append(numbers, num)
		}
	}

	return numbers
}

// IsLTS determines if a version is an LTS release
func (v NodeVersion) IsLTS() bool {
	switch lts := v.LTS.(type) {
	case bool:
		return lts
	case string:
		return lts != ""
	default:
		return false
	}
}

// LTSName returns the name of the LTS version (like "Hydrogen") or empty string if not LTS
func (v NodeVersion) LTSName() string {
	switch lts := v.LTS.(type) {
	case string:
		return lts
	default:
		return ""
	}
}

// GetVersionString returns a user-friendly version string with LTS information if applicable
func (v NodeVersion) GetVersionString() string {
	if v.IsLTS() {
		ltsName := v.LTSName()
		if ltsName != "" {
			return fmt.Sprintf("%s (LTS: %s)", v.Version, ltsName)
		}
		return fmt.Sprintf("%s (LTS)", v.Version)
	}
	return v.Version
}

// CleanVersion returns the version string without the 'v' prefix, suitable for directory names
func (v NodeVersion) CleanVersion() string {
	return strings.TrimPrefix(v.Version, "v")
}

// FetchVersionDetails fetches details for a specific version if available in the index.
func FetchVersionDetails(versionQuery string) (NodeVersion, error) {
	versions, err := FetchAvailableVersions()
	if err != nil {
		return NodeVersion{}, err
	}

	// Standardize version query (add v prefix if needed)
	if !strings.HasPrefix(versionQuery, "v") {
		versionQuery = "v" + versionQuery
	}

	for _, v := range versions {
		if v.Version == versionQuery {
			return v, nil
		}
	}

	return NodeVersion{}, fmt.Errorf("version %s not found in Node.js index", versionQuery)
}

// DownloadFile downloads a file from a URL to a local path with progress reporting
func DownloadFile(filepath string, url string) error {
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Set a user agent to avoid being blocked
	req.Header.Set("User-Agent", "node-spark/1.0")

	// Get the data
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Set up progress reader
	progressReader := &ProgressReader{
		Reader:        resp.Body,
		ContentLength: resp.ContentLength,
		OnProgress:    PrintProgressBar,
	}

	// Write the body to file with progress reporting
	_, err = io.Copy(out, progressReader)

	// Print a newline after the progress bar completes
	fmt.Println()

	return err
}
