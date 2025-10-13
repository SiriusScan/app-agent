package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/SiriusScan/app-agent/internal/modules/filehash" // Register module
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/parser"
)

func main() {
	fmt.Println("🔍 FileHash Module Integration Test")
	fmt.Println("=" + "==================================")

	// Step 1: Parse the template
	fmt.Println("\n📄 Step 1: Parsing template...")
	templatePath := "testing/test-templates/01-file-hash.yaml"
	template, err := parser.ParseTemplate(templatePath)
	if err != nil {
		fmt.Printf("❌ Failed to parse template: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Parsed template: %s\n", template.ID)
	fmt.Printf("  Name: %s\n", template.Info.Name)
	fmt.Printf("  Severity: %s\n", template.Info.Severity)

	// Step 2: Validate the template
	fmt.Println("\n✅ Step 2: Validating template...")
	if err := parser.ValidateTemplate(template); err != nil {
		fmt.Printf("❌ Template validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Template is valid")

	// Step 3: Get the module from registry
	fmt.Println("\n🔌 Step 3: Getting module from registry...")
	module := registry.Get("file_hash")
	if module == nil {
		fmt.Println("❌ FileHash module not found in registry")
		os.Exit(1)
	}
	fmt.Println("✓ FileHash module found")

	// Step 4: Execute the detection step
	fmt.Println("\n🚀 Step 4: Executing detection...")
	if len(template.Detection.Steps) == 0 {
		fmt.Println("❌ No detection steps in template")
		os.Exit(1)
	}

	step := template.Detection.Steps[0]
	fmt.Printf("  Module type: %s\n", step.Type)
	fmt.Printf("  Config: %v\n", step.Config)

	ctx := context.Background()
	result, err := module.Execute(ctx, step.Config)
	if err != nil {
		fmt.Printf("❌ Module execution failed: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Display results
	fmt.Println("\n📊 Step 5: Results")
	fmt.Println("-" + "-----------------")

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(resultJSON))

	// Check expected outcome
	fmt.Println("\n🎯 Step 6: Verification")
	if result.Matched {
		fmt.Println("✅ MATCHED: Vulnerable file detected!")
	} else {
		fmt.Println("❌ NOT MATCHED: File is safe")
	}

	if result.Error != "" {
		fmt.Printf("⚠️  Error: %s\n", result.Error)
	}

	// Verify evidence
	if evidence := result.Evidence; evidence != nil {
		fmt.Println("\n📋 Evidence:")
		fmt.Printf("  Path: %v\n", evidence["path"])
		fmt.Printf("  Algorithm: %v\n", evidence["algorithm"])
		fmt.Printf("  Expected Hash: %v\n", evidence["expected_hash"])
		fmt.Printf("  Actual Hash: %v\n", evidence["actual_hash"])
		fmt.Printf("  Matched: %v\n", evidence["matched"])
	}

	// Exit with success
	fmt.Println("\n✅ Integration test completed successfully!")
}

