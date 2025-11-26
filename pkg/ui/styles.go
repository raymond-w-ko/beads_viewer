package ui

import "github.com/charmbracelet/lipgloss"

// Dracula-inspired color palette
var (
	// Core Colors
	ColorPrimary     = lipgloss.Color("#BD93F9") // Purple
	ColorSecondary   = lipgloss.Color("#6272A4") // Blue-Gray
	ColorBg          = lipgloss.Color("#282A36") // Background
	ColorBgDark      = lipgloss.Color("#1E1F29") // Darker Background
	ColorBgHighlight = lipgloss.Color("#44475A") // Selection
	ColorText        = lipgloss.Color("#F8F8F2") // Foreground
	ColorSubtext     = lipgloss.Color("#BFBFBF") // Dimmer text

	// Status Colors
	ColorStatusOpen       = lipgloss.Color("#50FA7B") // Green
	ColorStatusInProgress = lipgloss.Color("#8BE9FD") // Cyan
	ColorStatusBlocked    = lipgloss.Color("#FF5555") // Red
	ColorStatusClosed     = lipgloss.Color("#6272A4") // Gray/Dim

	// Type Colors
	ColorTypeBug     = lipgloss.Color("#FF5555") // Red
	ColorTypeFeature = lipgloss.Color("#FFB86C") // Orange
	ColorTypeEpic    = lipgloss.Color("#BD93F9") // Purple
	ColorTypeTask    = lipgloss.Color("#F1FA8C") // Yellow
	ColorTypeChore   = lipgloss.Color("#8BE9FD") // Cyan
)

// Global Styles (using lipgloss.NewStyle() instead of deprecated patterns)
var (
	// App Layout
	AppStyle = lipgloss.NewStyle().Padding(0, 0)

	// Panels
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1)

	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// List Item Styles
	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Border(lipgloss.HiddenBorder(), false, false, false, true).
			BorderForeground(ColorBg)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				PaddingRight(1).
				Background(ColorBgHighlight).
				Border(lipgloss.HiddenBorder(), false, false, false, true).
				BorderForeground(ColorPrimary).
				Bold(true)

	// Column Styles
	ColIDStyle       = lipgloss.NewStyle().Width(8).Foreground(ColorSecondary).Bold(true)
	ColTypeStyle     = lipgloss.NewStyle().Width(2).Align(lipgloss.Center)
	ColPrioStyle     = lipgloss.NewStyle().Width(3).Align(lipgloss.Center)
	ColStatusStyle   = lipgloss.NewStyle().Width(12).Align(lipgloss.Center).Bold(true)
	ColTitleStyle    = lipgloss.NewStyle().Foreground(ColorText)
	ColAssigneeStyle = lipgloss.NewStyle().Width(12).Foreground(ColorSecondary).Align(lipgloss.Right)
	ColAgeStyle      = lipgloss.NewStyle().Width(8).Foreground(ColorSecondary).Align(lipgloss.Right)
	ColCommentsStyle = lipgloss.NewStyle().Width(4).Foreground(ColorSubtext).Align(lipgloss.Right)

	// Detail View Styles
	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Background(ColorBgHighlight).
				Bold(true).
				Padding(0, 1).
				MarginBottom(1)

	DetailMetaStyle = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			MarginBottom(1)

	// Header/Footer
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(0, 1)
)

// GetStatusColor returns the color for a given status
func GetStatusColor(s string) lipgloss.Color {
	switch s {
	case "open":
		return ColorStatusOpen
	case "in_progress":
		return ColorStatusInProgress
	case "blocked":
		return ColorStatusBlocked
	case "closed":
		return ColorStatusClosed
	default:
		return ColorText
	}
}

// GetTypeIcon returns the emoji and color for an issue type
func GetTypeIcon(t string) (string, lipgloss.Color) {
	switch t {
	case "bug":
		return "🐛", ColorTypeBug
	case "feature":
		return "✨", ColorTypeFeature
	case "task":
		return "📋", ColorTypeTask
	case "epic":
		return "🏔️", ColorTypeEpic
	case "chore":
		return "🧹", ColorTypeChore
	default:
		return "•", ColorText
	}
}

// GetPriorityIcon returns the emoji for a priority level
func GetPriorityIcon(p int) string {
	switch p {
	case 0:
		return "🔥" // Critical
	case 1:
		return "⚡" // High
	case 2:
		return "🔹" // Medium
	case 3:
		return "☕" // Low
	case 4:
		return "💤" // Backlog
	default:
		return ""
	}
}
