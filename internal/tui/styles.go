package tui

import (
	"github.com/charmbracelet/lipgloss"

	"nitui/internal/config"
)

// styles holds every lipgloss.Style derived from the user's theme, built
// once per Run() so views don't repeatedly reparse colors.
type styles struct {
	Title    lipgloss.Style
	Help     lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Border   lipgloss.Style
	Input    lipgloss.Style
}

func newStyles(t config.Theme) styles {
	base := lipgloss.NewStyle()
	if t.Colors.Background != "" {
		base = base.Background(lipgloss.Color(t.Colors.Background))
	}

	border := borderFor(t.BorderStyle)

	return styles{
		Title:    base.Foreground(lipgloss.Color(t.Colors.Primary)).Bold(true),
		Help:     base.Foreground(lipgloss.Color(t.Colors.Muted)),
		Selected: base.Foreground(lipgloss.Color(t.Colors.Accent)).Bold(true),
		Normal:   base.Foreground(lipgloss.Color(t.Colors.Text)),
		Success:  base.Foreground(lipgloss.Color(t.Colors.Success)),
		Warning:  base.Foreground(lipgloss.Color(t.Colors.Warning)),
		Error:    base.Foreground(lipgloss.Color(t.Colors.Error)).Bold(true),
		Border: base.
			Border(border).
			BorderForeground(lipgloss.Color(t.Colors.Secondary)).
			Padding(0, 1),
		Input: base.Foreground(lipgloss.Color(t.Colors.Text)),
	}
}

func borderFor(name string) lipgloss.Border {
	switch name {
	case "normal":
		return lipgloss.NormalBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}
