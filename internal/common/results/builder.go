package results

import (
	"github.com/SiriusScan/app-agent/internal/modules"
)

// Build creates a module Result with the given parameters.
func Build(matched bool, evidence map[string]interface{}) *modules.Result {
	return &modules.Result{
		Matched:  matched,
		Evidence: evidence,
	}
}

// BuildWithError creates a Result indicating an error occurred.
func BuildWithError(err error) *modules.Result {
	return &modules.Result{
		Matched: false,
		Error:   err.Error(),
	}
}

// BuildSuccess creates a successful Result with matched=true.
func BuildSuccess(evidence map[string]interface{}) *modules.Result {
	return &modules.Result{
		Matched:  true,
		Evidence: evidence,
	}
}

// BuildFailure creates a non-matching Result with matched=false.
func BuildFailure(evidence map[string]interface{}) *modules.Result {
	return &modules.Result{
		Matched:  false,
		Evidence: evidence,
	}
}

// EvidenceBuilder helps construct evidence maps fluently.
type EvidenceBuilder struct {
	evidence map[string]interface{}
}

// NewEvidence creates a new EvidenceBuilder.
func NewEvidence() *EvidenceBuilder {
	return &EvidenceBuilder{
		evidence: make(map[string]interface{}),
	}
}

// Add adds a key-value pair to the evidence.
func (b *EvidenceBuilder) Add(key string, value interface{}) *EvidenceBuilder {
	b.evidence[key] = value
	return b
}

// AddString adds a string value to the evidence.
func (b *EvidenceBuilder) AddString(key, value string) *EvidenceBuilder {
	b.evidence[key] = value
	return b
}

// AddInt adds an integer value to the evidence.
func (b *EvidenceBuilder) AddInt(key string, value int) *EvidenceBuilder {
	b.evidence[key] = value
	return b
}

// AddBool adds a boolean value to the evidence.
func (b *EvidenceBuilder) AddBool(key string, value bool) *EvidenceBuilder {
	b.evidence[key] = value
	return b
}

// AddStringSlice adds a string slice to the evidence.
func (b *EvidenceBuilder) AddStringSlice(key string, value []string) *EvidenceBuilder {
	b.evidence[key] = value
	return b
}

// Build returns the evidence map.
func (b *EvidenceBuilder) Build() map[string]interface{} {
	return b.evidence
}

// BuildResult creates a Result with the evidence.
func (b *EvidenceBuilder) BuildResult(matched bool) *modules.Result {
	return &modules.Result{
		Matched:  matched,
		Evidence: b.evidence,
	}
}

// Helper functions for common evidence patterns

// FileEvidence creates evidence for file-based detection.
func FileEvidence(path string, extraFields map[string]interface{}) map[string]interface{} {
	evidence := map[string]interface{}{
		"file": path,
	}
	for k, v := range extraFields {
		evidence[k] = v
	}
	return evidence
}

// HashEvidence creates evidence for hash-based detection.
func HashEvidence(path, expectedHash, actualHash, algorithm string) map[string]interface{} {
	return map[string]interface{}{
		"file":          path,
		"expected_hash": expectedHash,
		"actual_hash":   actualHash,
		"algorithm":     algorithm,
	}
}

// PatternEvidence creates evidence for pattern-based detection.
func PatternEvidence(path, pattern, matchedText string, lineNumber int) map[string]interface{} {
	return map[string]interface{}{
		"file":         path,
		"pattern":      pattern,
		"matched_text": matchedText,
		"line":         lineNumber,
	}
}

// CommandEvidence creates evidence for command execution.
func CommandEvidence(command []string, stdout, stderr string, exitCode int) map[string]interface{} {
	return map[string]interface{}{
		"command":   command,
		"stdout":    stdout,
		"stderr":    stderr,
		"exit_code": exitCode,
	}
}

// VersionEvidence creates evidence for version detection.
func VersionEvidence(software, version, extractedFrom string) map[string]interface{} {
	return map[string]interface{}{
		"software":       software,
		"version":        version,
		"extracted_from": extractedFrom,
	}
}

