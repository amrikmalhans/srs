package store

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/amrikmalhans/srs/internal/domain"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

// ParseCardFile parses a card from a file path
func ParseCardFile(path string) (*domain.Card, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return ParseCardContent(content)
}

// ParseCardContent parses a card from byte content
func ParseCardContent(content []byte) (*domain.Card, error) {
	// Split by frontmatter delimiter
	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid card format: missing frontmatter delimiters")
	}

	// Parse YAML frontmatter (parts[1] is the frontmatter content)
	var frontmatter struct {
		ID      string   `yaml:"id"`
		Tags    []string `yaml:"tags"`
		Created string   `yaml:"created"`
	}

	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Validate required fields
	if frontmatter.ID == "" {
		return nil, fmt.Errorf("missing required field: id")
	}

	id, err := uuid.Parse(frontmatter.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	if frontmatter.Created == "" {
		return nil, fmt.Errorf("missing required field: created")
	}

	created, err := time.Parse(time.RFC3339, frontmatter.Created)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format (expected RFC3339): %w", err)
	}

	// Parse markdown body (parts[2] is the content after second ---)
	body := strings.TrimSpace(parts[2])
	cardContent, err := parseQASections(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Q/A sections: %w", err)
	}

	card := &domain.Card{
		ID:      id,
		Created: created,
		Tags:    frontmatter.Tags,
		Content: cardContent,
	}

	return card, nil
}

// parseQASections extracts # Q and # A sections from markdown body
// Returns the combined content with Q and A sections
func parseQASections(body string) (string, error) {
	lines := strings.Split(body, "\n")

	var qStart, qEnd, aStart, aEnd int = -1, -1, -1, -1

	// Find # Q and # A headers (must be exact level 1 headers)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "# Q" {
			qStart = i
		} else if trimmed == "# A" {
			if qStart == -1 {
				return "", fmt.Errorf("found # A section before # Q section")
			}
			qEnd = i
			aStart = i
		}
	}

	if qStart == -1 {
		return "", fmt.Errorf("missing required section: # Q")
	}

	if aStart == -1 {
		return "", fmt.Errorf("missing required section: # A")
	}

	// Extract Q section content (from line after # Q to line before # A)
	if qEnd == -1 {
		qEnd = len(lines)
	}

	// Extract A section content (from line after # A to end)
	aEnd = len(lines)

	// Build content: Q section + A section
	var content strings.Builder

	// Add Q section
	content.WriteString("# Q\n")
	if qStart+1 < qEnd {
		qContent := strings.Join(lines[qStart+1:qEnd], "\n")
		content.WriteString(strings.TrimSpace(qContent))
		content.WriteString("\n")
	}

	// Add A section
	content.WriteString("\n# A\n")
	if aStart+1 < aEnd {
		aContent := strings.Join(lines[aStart+1:aEnd], "\n")
		content.WriteString(strings.TrimSpace(aContent))
		content.WriteString("\n")
	}

	return content.String(), nil
}

// ExtractQuestion extracts the Q section from card content
// Returns the question text without the "# Q" header
func ExtractQuestion(content string) string {
	lines := strings.Split(content, "\n")

	var qStart, qEnd int = -1, -1

	// Find # Q header
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "# Q" {
			qStart = i
		} else if trimmed == "# A" && qStart != -1 {
			qEnd = i
			break
		}
	}

	if qStart == -1 {
		return ""
	}

	if qEnd == -1 {
		qEnd = len(lines)
	}

	// Extract content between # Q and # A (or end)
	if qStart+1 < qEnd {
		qContent := strings.Join(lines[qStart+1:qEnd], "\n")
		return strings.TrimSpace(qContent)
	}

	return ""
}

// ExtractAnswer extracts the A section from card content
// Returns the answer text without the "# A" header
func ExtractAnswer(content string) string {
	lines := strings.Split(content, "\n")

	var aStart int = -1

	// Find # A header
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "# A" {
			aStart = i
			break
		}
	}

	if aStart == -1 {
		return ""
	}

	// Extract content from # A to end
	if aStart+1 < len(lines) {
		aContent := strings.Join(lines[aStart+1:], "\n")
		return strings.TrimSpace(aContent)
	}

	return ""
}
