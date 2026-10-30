//go:build matprofile

package imaging

import (
	"errors"
	"testing"

	"gocv.io/x/gocv"
)

func TestDecoderErrorReleasesNativeAllocations(t *testing.T) {
	before := gocv.MatProfile.Count()
	for range 100 {
		img := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
		if err := validateDecodedImage(&img, errors.New("native failure after allocation")); err == nil {
			t.Fatal("expected decoder error")
		}
	}
	if after := gocv.MatProfile.Count(); after != before {
		t.Fatalf("native matrices leaked on decoder errors: before=%d, after=%d", before, after)
	}
}
