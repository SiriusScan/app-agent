package main

import (
	"fmt"
	"os"

	_ "github.com/SiriusScan/app-agent/internal/modules/filecontent" // Register modules
	_ "github.com/SiriusScan/app-agent/internal/modules/filehash"
	"github.com/SiriusScan/app-agent/internal/cmd"
)

var version = "1.0.0-mvp"

func main() {
	rootCmd := cmd.NewRootCommand(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

