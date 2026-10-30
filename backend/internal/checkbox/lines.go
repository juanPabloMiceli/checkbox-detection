package checkbox

import (
	"context"
	"image"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// DetectLineCandidates finds axis-aligned checkbox borders.
//
// Input:
//   - binary: CV8UC1 image
//   - foreground ink: 255
//   - background: 0
//
// Output:
//   - candidate rectangles with four pixel-coordinate corners
func DetectLineCandidates(ctx context.Context, binary gocv.Mat, cfg config.Line, interior config.Interior) ([]Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if binary.Empty() {
		return nil, nil
	}

	// Extract horizontal and vertical structures.
	horizontal := extractLines(binary, image.Pt(cfg.HorizontalKernelLengthPx, 1))
	defer horizontal.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	vertical := extractLines(binary, image.Pt(1, cfg.VerticalKernelLengthPx))
	defer vertical.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Generate candidate boxes from horizontal-line components.
	contours := gocv.FindContours(
		horizontal,
		gocv.RetrievalList,
		gocv.ChainApproxSimple,
	)
	defer contours.Close()

	var candidates []Candidate

	for i := 0; i < contours.Size(); i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		contour := contours.At(i)
		rect := gocv.BoundingRect(contour)

		// Each horizontal line may represent the top or bottom
		// edge of a checkbox.
		width := rect.Dx()

		if width < cfg.MinWidthPx || width > cfg.MaxWidthPx {
			continue
		}

		y := rect.Min.Y + rect.Dy()/2

		// Look for a matching horizontal line below this one.
		// Width and height are independent; document geometry filters
		// implausible shapes after candidate generation.
		minHeight := cfg.MinEdgeSeparationPx
		maxHeight := cfg.MaxEdgeSeparationPx

		for height := minHeight; height <= maxHeight; height++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			bottomY := y + height

			if bottomY >= binary.Rows() {
				break
			}

			x0 := rect.Min.X
			x1 := rect.Max.X - 1

			if x0 < 0 || x1 >= binary.Cols() {
				continue
			}

			// Verify four sides independently.
			topScore := horizontalCoverage(
				horizontal, x0, x1, y,
			)

			bottomScore := horizontalCoverage(
				horizontal, x0, x1, bottomY,
			)

			leftScore := verticalCoverage(
				vertical, x0, y, bottomY,
			)

			rightScore := verticalCoverage(
				vertical, x1, y, bottomY,
			)

			// Require all four sides.
			if topScore < cfg.MinSideCoverage ||
				bottomScore < cfg.MinSideCoverage ||
				leftScore < cfg.MinSideCoverage ||
				rightScore < cfg.MinSideCoverage {
				continue
			}

			box := image.Rect(
				x0,
				y,
				x1+1,
				bottomY+1,
			)

			if !hasOpenInterior(binary, box, interior) {
				continue
			}
			corners := [4]image.Point{
				image.Pt(x0, y),
				image.Pt(x1, y),
				image.Pt(x1, bottomY),
				image.Pt(x0, bottomY),
			}

			candidates = append(candidates, Candidate{
				Bounds:  box,
				Corners: corners,
				Source:  SourceLine,
			})

			// Keep the first valid matching bottom edge.
			break
		}
	}

	return candidates, ctx.Err()
}

// extractLines applies morphological opening to retain
// horizontal or vertical structures.
func extractLines(binary gocv.Mat, size image.Point) gocv.Mat {
	kernel := gocv.GetStructuringElement(
		gocv.MorphRect,
		size,
	)
	defer kernel.Close()

	eroded := gocv.NewMat()
	defer eroded.Close()

	result := gocv.NewMat()

	gocv.Erode(binary, &eroded, kernel)
	gocv.Dilate(eroded, &result, kernel)

	return result
}

func horizontalCoverage(
	binary gocv.Mat,
	x0, x1, y int,
) float64 {
	if y < 0 || y >= binary.Rows() {
		return 0
	}

	count := 0
	total := 0

	for x := x0; x <= x1; x++ {
		if x < 0 || x >= binary.Cols() {
			continue
		}

		total++

		if binary.GetUCharAt(y, x) > 0 {
			count++
		}
	}

	if total == 0 {
		return 0
	}

	return float64(count) / float64(total)
}

func verticalCoverage(
	binary gocv.Mat,
	x, y0, y1 int,
) float64 {
	if x < 0 || x >= binary.Cols() {
		return 0
	}

	count := 0
	total := 0

	for y := y0; y <= y1; y++ {
		if y < 0 || y >= binary.Rows() {
			continue
		}

		total++

		if binary.GetUCharAt(y, x) > 0 {
			count++
		}
	}

	if total == 0 {
		return 0
	}

	return float64(count) / float64(total)
}

func hasOpenInterior(binary gocv.Mat, rect image.Rectangle, cfg config.Interior) bool {
	w, h := rect.Dx(), rect.Dy()

	if w < cfg.MinSidePx || h < cfg.MinSidePx {
		return false
	}

	marginX := max(cfg.MinMarginPx, w/cfg.MarginDivisor)
	marginY := max(cfg.MinMarginPx, h/cfg.MarginDivisor)

	x0 := rect.Min.X + marginX
	x1 := rect.Max.X - marginX
	y0 := rect.Min.Y + marginY
	y1 := rect.Max.Y - marginY

	if x0 >= x1 || y0 >= y1 {
		return false
	}

	return foregroundDensity(binary, image.Rect(x0, y0, x1, y1)) < cfg.MaxInkRatio
}
