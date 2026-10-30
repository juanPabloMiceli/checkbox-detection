package checkbox

import (
	"image"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// IsChecked classifies a detected checkbox using the binary image.
// In this image, foreground ink is white (255).
func IsChecked(binary gocv.Mat, bounds image.Rectangle, cfg config.Classification) bool {
	w, h := bounds.Dx(), bounds.Dy()

	// Exclude the border before measuring interior ink.
	marginX := w / cfg.MarginDivisor
	marginY := h / cfg.MarginDivisor

	x0 := bounds.Min.X + marginX
	x1 := bounds.Max.X - marginX
	y0 := bounds.Min.Y + marginY
	y1 := bounds.Max.Y - marginY

	if x1 <= x0 || y1 <= y0 {
		return false
	}

	return foregroundDensity(binary, image.Rect(x0, y0, x1, y1)) > cfg.CheckedInkRatio
}
