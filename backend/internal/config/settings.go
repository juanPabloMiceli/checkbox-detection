package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DetectionSettings is the supported per-request surface; preprocessing stays private.
type DetectionSettings struct {
	OutlineEnabled      bool    `json:"outlineEnabled"`
	LineEnabled         bool    `json:"lineEnabled"`
	CheckedInkRatio     float64 `json:"checkedInkRatio"`
	MinSidePx           int     `json:"minSidePx"`
	MaxSidePx           int     `json:"maxSidePx"`
	MatchToleranceRatio float64 `json:"matchToleranceRatio"`
	MinRectangularity   float64 `json:"minRectangularity"`
	FinalIoUThreshold   float64 `json:"finalIoUThreshold"`
	MinWidthPx          int     `json:"minWidthPx"`
	MaxWidthPx          int     `json:"maxWidthPx"`
	MinEdgeSeparationPx int     `json:"minEdgeSeparationPx"`
	MaxEdgeSeparationPx int     `json:"maxEdgeSeparationPx"`
	MinStrongSides      int     `json:"minStrongSides"`
	MinCoverage         float64 `json:"minCoverage"`
}

type SettingLimit struct {
	ExclusiveMin bool    `json:"exclusiveMin,omitempty"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Integer      bool    `json:"integer"`
}

type SettingsDefinition struct {
	Defaults DetectionSettings       `json:"defaults"`
	Limits   map[string]SettingLimit `json:"limits"`
}

// SettingsDefinition publishes defaults and UI bounds from the loaded configuration.
// Ratios use fractions, and pixel dimensions refer to the original image.
func (c Config) SettingsDefinition() SettingsDefinition {
	limits := map[string]SettingLimit{}
	for _, key := range []string{"minSidePx", "maxSidePx", "minWidthPx", "maxWidthPx", "minEdgeSeparationPx", "maxEdgeSeparationPx"} {
		limits[key] = SettingLimit{Min: 1, Max: 1_000_000, Integer: true}
	}
	for _, key := range []string{"checkedInkRatio", "matchToleranceRatio", "minCoverage"} {
		limits[key] = SettingLimit{Min: 0, Max: 1}
	}
	limits["minRectangularity"] = SettingLimit{Min: 0, Max: c.Contour.ReferenceMinRectangularity, ExclusiveMin: true}
	limits["finalIoUThreshold"] = SettingLimit{Min: 0, Max: 1, ExclusiveMin: true}
	limits["minStrongSides"] = SettingLimit{Min: 1, Max: 4, Integer: true}
	return SettingsDefinition{Defaults: DetectionSettings{
		OutlineEnabled: c.Detectors.OutlineEnabled, LineEnabled: c.Detectors.LineEnabled,
		CheckedInkRatio: c.Classification.CheckedInkRatio, MinSidePx: c.Contour.MinSidePx, MaxSidePx: c.Contour.MaxSidePx,
		MatchToleranceRatio: c.Geometry.MatchToleranceRatio, MinRectangularity: c.Contour.MinRectangularity,
		FinalIoUThreshold: c.Dedup.FinalIoUThreshold, MinWidthPx: c.Line.MinWidthPx, MaxWidthPx: c.Line.MaxWidthPx,
		MinEdgeSeparationPx: c.Line.MinEdgeSeparationPx, MaxEdgeSeparationPx: c.Line.MaxEdgeSeparationPx,
		MinStrongSides: c.Contour.Border.MinStrongSides, MinCoverage: c.Contour.Border.MinCoverage,
	}, Limits: limits}
}

// ApplySettings validates overrides against the published bounds and returns a
// configuration copy. The receiver and its preprocessing settings are unchanged.
func (c Config) ApplySettings(s DetectionSettings) (Config, error) {
	if err := c.validateSettings(s); err != nil {
		return Config{}, err
	}
	c.Detectors = Detectors{OutlineEnabled: s.OutlineEnabled, LineEnabled: s.LineEnabled}
	c.Classification.CheckedInkRatio = s.CheckedInkRatio
	c.Contour.MinSidePx = s.MinSidePx
	c.Contour.MaxSidePx = s.MaxSidePx
	c.Geometry.MatchToleranceRatio = s.MatchToleranceRatio
	c.Contour.MinRectangularity = s.MinRectangularity
	c.Dedup.FinalIoUThreshold = s.FinalIoUThreshold
	c.Line.MinWidthPx = s.MinWidthPx
	c.Line.MaxWidthPx = s.MaxWidthPx
	c.Line.MinEdgeSeparationPx = s.MinEdgeSeparationPx
	c.Line.MaxEdgeSeparationPx = s.MaxEdgeSeparationPx
	c.Contour.Border.MinStrongSides = s.MinStrongSides
	c.Contour.Border.MinCoverage = s.MinCoverage
	return c, nil
}

// DecodeSettings merges a partial JSON object over server defaults and validates
// the result. Unknown fields, nulls, trailing values, and oversized input are rejected.
func (c Config) DecodeSettings(data []byte) (DetectionSettings, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || len(data) > 8192 || data[0] != '{' {
		return DetectionSettings{}, fmt.Errorf("settings must be a JSON object of at most 8192 bytes")
	}
	settings := c.SettingsDefinition().Defaults
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		return settings, fmt.Errorf("invalid settings: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return settings, fmt.Errorf("settings must contain exactly one JSON object")
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return settings, err
		}
		if token == nil {
			return settings, fmt.Errorf("settings must not contain null")
		}
	}
	_, err := c.ApplySettings(settings)
	return settings, err
}
