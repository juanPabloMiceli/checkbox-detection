package checkbox

import "image"

// CandidateSource records how a proposal was generated, independently of its
// quality measurements. Unknown candidates cannot vote on document geometry.
type CandidateSource uint8

const (
	SourceUnknown CandidateSource = iota
	SourceContour
	SourceLine
)

// Candidate is internal geometry, not the external JSON representation.
type Candidate struct {
	Bounds         image.Rectangle `json:"bounds"`
	Corners        [4]image.Point  `json:"corners"`
	Source         CandidateSource `json:"source"`
	Rectangularity float64         `json:"rectangularity"`
}

// MarshalText keeps diagnostic artifacts readable without changing the
// comparable numeric representation used inside the detector.
func (source CandidateSource) MarshalText() ([]byte, error) {
	switch source {
	case SourceContour:
		return []byte("contour"), nil
	case SourceLine:
		return []byte("line"), nil
	default:
		return []byte("unknown"), nil
	}
}

func (candidate Candidate) needsGeometrySupport(referenceMin float64) bool {
	return candidate.Source == SourceContour && candidate.Rectangularity < referenceMin
}
