package checkbox

import (
	"context"
	"image"
	"sort"
)

func intersectionOverUnion(a, b image.Rectangle) float64 {
	intersection := a.Intersect(b)

	if intersection.Empty() {
		return 0
	}

	intersectionArea := intersection.Dx() * intersection.Dy()

	areaA := a.Dx() * a.Dy()
	areaB := b.Dx() * b.Dy()

	unionArea := areaA + areaB - intersectionArea

	if unionArea <= 0 {
		return 0
	}

	return float64(intersectionArea) / float64(unionArea)
}

// Prefer larger bounding boxes, which generally keeps outer borders. Run
// geometry/background filtering first so oversized outliers cannot suppress boxes.
func deduplicate(ctx context.Context, candidates []Candidate, iouThreshold float64, diagnostics bool) (DedupResult, error) {
	var result DedupResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if len(candidates) == 0 {
		return result, nil
	}
	sorted := append([]Candidate(nil), candidates...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if rectangleArea(a.Bounds) != rectangleArea(b.Bounds) {
			return rectangleArea(a.Bounds) > rectangleArea(b.Bounds)
		}
		return candidateLess(a, b)
	})
	for _, candidate := range sorted {
		if err := ctx.Err(); err != nil {
			return DedupResult{}, err
		}
		duplicate := false
		for _, existing := range result.Candidates {
			if err := ctx.Err(); err != nil {
				return DedupResult{}, err
			}
			if intersectionOverUnion(candidate.Bounds, existing.Bounds) >= iouThreshold {
				duplicate = true
				break
			}
		}
		if duplicate {
			result.DuplicateCount++
			recordRejection(&result.Rejected, diagnostics, candidate, RejectDuplicate)
		} else {
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	sort.Slice(result.Candidates, func(i, j int) bool { return candidateLess(result.Candidates[i], result.Candidates[j]) })
	return result, ctx.Err()
}

// Equal-area proposals use a total geometric order, then source and measured
// corners. The winner no longer depends on sort internals or input permutation.
func candidateLess(a, b Candidate) bool {
	av := [...]int{a.Bounds.Min.Y, a.Bounds.Min.X, a.Bounds.Max.Y, a.Bounds.Max.X, int(a.Source)}
	bv := [...]int{b.Bounds.Min.Y, b.Bounds.Min.X, b.Bounds.Max.Y, b.Bounds.Max.X, int(b.Source)}
	for i := range av {
		if av[i] != bv[i] {
			return av[i] < bv[i]
		}
	}
	for i := range a.Corners {
		if a.Corners[i].Y != b.Corners[i].Y {
			return a.Corners[i].Y < b.Corners[i].Y
		}
		if a.Corners[i].X != b.Corners[i].X {
			return a.Corners[i].X < b.Corners[i].X
		}
	}
	return a.Rectangularity > b.Rectangularity
}

func rectangleArea(rect image.Rectangle) int { return rect.Dx() * rect.Dy() }
