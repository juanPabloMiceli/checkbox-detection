package checkbox

import (
	"context"
	"image"
	"math"
	"sort"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// GeometryCluster describes a repeated width-height pair learned from distinct
// boxes with strong evidence on all four sides. Width and height stay separate:
// a document can use rectangular boxes, or more than one checkbox component.
type GeometryCluster struct {
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Support int     `json:"support"`
}

// FilterByDocumentGeometry learns from strict contour references, deduplicated
// so inner/outer contours cannot inflate support. Line candidates can vastly
// outnumber real boxes, so they do not vote on the model. Both detectors are
// filtered against that model before final deduplication: a large outlier must
// not suppress a valid box and then be discarded itself. Without enough strong
// references, established candidates pass through; imperfect contours requiring
// geometry support are withheld. Original corners are preserved.
func FilterByDocumentGeometry(ctx context.Context, candidates, references []Candidate, binary gocv.Mat, cfg config.Config, diagnostics bool) (GeometryResult, error) {
	result := GeometryResult{}
	seeds, rejectedReferences, err := selectGeometryReferences(ctx, references, binary, cfg, diagnostics)
	if err != nil {
		return result, err
	}
	result.ReferenceCount, result.ReferenceRejections = len(seeds), rejectedReferences
	result.Clusters, err = learnCheckboxSizeClusters(ctx, seeds, cfg.Geometry)
	if err != nil {
		return GeometryResult{}, err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return GeometryResult{}, err
		}
		reason := geometryRejection(candidate, result.Clusters, binary, cfg)
		if reason != "" {
			recordRejection(&result.Rejected, diagnostics, candidate, reason)
		} else {
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	return result, ctx.Err()
}

// selectGeometryReferences filters weak references before deduplication, so a
// larger weak outer contour cannot suppress a strong inner learning reference.
func selectGeometryReferences(ctx context.Context, references []Candidate, binary gocv.Mat, cfg config.Config, diagnostics bool) ([]Candidate, []CandidateRejection, error) {
	var eligible, seeds []Candidate
	var rejected []CandidateRejection
	for _, candidate := range references {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		switch {
		case candidate.Source != SourceContour:
			recordRejection(&rejected, diagnostics, candidate, RejectReferenceSource)
		case candidate.needsGeometrySupport(cfg.Contour.ReferenceMinRectangularity):
			recordRejection(&rejected, diagnostics, candidate, RejectReferenceRectangularity)
		default:
			eligible = append(eligible, candidate)
		}
	}
	unique, err := deduplicate(ctx, eligible, cfg.Dedup.ReferenceIoUThreshold, diagnostics)
	if err != nil {
		return nil, nil, err
	}
	for _, rejection := range unique.Rejected {
		rejection.Reason = RejectReferenceDuplicate
		rejected = append(rejected, rejection)
	}
	for _, candidate := range unique.Candidates {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if reason := referenceRejection(candidate, binary, cfg); reason != "" {
			recordRejection(&rejected, diagnostics, candidate, reason)
		} else {
			seeds = append(seeds, candidate)
		}
	}
	return seeds, rejected, nil
}

// referenceRejection returns the first failed check for an outline used to learn
// document sizes. An empty reason means it is suitable as a reference.
func referenceRejection(candidate Candidate, binary gocv.Mat, cfg config.Config) RejectionReason {
	bounds := candidate.Bounds
	switch {
	case bounds.Empty() || !bounds.In(image.Rect(0, 0, binary.Cols(), binary.Rows())):
		return RejectReferenceBounds
	case !hasStrongBordersOnAllSides(binary, bounds, cfg.Geometry.ReferenceBorder):
		return RejectReferenceBorder
	case !hasOpenInterior(binary, bounds, cfg.Interior):
		return RejectReferenceInterior
	case hasInkOnOppositeExteriorSides(binary, bounds, cfg.Exterior):
		return RejectReferenceExterior
	default:
		return ""
	}
}

// geometryRejection checks a proposal against learned sizes. Without a model,
// established candidates pass but weak contours needing size support do not.
func geometryRejection(candidate Candidate, clusters []GeometryCluster, binary gocv.Mat, cfg config.Config) RejectionReason {
	recovery := candidate.needsGeometrySupport(cfg.Contour.ReferenceMinRectangularity)
	if len(clusters) == 0 {
		if recovery {
			return RejectRecoveryWithoutModel
		}
		return ""
	}
	if hasInkOnOppositeExteriorSides(binary, candidate.Bounds, cfg.Exterior) {
		return RejectExteriorInk
	}
	tolerance := cfg.Geometry.MatchToleranceRatio
	if recovery {
		tolerance = cfg.Geometry.RecoveryToleranceRatio
	}
	for _, cluster := range clusters {
		if matchesGeometry(candidate, cluster, tolerance, cfg.Geometry.MinTolerancePx) {
			return ""
		}
	}
	return RejectGeometryMismatch
}

// hasInkOnOppositeExteriorSides detects shapes embedded in solid ink, such as
// white lettering in a dark section bar.
// Unlike an outlined box, that dark region continues beyond opposite borders.
// Sample a full box dimension beyond each side: narrower strips confuse thick
// borders and nearby boxes with a solid background. One adjoining form rule or
// a checkmark extending outside the box is allowed.
func hasInkOnOppositeExteriorSides(binary gocv.Mat, rect image.Rectangle, cfg config.Exterior) bool {
	if binary.Empty() || rect.Empty() {
		return false
	}
	pad := max(cfg.MinStripDepthPx, min(rect.Dx(), rect.Dy()))
	mx, my := rect.Dx()/cfg.EndMarginDivisor, rect.Dy()/cfg.EndMarginDivisor
	strips := []image.Rectangle{
		image.Rect(rect.Min.X+mx, rect.Min.Y-pad, rect.Max.X-mx, rect.Min.Y),
		image.Rect(rect.Min.X+mx, rect.Max.Y, rect.Max.X-mx, rect.Max.Y+pad),
		image.Rect(rect.Min.X-pad, rect.Min.Y+my, rect.Min.X, rect.Max.Y-my),
		image.Rect(rect.Max.X, rect.Min.Y+my, rect.Max.X+pad, rect.Max.Y-my),
	}
	var dark [4]bool
	for i, strip := range strips {
		strip = strip.Intersect(image.Rect(0, 0, binary.Cols(), binary.Rows()))
		if strip.Empty() {
			continue
		}
		dark[i] = foregroundDensity(binary, strip) > cfg.DarkInkRatio
	}
	return (dark[0] && dark[1]) || (dark[2] && dark[3])
}

// matchesGeometry compares width and height independently, allowing the larger
// of the proportional tolerance and the minimum pixel tolerance for each axis.
func matchesGeometry(candidate Candidate, cluster GeometryCluster, tolerance float64, minTolerance int) bool {
	w, h := float64(candidate.Bounds.Dx()), float64(candidate.Bounds.Dy())
	return w > 0 && h > 0 &&
		math.Abs(w-cluster.Width) <= math.Max(float64(minTolerance), tolerance*cluster.Width) &&
		math.Abs(h-cluster.Height) <= math.Max(float64(minTolerance), tolerance*cluster.Height)
}

// learnCheckboxSizeClusters repeatedly selects the most-supported size group
// and uses median dimensions as its center. Isolated and weak secondary groups
// cannot establish a template. The input slice is not modified.
func learnCheckboxSizeClusters(ctx context.Context, seeds []Candidate, cfg config.Geometry) ([]GeometryCluster, error) {
	minSupport := cfg.MinClusterSupport
	remaining := append([]Candidate(nil), seeds...)
	var clusters []GeometryCluster
	for len(remaining) >= minSupport {
		var best []Candidate
		for _, seed := range remaining {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			center := GeometryCluster{Width: float64(seed.Bounds.Dx()), Height: float64(seed.Bounds.Dy())}
			var neighbors []Candidate
			for _, candidate := range remaining {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if matchesGeometry(candidate, center, cfg.ClusterToleranceRatio, cfg.MinTolerancePx) {
					neighbors = append(neighbors, candidate)
				}
			}
			if len(neighbors) > len(best) {
				best = neighbors
			}
		}
		if len(best) < minSupport || (len(clusters) > 0 && float64(len(best))/float64(clusters[0].Support) < cfg.MinSecondarySupportRatio) {
			break
		}
		widths, heights := make([]int, 0, len(best)), make([]int, 0, len(best))
		for _, candidate := range best {
			widths = append(widths, candidate.Bounds.Dx())
			heights = append(heights, candidate.Bounds.Dy())
		}
		cluster := GeometryCluster{Width: medianDimension(widths), Height: medianDimension(heights), Support: len(best)}
		clusters = append(clusters, cluster)
		var next []Candidate
		for _, candidate := range remaining {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !matchesGeometry(candidate, cluster, cfg.MatchToleranceRatio, cfg.MinTolerancePx) {
				next = append(next, candidate)
			}
		}
		remaining = next
	}
	return clusters, ctx.Err()
}

// medianDimension sorts a nonempty slice in place and returns its median.
// An even-length slice uses the mean of the two middle dimensions.
func medianDimension(values []int) float64 {
	sort.Ints(values)
	mid := len(values) / 2
	if len(values)%2 == 0 {
		return float64(values[mid-1]+values[mid]) / 2
	}
	return float64(values[mid])
}

// hasStrongBordersOnAllSides searches a narrow band normal to each side.
// Searching along a side would bridge gaps in letters and make them look like
// continuous checkbox borders. All four sides must meet the coverage threshold.
func hasStrongBordersOnAllSides(binary gocv.Mat, rect image.Rectangle, cfg config.ReferenceBorder) bool {
	if binary.Empty() || rect.Empty() || !rect.In(image.Rect(0, 0, binary.Cols(), binary.Rows())) {
		return false
	}
	band := max(cfg.MinSearchRadiusPx, min(rect.Dx(), rect.Dy())/cfg.SearchRadiusDivisor)
	coverage := func(horizontal bool, fixed, start, end int) float64 {
		ink := 0
		for along := start; along < end; along++ {
			for offset := -band; offset <= band; offset++ {
				x, y := along, fixed+offset
				if !horizontal {
					x, y = fixed+offset, along
				}
				if x >= 0 && y >= 0 && x < binary.Cols() && y < binary.Rows() && binary.GetUCharAt(y, x) > 0 {
					ink++
					break
				}
			}
		}
		return float64(ink) / float64(end-start)
	}
	return coverage(true, rect.Min.Y, rect.Min.X, rect.Max.X) >= cfg.MinCoverage &&
		coverage(true, rect.Max.Y-1, rect.Min.X, rect.Max.X) >= cfg.MinCoverage &&
		coverage(false, rect.Min.X, rect.Min.Y, rect.Max.Y) >= cfg.MinCoverage &&
		coverage(false, rect.Max.X-1, rect.Min.Y, rect.Max.Y) >= cfg.MinCoverage
}
