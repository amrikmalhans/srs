package ui

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// GetTerminalWidth returns the terminal width in characters.
// Falls back to 80 if terminal width cannot be determined.
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	if width < 40 {
		return 40 // Minimum reasonable width
	}
	return width
}

// WrapText wraps text at word boundaries to fit within the specified width.
// Preserves leading whitespace for indented lines.
func WrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	lines := strings.Split(text, "\n")
	var result strings.Builder

	for _, line := range lines {
		if len(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Preserve leading whitespace
		leadingWhitespace := getLeadingWhitespace(line)
		indentWidth := len(leadingWhitespace)
		content := strings.TrimLeft(line, " \t")
		effectiveWidth := width - indentWidth

		if effectiveWidth <= 0 {
			// Line is too indented, just output as-is
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Wrap the content
		wrapped := wrapLine(content, effectiveWidth)
		wrappedLines := strings.Split(wrapped, "\n")

		for i, wrappedLine := range wrappedLines {
			if i > 0 {
				result.WriteString(leadingWhitespace)
			}
			result.WriteString(wrappedLine)
			result.WriteString("\n")
		}
	}

	return strings.TrimRight(result.String(), "\n")
}

// wrapLine wraps a single line of text at word boundaries.
func wrapLine(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine) == 0 {
			currentLine = word
			continue
		}

		// Check if adding this word would exceed width
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			result.WriteString(currentLine)
			result.WriteString("\n")
			currentLine = word
		}
	}

	if currentLine != "" {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(currentLine)
	}

	return result.String()
}

// getLeadingWhitespace returns the leading whitespace (spaces and tabs) from a line.
func getLeadingWhitespace(line string) string {
	var result strings.Builder
	for _, r := range line {
		if r == ' ' || r == '\t' {
			result.WriteRune(r)
		} else {
			break
		}
	}
	return result.String()
}

// RenderCardContent renders card content with smart handling of code fences.
// Code fences (``` blocks) are preserved without wrapping.
func RenderCardContent(content string, width int) string {
	if width <= 0 {
		width = GetTerminalWidth()
	}

	lines := strings.Split(content, "\n")
	var result strings.Builder
	var codeFence strings.Builder
	inCodeFence := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code fence start/end
		if strings.HasPrefix(trimmed, "```") {
			if inCodeFence {
				// End of code fence - output accumulated code without wrapping
				codeFence.WriteString(line)
				result.WriteString(codeFence.String())
				result.WriteString("\n")
				codeFence.Reset()
				inCodeFence = false
			} else {
				// Start of code fence
				if codeFence.Len() > 0 {
					result.WriteString(codeFence.String())
					codeFence.Reset()
				}
				codeFence.WriteString(line)
				codeFence.WriteString("\n")
				inCodeFence = true
			}
			continue
		}

		if inCodeFence {
			// Accumulate code fence content without wrapping
			codeFence.WriteString(line)
			codeFence.WriteString("\n")
		} else {
			// Regular content - wrap it
			if result.Len() > 0 && !strings.HasSuffix(result.String(), "\n") {
				result.WriteString("\n")
			}
			wrapped := WrapText(line, width)
			result.WriteString(wrapped)
			// Only add newline if this isn't the last line or if wrapped added content
			if i < len(lines)-1 || strings.HasSuffix(wrapped, "\n") {
				if !strings.HasSuffix(result.String(), "\n") {
					result.WriteString("\n")
				}
			}
		}
	}

	// If we ended in a code fence, output it
	if inCodeFence && codeFence.Len() > 0 {
		result.WriteString(codeFence.String())
	}

	return strings.TrimRight(result.String(), "\n")
}
