package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"

	"gocv.io/x/gocv"
)

func TestDecodedImageOwnership(t *testing.T) {
	decodeErr := errors.New("native decoder failure")
	for _, test := range []struct {
		name   string
		create func() gocv.Mat
		err    error
		valid  bool
	}{
		{"valid", func() gocv.Mat { return gocv.NewMatWithSize(10, 10, gocv.MatTypeCV8UC3) }, nil, true},
		{"allocated with error", func() gocv.Mat { return gocv.NewMatWithSize(10, 10, gocv.MatTypeCV8UC3) }, decodeErr, false},
		{"empty with error", gocv.NewMat, decodeErr, false},
		{"empty without error", gocv.NewMat, nil, false},
		{"closed with error", func() gocv.Mat { return gocv.Mat{} }, decodeErr, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			img := test.create()
			defer func() {
				if !img.Closed() {
					img.Close()
				}
			}()
			err := validateDecodedImage(&img, test.err)
			if test.valid {
				if err != nil || img.Closed() || img.Empty() {
					t.Fatalf("valid matrix was not returned to its caller: %v", err)
				}
			} else if err == nil || !img.Closed() {
				t.Fatalf("decode failure must release its matrix: err=%v, closed=%v", err, img.Closed())
			}
			if test.err != nil && !errors.Is(err, test.err) {
				t.Fatalf("original decoder error was lost: %v", err)
			}
		})
	}
}

func TestDecodeRejectsTruncatedImage(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 100, 100))); err != nil {
		t.Fatal(err)
	}
	// A complete IHDR lets header inspection succeed, but has no image pixels.
	truncated := encoded.Bytes()[:33]
	if _, _, err := image.DecodeConfig(bytes.NewReader(truncated)); err != nil {
		t.Fatalf("fixture must pass header inspection: %v", err)
	}
	img, err := Decode(truncated)
	if err == nil || !img.Closed() {
		if !img.Closed() {
			img.Close()
		}
		t.Fatalf("truncated image must return an error and closed matrix: %v", err)
	}
}
