package config

import (
	"fmt"
	"math"
)

// Validate checks settings exposed by the detection API. Internal algorithm and
// preprocessing values are trusted server configuration.
func (c Config) Validate() error {
	return c.validateSettings(c.SettingsDefinition().Defaults)
}

func (c Config) validateSettings(s DetectionSettings) error {
	if !s.OutlineEnabled && !s.LineEnabled {
		return fmt.Errorf("enable at least one detector")
	}
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
			return fmt.Errorf("%s must be greater than %g", key, limit.Min)
		}
		if math.IsNaN(v) || math.IsInf(v, 0) || v < limit.Min || v > limit.Max {
			return fmt.Errorf("%s must be between %g and %g", key, limit.Min, limit.Max)
		}
	}
	for _, pair := range []struct {
		name             string
		minimum, maximum int
	}{
		{"minSidePx", s.MinSidePx, s.MaxSidePx},
		{"minWidthPx", s.MinWidthPx, s.MaxWidthPx},
		{"minEdgeSeparationPx", s.MinEdgeSeparationPx, s.MaxEdgeSeparationPx},
	} {
		if pair.minimum > pair.maximum {
			return fmt.Errorf("%s must not exceed its maximum", pair.name)
		}
	}
	return nil
}
