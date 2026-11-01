package checkbox

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"checkbox-detector/internal/config"
)

func whitePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func newTestDetector(t *testing.T, cfg config.Config) *Detector {
	t.Helper()
	detector, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return detector
}

func TestDetectorCopiesConfigAndSupportsIndependentInstances(t *testing.T) {
	data, err := os.ReadFile("../../testdata/im3.png")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	ordinary := newTestDetector(t, cfg)
	cfg.Classification.CheckedInkRatio = 1
	strict := newTestDetector(t, cfg)
	cfg.Preprocess.ThresholdBlockSizePx = 0 // Cannot mutate either constructed detector.
	expected, err := ordinary.Process(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(expected.Boxes) == 0 {
		t.Fatal("sample must contain detections to compare")
	}
	for name, detector := range map[string]*Detector{"ordinary": ordinary, "strict": strict} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := detector.Process(context.Background(), data)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Boxes) != len(expected.Boxes) {
				t.Fatal("classification setting changed candidate geometry")
			}
			for i, box := range result.Boxes {
				wantChecked := expected.Boxes[i].Checked && name == "ordinary"
				if box.BBox != expected.Boxes[i].BBox || box.Checked != wantChecked {
					t.Fatalf("box %d: %+v", i, box)
				}
			}
		})
	}
}

func TestConstructorRejectsInvalidConfiguration(t *testing.T) {
	cfg := config.Default()
	cfg.Contour.MinSidePx = 0
	if detector, err := New(cfg); err == nil || detector != nil {
		t.Fatal("invalid config constructed a detector")
	}
}

func TestServiceValidatesImagesBeforeDecoding(t *testing.T) {
	detector := newTestDetector(t, config.Default())
	if _, err := detector.Process(context.Background(), []byte("invalid")); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("got %v", err)
	}
	data := whitePNG(t)
	if _, err := detector.Process(context.Background(), data[:33]); !errors.Is(err, ErrDecodeImage) {
		t.Fatalf("got %v", err)
	}
	// A huge but syntactically valid PNG header must be rejected before native
	// decoding. No image allocation is needed to exercise the size guard.
	large := bytes.Clone(data[:33])
	binary.BigEndian.PutUint32(large[16:20], 10_000)
	binary.BigEndian.PutUint32(large[20:24], 10_000)
	binary.BigEndian.PutUint32(large[29:33], crc32.ChecksumIEEE(large[12:29]))
	_, err := detector.Process(context.Background(), large)
	var sizeError *ImageSizeError
	if !errors.As(err, &sizeError) || sizeError.MaxPixels != config.Default().MaxImagePixels {
		t.Fatalf("want pixel limit error, got %v", err)
	}
	cfg := config.Default()
	cfg.MaxImagePixels = 9_999
	if _, err := newTestDetector(t, cfg).Process(context.Background(), data); !errors.As(err, &sizeError) {
		t.Fatalf("custom pixel limit was ignored: %v", err)
	}
}

func TestServiceReturnsEmptyBoxesForBlankImage(t *testing.T) {
	result, err := newTestDetector(t, config.Default()).Process(context.Background(), whitePNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Boxes == nil || len(result.Boxes) != 0 {
		t.Fatal("blank image must produce a nonnil empty boxes array")
	}
}

// Cancel deterministically at a stage boundary, including after native work.
type stageContext struct {
	context.Context
	calls, cancelAt int
}

func (c *stageContext) Err() error {
	c.calls++
	if c.cancelAt > 0 && c.calls >= c.cancelAt {
		return context.Canceled
	}
	return nil
}

func TestServiceChecksCancellationBetweenStages(t *testing.T) {
	detector := newTestDetector(t, config.Default())
	data := whitePNG(t)
	probe := &stageContext{Context: context.Background()}
	if _, err := detector.Process(probe, data); err != nil {
		t.Fatal(err)
	}
	for stage := 1; stage <= probe.calls; stage++ {
		ctx := &stageContext{Context: context.Background(), cancelAt: stage}
		result, err := detector.Process(ctx, data)
		if !errors.Is(err, context.Canceled) || result.Boxes != nil {
			t.Fatalf("stage %d returned partial results or wrong error: %v", stage, err)
		}
	}
}
