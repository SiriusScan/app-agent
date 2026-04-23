package server

import (
	"encoding/json"
	"testing"
)

func TestIsTemplateNotifyCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"internal:template upload", true},
		{"internal:template delete", true},
		{"internal:template-scan --template foo", false},
		{"scan:start", false},
		{"", false},
		{"INTERNAL:template upload", false}, // case-sensitive on purpose; legacy producers always lowercase
	}
	for _, tc := range cases {
		if got := isTemplateNotifyCommand(tc.cmd); got != tc.want {
			t.Errorf("isTemplateNotifyCommand(%q) = %v, want %v", tc.cmd, got, tc.want)
		}
	}
}

func TestEngineCommandMessage_Unmarshal(t *testing.T) {
	const payload = `{"command":"internal:template upload","template_id":"smoke-test","timestamp":"2026-04-22T00:00:00Z"}`
	var msg EngineCommandMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Command != "internal:template upload" {
		t.Errorf("Command = %q", msg.Command)
	}
	if msg.TemplateID != "smoke-test" {
		t.Errorf("TemplateID = %q", msg.TemplateID)
	}
	if !isTemplateNotifyCommand(msg.Command) {
		t.Error("expected isTemplateNotifyCommand true for parsed upload command")
	}
}
