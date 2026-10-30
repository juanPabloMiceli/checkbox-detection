package checkbox

import (
	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// DetectionResult contains checkbox bounds and checked states in original-image coordinates.
type DetectionResult struct {
	Boxes []Detection `json:"boxes"`
}

type Detection struct {
	// BBox is [left, top, right, bottom] in image pixels. Right and bottom
	// follow image.Rectangle's exclusive maximum-coordinate convention.
	BBox    [4]int `json:"bbox"`
	Checked bool   `json:"is_checked"`
}

func ToDetections(
	candidates []Candidate,
	binary gocv.Mat,
	cfg config.Classification,
) []Detection {
	detections := make([]Detection, 0, len(candidates))

	for _, candidate := range candidates {
		detections = append(detections, Detection{
			BBox: [4]int{
				candidate.Bounds.Min.X,
				candidate.Bounds.Min.Y,
				candidate.Bounds.Max.X,
				candidate.Bounds.Max.Y,
			},
			Checked: IsChecked(binary, candidate.Bounds, cfg),
		})
	}

	return detections
}
