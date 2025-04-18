// internal/use.go

package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/s42yt/node-spark/pkg"
)

// UseVersion switches the active Node.js version by creating appropriate symlinks
// or, on Windows, by modifying PATH-related registry entries and creating shims.
func UseVersion(version string, config *pkg.Config) error {
	// Check if the version is installed
	versionPath := filepath.Join(pkg.GetInstallPath(config), version)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("version %s is not installed", version)
	}

	fmt.Printf("Switching to Node.js version %s...\n", version)

	// The current implementation approach differs by OS
	if runtime.GOOS == "windows" {
		return useVersionWindows(version, versionPath, config)
	}

	return useVersionPosix(version, versionPath, config)
}

// useVersionWindows implements version switching for Windows
// by creating batch script shims in a central location
func useVersionWindows(version, versionPath string, config *pkg.Config) error {
	// Base directory for node-spark
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	nodeBinPath := filepath.Join(versionPath, "bin")
	if _, err := os.Stat(nodeBinPath); os.IsNotExist(err) {
		// On Windows, executables might be directly in the version folder
		nodeBinPath = versionPath
	}

	// Create shims directory
	shimDir := filepath.Join(homeDir, ".node-spark", "shims")

	// Create the shim files using our Windows helper
	if err := CreateWindowsShims(nodeBinPath, shimDir); err != nil {
		return err
	}

	// Update the PATH environment variable
	if err := UpdateWindowsPath(shimDir); err != nil {
		fmt.Printf("Warning: Could not fully update PATH environment: %v\n", err)
		// Continue anyway - not fatal
	}

	// Create activation script for immediate use
	activateScript, err := CreateWindowsActivationScript(shimDir, version)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		// Continue anyway - not fatal
	}

	// Set or update the ActiveVersion in config
	config.ActiveVersion = version

	// Verify the node executable is compatible with the system
	nodePath := filepath.Join(shimDir, "node.exe")
	if !IsProperArchForSystem(nodePath) {
		fmt.Printf("Warning: The installed Node.js binary may not be compatible with your system.\n")
		fmt.Printf("You might need to install a different architecture version.\n")
	}

	fmt.Printf("Created shims in %s\n", shimDir)
	fmt.Printf("Node.js %s should now be available in new terminal windows.\n", version)
	fmt.Printf("For immediate use in your current terminal, run: . %s\n", activateScript)

	return nil
}

// useVersionPosix implements version switching for Unix-like systems
// by creating symlinks in a central location
func useVersionPosix(version, versionPath string, config *pkg.Config) error {
	// Base directory for node-spark
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	// Create the symlink directory
	symlinkDir := filepath.Join(homeDir, ".node-spark", "current")
	if err := os.MkdirAll(filepath.Dir(symlinkDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for symlink: %w", err)
	}

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkDir); err == nil {
		if err := os.Remove(symlinkDir); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(versionPath, symlinkDir); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Set the ActiveVersion in config
	config.ActiveVersion = version

	fmt.Printf("Successfully switched to Node.js %s\n", version)
	fmt.Printf("Make sure %s/bin is in your PATH\n", symlinkDir)

	// Suggest adding to shell config if not already there
	fmt.Println("\nTo ensure the Node.js version persists in new terminal sessions, add this to your shell config file:")
	fmt.Printf("export PATH=\"%s/bin:$PATH\"\n", symlinkDir)

	return nil
}

// GetActiveNodeVersion retrieves the currently active Node.js version
// by checking the symlink or current configuration.
func GetActiveNodeVersion() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	// Try to determine by checking the active version symlink/dir
	symlinkPath := filepath.Join(homeDir, ".node-spark", "current")

	// Check if the symlink exists and where it points to
	if linkDest, err := filepath.EvalSymlinks(symlinkPath); err == nil {
		// Extract version from the path
		parts := strings.Split(linkDest, string(filepath.Separator))
		if len(parts) > 0 {
			// Assume the last directory component is the version
			return parts[len(parts)-1], nil
		}
	}

	// If we couldn't determine from symlink, try running node --version
	// This could return the system's default Node version if our tool's version isn't active
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		// Remove the 'v' prefix if present
		version = strings.TrimPrefix(version, "v")
		return version, nil
	}

	return "", fmt.Errorf("no active Node.js version found")
}

// ListAvailableNodeVersions lists all available Node.js versions
// that can be installed from nodejs.org.
func ListAvailableNodeVersions() ([]string, error) {
	versions, err := FetchAvailableVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available versions: %w", err)
	}

	result := make([]string, len(versions))
	for i, version := range versions {
		// Use our new GetVersionString method
		result[i] = version.GetVersionString()
	}

	return result, nil
}

// SetActiveVersion updates the configuration to mark a version as active.
// This currently only updates the config file. Actual PATH modification
// or symlinking needs to be handled separately (e.g., by user's shell profile
// sourcing a script generated by node-spark, or by direct symlinking if feasible).
func SetActiveVersion(version string, config *pkg.Config) error {
	// Verify the version is actually installed
	versionPath := filepath.Join(pkg.GetInstallPath(config), version)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("version %s is not installed", version)
	}

	// Now use our implementation that actually creates symlinks or shims
	if err := UseVersion(version, config); err != nil {
		return err
	}

	config.ActiveVersion = version
	fmt.Printf("Set active Node.js version to %s\n", version)
	return nil
}

// GetActiveVersion retrieves the currently active Node.js version from the config.
func GetActiveVersion(config *pkg.Config) (string, error) {
	if config.ActiveVersion == "" {
		return "", fmt.Errorf("no active Node.js version set")
	}
	return config.ActiveVersion, nil
}

// ListInstalledVersions retrieves the list of installed Node.js versions from the config.
func ListInstalledVersions(config *pkg.Config) ([]string, error) {
	return config.InstalledVersions, nil
}

// UninstallVersion removes an installed Node.js version
func UninstallVersion(version string, config *pkg.Config) error {
	// Check if version is currently active
	if config.ActiveVersion == version {
		return fmt.Errorf("cannot uninstall the currently active version; switch to another version first")
	}

	// Check if the version actually exists
	versionPath := filepath.Join(pkg.GetInstallPath(config), version)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("version %s is not installed", version)
	}

	// Remove the version directory
	if err := os.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	// Update the config to remove this version
	for i, v := range config.InstalledVersions {
		if v == version {
			// Remove this version from the slice
			config.InstalledVersions = append(
				config.InstalledVersions[:i],
				config.InstalledVersions[i+1:]...,
			)
			break
		}
	}

	fmt.Printf("Successfully uninstalled Node.js version %s\n", version)
	return nil
}

// --- TUI Implementation ---

// Model represents the state of the TUI
type TUIModel struct {
	list          list.Model
	state         string
	selectedIndex int
	config        *pkg.Config
	error         string
	spinner       spinner.Model
	loading       bool
	input         textinput.Model
}

// item represents an item in the TUI list
type item struct {
	title       string
	description string
	isActive    bool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// initTUI initializes the TUI model
func InitTUI(config *pkg.Config) tea.Model {
	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	// Create text input for version installation
	ti := textinput.New()
	ti.Placeholder = "Enter Node.js version to install (e.g., 18.17.0)"
	ti.Focus()
	ti.Width = 40

	// Create the model
	m := TUIModel{
		state:   "menu",
		config:  config,
		spinner: s,
		input:   ti,
	}

	return m
}

// Init initializes the TUI
func (m TUIModel) Init() tea.Cmd {
	return tea.Batch(loadInstalledVersions(m.config), m.spinner.Tick)
}

// Update updates the TUI state
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == "menu" {
				return m, tea.Quit
			} else {
				// Go back to main menu
				m.state = "menu"
				return m, loadInstalledVersions(m.config)
			}
		case "enter":
			if m.state == "install" {
				version := m.input.Value()
				m.input.Reset()
				m.state = "installing"
				m.loading = true
				return m, installVersion(version, m.config)
			}
		}

	case loadedVersionsMsg:
		m.loading = false
		if msg.err != nil {
			m.error = msg.err.Error()
			return m, nil
		}

		// Check if there are any versions to display
		if len(msg.versions) == 0 {
			// Create an empty list but with proper initialization
			delegate := list.NewDefaultDelegate()
			emptyList := list.New([]list.Item{}, delegate, 0, 0)
			emptyList.Title = "Installed Node.js Versions"
			emptyList.SetStatusBarItemName("version", "versions")
			emptyList.SetShowHelp(true)
			// Add a placeholder item so the list doesn't panic
			emptyItem := item{title: "No Node.js versions installed", description: "Press 'i' to install a version"}
			emptyList.SetItems([]list.Item{emptyItem})

			m.list = emptyList

			// Add key bindings for empty list too
			m.list.AdditionalFullHelpKeys = func() []key.Binding {
				return []key.Binding{
					key.NewBinding(
						key.WithKeys("i"),
						key.WithHelp("i", "install new version"),
					),
				}
			}
			return m, nil
		}

		items := make([]list.Item, len(msg.versions))
		for i, v := range msg.versions {
			isActive := v == m.config.ActiveVersion
			var desc string
			if isActive {
				desc = "Currently active version"
			}
			items[i] = item{title: v, description: desc, isActive: isActive}
		}

		delegate := list.NewDefaultDelegate()
		listModel := list.New(items, delegate, 0, 0)
		listModel.Title = "Installed Node.js Versions"
		listModel.SetStatusBarItemName("version", "versions")
		listModel.SetShowHelp(true) // Make sure help is shown
		listModel.AdditionalFullHelpKeys = func() []key.Binding {
			return []key.Binding{
				key.NewBinding(
					key.WithKeys("u"),
					key.WithHelp("u", "use version"),
				),
				key.NewBinding(
					key.WithKeys("x"),
					key.WithHelp("x", "uninstall version"),
				),
				key.NewBinding(
					key.WithKeys("i"),
					key.WithHelp("i", "install new version"),
				),
			}
		}

		m.list = listModel
		return m, nil

	case installedVersionMsg:
		m.loading = false
		m.state = "menu"
		if msg.err != nil {
			m.error = msg.err.Error()
		}
		return m, loadInstalledVersions(m.config)

	case versionActivatedMsg:
		m.loading = false
		m.state = "menu"
		if msg.err != nil {
			m.error = msg.err.Error()
		}
		return m, loadInstalledVersions(m.config)

	case versionUninstalledMsg:
		m.loading = false
		m.state = "menu"
		if msg.err != nil {
			m.error = msg.err.Error()
		}
		return m, loadInstalledVersions(m.config)
	}

	// Handle list events when in menu state
	if m.state == "menu" && m.list.Items() != nil && len(m.list.Items()) > 0 {
		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		cmds = append(cmds, listCmd)

		// Handle custom key presses for list items
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "u": // Use the selected version
				if len(m.list.Items()) > 0 && m.list.SelectedItem() != nil {
					selected := m.list.SelectedItem().(item)
					// Don't try to "use" our placeholder message
					if selected.title != "No Node.js versions installed" {
						m.loading = true
						cmds = append(cmds, useVersionCmd(selected.title, m.config)) // Use the command wrapper
					}
				}
			case "x": // Uninstall the selected version
				if len(m.list.Items()) > 0 && m.list.SelectedItem() != nil {
					selected := m.list.SelectedItem().(item)
					// Don't try to uninstall our placeholder message
					if selected.title != "No Node.js versions installed" {
						m.loading = true
						cmds = append(cmds, uninstallVersion(selected.title, m.config))
					}
				}
			case "i": // Install a new version
				m.state = "install"
				m.input.Focus()
				return m, textinput.Blink
			}
		}
	} else if m.state == "menu" {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			// Special handling when we don't have a list or it's empty
			if keyMsg.String() == "i" {
				m.state = "install"
				m.input.Focus()
				return m, textinput.Blink
			}
		}
	}

	// Handle input events when in install state
	if m.state == "install" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Handle spinner updates when loading
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m TUIModel) View() string {
	var view string
	if m.loading {
		view = fmt.Sprintf("\n  %s Loading...\n\n", m.spinner.View())
		return view
	}

	switch m.state {
	case "menu":
		view = "\n" + m.list.View()
	case "install", "installing":
		view = fmt.Sprintf("\n  Install a new Node.js version:\n\n  %s\n\n  (Enter to install, Esc to cancel)\n", m.input.View())
	default:
		view = "\nLoading..."
	}
	return view
}

// Custom message types for TUI updates
type loadedVersionsMsg struct {
	versions []string
	err      error
}

type installedVersionMsg struct {
	version string
	err     error
}

type versionActivatedMsg struct {
	version string
	err     error
}

type versionUninstalledMsg struct {
	version string
	err     error
}

// Commands for asynchronous operations
func loadInstalledVersions(config *pkg.Config) tea.Cmd {
	return func() tea.Msg {
		versions, err := ListInstalledVersions(config)
		return loadedVersionsMsg{versions: versions, err: err}
	}
}

func installVersion(version string, config *pkg.Config) tea.Cmd {
	return func() tea.Msg {
		// We need to import the installation function from install.go
		// Assuming InstallNodeVersion exists in another package or needs to be defined/imported
		// For now, let's assume it's available. If not, that's a separate issue.
		err := InstallNodeVersion(version, config) // Placeholder if not defined
		return installedVersionMsg{version: version, err: err}
	}
}

// useVersionCmd wraps the UseVersion logic in a tea.Cmd
func useVersionCmd(version string, config *pkg.Config) tea.Cmd {
	return func() tea.Msg {
		err := UseVersion(version, config)
		return versionActivatedMsg{version: version, err: err}
	}
}

// uninstallVersion wraps the UninstallVersion logic in a tea.Cmd
func uninstallVersion(version string, config *pkg.Config) tea.Cmd {
	return func() tea.Msg {
		err := UninstallVersion(version, config)
		return versionUninstalledMsg{version: version, err: err}
	}
}

// Assuming InstallNodeVersion is defined elsewhere or needs to be added.
// If InstallNodeVersion is not defined, you'll need to implement or import it.
// For example, if it's in the same package:
/*
func InstallNodeVersion(version string, config *pkg.Config) error {
	// Implementation for installing Node.js version
	fmt.Printf("Simulating installation of Node.js version %s...\n", version)
	// Add version to config (simulate success)
	config.InstalledVersions = append(config.InstalledVersions, version)
	// In a real scenario, this would download and extract the version
	return nil
}
*/

