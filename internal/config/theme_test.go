package config

import "testing"

func TestThemeMerge_FillsBlanksOnly(t *testing.T) {
	user := Theme{
		Colors: Colors{Accent: "#123456"}, // everything else left blank
	}
	got := user.merge(DefaultTheme())

	if got.Colors.Accent != "#123456" {
		t.Errorf("Accent = %q, want user override preserved", got.Colors.Accent)
	}
	if got.Colors.Primary != DefaultTheme().Colors.Primary {
		t.Errorf("Primary = %q, want default fallback", got.Colors.Primary)
	}
	if got.BorderStyle != DefaultTheme().BorderStyle {
		t.Errorf("BorderStyle = %q, want default fallback", got.BorderStyle)
	}
}

func TestThemeMerge_EmptyBackgroundStaysEmpty(t *testing.T) {
	got := Theme{}.merge(DefaultTheme())
	if got.Colors.Background != "" {
		t.Errorf("Background = %q, want empty (terminal default) since DefaultTheme leaves it unset", got.Colors.Background)
	}
}
