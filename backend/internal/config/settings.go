package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
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

func (c Config) ApplySettings(s DetectionSettings) (Config, error) {
	// Check the same bounds published to the frontend.
	values := map[string]float64{
		"checkedInkRatio":     s.CheckedInkRatio,
		"minSidePx":           float64(s.MinSidePx),
		"maxSidePx":           float64(s.MaxSidePx),
		"matchToleranceRatio": s.MatchToleranceRatio,
		"minRectangularity":   s.MinRectangularity,
		"finalIoUThreshold":   s.FinalIoUThreshold,
		"minWidthPx":          float64(s.MinWidthPx),
		"maxWidthPx":          float64(s.MaxWidthPx),
		"minEdgeSeparationPx": float64(s.MinEdgeSeparationPx),
		"maxEdgeSeparationPx": float64(s.MaxEdgeSeparationPx),
		"minStrongSides":      float64(s.MinStrongSides),
		"minCoverage":         s.MinCoverage,
	}
	for key, limit := range c.SettingsDefinition().Limits {
		v := values[key]
		if limit.ExclusiveMin && v == limit.Min {
			return Config{}, fmt.Errorf("%s must be greater than %g", key, limit.Min)
		}
		if math.IsNaN(v) || math.IsInf(v, 0) || v < limit.Min || v > limit.Max {
			return Config{}, fmt.Errorf("%s must be between %g and %g", key, limit.Min, limit.Max)
		}
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
	return c, c.Validate()
}

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
