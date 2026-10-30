//go:build matprofile

package checkbox

import (
	"context"
	"errors"
	"testing"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

func TestServiceReleasesNativeMatrices(t *testing.T) {
	detector := newTestDetector(t, config.Default())
	data := whitePNG(t)
	probe := &stageContext{Context: context.Background()}
	if _, err := detector.Process(probe, data); err != nil {
		t.Fatal(err)
	}
	before := gocv.MatProfile.Count()
	for range 10 {
		if _, err := detector.Process(context.Background(), data); err != nil {
			t.Fatal(err)
		}
		if _, err := detector.Process(context.Background(), data[:33]); !errors.Is(err, ErrDecodeImage) {
			t.Fatal(err)
		}
		for stage := 3; stage <= probe.calls; stage++ {
			ctx := &stageContext{Context: context.Background(), cancelAt: stage}
			if _, err := detector.Process(ctx, data); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		}
		if after := gocv.MatProfile.Count(); after != before {
			t.Fatalf("native matrices leaked: before=%d, after=%d", before, after)
		}
	}
}
