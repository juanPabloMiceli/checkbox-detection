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

func TestValidationRejectsUnsafeSettings(t *testing.T) {
	for _, test := range []struct {
		name, field string
		change      func(*Config)
	}{
		{"even blur", "preprocess.blurKernelWidthPx", func(c *Config) { c.Preprocess.BlurKernelWidthPx = 4 }},
		{"zero blur", "preprocess.blurKernelHeightPx", func(c *Config) { c.Preprocess.BlurKernelHeightPx = 0 }},
		{"large blur", "preprocess.blurKernelWidthPx", func(c *Config) { c.Preprocess.BlurKernelWidthPx = 513 }},
		{"small block", "preprocess.thresholdBlockSizePx", func(c *Config) { c.Preprocess.ThresholdBlockSizePx = 1 }},
		{"negative sigma", "preprocess.blurSigma", func(c *Config) { c.Preprocess.BlurSigmaX = -1 }},
		{"nan sigma", "preprocess.blurSigmaY", func(c *Config) { c.Preprocess.BlurSigmaY = math.NaN() }},
		{"infinite threshold", "preprocess.thresholdOffset", func(c *Config) { c.Preprocess.ThresholdOffset = math.Inf(1) }},
		{"float32 overflow", "preprocess.thresholdOffset", func(c *Config) { c.Preprocess.ThresholdOffset = math.MaxFloat64 }},
		{"nan fraction", "classification.checkedInkRatio", func(c *Config) { c.Classification.CheckedInkRatio = math.NaN() }},
		{"negative fraction", "background.darkRatio", func(c *Config) { c.Background.DarkRatio = -0.1 }},
		{"fraction above one", "line.minSideCoverage", func(c *Config) { c.Line.MinSideCoverage = 1.1 }},
		{"zero quantile", "background.referenceQuantile", func(c *Config) { c.Background.ReferenceQuantile = 0 }},
		{"zero dedup", "dedup.finalIoUThreshold", func(c *Config) { c.Dedup.FinalIoUThreshold = 0 }},
		{"zero divisor", "background.gapDivisor", func(c *Config) { c.Background.GapDivisor = 0 }},
		{"empty interior", "interior.marginDivisor", func(c *Config) { c.Interior.MarginDivisor = 2 }},
		{"negative radius", "contour.border.minSearchRadiusPx", func(c *Config) { c.Contour.Border.MinSearchRadiusPx = -1 }},
		{"empty histogram", "background.sampleBudget", func(c *Config) { c.Background.SampleBudget = 0 }},
		{"no cluster support", "geometry.minClusterSupport", func(c *Config) { c.Geometry.MinClusterSupport = 0 }},
		{"invalid side count", "contour.border.minStrongSides", func(c *Config) { c.Contour.Border.MinStrongSides = 5 }},
		{"reversed contour sizes", "contour.minSidePx", func(c *Config) { c.Contour.MinSidePx = 91 }},
		{"reversed widths", "line.minWidthPx", func(c *Config) { c.Line.MaxWidthPx = 11 }},
		{"reversed heights", "line.minEdgeSeparationPx", func(c *Config) { c.Line.MaxEdgeSeparationPx = 11 }},
		{"weak references", "contour.referenceMinRectangularity", func(c *Config) { c.Contour.ReferenceMinRectangularity = 0.8 }},
		{"zero line kernel", "line.horizontalKernelLengthPx", func(c *Config) { c.Line.HorizontalKernelLengthPx = 0 }},
		{"zero pixel limit", "maxImagePixels", func(c *Config) { c.MaxImagePixels = 0 }},
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
