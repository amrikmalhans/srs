// Package ui provides terminal rendering utilities for the SRS tool.
// It handles text wrapping, terminal width detection, and smart rendering
// of card content including code fences that should not be wrapped.
package ui

import (
	"os"
	"strings"

	"github.com/fatih/color"
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

// ColorCardHeader returns a colorized string for card headers.
// Uses subtle cyan/blue for visual separation.
func ColorCardHeader(text string) string {
	return color.New(color.FgHiCyan).Sprint(text)
}

// ColorQuestion returns a colorized string for question text.
// Uses subtle yellow for questions.
func ColorQuestion(text string) string {
	return color.New(color.FgHiYellow).Sprint(text)
}

// ColorAnswer returns a colorized string for answer text.
// Uses subtle green for answers.
func ColorAnswer(text string) string {
	return color.New(color.FgHiGreen).Sprint(text)
}

// ColorPrompt returns a colorized string for prompts and instructions.
// Uses subtle gray for prompts.
func ColorPrompt(text string) string {
	return color.New(color.FgHiBlack).Sprint(text)
}

// ColorGrade returns a colorized string for grading options.
// Uses different colors: 1=Again (red), 2=Hard (yellow), 3=Good (green), 4=Easy (cyan).
func ColorGrade(gradeNum int, text string) string {
	var c *color.Color
	switch gradeNum {
	case 1: // Again
		c = color.New(color.FgHiRed)
	case 2: // Hard
		c = color.New(color.FgHiYellow)
	case 3: // Good
		c = color.New(color.FgHiGreen)
	case 4: // Easy
		c = color.New(color.FgHiCyan)
	default:
		return text
	}
	return c.Sprint(text)
}

// ColorTag returns a colorized string for tags.
// Uses subtle blue/cyan for tags.
func ColorTag(text string) string {
	return color.New(color.FgHiBlue).Sprint(text)
}

// ColorID returns a colorized string for card IDs.
// Uses subtle gray for IDs.
func ColorID(text string) string {
	return color.New(color.FgHiBlack).Sprint(text)
}

// ColorMatch returns a colorized string for search matches.
// Uses subtle magenta for highlights.
func ColorMatch(text string) string {
	return color.New(color.FgHiMagenta).Sprint(text)
}

// ColorSummary returns a colorized string for summary headers.
// Uses subtle bold cyan for summary headers.
func ColorSummary(text string) string {
	return color.New(color.FgHiCyan, color.Bold).Sprint(text)
}

// ColorMatchField returns a colorized string for match field indicators (Q, A, tag).
// Uses different colors: Q (yellow), A (green), tag (blue).
func ColorMatchField(field string) string {
	var c *color.Color
	switch field {
	case "Q":
		c = color.New(color.FgHiYellow)
	case "A":
		c = color.New(color.FgHiGreen)
	case "tag":
		c = color.New(color.FgHiBlue)
	default:
		return field
	}
	return c.Sprint(field)
}
