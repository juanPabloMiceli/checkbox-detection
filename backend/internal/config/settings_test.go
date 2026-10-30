package config

import (
	"reflect"
	"testing"
)

func TestSettingsAreIsolatedAndValidateRanges(t *testing.T) {
	base := Default()
	for _, payload := range []string{`{"minSidePx":91}`, `{"minWidthPx":91}`, `{"minEdgeSeparationPx":91}`, `{"checkedInkRatio":1.1}`, `{"minStrongSides":5}`, `{"minStrongSides":1.5}`, `{"minRectangularity":0.95}`, `{"finalIoUThreshold":0}`, `{"minCoverage":null}`, `{"preprocess":{}}`, `{} {}`, `{"outlineEnabled":false,"lineEnabled":false}`, `{"outlineEnabled":null}`, `{"lineEnabled":"false"}`} {
		if _, err := base.DecodeSettings([]byte(payload)); err == nil {
			t.Errorf("accepted invalid settings %s", payload)
		}
	}
	s, err := base.DecodeSettings([]byte(`{"checkedInkRatio":1,"minStrongSides":4}`))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := base.ApplySettings(s)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Classification.CheckedInkRatio != 1 || base.Classification.CheckedInkRatio != 0.12 {
		t.Fatal("override not applied independently")
	}
	if !reflect.DeepEqual(base.Preprocess, changed.Preprocess) {
		t.Fatal("preprocessing changed")
	}
	if len(base.SettingsDefinition().Limits) != 12 {
		t.Fatal("expected exactly twelve supported settings")
	}
}

func TestDetectorSwitchDefaultsAndIsolation(t *testing.T) {
	base := Default()
	defaults := base.SettingsDefinition().Defaults
	if !defaults.OutlineEnabled || !defaults.LineEnabled {
		t.Fatal("both detectors should default to enabled")
	}
	for _, payload := range []string{`{"outlineEnabled":false}`, `{"lineEnabled":false}`} {
		settings, err := base.DecodeSettings([]byte(payload))
		if err != nil {
			t.Fatal(err)
		}
		overridden, err := base.ApplySettings(settings)
		if err != nil {
			t.Fatal(err)
		}
		if overridden.Detectors.OutlineEnabled && overridden.Detectors.LineEnabled {
			t.Fatal("switch was ignored")
		}
		if !base.Detectors.OutlineEnabled || !base.Detectors.LineEnabled {
			t.Fatal("shared defaults changed")
		}
	}
}
