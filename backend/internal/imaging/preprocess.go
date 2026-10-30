package imaging

import (
	"image"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

// Preprocess converts a document image into a binary image
// with dark foreground pixels represented as white.
func Preprocess(src gocv.Mat, cfg config.Preprocess) gocv.Mat {
	gray := gocv.NewMat()
	defer gray.Close()

	if src.Channels() == 1 {
		src.CopyTo(&gray)
	} else {
		gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	}

	blurred := gocv.NewMat()
	defer blurred.Close()

	gocv.GaussianBlur(
		gray,
		&blurred,
		image.Pt(cfg.BlurKernelWidthPx, cfg.BlurKernelHeightPx),
		cfg.BlurSigmaX,
		cfg.BlurSigmaY,
		gocv.BorderDefault,
	)

	binary := gocv.NewMat()

	gocv.AdaptiveThreshold(
		blurred,
		&binary,
		255,
		gocv.AdaptiveThresholdGaussian,
		gocv.ThresholdBinaryInv,
		cfg.ThresholdBlockSizePx,
		float32(cfg.ThresholdOffset),
	)

	return binary
}
