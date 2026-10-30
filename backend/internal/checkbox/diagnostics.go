package checkbox

// Diagnostics reuses stage results for inspection without recomputing the pipeline.
// Normal HTTP processing does not retain this data or generate debug images.
type Diagnostics struct {
	Contours   ContourResult
	LineCount  int
	Geometry   GeometryResult
	Background BackgroundStats
	Dedup      DedupResult
}

// RejectionReason identifies the first failing check at a stage. Reference
// rejections concern geometry learning, not necessarily a final detection.
type RejectionReason string

const (
	RejectContourSize             RejectionReason = "contour_size"
	RejectContourPolygon          RejectionReason = "contour_polygon"
	RejectContourConvexity        RejectionReason = "contour_convexity"
	RejectContourRectangularity   RejectionReason = "contour_rectangularity"
	RejectContourBorder           RejectionReason = "contour_border"
	RejectReferenceSource         RejectionReason = "reference_source"
	RejectReferenceRectangularity RejectionReason = "reference_rectangularity"
	RejectReferenceDuplicate      RejectionReason = "reference_duplicate"
	RejectReferenceBounds         RejectionReason = "reference_bounds"
	RejectReferenceBorder         RejectionReason = "reference_border"
	RejectReferenceInterior       RejectionReason = "reference_interior"
	RejectReferenceExterior       RejectionReason = "reference_exterior"
	RejectRecoveryWithoutModel    RejectionReason = "recovery_without_model"
	RejectGeometryMismatch        RejectionReason = "geometry_mismatch"
	RejectExteriorInk             RejectionReason = "exterior_ink"
	RejectDuplicate               RejectionReason = "duplicate"
)

type CandidateRejection struct {
	Candidate Candidate       `json:"candidate"`
	Reason    RejectionReason `json:"reason"`
}

func recordRejection(rejections *[]CandidateRejection, enabled bool, candidate Candidate, reason RejectionReason) {
	if enabled {
		*rejections = append(*rejections, CandidateRejection{Candidate: candidate, Reason: reason})
	}
}

type ContourResult struct {
	Candidates []Candidate
	Rejections RejectionStats
	Rejected   []CandidateRejection
}

type GeometryResult struct {
	Candidates          []Candidate
	Clusters            []GeometryCluster
	ReferenceCount      int
	ReferenceRejections []CandidateRejection
	Rejected            []CandidateRejection
}

type DedupResult struct {
	Candidates     []Candidate
	Rejected       []CandidateRejection
	DuplicateCount int
}
