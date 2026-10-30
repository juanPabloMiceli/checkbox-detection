package checkbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"checkbox-detector/internal/config"
	"checkbox-detector/internal/imaging"
)

var (
	ErrInvalidImage = errors.New("image must be a valid PNG or JPEG")
	ErrDecodeImage  = errors.New("could not decode image")
)

type ImageSizeError struct{ MaxPixels int64 }

func (e *ImageSizeError) Error() string {
	return fmt.Sprintf("image exceeds the %g megapixel limit", float64(e.MaxPixels)/1_000_000)
}

// Detector keeps an immutable copy of validated settings. Requests share no
// matrices or other mutable processing state.
type Detector struct{ config config.Config }

func New(cfg config.Config) (*Detector, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid detector configuration: %w", err)
	}
	return &Detector{config: cfg}, nil
}

// Process is the image-to-result entry point used by the HTTP server. Context
// cancellation is checked between stages; it cannot interrupt an OpenCV call.
// All native allocations are closed before it returns plain Go detections.
func (d *Detector) Process(ctx context.Context, data []byte) (DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}
	dimensions, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return DetectionResult{}, ErrInvalidImage
	}
	// Division avoids overflow even with extreme dimensions in an image header.
	if dimensions.Width <= 0 || dimensions.Height <= 0 || int64(dimensions.Width) > d.config.MaxImagePixels/int64(dimensions.Height) {
		return DetectionResult{}, &ImageSizeError{MaxPixels: d.config.MaxImagePixels}
	}
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}
	source, err := imaging.Decode(data)
	if err != nil {
		return DetectionResult{}, fmt.Errorf("%w: %v", ErrDecodeImage, err)
	}
	defer source.Close()
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}
	binary := imaging.Preprocess(source, d.config.Preprocess)
	defer binary.Close()
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}
	pipeline, err := Detect(ctx, source, binary, d.config, false)
	if err != nil {
		return DetectionResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}
	result := DetectionResult{Boxes: ToDetections(pipeline.Candidates, binary, d.config.Classification)}
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}
	return result, nil
}

// DetectorConfig returns a value copy; HTTP overrides never modify shared defaults.
func (d *Detector) DetectorConfig() config.Config { return d.config }

// ProcessWithSettings runs detection with an isolated configuration copy, so
// concurrent requests cannot alter each other's settings or the server defaults.
func (d *Detector) ProcessWithSettings(ctx context.Context, data []byte, settings config.DetectionSettings) (DetectionResult, error) {
	cfg, err := d.config.ApplySettings(settings)
	if err != nil {
		return DetectionResult{}, err
	}
	requestDetector := Detector{config: cfg}
	return requestDetector.Process(ctx, data)
}
