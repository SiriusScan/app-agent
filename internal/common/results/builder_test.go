package results

import (
	"errors"
	"testing"
)

func TestBuild(t *testing.T) {
	evidence := map[string]interface{}{
		"file": "/test/file.txt",
		"hash": "abc123",
	}

	result := Build(true, evidence)

	if !result.Matched {
		t.Error("Expected matched=true")
	}

	if result.Evidence["file"] != "/test/file.txt" {
		t.Error("Evidence not preserved")
	}

	t.Log("✅ Build working")
}

func TestBuildWithError(t *testing.T) {
	err := errors.New("test error")
	result := BuildWithError(err)

	if result.Matched {
		t.Error("Expected matched=false for error result")
	}

	if result.Error != "test error" {
		t.Errorf("Expected error message %q, got %q", "test error", result.Error)
	}

	t.Log("✅ BuildWithError working")
}

func TestBuildSuccessFailure(t *testing.T) {
	// Test BuildSuccess
	successEvidence := map[string]interface{}{"key": "value"}
	success := BuildSuccess(successEvidence)

	if !success.Matched {
		t.Error("BuildSuccess should set matched=true")
	}

	// Test BuildFailure
	failureEvidence := map[string]interface{}{"reason": "not found"}
	failure := BuildFailure(failureEvidence)

	if failure.Matched {
		t.Error("BuildFailure should set matched=false")
	}

	t.Log("✅ BuildSuccess/BuildFailure working")
}

func TestEvidenceBuilder(t *testing.T) {
	evidence := NewEvidence().
		AddString("file", "/test/file.txt").
		AddInt("line", 42).
		AddBool("found", true).
		AddStringSlice("tags", []string{"tag1", "tag2"}).
		Build()

	if evidence["file"] != "/test/file.txt" {
		t.Error("String not added correctly")
	}

	if evidence["line"] != 42 {
		t.Error("Int not added correctly")
	}

	if evidence["found"] != true {
		t.Error("Bool not added correctly")
	}

	tags, ok := evidence["tags"].([]string)
	if !ok || len(tags) != 2 {
		t.Error("String slice not added correctly")
	}

	t.Log("✅ EvidenceBuilder working")
}

func TestEvidenceBuilderResult(t *testing.T) {
	result := NewEvidence().
		AddString("test", "value").
		BuildResult(true)

	if !result.Matched {
		t.Error("Expected matched=true")
	}

	if result.Evidence["test"] != "value" {
		t.Error("Evidence not built correctly")
	}

	t.Log("✅ EvidenceBuilder.BuildResult working")
}

func TestFileEvidence(t *testing.T) {
	evidence := FileEvidence("/test/file.txt", map[string]interface{}{
		"size": 1024,
		"hash": "abc123",
	})

	if evidence["file"] != "/test/file.txt" {
		t.Error("File path not set")
	}

	if evidence["size"] != 1024 {
		t.Error("Extra fields not preserved")
	}

	t.Log("✅ FileEvidence working")
}

func TestHashEvidence(t *testing.T) {
	evidence := HashEvidence(
		"/usr/bin/test",
		"expected123",
		"actual123",
		"sha256",
	)

	if evidence["file"] != "/usr/bin/test" {
		t.Error("File path not set")
	}

	if evidence["expected_hash"] != "expected123" {
		t.Error("Expected hash not set")
	}

	if evidence["actual_hash"] != "actual123" {
		t.Error("Actual hash not set")
	}

	if evidence["algorithm"] != "sha256" {
		t.Error("Algorithm not set")
	}

	t.Log("✅ HashEvidence working")
}

func TestPatternEvidence(t *testing.T) {
	evidence := PatternEvidence(
		"/etc/config",
		"debug=true",
		"debug=true",
		42,
	)

	if evidence["file"] != "/etc/config" {
		t.Error("File path not set")
	}

	if evidence["pattern"] != "debug=true" {
		t.Error("Pattern not set")
	}

	if evidence["matched_text"] != "debug=true" {
		t.Error("Matched text not set")
	}

	if evidence["line"] != 42 {
		t.Error("Line number not set")
	}

	t.Log("✅ PatternEvidence working")
}

func TestCommandEvidence(t *testing.T) {
	evidence := CommandEvidence(
		[]string{"ssh", "-V"},
		"OpenSSH_8.0",
		"",
		0,
	)

	cmd, ok := evidence["command"].([]string)
	if !ok || len(cmd) != 2 {
		t.Error("Command not set correctly")
	}

	if evidence["stdout"] != "OpenSSH_8.0" {
		t.Error("Stdout not set")
	}

	if evidence["exit_code"] != 0 {
		t.Error("Exit code not set")
	}

	t.Log("✅ CommandEvidence working")
}

func TestVersionEvidence(t *testing.T) {
	evidence := VersionEvidence(
		"nginx",
		"1.18.0",
		"nginx -v output",
	)

	if evidence["software"] != "nginx" {
		t.Error("Software not set")
	}

	if evidence["version"] != "1.18.0" {
		t.Error("Version not set")
	}

	if evidence["extracted_from"] != "nginx -v output" {
		t.Error("Extracted from not set")
	}

	t.Log("✅ VersionEvidence working")
}

