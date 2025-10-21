package main

import (
	"fmt"
	"os"

	"github.com/SiriusScan/app-agent/internal/cmd"
	_ "github.com/SiriusScan/app-agent/internal/commands/help" // Register help command
	_ "github.com/SiriusScan/app-agent/internal/commands/repo" // Register repo command
	_ "github.com/SiriusScan/app-agent/internal/commands/scan" // Register scan command
	_ "github.com/SiriusScan/app-agent/internal/commands/status" // Register status command
	_ "github.com/SiriusScan/app-agent/internal/commands/sync" // Register sync command
	_ "github.com/SiriusScan/app-agent/internal/commands/template" // Register template commands
	_ "github.com/SiriusScan/app-agent/internal/commands/templatescan" // Register template scan command
	_ "github.com/SiriusScan/app-agent/internal/modules/filecontent" // Register modules
	_ "github.com/SiriusScan/app-agent/internal/modules/filehash"
	_ "github.com/SiriusScan/app-agent/internal/modules/versioncmd"
)

var version = "1.0.0-mvp"

func main() {
	rootCmd := cmd.NewRootCommand(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
