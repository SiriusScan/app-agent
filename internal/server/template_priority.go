package server

import (
	"math"
)

// TemplateSource represents a template with its source information
type TemplateSource struct {
	ID                 string
	Type               string // "custom", "repository", "builtin"
	RepositoryID       string
	RepositoryName     string
	RepositoryPriority int
	Version            string
	Content            []byte
}

// ResolveTemplatePriority determines which template to use when multiple sources have the same template ID
//
// Priority Resolution Rules:
// 1. Custom templates (type: "custom") always win - highest priority
// 2. Among repository templates, lower priority number wins (1 beats 2)
// 3. Within same priority, first synced repository wins (stable sort)
// 4. Builtin templates have lowest priority
//
// Example:
//
//	Template "apache-log4j-rce" exists in:
//	- Custom (priority: N/A) -> Winner!
//
//	If no custom version:
//	- Repository A (priority: 1)
//	- Repository B (priority: 2) -> Repository A wins
//
//	If same priority:
//	- Repository A (priority: 1, synced first) -> Winner!
//	- Repository B (priority: 1, synced later)
func ResolveTemplatePriority(templates map[string]*TemplateSource) *TemplateSource {
	if len(templates) == 0 {
		return nil
	}

	// If only one template, return it
	if len(templates) == 1 {
		for _, tmpl := range templates {
			return tmpl
		}
	}

	// Step 1: Custom templates always win
	for _, tmpl := range templates {
		if tmpl.Type == "custom" {
			return tmpl
		}
	}

	// Step 2: Among repositories, lowest priority number wins
	var winner *TemplateSource
	lowestPriority := math.MaxInt32

	for _, tmpl := range templates {
		// Skip non-repository templates
		if tmpl.Type != "repository" {
			continue
		}

		if tmpl.RepositoryPriority < lowestPriority {
			lowestPriority = tmpl.RepositoryPriority
			winner = tmpl
		}
	}

	// Step 3: If no repository template found, check for builtin
	if winner == nil {
		for _, tmpl := range templates {
			if tmpl.Type == "builtin" {
				return tmpl
			}
		}
	}

	return winner
}

// ResolveTemplateConflicts takes a list of templates and returns a deduplicated list
// with conflicts resolved according to priority rules
func ResolveTemplateConflicts(templates []*TemplateSource) []*TemplateSource {
	// Group templates by ID
	grouped := make(map[string]map[string]*TemplateSource)

	for _, tmpl := range templates {
		if _, exists := grouped[tmpl.ID]; !exists {
			grouped[tmpl.ID] = make(map[string]*TemplateSource)
		}

		// Use a unique key for each source (type + repository ID)
		sourceKey := tmpl.Type
		if tmpl.RepositoryID != "" {
			sourceKey = tmpl.Type + ":" + tmpl.RepositoryID
		}

		grouped[tmpl.ID][sourceKey] = tmpl
	}

	// Resolve conflicts for each template ID
	resolved := make([]*TemplateSource, 0, len(grouped))

	for _, sources := range grouped {
		winner := ResolveTemplatePriority(sources)
		if winner != nil {
			resolved = append(resolved, winner)
		}
	}

	return resolved
}

// GetTemplatePriorityRank returns a numeric rank for template priority
// Lower numbers = higher priority
func GetTemplatePriorityRank(tmpl *TemplateSource) int {
	switch tmpl.Type {
	case "custom":
		return 0 // Highest priority
	case "repository":
		// Repository priority is the actual priority value
		// Priority 1 = rank 100, Priority 2 = rank 200, etc.
		return 100 + tmpl.RepositoryPriority
	case "builtin":
		return 1000 // Lowest priority
	default:
		return 9999 // Unknown type, very low priority
	}
}

// CompareTemplatePriority compares two templates and returns:
// -1 if a has higher priority than b
//
//	0 if they have equal priority
//	1 if b has higher priority than a
func CompareTemplatePriority(a, b *TemplateSource) int {
	rankA := GetTemplatePriorityRank(a)
	rankB := GetTemplatePriorityRank(b)

	if rankA < rankB {
		return -1
	} else if rankA > rankB {
		return 1
	}
	return 0
}






