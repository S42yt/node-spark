package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the application's configuration.
type Config struct {
	InstallPath       string   `json:"installPath"`
	InstalledVersions []string `json:"installedVersions"`
	ActiveVersion     string   `json:"activeVersion"` // Track the currently active version
}

// LoadConfig loads the configuration from a file.
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			installPath := filepath.Join(filepath.Dir(configPath), "versions")
			return &Config{InstallPath: installPath, InstalledVersions: []string{}}, nil
		}
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	// Ensure InstallPath is set if loaded config is old/missing it
	if config.InstallPath == "" {
		config.InstallPath = filepath.Join(filepath.Dir(configPath), "versions")
	}
	return &config, nil
}

// SaveConfig saves the configuration to a file.
func SaveConfig(configPath string, config *Config) error {
	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetConfigPath returns the path to the configuration file.
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback or handle error appropriately
		panic("Could not get user home directory: " + err.Error())
	}
	return filepath.Join(homeDir, ".node-spark", "config.json")
}

// GetInstallPath returns the path where Node versions are installed.
func GetInstallPath(config *Config) string {
	return config.InstallPath
}
