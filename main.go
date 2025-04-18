package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/s42yt/node-spark/cmd"
	"github.com/s42yt/node-spark/internal"
)

func main() {
	// Check if this is first run for auto global installation
	if internal.IsFirstRun() && len(os.Args) == 1 {
		// If no command arguments are provided, and this is first run
		fmt.Println("╔════════════════════════════════════════════════════╗")
		fmt.Println("║ Welcome to Node Spark!                             ║")
		fmt.Println("║                                                    ║")
		fmt.Println("║ This appears to be your first time running the     ║")
		fmt.Println("║ application. Installing Node Spark globally...     ║")
		fmt.Println("╚════════════════════════════════════════════════════╝")

		// Install globally silently
		if err := internal.InstallGlobalSilently(); err != nil {
			fmt.Printf("Warning: Failed to install globally: %v\n", err)
			fmt.Println("You can run 'nsk install-global' manually later.")
		} else {
			fmt.Println("✅ Node Spark has been installed globally!")
			fmt.Println("You can now use 'nsk' from anywhere on your system.")
		}
		fmt.Println()
	}

	// Check for --wipe option separately to ensure it works without cobra
	for _, arg := range os.Args {
		if strings.ToLower(arg) == "--wipe" {
			fmt.Println("🧹 Wiping all Node Spark data...")
			if err := internal.WipeAllData(); err != nil {
				fmt.Printf("❌ Error wiping data: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ All Node Spark data has been wiped successfully.")
			os.Exit(0)
		}
	}

	// Continue with normal execution
	cmd.Execute()
}
