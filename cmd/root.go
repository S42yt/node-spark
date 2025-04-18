package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/s42yt/node-spark/internal"
	"github.com/s42yt/node-spark/pkg"
	"github.com/spf13/cobra"
)

// version holds the application version, set at build time.
var version = "0.10"

var cfg *pkg.Config
var cfgPath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "node-spark",
	Short:   "A fast Node.js version manager",
	Long:    `NodeSpark is a CLI tool to manage multiple Node.js versions easily.`,
	Aliases: []string{"nsk"}, // Add nsk as an alias
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check for wipe flag first
		if wipeAll, _ := cmd.Flags().GetBool("wipe"); wipeAll {
			if err := internal.WipeAllData(); err != nil {
				return err
			}
			fmt.Println(" All node-spark data has been wiped successfully.")
			os.Exit(0)
		}

		// Load configuration before any command runs
		var err error
		cfgPath = pkg.GetConfigPath()
		cfg, err = pkg.LoadConfig(cfgPath)
		return err // Return error to stop execution if config loading fails
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.node-spark.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Hidden wipe flag
	rootCmd.PersistentFlags().Bool("wipe", false, "Wipe all node-spark data (hidden)")
	rootCmd.PersistentFlags().MarkHidden("wipe")

	// Add subcommands
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(listRemoteCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(uninstallCmd) // Add uninstall command
	rootCmd.AddCommand(tuiCmd)       // Add TUI command

	// Add global installation commands
	rootCmd.AddCommand(installGlobalCmd)
	rootCmd.AddCommand(uninstallGlobalCmd)
}

// --- Subcommands ---

var installCmd = &cobra.Command{
	Use:   "install [version]",
	Short: "Install a specific Node.js version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		err := internal.InstallVersion(version, cfg)
		if err != nil {
			return err
		}
		// Save config after successful install
		return pkg.SaveConfig(cfgPath, cfg)
	},
}

var useCmd = &cobra.Command{
	Use:   "use [version]",
	Short: "Switch to use a specific Node.js version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		err := internal.SetActiveVersion(version, cfg)
		if err != nil {
			return err
		}
		// Save config after successful use command
		return pkg.SaveConfig(cfgPath, cfg)
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List installed Node.js versions",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		versions, err := internal.ListInstalledVersions(cfg)
		if err != nil {
			return err
		}
		activeVersion, _ := internal.GetActiveVersion(cfg) // Ignore error if none active

		if len(versions) == 0 {
			fmt.Println("No Node.js versions installed yet.")
			return nil
		}

		fmt.Println("Installed Node.js versions:")
		for _, v := range versions {
			if v == activeVersion {
				fmt.Printf(" * %s (active)\n", v)
			} else {
				fmt.Printf("   %s\n", v)
			}
		}
		return nil
	},
}

var listRemoteCmd = &cobra.Command{
	Use:     "list-remote",
	Short:   "List available Node.js versions for installation",
	Aliases: []string{"ls-remote"},
	RunE: func(cmd *cobra.Command, args []string) error {
		versions, err := internal.FetchAvailableVersions()
		if err != nil {
			return fmt.Errorf("failed to fetch remote versions: %w", err)
		}

		fmt.Println("Available Node.js versions:")
		for _, v := range versions {
			fmt.Printf("  %s\n", v.GetVersionString())
		}
		return nil
	},
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Display the currently active Node.js version",
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := internal.GetActiveVersion(cfg)
		if err != nil {
			// Handle case where no version is active yet
			if err.Error() == "no active Node.js version set" {
				fmt.Println("No active Node.js version set. Use 'nsk use <version>' to set one.")
				return nil
			}
			return err
		}
		fmt.Printf("Currently active Node.js version: %s\n", version)
		return nil
	},
}

// New uninstall command
var uninstallCmd = &cobra.Command{
	Use:     "uninstall [version]",
	Short:   "Uninstall a specific Node.js version",
	Aliases: []string{"remove", "rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		err := internal.UninstallVersion(version, cfg)
		if err != nil {
			return err
		}
		// Save config after successful uninstall
		return pkg.SaveConfig(cfgPath, cfg)
	},
}

// TUI command to launch the terminal UI for managing Node.js versions
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the terminal UI for managing Node.js versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create and start the TUI
		p := tea.NewProgram(internal.InitTUI(cfg))
		_, err := p.Run()

		// If there was a TUI error, return it
		if err != nil {
			return err
		}

		// Save any changes made in the TUI to config
		return pkg.SaveConfig(cfgPath, cfg)
	},
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Show the current version of Node Spark",
	Aliases: []string{"v"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Node Spark version: %s\n", version)
	},
}

// --- Global Installation Commands ---

var installGlobalCmd = &cobra.Command{
	Use:     "install-global",
	Short:   "Install node-spark globally on your system",
	Aliases: []string{"global-install", "ig"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return internal.InstallGlobal()
	},
}

var uninstallGlobalCmd = &cobra.Command{
	Use:     "uninstall-global",
	Short:   "Uninstall node-spark from your system",
	Aliases: []string{"global-uninstall", "ug"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return internal.UninstallGlobal()
	},
}
