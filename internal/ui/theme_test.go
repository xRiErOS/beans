package ui

import "testing"

func TestThemeByNameReturnsEveryFlavour(t *testing.T) {
	for _, name := range []string{"latte", "frappe", "macchiato", "mocha"} {
		th, ok := ThemeByName(name)
		if !ok {
			t.Fatalf("ThemeByName(%q) = not found, want a theme", name)
		}
		if th.Name != name {
			t.Errorf("Theme.Name = %q, want %q", th.Name, name)
		}
	}
}

func TestThemeByNameRejectsUnknown(t *testing.T) {
	if _, ok := ThemeByName("dracula"); ok {
		t.Error(`ThemeByName("dracula") = found, want not found`)
	}
}

func TestDefaultThemeIsMocha(t *testing.T) {
	if got := DefaultTheme().Name; got != "mocha" {
		t.Errorf(`DefaultTheme().Name = %q, want "mocha"`, got)
	}
}

func TestMochaToneValues(t *testing.T) {
	// Verified against catppuccin/palette palette.json and against the ANSI
	// palette configured in ghostty. Do not edit these from memory.
	want := map[string]string{
		"mauve": "#cba6f7", "blue": "#89b4fa", "sapphire": "#74c7ec",
		"maroon": "#eba0ac", "green": "#a6e3a1", "peach": "#fab387",
		"yellow": "#f9e2af", "red": "#f38ba8", "overlay2": "#9399b2",
		"overlay1": "#7f849c", "overlay0": "#6c7086", "surface2": "#585b70",
		"subtext0": "#a6adc8",
	}
	th := DefaultTheme()
	for tone, hex := range want {
		if got := th.Hex(tone); got != hex {
			t.Errorf("mocha.Hex(%q) = %q, want %q", tone, got, hex)
		}
	}
}

func TestEveryThemeDefinesTheSameTones(t *testing.T) {
	base := DefaultTheme()
	for name, th := range Themes() {
		for tone := range base.Colors {
			if th.Hex(tone) == "" {
				t.Errorf("theme %q is missing tone %q", name, tone)
			}
		}
	}
}

func TestHexOfUnknownToneIsEmpty(t *testing.T) {
	if got := DefaultTheme().Hex("chartreuse"); got != "" {
		t.Errorf(`Hex("chartreuse") = %q, want ""`, got)
	}
}
