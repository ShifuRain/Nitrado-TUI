package tui

import (
	"fmt"

	"nitui/internal/api"
)

// serviceItem adapts api.Service to bubbles/list.Item.
type serviceItem struct{ api.Service }

func (s serviceItem) Title() string { return fmt.Sprintf("Server %d — %s", s.ID, s.Details.Game) }
func (s serviceItem) Description() string {
	return fmt.Sprintf("%s — %s", s.Status, s.Details.Address)
}
func (s serviceItem) FilterValue() string { return s.Title() }

// gameItem adapts api.Game to bubbles/list.Item.
type gameItem struct{ api.Game }

func (g gameItem) Title() string {
	switch {
	case g.Active:
		return g.Name + " (active)"
	case g.Installed:
		return g.Name + " (installed)"
	default:
		return g.Name
	}
}

func (g gameItem) Description() string {
	if !g.EnoughSlots {
		return g.Slug + " — needs more slots than this server has"
	}
	if g.TooManySlots {
		return g.Slug + " — this server has more slots than recommended"
	}
	return g.Slug
}

func (g gameItem) FilterValue() string { return g.Name }
