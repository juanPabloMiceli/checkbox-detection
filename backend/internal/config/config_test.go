package config

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestDefaultIsValidAndIndependent(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.Contour.Border.SearchRadiusDivisor = 99
	if Default().Contour.Border.SearchRadiusDivisor != 12 {
		t.Fatal("default configuration was mutated")
	}
	encoded, err := json.Marshal(Default())
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Load(strings.NewReader(string(encoded)))
	if err != nil || roundTrip != Default() {
		t.Fatalf("defaults did not round-trip: %v", err)
	}
}

func TestValidationRejectsInvalidDetectionSettings(t *testing.T) {
	for _, test := range []struct {
		name, field string
		change      func(*Config)
	}{
		{"nan fraction", "checkedInkRatio", func(c *Config) { c.Classification.CheckedInkRatio = math.NaN() }},
		{"zero dedup", "finalIoUThreshold", func(c *Config) { c.Dedup.FinalIoUThreshold = 0 }},
		{"invalid side count", "minStrongSides", func(c *Config) { c.Contour.Border.MinStrongSides = 5 }},
		{"reversed contour sizes", "minSidePx", func(c *Config) { c.Contour.MinSidePx = 91 }},
		{"reversed widths", "minWidthPx", func(c *Config) { c.Line.MaxWidthPx = 11 }},
		{"reversed heights", "minEdgeSeparationPx", func(c *Config) { c.Line.MaxEdgeSeparationPx = 11 }},
		{"shape above reference limit", "minRectangularity", func(c *Config) { c.Contour.MinRectangularity = 0.95 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := Default()
			test.change(&cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("want validation error for %s, got %v", test.field, err)
			}
		})
	}
}

func TestLoadMergesPartialConfigAndRejectsMistakes(t *testing.T) {
	cfg, err := Load(strings.NewReader(`{"classification":{"checkedInkRatio":0.25},"contour":{"border":{"minStrongSides":4}}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.Classification.CheckedInkRatio = 0.25
	want.Contour.Border.MinStrongSides = 4
	if cfg != want {
		t.Fatalf("partial override lost defaults: %+v", cfg)
	}
	for _, value := range []string{
		``, `null`, `[]`, `true`, `{} {}`, `{} garbage`,
		`{"unknown":1}`, `{"classification":{"checkedInkRato":0.25}}`,
		`{"classification":null}`, `{"classification":{"checkedInkRatio":null}}`,
		`{"classification":{"checkedInkRatio":"high"}}`, `{"maxImagePixels":1.5}`,
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := Load(strings.NewReader(value)); err == nil {
				t.Fatalf("accepted invalid configuration: %s", value)
			}
		})
	}
}
