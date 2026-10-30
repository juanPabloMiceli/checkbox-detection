package checkbox

import (
	"image"
	"image/color"
	"testing"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

func backgroundTestPage(paper uint8) gocv.Mat {
	page := gocv.NewMatWithSize(300, 400, gocv.MatTypeCV8UC3)
	v := float64(paper)
	page.SetTo(gocv.NewScalar(v, v, v, 0))
	return page
}

func TestBackgroundPreservesShadedAndMarkedCheckboxes(t *testing.T) {
	for _, background := range []struct {
		name  string
		color color.RGBA
	}{
		{"white", color.RGBA{255, 255, 255, 255}},
		{"off-white", color.RGBA{235, 228, 210, 255}},
		{"gray", color.RGBA{160, 160, 160, 255}},
		{"blue", color.RGBA{100, 160, 220, 255}},
		{"deep shadow", color.RGBA{40, 40, 40, 255}},
	} {
		for _, marked := range []bool{false, true} {
			t.Run(background.name+map[bool]string{false: "/unchecked", true: "/filled"}[marked], func(t *testing.T) {
				page := backgroundTestPage(255)
				defer page.Close()
				gocv.Rectangle(&page, image.Rect(50, 50, 200, 200), background.color, -1)
				candidate := testCandidate(100, 100, 40, 30)
				gocv.Rectangle(&page, candidate.Bounds, color.RGBA{}, 2)
				if marked {
					gocv.Rectangle(&page, candidate.Bounds.Inset(3), color.RGBA{}, -1)
				}
				kept, stats := testBackground(t, page, []Candidate{candidate}, config.Default().Background)
				if len(kept) != 1 || kept[0] != candidate || len(stats.Rejected) != 0 {
					t.Fatalf("lost a genuine checkbox or changed its coordinates: kept=%+v, stats=%+v", kept, stats)
				}
			})
		}
	}
}

func TestBackgroundRejectsLetterCountersAcrossScanExposures(t *testing.T) {
	for _, paper := range []uint8{255, 160, 90} {
		page := backgroundTestPage(paper)
		candidate := testCandidate(100, 100, 40, 30)
		// A pale rectangular counter inside a long dark section label.
		gocv.Rectangle(&page, image.Rect(100, 0, 140, 250), color.RGBA{}, -1)
		gocv.Rectangle(&page, candidate.Bounds.Inset(3), color.RGBA{paper, paper, paper, 255}, -1)
		kept, stats := testBackground(t, page, []Candidate{candidate}, config.Default().Background)
		page.Close()
		if len(kept) != 0 || len(stats.Rejected) != 1 || stats.Rejected[0].Candidate != candidate {
			t.Fatalf("paper brightness %d: dark section-bar counter survived: %+v", paper, stats)
		}
		if stats.ReferenceBrightness != float64(paper) {
			t.Fatalf("want reference brightness %d, got %.0f", paper, stats.ReferenceBrightness)
		}
	}
}

func TestBackgroundPreservesBoxesBesideRulesAndDenseRows(t *testing.T) {
	page := backgroundTestPage(255)
	defer page.Close()
	var candidates []Candidate
	for i := 0; i < 5; i++ {
		candidate := testCandidate(100, 30+i*36, 40, 30)
		gocv.Rectangle(&page, candidate.Bounds, color.RGBA{}, 2)
		gocv.Line(&page, candidate.Corners[0], candidate.Corners[2], color.RGBA{}, 2)
		candidates = append(candidates, candidate)
	}
	// One strongly dark adjoining side must not reject the row of checkboxes.
	gocv.Rectangle(&page, image.Rect(140, 0, 190, 250), color.RGBA{}, -1)
	kept, stats := testBackground(t, page, candidates, config.Default().Background)
	if len(kept) != len(candidates) || len(stats.Rejected) != 0 {
		t.Fatalf("lost checkbox beside a dark rule or neighboring marks: %+v", stats)
	}
}

func TestBackgroundHandlesCroppedAndMissingEvidence(t *testing.T) {
	page := backgroundTestPage(255)
	defer page.Close()
	candidates := []Candidate{testCandidate(0, 0, 40, 30), testCandidate(360, 270, 40, 30)}
	for _, candidate := range candidates {
		gocv.Rectangle(&page, candidate.Bounds, color.RGBA{}, 2)
	}
	kept, stats := testBackground(t, page, candidates, config.Default().Background)
	if len(kept) != 2 || len(stats.Rejected) != 0 {
		t.Fatalf("crop boundaries removed real boxes: %+v", stats)
	}
	empty := gocv.NewMat()
	defer empty.Close()
	kept, _ = testBackground(t, empty, candidates, config.Default().Background)
	if len(kept) != 2 {
		t.Fatal("missing source evidence must preserve candidates")
	}
	if kept, _ := testBackground(t, page, nil, config.Default().Background); len(kept) != 0 {
		t.Fatal("empty candidate list must remain empty")
	}
	black := backgroundTestPage(0)
	defer black.Close()
	if kept, _ := testBackground(t, black, candidates, config.Default().Background); len(kept) != 2 {
		t.Fatal("a page with no light reference must preserve candidates")
	}
}

func TestBackgroundSupportsGrayscaleAndBGRA(t *testing.T) {
	for _, matType := range []gocv.MatType{gocv.MatTypeCV8U, gocv.MatTypeCV8UC4} {
		page := gocv.NewMatWithSize(150, 150, matType)
		page.SetTo(gocv.NewScalar(255, 255, 255, 255))
		candidate := testCandidate(50, 50, 40, 30)
		gocv.Rectangle(&page, candidate.Bounds, color.RGBA{A: 255}, 2)
		kept, stats := testBackground(t, page, []Candidate{candidate}, config.Default().Background)
		page.Close()
		if len(kept) != 1 || len(stats.Rejected) != 0 {
			t.Fatalf("unsupported image layout %v: %+v", matType, stats)
		}
	}
}
