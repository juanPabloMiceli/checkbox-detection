package checkbox

import (
	"image"

	"gocv.io/x/gocv"
)

// foregroundDensity borrows an 8-bit binary image and an in-image rectangle.
// Callers choose their own margins and strict/inclusive threshold comparison.
func foregroundDensity(binary gocv.Mat, bounds image.Rectangle) float64 {
	if bounds.Empty() {
		return 0
	}
	ink := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if binary.GetUCharAt(y, x) > 0 {
				ink++
			}
		}
	}
	return float64(ink) / float64(rectangleArea(bounds))
}
