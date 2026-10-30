package checkbox

import (
	"context"
	"image"
	"testing"

	"checkbox-detector/internal/config"
	"checkbox-detector/internal/imaging"

	"gocv.io/x/gocv"
)

func TestRelaxedContoursRecoverMarkedCheckboxes(t *testing.T) {
	img := gocv.IMRead("../../testdata/im3.png", gocv.IMReadColor)
	if img.Empty() {
		t.Fatal("failed to load im3 fixture")
	}
	defer img.Close()
	binary := imaging.Preprocess(img, config.Default().Preprocess)
	defer binary.Close()
	result, err := Detect(context.Background(), img, binary, config.Default(), false)
	if err != nil {
		t.Fatal(err)
	}
	// Evaluation annotations for genuinely checked boxes previously rejected
	// by rectangularity. These coordinates are not used by the detector.
	for name, bounds := range map[string]image.Rectangle{
		"occupant owner":          image.Rect(456, 398, 488, 430),
		"offered for sale no":     image.Rect(2023, 593, 2055, 624),
		"electricity private":     image.Rect(2143, 1966, 2175, 1997),
		"utilities typical yes":   image.Rect(1126, 2083, 1158, 2114),
		"adequate vehicle access": image.Rect(1005, 2239, 1037, 2270),
		"adverse conditions no":   image.Rect(1867, 2473, 1899, 2504),
	} {
		found := false
		for _, candidate := range result.Candidates {
			if intersectionOverUnion(candidate.Bounds, bounds) >= 0.8 && IsChecked(binary, candidate.Bounds, config.Default().Classification) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missed checked checkbox: %s", name)
		}
	}
}

func TestRelaxedContoursRejectLabelBetweenCheckboxes(t *testing.T) {
	img := gocv.IMRead("../../testdata/im0.png", gocv.IMReadColor)
	if img.Empty() {
		t.Fatal("failed to load im0 fixture")
	}
	defer img.Close()
	binary := imaging.Preprocess(img, config.Default().Preprocess)
	defer binary.Close()
	result, err := Detect(context.Background(), img, binary, config.Default(), false)
	if err != nil {
		t.Fatal(err)
	}
	// The Att. label between two real boxes formed a convincing contour when
	// rectangularity was loosened without tighter document geometry support.
	textCell := image.Rect(433, 2757, 490, 2807)
	for _, candidate := range result.Candidates {
		if intersectionOverUnion(candidate.Bounds, textCell) >= 0.5 {
			t.Fatalf("mistook text between neighboring boxes for a checkbox: %v", candidate.Bounds)
		}
	}
}

func TestRecoveryCandidatesCannotTeachTheirOwnGeometry(t *testing.T) {
	binary := gocv.NewMatWithSize(500, 700, gocv.MatTypeCV8U)
	defer binary.Close()
	var references []Candidate
	for i := 0; i < 3; i++ {
		box := testCandidate(30+i*120, 30, 40, 30)
		box.Rectangularity = 0.95
		drawTestBox(&binary, box)
		references = append(references, box)
	}
	good := testCandidate(30, 130, 41, 31)
	good.Rectangularity = 0.88
	drawTestBox(&binary, good)
	all := append(append([]Candidate(nil), references...), good)
	// More weak outliers than strict references. They fall inside the old 20%
	// acceptance tolerance but outside the recovery tolerance in height.
	for i := 0; i < 6; i++ {
		box := testCandidate(30+i*100, 250, 43, 35)
		box.Rectangularity = 0.88
		drawTestBox(&binary, box)
		references = append(references, box)
		all = append(all, box)
	}
	filtered, clusters := testGeometry(t, all, references, binary, config.Default())
	if len(clusters) != 1 || clusters[0].Support != 3 || clusters[0].Width != 40 || clusters[0].Height != 30 {
		t.Fatalf("weak candidates changed the learned geometry: %+v", clusters)
	}
	if len(filtered) != 4 || filtered[3] != good {
		t.Fatalf("want the strict references and supported weak candidate only, got %+v", filtered)
	}
	withoutModel, clusters := testGeometry(t, []Candidate{good, all[0]}, []Candidate{good}, binary, config.Default())
	if len(clusters) != 0 || len(withoutModel) != 1 || withoutModel[0] != all[0] {
		t.Fatalf("unsupported recovery contour survived without a model: %+v", withoutModel)
	}
}

func TestWeakOuterContourCannotSuppressLearningReference(t *testing.T) {
	binary := gocv.NewMatWithSize(200, 500, gocv.MatTypeCV8U)
	defer binary.Close()
	var references []Candidate
	for i := 0; i < 3; i++ {
		inner := testCandidate(30+i*120, 30, 38, 30)
		inner.Rectangularity = 0.95
		outer := testCandidate(29+i*120, 29, 40, 32)
		outer.Rectangularity = 0.88
		drawTestBox(&binary, inner)
		drawTestBox(&binary, outer)
		references = append(references, outer, inner)
	}
	_, clusters := testGeometry(t, references, references, binary, config.Default())
	if len(clusters) != 1 || clusters[0].Support != 3 || clusters[0].Width != 38 || clusters[0].Height != 30 {
		t.Fatalf("larger weak contours suppressed the strong inner references: %+v", clusters)
	}
}
