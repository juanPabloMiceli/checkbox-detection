package checkbox

import (
	"context"
	"image"
	"image/color"
	"testing"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

func testCandidate(x, y, w, h int) Candidate {
	return Candidate{
		Bounds:         image.Rect(x, y, x+w, y+h),
		Source:         SourceContour,
		Rectangularity: 1,
		Corners: [4]image.Point{
			image.Pt(x, y), image.Pt(x+w-1, y),
			image.Pt(x+w-1, y+h-1), image.Pt(x, y+h-1),
		},
	}
}

func drawTestBox(binary *gocv.Mat, candidate Candidate) {
	for i, point := range candidate.Corners {
		gocv.Line(binary, point, candidate.Corners[(i+1)%4], color.RGBA{255, 255, 255, 255}, 1)
	}
}

func TestDocumentGeometryKeepsRectangularComponentsAndMarks(t *testing.T) {
	binary := gocv.NewMatWithSize(700, 700, gocv.MatTypeCV8U)
	defer binary.Close()
	var references []Candidate
	for i := 0; i < 4; i++ {
		// Two distinct components with opposite orientations.
		references = append(references, testCandidate(30+i*130, 30, 60, 24), testCandidate(30+i*130, 150, 24, 60))
	}
	for _, candidate := range references {
		drawTestBox(&binary, candidate)
	}
	checked := testCandidate(30, 300, 60, 24)
	drawTestBox(&binary, checked)
	gocv.Line(&binary, checked.Corners[0], checked.Corners[2], color.RGBA{255, 255, 255, 255}, 2)
	filled := testCandidate(160, 300, 60, 24)
	gocv.Rectangle(&binary, filled.Bounds, color.RGBA{255, 255, 255, 255}, -1)
	all := append(append([]Candidate(nil), references...), checked, filled)
	// A large, repeated population from a weaker detector must not vote on
	// document geometry, even if its aspect ratio matches the real boxes.
	for i := 0; i < 80; i++ {
		noise := testCandidate(20+i%10*60, 400+i/10*30, 30, 12)
		drawTestBox(&binary, noise)
		all = append(all, noise)
	}
	filtered, clusters := testGeometry(t, all, references, binary, config.Default())
	if len(clusters) != 2 || len(filtered) != 10 {
		t.Fatalf("want both components and all 10 boxes, got %d clusters and %d boxes: %+v", len(clusters), len(filtered), clusters)
	}
	for i, candidate := range filtered {
		if candidate != all[i] {
			t.Fatalf("candidate %d changed, including bounds or corners", i)
		}
	}
}

func TestDocumentGeometryFallsBackWithoutIndependentReferences(t *testing.T) {
	binary := gocv.NewMatWithSize(200, 200, gocv.MatTypeCV8U)
	defer binary.Close()
	box := testCandidate(30, 30, 60, 24)
	drawTestBox(&binary, box)
	all := []Candidate{box, testCandidate(110, 100, 24, 60)}
	for _, refs := range [][]Candidate{nil, {box}, {box, box, box, box, box}} {
		filtered, clusters := testGeometry(t, all, refs, binary, config.Default())
		if len(clusters) != 0 || len(filtered) != len(all) {
			t.Fatalf("insufficient independent evidence must pass through: got %+v, %d boxes", clusters, len(filtered))
		}
	}
	empty, clusters := testGeometry(t, nil, nil, binary, config.Default())
	if len(empty) != 0 || len(clusters) != 0 {
		t.Fatal("empty input must stay empty")
	}
}

func TestDocumentGeometryRejectsDarkBarButKeepsDenseRows(t *testing.T) {
	binary := gocv.NewMatWithSize(400, 500, gocv.MatTypeCV8U)
	defer binary.Close()
	var boxes []Candidate
	for row := 0; row < 5; row++ {
		box := testCandidate(100, 30+row*35, 40, 30)
		drawTestBox(&binary, box)
		boxes = append(boxes, box)
	}
	// A long dark section bar with a white rectangular text counter.
	gocv.Rectangle(&binary, image.Rect(300, 0, 340, 350), color.RGBA{255, 255, 255, 255}, -1)
	gocv.Rectangle(&binary, image.Rect(303, 104, 337, 126), color.RGBA{}, -1)
	falseBox := testCandidate(300, 100, 40, 30)
	all := append(append([]Candidate(nil), boxes...), falseBox)
	filtered, _ := testGeometry(t, all, boxes, binary, config.Default())
	if len(filtered) != len(boxes) {
		t.Fatalf("want all %d dense-row boxes and no dark-bar text, got %d", len(boxes), len(filtered))
	}
	for i := range boxes {
		if filtered[i] != boxes[i] {
			t.Fatalf("lost real box %d", i)
		}
	}
}

func TestBothDetectorsAllowLandscapeAndPortraitBoxes(t *testing.T) {
	binary := gocv.NewMatWithSize(300, 500, gocv.MatTypeCV8U)
	defer binary.Close()
	want := []Candidate{testCandidate(40, 40, 60, 24), testCandidate(200, 40, 24, 60)}
	for _, box := range want {
		drawTestBox(&binary, box)
	}
	contours := testContours(t, binary, config.Default().Contour)
	for name, candidates := range map[string][]Candidate{"contours": contours, "lines": testLines(t, binary, config.Default().Line, config.Default().Interior)} {
		for _, box := range want {
			found := false
			for _, candidate := range candidates {
				if intersectionOverUnion(box.Bounds, candidate.Bounds) > 0.8 {
					found = true
				}
			}
			if !found {
				t.Errorf("%s missed rectangular box %v; candidates: %+v", name, box.Bounds, candidates)
			}
		}
	}
}

func TestGeometryFilteringBeforeDedupPreservesCorrectBox(t *testing.T) {
	binary := gocv.NewMatWithSize(300, 500, gocv.MatTypeCV8U)
	defer binary.Close()
	var references []Candidate
	for i := 0; i < 3; i++ {
		box := testCandidate(30+i*120, 30, 40, 30)
		drawTestBox(&binary, box)
		references = append(references, box)
	}
	real := testCandidate(30, 150, 40, 30)
	drawTestBox(&binary, real)
	oversized := testCandidate(30, 137, 40, 43)
	all := append(append([]Candidate(nil), references...), oversized, real)
	filtered, _ := testGeometry(t, all, references, binary, config.Default())
	unique, err := deduplicate(context.Background(), filtered, 0.5, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(unique.Candidates) != 4 || unique.Candidates[3] != real {
		t.Fatalf("larger outlier must not suppress the correct box: %+v", unique)
	}
}

func TestOnlyContourSourcesCanTeachDocumentGeometry(t *testing.T) {
	for _, source := range []CandidateSource{SourceUnknown, SourceLine, SourceContour} {
		binary := gocv.NewMatWithSize(150, 500, gocv.MatTypeCV8U)
		var candidates []Candidate
		for i := range 3 {
			candidate := testCandidate(30+i*120, 30, 40, 30)
			candidate.Source = source
			drawTestBox(&binary, candidate)
			candidates = append(candidates, candidate)
		}
		_, clusters := testGeometry(t, candidates, candidates, binary, config.Default())
		binary.Close()
		if (len(clusters) > 0) != (source == SourceContour) {
			t.Fatalf("source %d taught unexpected geometry: %+v", source, clusters)
		}
	}
}
