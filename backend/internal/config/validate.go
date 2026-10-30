package config

import (
	"errors"
	"fmt"
	"math"
)

// Validate rejects invalid numeric settings before any native processing.
func (c Config) Validate() error {
	var issues []error
	if !c.Detectors.OutlineEnabled && !c.Detectors.LineEnabled {
		issues = append(issues, fmt.Errorf("enable at least one detector"))
	}
	check := func(ok bool, name, requirement string) {
		if !ok {
			issues = append(issues, fmt.Errorf("%s %s", name, requirement))
		}
	}
	integer := func(name string, value, minimum int) {
		check(value >= minimum && value <= 1_000_000, name, fmt.Sprintf("must be between %d and 1000000", minimum))
	}
	fraction := func(name string, value float64, positive bool) {
		ok := !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
		if positive {
			ok = ok && value > 0
		}
		requirement := "must be a finite fraction in [0,1]"
		if positive {
			requirement += " excluding zero"
		}
		check(ok, name, requirement)
	}
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"preprocess.blurKernelWidthPx", c.Preprocess.BlurKernelWidthPx},
		{"preprocess.blurKernelHeightPx", c.Preprocess.BlurKernelHeightPx},
		{"preprocess.thresholdBlockSizePx", c.Preprocess.ThresholdBlockSizePx},
	} {
		check(setting.value > 0 && setting.value <= 511 && setting.value%2 == 1, setting.name, "must be positive, odd, and at most 511")
	}
	check(c.Preprocess.ThresholdBlockSizePx >= 3, "preprocess.thresholdBlockSizePx", "must be at least 3")
	for _, setting := range []struct {
		name  string
		value float64
	}{
		{"preprocess.blurSigmaX", c.Preprocess.BlurSigmaX}, {"preprocess.blurSigmaY", c.Preprocess.BlurSigmaY},
		{"preprocess.thresholdOffset", c.Preprocess.ThresholdOffset},
	} {
		check(!math.IsNaN(setting.value) && !math.IsInf(setting.value, 0), setting.name, "must be finite")
	}
	check(c.Preprocess.BlurSigmaX >= 0 && c.Preprocess.BlurSigmaY >= 0, "preprocess.blurSigma", "must be nonnegative")
	check(math.Abs(c.Preprocess.ThresholdOffset) <= math.MaxFloat32, "preprocess.thresholdOffset", "must fit in a finite float32")
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"contour.minSidePx", c.Contour.MinSidePx}, {"contour.maxSidePx", c.Contour.MaxSidePx},
		{"contour.border.minSidePx", c.Contour.Border.MinSidePx},
		{"contour.border.searchRadiusDivisor", c.Contour.Border.SearchRadiusDivisor},
		{"line.minWidthPx", c.Line.MinWidthPx}, {"line.maxWidthPx", c.Line.MaxWidthPx},
		{"line.minEdgeSeparationPx", c.Line.MinEdgeSeparationPx}, {"line.maxEdgeSeparationPx", c.Line.MaxEdgeSeparationPx},
		{"interior.minSidePx", c.Interior.MinSidePx},
		{"geometry.minClusterSupport", c.Geometry.MinClusterSupport},
		{"geometry.referenceBorder.searchRadiusDivisor", c.Geometry.ReferenceBorder.SearchRadiusDivisor},
		{"exterior.minStripDepthPx", c.Exterior.MinStripDepthPx},
		{"background.sampleBudget", c.Background.SampleBudget}, {"background.minStripDepthPx", c.Background.MinStripDepthPx},
		{"background.gapDivisor", c.Background.GapDivisor},
		{"background.borderSearchRadiusDivisor", c.Background.BorderSearchRadiusDivisor},
	} {
		integer(setting.name, setting.value, 1)
	}
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"contour.border.minSearchRadiusPx", c.Contour.Border.MinSearchRadiusPx},
		{"contour.border.minEndMarginPx", c.Contour.Border.MinEndMarginPx},
		{"interior.minMarginPx", c.Interior.MinMarginPx}, {"geometry.minTolerancePx", c.Geometry.MinTolerancePx},
		{"geometry.referenceBorder.minSearchRadiusPx", c.Geometry.ReferenceBorder.MinSearchRadiusPx},
		{"background.minGapPx", c.Background.MinGapPx}, {"background.borderMinSearchRadiusPx", c.Background.BorderMinSearchRadiusPx},
	} {
		integer(setting.name, setting.value, 0)
	}
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"contour.border.endMarginDivisor", c.Contour.Border.EndMarginDivisor},
		{"interior.marginDivisor", c.Interior.MarginDivisor}, {"exterior.endMarginDivisor", c.Exterior.EndMarginDivisor},
		{"background.endMarginDivisor", c.Background.EndMarginDivisor}, {"classification.marginDivisor", c.Classification.MarginDivisor},
	} {
		integer(setting.name, setting.value, 3)
	}
	check(c.Contour.MinSidePx <= c.Contour.MaxSidePx, "contour.minSidePx", "must not exceed maxSidePx")
	check(c.Line.MinWidthPx <= c.Line.MaxWidthPx, "line.minWidthPx", "must not exceed maxWidthPx")
	check(c.Line.MinEdgeSeparationPx <= c.Line.MaxEdgeSeparationPx, "line.minEdgeSeparationPx", "must not exceed maxEdgeSeparationPx")
	check(c.Contour.Border.MinStrongSides >= 1 && c.Contour.Border.MinStrongSides <= 4, "contour.border.minStrongSides", "must be between 1 and 4")
	for _, setting := range []struct {
		name  string
		value int
	}{
		{"line.horizontalKernelLengthPx", c.Line.HorizontalKernelLengthPx}, {"line.verticalKernelLengthPx", c.Line.VerticalKernelLengthPx},
	} {
		check(setting.value > 0 && setting.value <= 511, setting.name, "must be between 1 and 511")
	}
	for _, setting := range []struct {
		name  string
		value float64
	}{
		{"contour.approximationEpsilonRatio", c.Contour.ApproximationEpsilonRatio},
		{"contour.minRectangularity", c.Contour.MinRectangularity}, {"contour.referenceMinRectangularity", c.Contour.ReferenceMinRectangularity},
		{"dedup.referenceIoUThreshold", c.Dedup.ReferenceIoUThreshold}, {"dedup.finalIoUThreshold", c.Dedup.FinalIoUThreshold},
		{"background.referenceQuantile", c.Background.ReferenceQuantile},
	} {
		fraction(setting.name, setting.value, true)
	}
	for _, setting := range []struct {
		name  string
		value float64
	}{
		{"contour.border.minCoverage", c.Contour.Border.MinCoverage}, {"line.minSideCoverage", c.Line.MinSideCoverage},
		{"interior.maxInkRatio", c.Interior.MaxInkRatio}, {"geometry.clusterToleranceRatio", c.Geometry.ClusterToleranceRatio},
		{"geometry.matchToleranceRatio", c.Geometry.MatchToleranceRatio}, {"geometry.recoveryToleranceRatio", c.Geometry.RecoveryToleranceRatio},
		{"geometry.minSecondarySupportRatio", c.Geometry.MinSecondarySupportRatio},
		{"geometry.referenceBorder.minCoverage", c.Geometry.ReferenceBorder.MinCoverage},
		{"exterior.darkInkRatio", c.Exterior.DarkInkRatio}, {"background.darkRatio", c.Background.DarkRatio},
		{"background.minDarkFraction", c.Background.MinDarkFraction}, {"background.minContrastRatio", c.Background.MinContrastRatio},
		{"background.minVisibleRatio", c.Background.MinVisibleRatio}, {"classification.checkedInkRatio", c.Classification.CheckedInkRatio},
	} {
		fraction(setting.name, setting.value, false)
	}
	check(c.Contour.ReferenceMinRectangularity >= c.Contour.MinRectangularity, "contour.referenceMinRectangularity", "must not be below minRectangularity")
	check(c.MaxImagePixels > 0, "maxImagePixels", "must be positive")
	return errors.Join(issues...)
}
