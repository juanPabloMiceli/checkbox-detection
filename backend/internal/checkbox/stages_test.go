package checkbox

import (
	"context"
	"testing"

	"checkbox-detector/internal/config"
	"gocv.io/x/gocv"
)

func testGeometry(t *testing.T, candidates, references []Candidate, binary gocv.Mat, cfg config.Config) ([]Candidate, []GeometryCluster) {
	t.Helper()
	result, err := FilterByDocumentGeometry(context.Background(), candidates, references, binary, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	return result.Candidates, result.Clusters
}

func testContours(t *testing.T, binary gocv.Mat, cfg config.Contour) []Candidate {
	t.Helper()
	result, err := DetectCandidates(context.Background(), binary, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	return result.Candidates
}

func testLines(t *testing.T, binary gocv.Mat, cfg config.Line, interior config.Interior) []Candidate {
	t.Helper()
	candidates, err := DetectLineCandidates(context.Background(), binary, cfg, interior)
	if err != nil {
		t.Fatal(err)
	}
	return candidates
}

func testBackground(t *testing.T, source gocv.Mat, candidates []Candidate, cfg config.Background) ([]Candidate, BackgroundStats) {
	t.Helper()
	kept, stats, err := FilterByBackground(context.Background(), source, candidates, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	return kept, stats
}
