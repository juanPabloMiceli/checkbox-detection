// Package config defines the detector's defaults and startup validation.
package config

// Config contains only values. Copying it gives each detector independent,
// immutable settings; no maps, slices, or package-level mutable defaults exist.
type Config struct {
	Detectors      Detectors      `json:"detectors"`
	Preprocess     Preprocess     `json:"preprocess"`
	Contour        Contour        `json:"contour"`
	Line           Line           `json:"line"`
	Interior       Interior       `json:"interior"`
	Geometry       Geometry       `json:"geometry"`
	Exterior       Exterior       `json:"exterior"`
	Background     Background     `json:"background"`
	Classification Classification `json:"classification"`
	Dedup          Dedup          `json:"dedup"`
	MaxImagePixels int64          `json:"maxImagePixels"`
}

type Detectors struct {
	OutlineEnabled bool `json:"outlineEnabled"`
	LineEnabled    bool `json:"lineEnabled"`
}

type Preprocess struct {
	BlurKernelWidthPx    int     `json:"blurKernelWidthPx"`
	BlurKernelHeightPx   int     `json:"blurKernelHeightPx"`
	BlurSigmaX           float64 `json:"blurSigmaX"`
	BlurSigmaY           float64 `json:"blurSigmaY"`
	ThresholdBlockSizePx int     `json:"thresholdBlockSizePx"`
	ThresholdOffset      float64 `json:"thresholdOffset"`
}

type Contour struct {
	MinSidePx                  int           `json:"minSidePx"`
	MaxSidePx                  int           `json:"maxSidePx"`
	ApproximationEpsilonRatio  float64       `json:"approximationEpsilonRatio"`
	MinRectangularity          float64       `json:"minRectangularity"`
	ReferenceMinRectangularity float64       `json:"referenceMinRectangularity"`
	Border                     ContourBorder `json:"border"`
}

type ContourBorder struct {
	MinSidePx           int     `json:"minSidePx"`
	MinSearchRadiusPx   int     `json:"minSearchRadiusPx"`
	SearchRadiusDivisor int     `json:"searchRadiusDivisor"`
	MinEndMarginPx      int     `json:"minEndMarginPx"`
	EndMarginDivisor    int     `json:"endMarginDivisor"`
	MinCoverage         float64 `json:"minCoverage"`
	MinStrongSides      int     `json:"minStrongSides"`
}

type Line struct {
	HorizontalKernelLengthPx int     `json:"horizontalKernelLengthPx"`
	VerticalKernelLengthPx   int     `json:"verticalKernelLengthPx"`
	MinWidthPx               int     `json:"minWidthPx"`
	MaxWidthPx               int     `json:"maxWidthPx"`
	MinEdgeSeparationPx      int     `json:"minEdgeSeparationPx"`
	MaxEdgeSeparationPx      int     `json:"maxEdgeSeparationPx"`
	MinSideCoverage          float64 `json:"minSideCoverage"`
}

type Interior struct {
	MinSidePx     int     `json:"minSidePx"`
	MinMarginPx   int     `json:"minMarginPx"`
	MarginDivisor int     `json:"marginDivisor"`
	MaxInkRatio   float64 `json:"maxInkRatio"`
}

type ReferenceBorder struct {
	MinSearchRadiusPx   int     `json:"minSearchRadiusPx"`
	SearchRadiusDivisor int     `json:"searchRadiusDivisor"`
	MinCoverage         float64 `json:"minCoverage"`
}

type Geometry struct {
	ClusterToleranceRatio    float64         `json:"clusterToleranceRatio"`
	MatchToleranceRatio      float64         `json:"matchToleranceRatio"`
	RecoveryToleranceRatio   float64         `json:"recoveryToleranceRatio"`
	MinTolerancePx           int             `json:"minTolerancePx"`
	MinClusterSupport        int             `json:"minClusterSupport"`
	MinSecondarySupportRatio float64         `json:"minSecondarySupportRatio"`
	ReferenceBorder          ReferenceBorder `json:"referenceBorder"`
}

type Exterior struct {
	MinStripDepthPx  int     `json:"minStripDepthPx"`
	EndMarginDivisor int     `json:"endMarginDivisor"`
	DarkInkRatio     float64 `json:"darkInkRatio"`
}

type Background struct {
	ReferenceQuantile         float64 `json:"referenceQuantile"`
	SampleBudget              int     `json:"sampleBudget"`
	DarkRatio                 float64 `json:"darkRatio"`
	MinDarkFraction           float64 `json:"minDarkFraction"`
	MinContrastRatio          float64 `json:"minContrastRatio"`
	MinVisibleRatio           float64 `json:"minVisibleRatio"`
	MinStripDepthPx           int     `json:"minStripDepthPx"`
	EndMarginDivisor          int     `json:"endMarginDivisor"`
	MinGapPx                  int     `json:"minGapPx"`
	GapDivisor                int     `json:"gapDivisor"`
	BorderMinSearchRadiusPx   int     `json:"borderMinSearchRadiusPx"`
	BorderSearchRadiusDivisor int     `json:"borderSearchRadiusDivisor"`
}

type Classification struct {
	MarginDivisor   int     `json:"marginDivisor"`
	CheckedInkRatio float64 `json:"checkedInkRatio"`
}

type Dedup struct {
	ReferenceIoUThreshold float64 `json:"referenceIoUThreshold"`
	FinalIoUThreshold     float64 `json:"finalIoUThreshold"`
}

// Default returns a fresh copy of the settings used by the frozen fixtures.
func Default() Config {
	return Config{
		Detectors: Detectors{OutlineEnabled: true, LineEnabled: true},
		Preprocess: Preprocess{
			BlurKernelWidthPx: 3, BlurKernelHeightPx: 3,
			BlurSigmaX: 0, BlurSigmaY: 0,
			ThresholdBlockSizePx: 31, ThresholdOffset: 10,
		},
		Contour: Contour{
			MinSidePx: 15, MaxSidePx: 90, ApproximationEpsilonRatio: 0.05,
			MinRectangularity: 0.85, ReferenceMinRectangularity: 0.90,
			Border: ContourBorder{
				MinSidePx: 8, MinSearchRadiusPx: 2, SearchRadiusDivisor: 12,
				MinEndMarginPx: 2, EndMarginDivisor: 8, MinCoverage: 0.65, MinStrongSides: 3,
			},
		},
		Line: Line{
			HorizontalKernelLengthPx: 9, VerticalKernelLengthPx: 9,
			MinWidthPx: 12, MaxWidthPx: 90, MinEdgeSeparationPx: 12, MaxEdgeSeparationPx: 90,
			MinSideCoverage: 0.65,
		},
		Interior: Interior{MinSidePx: 10, MinMarginPx: 2, MarginDivisor: 5, MaxInkRatio: 0.70},
		Geometry: Geometry{
			ClusterToleranceRatio: 0.12, MatchToleranceRatio: 0.20, RecoveryToleranceRatio: 0.12,
			MinTolerancePx: 2, MinClusterSupport: 3, MinSecondarySupportRatio: 0.20,
			ReferenceBorder: ReferenceBorder{MinSearchRadiusPx: 1, SearchRadiusDivisor: 16, MinCoverage: 0.90},
		},
		Exterior: Exterior{MinStripDepthPx: 4, EndMarginDivisor: 5, DarkInkRatio: 0.55},
		Background: Background{
			ReferenceQuantile: 0.90, SampleBudget: 65_536, DarkRatio: 0.35,
			MinDarkFraction: 0.65, MinContrastRatio: 0.06, MinVisibleRatio: 0.25,
			MinStripDepthPx: 4, EndMarginDivisor: 5, MinGapPx: 1, GapDivisor: 12,
			BorderMinSearchRadiusPx: 1, BorderSearchRadiusDivisor: 16,
		},
		Classification: Classification{MarginDivisor: 4, CheckedInkRatio: 0.12},
		Dedup:          Dedup{ReferenceIoUThreshold: 0.50, FinalIoUThreshold: 0.50},
		MaxImagePixels: 40_000_000,
	}
}
