package checkbox

import (
	"context"
	"image"
	"image/color"
	"reflect"
	"testing"

	"checkbox-detector/internal/config"
	"gocv.io/x/gocv"
)

func TestGeometryReportsCandidateAndReferenceReasonsSeparately(t *testing.T) {
	binary := gocv.NewMatWithSize(400, 600, gocv.MatTypeCV8U)
	defer binary.Close()
	var references []Candidate
	for i := range 3 {
		candidate := testCandidate(30+i*120, 30, 40, 30)
		drawTestBox(&binary, candidate)
		references = append(references, candidate)
	}
	weak := testCandidate(30, 150, 41, 31)
	weak.Rectangularity = .88
	drawTestBox(&binary, weak)
	outlier := testCandidate(150, 150, 80, 20)
	all := append(append([]Candidate(nil), references...), weak, outlier)
	refs := append(append([]Candidate(nil), references...), weak)
	result, err := FilterByDocumentGeometry(context.Background(), all, refs, binary, config.Default(), true)
	if err != nil {
		t.Fatal(err)
	}
	if result.ReferenceCount != 3 || len(result.Candidates) != 4 {
		t.Fatalf("unexpected selection: %+v", result)
	}
	if len(result.ReferenceRejections) != 1 || result.ReferenceRejections[0].Reason != RejectReferenceRectangularity {
		t.Fatalf("bad reference evidence: %+v", result.ReferenceRejections)
	}
	if len(result.Rejected) != 1 || result.Rejected[0].Candidate != outlier || result.Rejected[0].Reason != RejectGeometryMismatch {
		t.Fatalf("bad candidate evidence: %+v", result.Rejected)
	}
	withoutModel, err := FilterByDocumentGeometry(context.Background(), []Candidate{weak}, nil, binary, config.Default(), true)
	if err != nil || len(withoutModel.Rejected) != 1 || withoutModel.Rejected[0].Reason != RejectRecoveryWithoutModel {
		t.Fatalf("missing recovery rejection: %+v, %v", withoutModel, err)
	}
	quiet, err := FilterByDocumentGeometry(context.Background(), all, refs, binary, config.Default(), false)
	if err != nil || !reflect.DeepEqual(quiet.Candidates, result.Candidates) || quiet.ReferenceRejections != nil || quiet.Rejected != nil {
		t.Fatal("diagnostic recording changed behavior or allocated detail")
	}
}

func TestContourDiagnosticsCountOnlyFirstFailure(t *testing.T) {
	binary := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8U)
	defer binary.Close()
	gocv.Rectangle(&binary, image.Rect(10, 10, 16, 16), color.RGBA{255, 255, 255, 255}, 1)
	result, err := DetectCandidates(context.Background(), binary, config.Default().Contour, true)
	if err != nil {
		t.Fatal(err)
	}
	s := result.Rejections
	if s.Size == 0 || len(result.Rejected) != s.Size+s.VertexCount+s.Convexity+s.Rectangularity+s.Border {
		t.Fatalf("first-failure counts do not match records: %+v", result)
	}
	for _, rejection := range result.Rejected {
		if rejection.Reason != RejectContourSize || rejection.Candidate.Source != SourceContour {
			t.Fatalf("unexpected reason: %+v", rejection)
		}
	}
	quiet, err := DetectCandidates(context.Background(), binary, config.Default().Contour, false)
	if err != nil || quiet.Rejected != nil || quiet.Rejections != result.Rejections {
		t.Fatal("optional recording changed counts")
	}
}

func TestEqualAreaDeduplicationIsIndependentOfInputOrder(t *testing.T) {
	a := testCandidate(10, 10, 20, 20)
	b := testCandidate(11, 10, 20, 20)
	c := a
	c.Source = SourceLine
	for _, permutation := range [][]Candidate{{a, b, c}, {a, c, b}, {b, a, c}, {b, c, a}, {c, a, b}, {c, b, a}} {
		original := append([]Candidate(nil), permutation...)
		result, err := deduplicate(context.Background(), permutation, .5, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Candidates) != 1 || result.Candidates[0] != a || result.DuplicateCount != 2 || len(result.Rejected) != 2 {
			t.Fatalf("tie result depends on input: %+v", result)
		}
		if !reflect.DeepEqual(permutation, original) {
			t.Fatal("input changed")
		}
	}
}
