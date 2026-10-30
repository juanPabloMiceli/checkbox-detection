package checkbox

import (
	"context"
	"image"
	"math"
	"sort"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// RejectionStats counts contours rejected by their first failing check.
type RejectionStats struct {
	Size           int `json:"size"`
	VertexCount    int `json:"vertex_count"`
	Convexity      int `json:"convexity"`
	Rectangularity int `json:"rectangularity"`
	Border         int `json:"border"`
}

func DetectCandidates(ctx context.Context, binary gocv.Mat, cfg config.Contour, diagnostics bool) (ContourResult, error) {
	var result ContourResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	contours := gocv.FindContours(
		binary,
		gocv.RetrievalList,
		gocv.ChainApproxSimple,
	)
	defer contours.Close()

	for i := 0; i < contours.Size(); i++ {
		if err := ctx.Err(); err != nil {
			return ContourResult{}, err
		}
		contour := contours.At(i)

		rect := gocv.BoundingRect(contour)
		w, h := rect.Dx(), rect.Dy()
		proposal := Candidate{Bounds: rect, Source: SourceContour}

		if w < cfg.MinSidePx || h < cfg.MinSidePx || w > cfg.MaxSidePx || h > cfg.MaxSidePx {
			result.Rejections.Size++
			recordRejection(&result.Rejected, diagnostics, proposal, RejectContourSize)
			continue
		}

		perimeter := gocv.ArcLength(contour, true)

		// Approximate the contour with a polygon.
		approx := gocv.ApproxPolyDP(
			contour,
			cfg.ApproximationEpsilonRatio*perimeter,
			true,
		)

		if approx.Size() != 4 {
			result.Rejections.VertexCount++
			recordRejection(&result.Rejected, diagnostics, proposal, RejectContourPolygon)
			approx.Close()
			continue
		}
		points := make([]image.Point, 4)
		for j := 0; j < 4; j++ {
			points[j] = approx.At(j)
		}
		approx.Close()

		if !isConvexQuad(points) {
			result.Rejections.Convexity++
			recordRejection(&result.Rejected, diagnostics, proposal, RejectContourConvexity)
			continue
		}

		// Reject contours that don't fill enough of their bounding rectangle.
		area := math.Abs(gocv.ContourArea(contour))
		boundingArea := float64(w * h)

		rectangularity := area / boundingArea
		proposal.Corners = orderCorners(points)
		proposal.Rectangularity = rectangularity

		if rectangularity < cfg.MinRectangularity {
			result.Rejections.Rectangularity++
			recordRejection(&result.Rejected, diagnostics, proposal, RejectContourRectangularity)
			continue
		}
		if !hasCheckboxBorder(binary, rect, cfg.Border) {
			result.Rejections.Border++
			recordRejection(&result.Rejected, diagnostics, proposal, RejectContourBorder)
			continue
		}
		result.Candidates = append(result.Candidates, proposal)
	}

	return result, ctx.Err()
}

func hasCheckboxBorder(binary gocv.Mat, rect image.Rectangle, cfg config.ContourBorder) bool {
	w, h := rect.Dx(), rect.Dy()

	if w < cfg.MinSidePx || h < cfg.MinSidePx {
		return false
	}

	// Search within a small band around each border.
	band := max(cfg.MinSearchRadiusPx, min(w, h)/cfg.SearchRadiusDivisor)
	margin := max(cfg.MinEndMarginPx, min(w, h)/cfg.EndMarginDivisor)

	hasInkNear := func(x, y int) bool {
		for dy := -band; dy <= band; dy++ {
			yy := y + dy

			if yy < 0 || yy >= binary.Rows() {
				continue
			}

			for dx := -band; dx <= band; dx++ {
				xx := x + dx

				if xx < 0 || xx >= binary.Cols() {
					continue
				}

				if binary.GetUCharAt(yy, xx) > 0 {
					return true
				}
			}
		}

		return false
	}

	horizontalCoverage := func(y int) float64 {
		ink, total := 0, 0

		for x := rect.Min.X + margin; x < rect.Max.X-margin; x++ {
			total++

			if hasInkNear(x, y) {
				ink++
			}
		}

		if total == 0 {
			return 0
		}

		return float64(ink) / float64(total)
	}

	verticalCoverage := func(x int) float64 {
		ink, total := 0, 0

		for y := rect.Min.Y + margin; y < rect.Max.Y-margin; y++ {
			total++

			if hasInkNear(x, y) {
				ink++
			}
		}

		if total == 0 {
			return 0
		}

		return float64(ink) / float64(total)
	}

	top := horizontalCoverage(rect.Min.Y)
	bottom := horizontalCoverage(rect.Max.Y - 1)
	left := verticalCoverage(rect.Min.X)
	right := verticalCoverage(rect.Max.X - 1)

	// Allow one damaged or partially missing border.
	sides := []float64{top, bottom, left, right}

	strongSides := 0

	for _, coverage := range sides {
		if coverage >= cfg.MinCoverage {
			strongSides++
		}
	}

	return strongSides >= cfg.MinStrongSides
}

func isConvexQuad(points []image.Point) bool {
	var sign int64

	for i := 0; i < 4; i++ {
		a := points[i]
		b := points[(i+1)%4]
		c := points[(i+2)%4]

		abx := int64(b.X - a.X)
		aby := int64(b.Y - a.Y)
		bcx := int64(c.X - b.X)
		bcy := int64(c.Y - b.Y)

		cross := abx*bcy - aby*bcx

		if cross == 0 {
			return false
		}

		if sign == 0 {
			sign = cross
		} else if (cross > 0) != (sign > 0) {
			return false
		}
	}

	return true
}

func orderCorners(points []image.Point) [4]image.Point {
	if len(points) != 4 {
		panic("orderCorners requires exactly four points")
	}

	ordered := append([]image.Point(nil), points...)

	var centerX, centerY float64
	for _, p := range ordered {
		centerX += float64(p.X)
		centerY += float64(p.Y)
	}

	centerX /= 4
	centerY /= 4

	sort.Slice(ordered, func(i, j int) bool {
		angleI := math.Atan2(
			float64(ordered[i].Y)-centerY,
			float64(ordered[i].X)-centerX,
		)
		angleJ := math.Atan2(
			float64(ordered[j].Y)-centerY,
			float64(ordered[j].X)-centerX,
		)

		return angleI < angleJ
	})

	// Start at the visually topmost point.
	// For ties, prefer the leftmost point.
	start := 0

	for i := 1; i < 4; i++ {
		if ordered[i].Y < ordered[start].Y ||
			(ordered[i].Y == ordered[start].Y &&
				ordered[i].X < ordered[start].X) {
			start = i
		}
	}

	var corners [4]image.Point

	for i := 0; i < 4; i++ {
		corners[i] = ordered[(start+i)%4]
	}

	return corners
}
