package templatescan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestParseArgsExactTemplateID(t *testing.T) {
	command := &ScanCommand{}
	for _, args := range []string{"--template-id selected", "--template-id=selected", "--template-id=selected --workers=1 --timeout=10"} {
		config, err := command.parseArgs(args)
		if err != nil {
			t.Fatalf("%q: %v", args, err)
		}
		if config.TemplateID != "selected" || config.TemplatePath != "" || config.Directory != "" {
			t.Fatalf("%q selected unexpected scope: %+v", args, config)
		}
	}
	for _, args := range []string{
		"--template-id", "--template-id=", "--template-id --all",
		"--template-id=selected --all", "--all --template-id=selected",
		"--template-id=selected --template file.yaml", "--template-id=selected --directory templates",
		"--template-id=one --template-id=two", "--template-id one --template-id two",
	} {
		if config, err := command.parseArgs(args); err == nil {
			t.Fatalf("%q should fail instead of broadening scope: %+v", args, config)
		}
	}
}

func TestParseArgsPreservesLegacySelectors(t *testing.T) {
	command := &ScanCommand{}
	for _, args := range []string{"", "--all", "--template file.yaml", "--directory fixtures"} {
		config, err := command.parseArgs(args)
		if err != nil {
			t.Fatalf("%q: %v", args, err)
		}
		if config.TemplateID != "" {
			t.Fatalf("legacy selector became an ID: %+v", config)
		}
		if args == "--template file.yaml" && config.TemplatePath != "file.yaml" {
			t.Fatal("lost legacy file selector")
		}
	}
}

// These fixtures are only parsed. No detection module, command, or network call
// is executed, and all storage is confined to t.TempDir().
func TestLoadTemplateByIDUsesExactIDAndSourcePrecedence(t *testing.T) {
	base := t.TempDir()
	t.Setenv("SIRIUS_TEMPLATE_DIR", base)
	write := func(source, filename, id, name string) {
		t.Helper()
		dir := filepath.Join(base, source)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		content := fmt.Sprintf(`id: %s
info:
  name: %s
  severity: high
detection:
  logic: any
  steps:
    - type: file_content
      config:
        path: /fixture-not-executed
        regex: fixture
`, id, name)
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("server", "different-filename.yaml", "selected", "Server version")
	write("custom", "custom-name.yaml", "selected", "Custom version")
	write("custom", "unrelated.yaml", "selected-extra", "Must not run")

	template, err := loadTemplateByID(context.Background(), zap.NewNop(), "selected")
	if err != nil {
		t.Fatal(err)
	}
	if template.ID != "selected" || template.Info.Name != "Custom version" {
		t.Fatalf("wrong template or precedence: %+v", template)
	}
	if template.FilePath != filepath.Join(base, "custom", "custom-name.yaml") {
		t.Fatalf("ID was treated as a filename: %s", template.FilePath)
	}
	for _, id := range []string{"missing", "select", "*", "selected,selected-extra"} {
		template, err := loadTemplateByID(context.Background(), zap.NewNop(), id)
		if err == nil || template != nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("ID %q must fail closed, got template=%v error=%v", id, template, err)
		}
	}
}
