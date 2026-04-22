package main

import (
	"fmt"
	"os"

	"github.com/SiriusScan/app-agent/internal/cmd"
	siriusbootstrap "github.com/SiriusScan/app-agent/internal/family/sirius/bootstrap"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	siriusbootstrap.LoadCompatibilityRuntime()
	rootCmd := cmd.NewRootCommand(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
