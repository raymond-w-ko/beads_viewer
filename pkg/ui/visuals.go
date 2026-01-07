package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)



// RenderSparkline creates a textual bar chart of value (0.0 - 1.0)
func RenderSparkline(val float64, width int) string {
	if width <= 0 {
		return ""
	}

	chars := []string{" ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	
	if math.IsNaN(val) {
		val = 0
	}
	if val < 0 {
		val = 0
	}
	if val > 1 {
		val = 1
	}

	// Calculate fullness
	fullChars := int(val * float64(width))
	remainder := (val * float64(width)) - float64(fullChars)

	var sb strings.Builder
	for i := 0; i < fullChars; i++ {
		sb.WriteString("█")
	}

	if fullChars < width {
		idx := int(remainder * float64(len(chars)))
		// Ensure non-zero values are visible
		if idx == 0 && remainder > 0 {
			idx = 1
		}
		if idx >= len(chars) {
			idx = len(chars) - 1
		}
		if idx > 0 {
			sb.WriteString(chars[idx])
		} else {
			sb.WriteString(" ")
		}
	}

	// Pad
	padding := width - fullChars - 1
	if padding > 0 {
		sb.WriteString(strings.Repeat(" ", padding))
	}

	return sb.String()
}

// GetHeatmapColor returns a color based on score (0-1)
func GetHeatmapColor(score float64, t Theme) lipgloss.TerminalColor {
	if score > 0.8 {
		return t.Primary // Peak/High
	} else if score > 0.5 {
		return t.Feature // Mid-High
	} else if score > 0.2 {
		return t.InProgress // Low-Mid
	}
	return t.Secondary // Low
}

// HeatmapGradientColors defines the color gradient for enhanced heatmap (bv-t4yg)
// Ordered from cold (low count) to hot (high count) - adaptive for light/dark mode
var HeatmapGradientColors = []lipgloss.AdaptiveColor{
	{Light: "#F5F5F5", Dark: "#1a1a2e"}, // 0: empty (light gray / dark blue)
	{Light: "#E3F2FD", Dark: "#16213e"}, // 1: very few (light blue / navy)
	{Light: "#BBDEFB", Dark: "#0f4c75"}, // 2: few (lighter blue / blue)
	{Light: "#90CAF9", Dark: "#3282b8"}, // 3: some (medium blue / light blue)
	{Light: "#FFF9C4", Dark: "#bbe1fa"}, // 4: moderate (light yellow / pale blue)
	{Light: "#FFE082", Dark: "#f7dc6f"}, // 5: above average (amber / gold)
	{Light: "#FFAB91", Dark: "#e94560"}, // 6: many (light orange / coral)
	{Light: "#EF5350", Dark: "#ff2e63"}, // 7: hot (red / hot pink)
}

// GetHeatGradientColor returns an interpolated color for heatmap intensity (0-1) (bv-t4yg)
func GetHeatGradientColor(intensity float64, t Theme) lipgloss.AdaptiveColor {
	if intensity <= 0 {
		return HeatmapGradientColors[0]
	}
	if intensity >= 1 {
		return HeatmapGradientColors[len(HeatmapGradientColors)-1]
	}

	// Map intensity to gradient index
	idx := int(intensity * float64(len(HeatmapGradientColors)-1))
	if idx >= len(HeatmapGradientColors)-1 {
		idx = len(HeatmapGradientColors) - 2
	}

	return HeatmapGradientColors[idx+1] // +1 because 0 is for empty cells
}

// GetHeatGradientColorBg returns a background-friendly color for heatmap cell (bv-t4yg)
// Returns both the background color and appropriate foreground for contrast
// Now adaptive for light/dark mode
func GetHeatGradientColorBg(intensity float64) (bg lipgloss.AdaptiveColor, fg lipgloss.AdaptiveColor) {
	if intensity <= 0 {
		// Empty cell: subtle background
		return lipgloss.AdaptiveColor{Light: "#F5F5F5", Dark: "#1a1a2e"},
			lipgloss.AdaptiveColor{Light: "#888888", Dark: "#6272a4"}
	}

	// Select background color based on intensity
	switch {
	case intensity >= 0.8:
		// Hot: red/pink with contrasting text
		return lipgloss.AdaptiveColor{Light: "#EF5350", Dark: "#ff2e63"},
			lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#ffffff"}
	case intensity >= 0.6:
		// Warm: coral/orange
		return lipgloss.AdaptiveColor{Light: "#FFAB91", Dark: "#e94560"},
			lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
	case intensity >= 0.4:
		// Medium: gold/amber
		return lipgloss.AdaptiveColor{Light: "#FFE082", Dark: "#f7dc6f"},
			lipgloss.AdaptiveColor{Light: "#000000", Dark: "#1a1a2e"}
	case intensity >= 0.2:
		// Cool: blue
		return lipgloss.AdaptiveColor{Light: "#90CAF9", Dark: "#3282b8"},
			lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
	default:
		// Cold: light blue/navy
		return lipgloss.AdaptiveColor{Light: "#E3F2FD", Dark: "#16213e"},
			lipgloss.AdaptiveColor{Light: "#555555", Dark: "#bbe1fa"}
	}
}

// RepoColors maps repo prefixes to distinctive colors for visual differentiation
// These colors are designed to be visible on both light and dark backgrounds
var RepoColors = []lipgloss.AdaptiveColor{
	{Light: "#CC5555", Dark: "#FF6B6B"}, // Coral red
	{Light: "#3BA89E", Dark: "#4ECDC4"}, // Teal
	{Light: "#3891A6", Dark: "#45B7D1"}, // Sky blue
	{Light: "#6B9E87", Dark: "#96CEB4"}, // Sage green
	{Light: "#AA7AAA", Dark: "#DDA0DD"}, // Plum
	{Light: "#C4A93D", Dark: "#F7DC6F"}, // Gold
	{Light: "#9370A8", Dark: "#BB8FCE"}, // Lavender
	{Light: "#5A9BC2", Dark: "#85C1E9"}, // Light blue
}

// GetRepoColor returns a consistent color for a repo prefix based on hash
func GetRepoColor(prefix string) lipgloss.AdaptiveColor {
	if prefix == "" {
		// Return a neutral muted color for empty prefix
		return lipgloss.Color("#888888")
	}
	// Simple hash based on prefix characters
	hash := 0
	for _, c := range prefix {
		hash = (hash*31 + int(c)) % len(RepoColors)
	}
	if hash < 0 {
		hash = -hash
	}
	return RepoColors[hash%len(RepoColors)]
}

// RenderRepoBadge creates a compact colored badge for a repository prefix
// Example: "api" -> "[API]" with distinctive color
func RenderRepoBadge(prefix string) string {
	if prefix == "" {
		return ""
	}
	// Uppercase and limit to 4 chars for compactness
	display := strings.ToUpper(prefix)
	if len(display) > 4 {
		display = display[:4]
	}

	color := GetRepoColor(prefix)
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render("[" + display + "]")
}
