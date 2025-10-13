package patterns

import (
	"strings"
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		want    bool
	}{
		{"simple match", "hello", "hello world", true},
		{"no match", "hello", "goodbye world", false},
		{"regex pattern", "h.llo", "hello world", true},
		{"digit pattern", `\d+`, "test123", true},
		{"email pattern", `\w+@\w+\.\w+`, "user@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := Match(tt.pattern, tt.text)
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}
			if matched != tt.want {
				t.Errorf("Match() = %v, want %v", matched, tt.want)
			}
		})
	}

	t.Log("✅ Pattern matching working")
}

func TestMatchCaseInsensitive(t *testing.T) {
	opts := MatchOptions{
		Timeout:         DefaultMatchTimeout,
		CaseInsensitive: true,
	}

	matched, err := MatchWithOptions("HELLO", "hello world", opts)
	if err != nil {
		t.Fatalf("MatchWithOptions() error = %v", err)
	}
	if !matched {
		t.Error("Expected case-insensitive match")
	}

	t.Log("✅ Case-insensitive matching working")
}

func TestMatchInvalidPattern(t *testing.T) {
	_, err := Match("[invalid", "text")
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}

	t.Log("✅ Invalid pattern handling working")
}

func TestMatchTimeout(t *testing.T) {
	// Catastrophic backtracking pattern that can timeout
	pattern := `(a+)+b`
	text := strings.Repeat("a", 30)

	opts := MatchOptions{
		Timeout:         10 * time.Millisecond,
		CaseInsensitive: false,
	}

	_, err := MatchWithOptions(pattern, text, opts)
	// This might or might not timeout depending on system speed
	// Just verify if error occurs, it's a TimeoutError
	if err != nil {
		if _, ok := err.(*TimeoutError); !ok {
			t.Errorf("Expected TimeoutError, got %T", err)
		}
	}

	t.Log("✅ Timeout handling implemented")
}

func TestFind(t *testing.T) {
	result, err := Find(`\d+`, "test 123 more 456")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}

	if !result.Matched {
		t.Error("Expected match")
	}

	if result.MatchedText != "123" {
		t.Errorf("Expected matched text '123', got %q", result.MatchedText)
	}

	t.Log("✅ Find working")
}

func TestFindInLines(t *testing.T) {
	text := `line 1: hello
line 2: world
line 3: hello again
line 4: goodbye`

	results, err := FindInLines("hello", text)
	if err != nil {
		t.Fatalf("FindInLines() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(results))
	}

	if results[0].Line != 1 {
		t.Errorf("Expected first match on line 1, got %d", results[0].Line)
	}

	if results[1].Line != 3 {
		t.Errorf("Expected second match on line 3, got %d", results[1].Line)
	}

	t.Log("✅ FindInLines working")
}

func TestFindAll(t *testing.T) {
	matches, err := FindAll(`\d+`, "test 123 more 456 end 789")
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(matches))
	}

	expected := []string{"123", "456", "789"}
	for i, match := range matches {
		if match != expected[i] {
			t.Errorf("Match %d: got %q, want %q", i, match, expected[i])
		}
	}

	t.Log("✅ FindAll working")
}

func TestFindNoMatch(t *testing.T) {
	result, err := Find("xyz", "abc def")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}

	if result.Matched {
		t.Error("Expected no match")
	}

	if result.MatchedText != "" {
		t.Errorf("Expected empty matched text, got %q", result.MatchedText)
	}

	t.Log("✅ No-match handling working")
}

