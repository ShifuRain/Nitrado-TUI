package config

// Theme controls the visual appearance of the interactive TUI. Every field
// is optional in the user's YAML file — anything left blank falls back to
// the value in DefaultTheme().
type Theme struct {
	// BorderStyle is one of: "rounded", "normal", "thick", "double", "hidden".
	BorderStyle string `yaml:"border_style"`
	Colors      Colors `yaml:"colors"`
}

// Colors accepts any value lipgloss.Color understands: a hex string like
// "#7D56F4", an ANSI 256 index as a string like "212", or a true color
// spec. Leave a field empty to use the default.
type Colors struct {
	Primary    string `yaml:"primary"`    // headers, active borders
	Secondary  string `yaml:"secondary"`  // secondary text, inactive borders
	Accent     string `yaml:"accent"`     // selected list item, highlights
	Text       string `yaml:"text"`       // normal body text
	Muted      string `yaml:"muted"`      // dim/help text
	Success    string `yaml:"success"`    // server running, success messages
	Warning    string `yaml:"warning"`    // approaching a limit, degraded state
	Error      string `yaml:"error"`      // failures, limit reached
	Background string `yaml:"background"` // empty = terminal's own background
}

// DefaultTheme is used for any color/style left unset by the user's config.
func DefaultTheme() Theme {
	return Theme{
		BorderStyle: "rounded",
		Colors: Colors{
			Primary:    "#7D56F4",
			Secondary:  "#5A5A7A",
			Accent:     "#F25D94",
			Text:       "#E4E4E7",
			Muted:      "#71717A",
			Success:    "#4ADE80",
			Warning:    "#FBBF24",
			Error:      "#F87171",
			Background: "",
		},
	}
}

// merge fills any blank field in t with the corresponding field from d.
func (t Theme) merge(d Theme) Theme {
	if t.BorderStyle == "" {
		t.BorderStyle = d.BorderStyle
	}
	c, dc := &t.Colors, d.Colors
	fields := []*string{&c.Primary, &c.Secondary, &c.Accent, &c.Text, &c.Muted, &c.Success, &c.Warning, &c.Error}
	defaults := []string{dc.Primary, dc.Secondary, dc.Accent, dc.Text, dc.Muted, dc.Success, dc.Warning, dc.Error}
	for i, f := range fields {
		if *f == "" {
			*f = defaults[i]
		}
	}
	// Background intentionally has no non-empty default: an empty value
	// means "use the terminal's own background", which matters for users
	// running a transparent/rice'd terminal.
	return t
}
