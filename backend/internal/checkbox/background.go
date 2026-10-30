package checkbox

import (
	"context"
	"image"
	"math"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// BackgroundRejection records the evidence behind a rejected proposal. Side
// arrays are ordered top, bottom, left, right; contrast is measured in gray
// levels, with positive values indicating a dark outline on lighter paper.
type BackgroundRejection struct {
	Candidate       Candidate  `json:"candidate"`
	DarkFractions   [4]float64 `json:"dark_fractions"`
	BorderContrasts [4]float64 `json:"border_contrasts"`
}

type BackgroundStats struct {
	ReferenceBrightness float64               `json:"reference_brightness"`
	DarkThreshold       float64               `json:"dark_threshold"`
	Rejected            []BackgroundRejection `json:"rejected"`
	RejectedCount       int                   `json:"rejected_count"`
}

// FilterByBackground examines original-image brightness, which adaptive
// thresholding can erase in uniform dark bars. It never samples the checkbox
// interior. Dark surroundings alone are insufficient: a visible dark outline
// on a shadowed background is retained. Sampling and rejection precede final
// deduplication, and the original candidate coordinates are preserved. Source
// is an 8-bit grayscale, BGR, or BGRA image, as produced by image loading.
func FilterByBackground(ctx context.Context, source gocv.Mat, candidates []Candidate, cfg config.Background, diagnostics bool) ([]Candidate, BackgroundStats, error) {
	var stats BackgroundStats
	if err := ctx.Err(); err != nil {
		return nil, stats, err
	}
	if source.Empty() || len(candidates) == 0 {
		return candidates, stats, ctx.Err()
	}
	gray := gocv.NewMat()
	defer gray.Close()
	switch source.Channels() {
	case 1:
		source.CopyTo(&gray)
	case 3:
		gocv.CvtColor(source, &gray, gocv.ColorBGRToGray)
	case 4:
		gocv.CvtColor(source, &gray, gocv.ColorBGRAToGray)
	default:
		return candidates, stats, ctx.Err()
	}

	// A sparse page histogram estimates the light paper level. Relative
	// thresholds accommodate off-white paper and overall scan exposure.
	var histogram [256]int
	count := 0
	stride := max(1, int(math.Sqrt(float64(gray.Rows())*float64(gray.Cols())/float64(cfg.SampleBudget))))
	for y := 0; y < gray.Rows(); y += stride {
		if err := ctx.Err(); err != nil {
			return nil, BackgroundStats{}, err
		}
		for x := 0; x < gray.Cols(); x += stride {
			histogram[gray.GetUCharAt(y, x)]++
			count++
		}
	}
	stats.ReferenceBrightness = histogramQuantile(histogram, count, cfg.ReferenceQuantile)
	stats.DarkThreshold = stats.ReferenceBrightness * cfg.DarkRatio
	if stats.ReferenceBrightness == 0 {
		return candidates, stats, ctx.Err()
	}

	kept := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, BackgroundStats{}, err
		}
		evidence, reject := inspectBackground(gray, candidate, stats, cfg)
		if reject {
			stats.RejectedCount++
			if diagnostics {
				stats.Rejected = append(stats.Rejected, evidence)
			}
		} else {
			kept = append(kept, candidate)
		}
	}
	return kept, stats, ctx.Err()
}

func inspectBackground(gray gocv.Mat, candidate Candidate, stats BackgroundStats, cfg config.Background) (BackgroundRejection, bool) {
	evidence := BackgroundRejection{Candidate: candidate}
	rect := candidate.Bounds
	canvas := image.Rect(0, 0, gray.Cols(), gray.Rows())
	if rect.Empty() || !rect.In(canvas) {
		return evidence, false
	}
	s := min(rect.Dx(), rect.Dy())
	depth := max(cfg.MinStripDepthPx, s)
	gap := max(cfg.MinGapPx, s/cfg.GapDivisor)
	mx, my := rect.Dx()/cfg.EndMarginDivisor, rect.Dy()/cfg.EndMarginDivisor
	strips := [4]image.Rectangle{
		image.Rect(rect.Min.X+mx, rect.Min.Y-gap-depth, rect.Max.X-mx, rect.Min.Y-gap),
		image.Rect(rect.Min.X+mx, rect.Max.Y+gap, rect.Max.X-mx, rect.Max.Y+gap+depth),
		image.Rect(rect.Min.X-gap-depth, rect.Min.Y+my, rect.Min.X-gap, rect.Max.Y-my),
		image.Rect(rect.Max.X+gap, rect.Min.Y+my, rect.Max.X+gap+depth, rect.Max.Y-my),
	}
	var darkWithoutOutline [4]bool
	for side, fullStrip := range strips {
		strip := fullStrip.Intersect(canvas)
		if strip.Empty() || float64(rectangleArea(strip)) < cfg.MinVisibleRatio*float64(rectangleArea(fullStrip)) {
			continue // Missing evidence near a crop boundary does not reject a box.
		}
		var histogram [256]int
		dark, total := 0, rectangleArea(strip)
		for y := strip.Min.Y; y < strip.Max.Y; y++ {
			for x := strip.Min.X; x < strip.Max.X; x++ {
				value := gray.GetUCharAt(y, x)
				histogram[value]++
				if float64(value) < stats.DarkThreshold {
					dark++
				}
			}
		}
		background := histogramQuantile(histogram, total, 0.5)
		border := borderBrightness(gray, rect, side, cfg)
		evidence.DarkFractions[side] = float64(dark) / float64(total)
		evidence.BorderContrasts[side] = background - border
		darkWithoutOutline[side] = evidence.DarkFractions[side] >= cfg.MinDarkFraction &&
			evidence.BorderContrasts[side] < cfg.MinContrastRatio*stats.ReferenceBrightness
	}
	reject := (darkWithoutOutline[0] && darkWithoutOutline[1]) || (darkWithoutOutline[2] && darkWithoutOutline[3])
	return evidence, reject
}

// Find the darkest pixel in a narrow perpendicular band at each edge position,
// then take their median. This tolerates inner/outer border measurements and
// resists a few extending checkmark strokes or adjacent characters.
func borderBrightness(gray gocv.Mat, rect image.Rectangle, side int, cfg config.Background) float64 {
	band := max(cfg.BorderMinSearchRadiusPx, min(rect.Dx(), rect.Dy())/cfg.BorderSearchRadiusDivisor)
	horizontal := side < 2
	start, end, fixed := rect.Min.X+rect.Dx()/cfg.EndMarginDivisor, rect.Max.X-rect.Dx()/cfg.EndMarginDivisor, rect.Min.Y
	if side == 1 {
		fixed = rect.Max.Y - 1
	} else if !horizontal {
		start, end, fixed = rect.Min.Y+rect.Dy()/cfg.EndMarginDivisor, rect.Max.Y-rect.Dy()/cfg.EndMarginDivisor, rect.Min.X
		if side == 3 {
			fixed = rect.Max.X - 1
		}
	}
	var histogram [256]int
	for along := start; along < end; along++ {
		value := uint8(255)
		for offset := -band; offset <= band; offset++ {
			x, y := along, fixed+offset
			if !horizontal {
				x, y = fixed+offset, along
			}
			if x >= 0 && y >= 0 && x < gray.Cols() && y < gray.Rows() {
				value = min(value, gray.GetUCharAt(y, x))
			}
		}
		histogram[value]++
	}
	return histogramQuantile(histogram, end-start, 0.5)
}

func histogramQuantile(histogram [256]int, count int, quantile float64) float64 {
	target, cumulative := max(1, int(math.Ceil(float64(count)*quantile))), 0
	for value, frequency := range histogram {
		cumulative += frequency
		if cumulative >= target {
			return float64(value)
		}
	}
	return 0
}
