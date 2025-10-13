package script

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SiriusScan/app-agent/internal/detect"
)

// ScriptMetadata represents metadata extracted from script files
type ScriptMetadata struct {
	Name               string                   `json:"name"`
	Description        string                   `json:"description"`
	Author             string                   `json:"author"`
	Version            string                   `json:"version"`
	VulnerabilityID    string                   `json:"vulnerability_id"`
	Severity           detect.SeverityLevel     `json:"severity"`
	Timeout            time.Duration            `json:"timeout"`
	RequiredPrivileges []string                 `json:"required_privileges"`
	Parameters         []detect.ScriptParameter `json:"parameters"`
	Tags               []string                 `json:"tags"`
}

// extractScriptMetadata parses script content to extract metadata from comments
func (se *ScriptExecutor) extractScriptMetadata(content []byte, language detect.ScriptLanguage) (*ScriptMetadata, error) {
	metadata := &ScriptMetadata{
		Severity:           detect.SeverityLevelMedium, // Default severity
		RequiredPrivileges: []string{},
		Parameters:         []detect.ScriptParameter{},
		Tags:               []string{},
	}

	// Parse metadata based on script language
	switch language {
	case detect.ScriptLanguagePowerShell:
		return se.extractPowerShellMetadata(content, metadata)
	case detect.ScriptLanguageBash:
		return se.extractBashMetadata(content, metadata)
	case detect.ScriptLanguagePython:
		return se.extractPythonMetadata(content, metadata)
	default:
		return metadata, fmt.Errorf("unsupported script language for metadata extraction: %s", language)
	}
}

// extractPowerShellMetadata extracts metadata from PowerShell comment-based help
func (se *ScriptExecutor) extractPowerShellMetadata(content []byte, metadata *ScriptMetadata) (*ScriptMetadata, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inCommentBlock := false

	// PowerShell comment-based help patterns
	patterns := map[string]*regexp.Regexp{
		"synopsis":      regexp.MustCompile(`^\s*\.SYNOPSIS\s*$`),
		"description":   regexp.MustCompile(`^\s*\.DESCRIPTION\s*$`),
		"author":        regexp.MustCompile(`^\s*\.AUTHOR\s*$`),
		"version":       regexp.MustCompile(`^\s*\.VERSION\s*$`),
		"vulnerability": regexp.MustCompile(`^\s*\.VULNERABILITY\s*$`),
		"severity":      regexp.MustCompile(`^\s*\.SEVERITY\s*$`),
		"timeout":       regexp.MustCompile(`^\s*\.TIMEOUT\s*$`),
		"parameter":     regexp.MustCompile(`^\s*\.PARAMETER\s+(\w+)\s*$`),
	}

	currentSection := ""
	var contentLines []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check for comment block start
		if strings.HasPrefix(trimmedLine, "<#") {
			inCommentBlock = true
			continue
		}

		// Check for comment block end
		if strings.HasPrefix(trimmedLine, "#>") {
			inCommentBlock = false
			break
		}

		// Only process lines inside comment blocks
		if !inCommentBlock {
			continue
		}

		// Check if line starts a new section
		foundSection := ""
		for section, pattern := range patterns {
			if pattern.MatchString(trimmedLine) {
				foundSection = section
				break
			}
		}

		if foundSection != "" {
			// Process previous section content
			if currentSection != "" && len(contentLines) > 0 {
				se.processPowerShellSection(metadata, currentSection, contentLines)
			}

			currentSection = foundSection
			contentLines = []string{}
		} else if currentSection != "" {
			// Add content to current section
			if trimmedLine != "" {
				contentLines = append(contentLines, trimmedLine)
			}
		}
	}

	// Process final section
	if currentSection != "" && len(contentLines) > 0 {
		se.processPowerShellSection(metadata, currentSection, contentLines)
	}

	// Set default name if not provided
	if metadata.Name == "" {
		metadata.Name = "PowerShell Detection Script"
	}

	return metadata, nil
}

// processPowerShellSection processes content for a specific PowerShell help section
func (se *ScriptExecutor) processPowerShellSection(metadata *ScriptMetadata, section string, lines []string) {
	content := strings.Join(lines, " ")
	content = strings.TrimSpace(content)

	switch section {
	case "synopsis":
		metadata.Name = content
	case "description":
		metadata.Description = content
	case "author":
		metadata.Author = content
	case "version":
		metadata.Version = content
	case "vulnerability":
		metadata.VulnerabilityID = content
	case "severity":
		metadata.Severity = se.parseSeverity(content)
	case "timeout":
		if timeout, err := se.parseTimeout(content); err == nil {
			metadata.Timeout = timeout
		}
	case "parameter":
		// Parameter parsing would be more complex, implementing basic version
		paramName := strings.Fields(content)[0]
		param := detect.ScriptParameter{
			Name:        paramName,
			Type:        "string",
			Required:    false,
			Description: content,
		}
		metadata.Parameters = append(metadata.Parameters, param)
	}
}

// extractBashMetadata extracts metadata from Bash script comments
func (se *ScriptExecutor) extractBashMetadata(content []byte, metadata *ScriptMetadata) (*ScriptMetadata, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Bash metadata patterns (using comments)
	patterns := map[string]*regexp.Regexp{
		"name":          regexp.MustCompile(`^#\s*Name:\s*(.+)$`),
		"description":   regexp.MustCompile(`^#\s*Description:\s*(.+)$`),
		"author":        regexp.MustCompile(`^#\s*Author:\s*(.+)$`),
		"version":       regexp.MustCompile(`^#\s*Version:\s*(.+)$`),
		"vulnerability": regexp.MustCompile(`^#\s*Vulnerability:\s*(.+)$`),
		"severity":      regexp.MustCompile(`^#\s*Severity:\s*(.+)$`),
		"timeout":       regexp.MustCompile(`^#\s*Timeout:\s*(.+)$`),
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Stop processing after script content starts (first non-comment, non-empty line)
		if !strings.HasPrefix(strings.TrimSpace(line), "#") && strings.TrimSpace(line) != "" {
			break
		}

		for section, pattern := range patterns {
			if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
				value := strings.TrimSpace(matches[1])

				switch section {
				case "name":
					metadata.Name = value
				case "description":
					metadata.Description = value
				case "author":
					metadata.Author = value
				case "version":
					metadata.Version = value
				case "vulnerability":
					metadata.VulnerabilityID = value
				case "severity":
					metadata.Severity = se.parseSeverity(value)
				case "timeout":
					if timeout, err := se.parseTimeout(value); err == nil {
						metadata.Timeout = timeout
					}
				}
			}
		}
	}

	// Set default name if not provided
	if metadata.Name == "" {
		metadata.Name = "Bash Detection Script"
	}

	return metadata, nil
}

// extractPythonMetadata extracts metadata from Python docstrings
func (se *ScriptExecutor) extractPythonMetadata(content []byte, metadata *ScriptMetadata) (*ScriptMetadata, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inDocstring := false
	docstringDelimiter := ""
	var docstringLines []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check for docstring start
		if !inDocstring {
			if strings.HasPrefix(trimmedLine, `"""`) || strings.HasPrefix(trimmedLine, `'''`) {
				inDocstring = true
				docstringDelimiter = trimmedLine[:3]

				// Check if docstring ends on same line
				if len(trimmedLine) > 3 && strings.HasSuffix(trimmedLine, docstringDelimiter) {
					docstringContent := trimmedLine[3 : len(trimmedLine)-3]
					if docstringContent != "" {
						docstringLines = append(docstringLines, docstringContent)
					}
					inDocstring = false
					break
				}
			}
			continue
		}

		// Check for docstring end
		if strings.HasSuffix(trimmedLine, docstringDelimiter) {
			if len(trimmedLine) > 3 {
				docstringContent := trimmedLine[:len(trimmedLine)-3]
				if docstringContent != "" {
					docstringLines = append(docstringLines, docstringContent)
				}
			}
			break
		}

		// Add line to docstring
		docstringLines = append(docstringLines, trimmedLine)
	}

	// Parse docstring content for metadata
	if len(docstringLines) > 0 {
		se.parsePythonDocstring(metadata, docstringLines)
	}

	// Set default name if not provided
	if metadata.Name == "" {
		metadata.Name = "Python Detection Script"
	}

	return metadata, nil
}

// parsePythonDocstring parses Python docstring content for metadata
func (se *ScriptExecutor) parsePythonDocstring(metadata *ScriptMetadata, lines []string) {
	// Simple parsing - first line is name/synopsis, rest is description
	if len(lines) > 0 {
		metadata.Name = strings.TrimSpace(lines[0])
	}

	if len(lines) > 1 {
		description := strings.Join(lines[1:], " ")
		metadata.Description = strings.TrimSpace(description)
	}

	// Look for structured metadata in description
	fullText := strings.Join(lines, "\n")

	// Extract vulnerability ID
	if vulnMatch := regexp.MustCompile(`(?i)vulnerability[:\s]+([A-Z]{3,4}-\d{4}-\d+)`).FindStringSubmatch(fullText); len(vulnMatch) > 1 {
		metadata.VulnerabilityID = vulnMatch[1]
	}

	// Extract severity
	if severityMatch := regexp.MustCompile(`(?i)severity[:\s]+(critical|high|medium|low|info)`).FindStringSubmatch(fullText); len(severityMatch) > 1 {
		metadata.Severity = se.parseSeverity(severityMatch[1])
	}

	// Extract author
	if authorMatch := regexp.MustCompile(`(?i)author[:\s]+(.+)`).FindStringSubmatch(fullText); len(authorMatch) > 1 {
		metadata.Author = strings.TrimSpace(authorMatch[1])
	}

	// Extract version
	if versionMatch := regexp.MustCompile(`(?i)version[:\s]+([0-9.]+)`).FindStringSubmatch(fullText); len(versionMatch) > 1 {
		metadata.Version = versionMatch[1]
	}
}

// parseSeverity converts string severity to SeverityLevel
func (se *ScriptExecutor) parseSeverity(severity string) detect.SeverityLevel {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return detect.SeverityLevelCritical
	case "high":
		return detect.SeverityLevelHigh
	case "medium":
		return detect.SeverityLevelMedium
	case "low":
		return detect.SeverityLevelLow
	case "info", "informational":
		return detect.SeverityLevelInfo
	default:
		return detect.SeverityLevelMedium
	}
}

// parseTimeout converts string timeout to time.Duration
func (se *ScriptExecutor) parseTimeout(timeout string) (time.Duration, error) {
	timeout = strings.TrimSpace(strings.ToLower(timeout))

	// Handle simple cases
	if timeout == "" {
		return 0, fmt.Errorf("empty timeout")
	}

	// Try parsing as duration string (e.g., "30s", "5m")
	if duration, err := time.ParseDuration(timeout); err == nil {
		return duration, nil
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(timeout); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}

	// Handle special formats
	if strings.HasSuffix(timeout, "min") || strings.HasSuffix(timeout, "minutes") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(timeout, "minutes"), "min")
		numStr = strings.TrimSpace(numStr)
		if minutes, err := strconv.Atoi(numStr); err == nil {
			return time.Duration(minutes) * time.Minute, nil
		}
	}

	if strings.HasSuffix(timeout, "sec") || strings.HasSuffix(timeout, "seconds") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(timeout, "seconds"), "sec")
		numStr = strings.TrimSpace(numStr)
		if seconds, err := strconv.Atoi(numStr); err == nil {
			return time.Duration(seconds) * time.Second, nil
		}
	}

	return 0, fmt.Errorf("unable to parse timeout: %s", timeout)
}
