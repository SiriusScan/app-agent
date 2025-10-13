package main

import (
	"fmt"

	"github.com/SiriusScan/app-agent/internal/template/parser"
)

func main() {
	fmt.Println("🔍 Template Discovery Test")
	fmt.Println("=" + "=========================")

	// Discover all templates in the test directory
	templates, errors := parser.DiscoverTemplates("testing/test-templates")

	fmt.Printf("\n📊 Discovery Results:\n")
	fmt.Printf("  Valid templates: %d\n", len(templates))
	fmt.Printf("  Errors: %d\n", len(errors))

	// Display valid templates
	if len(templates) > 0 {
		fmt.Println("\n✅ Valid Templates:")
		for i, tmpl := range templates {
			fmt.Printf("  %d. %s - %s (Severity: %s)\n", i+1, tmpl.ID, tmpl.Info.Name, tmpl.Info.Severity)
			fmt.Printf("     Steps: %d\n", len(tmpl.Detection.Steps))
		}
	}

	// Display errors if any
	if len(errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for i, err := range errors {
			fmt.Printf("  %d. %v\n", i+1, err)
		}
	}

	fmt.Println("\n✅ Discovery test completed!")
}

