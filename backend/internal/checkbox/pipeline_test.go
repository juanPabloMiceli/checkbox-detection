package checkbox_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"checkbox-detector/internal/checkbox"
	"checkbox-detector/internal/config"
	"checkbox-detector/internal/imaging"
	"checkbox-detector/internal/testfixtures"

	"gocv.io/x/gocv"
)

func TestDefaultPipelineMatchesCharacterization(t *testing.T) {
	for _, fixture := range testfixtures.Load(t, "../../testdata") {
		t.Run(fixture.Name, func(t *testing.T) {
			var expected checkbox.DetectionResult
			if err := json.Unmarshal(fixture.ExpectedJSON, &expected); err != nil {
				t.Fatal(err)
			}
			img := gocv.IMRead(fixture.ImagePath, gocv.IMReadColor)
			if img.Empty() {
				t.Fatal("could not load source image")
			}
			defer img.Close()
			binary := imaging.Preprocess(img, config.Default().Preprocess)
			defer binary.Close()
			pipeline, err := checkbox.Detect(context.Background(), img, binary, config.Default(), false)
			if err != nil {
				t.Fatal(err)
			}
			if pipeline.Diagnostics != nil {
				t.Fatal("normal processing retained optional diagnostics")
			}
			inspected, err := checkbox.Detect(context.Background(), img, binary, config.Default(), true)
			if err != nil {
				t.Fatal(err)
			}
			if inspected.Diagnostics == nil || !reflect.DeepEqual(inspected.Candidates, pipeline.Candidates) {
				t.Fatal("diagnostics are missing or changed the selected candidates")
			}
			diagnostics := inspected.Diagnostics
			if len(diagnostics.Dedup.Candidates) != len(pipeline.Candidates) ||
				len(diagnostics.Geometry.Candidates)-diagnostics.Background.RejectedCount-diagnostics.Dedup.DuplicateCount != len(pipeline.Candidates) {
				t.Fatal("diagnostic stage counts do not account for final candidates")
			}
			actual := checkbox.DetectionResult{Boxes: checkbox.ToDetections(pipeline.Candidates, binary, config.Default().Classification)}
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("default detections differ from reviewed fixture (%d actual boxes, %d expected); compare bounds, order, and classification before updating expectations", len(actual.Boxes), len(expected.Boxes))
				for i := 0; i < min(len(actual.Boxes), len(expected.Boxes)); i++ {
					if actual.Boxes[i] != expected.Boxes[i] {
						t.Errorf("first mismatch at box %d: got %+v, want %+v", i, actual.Boxes[i], expected.Boxes[i])
						break
					}
				}
			}
		})
	}
}

func TestDetectorSwitchesSkipDisabledAlgorithms(t *testing.T) {
	source := gocv.IMRead("../../testdata/im3.png", gocv.IMReadColor)
	if source.Empty() {
		t.Fatal("could not load fixture")
	}
	defer source.Close()
	binary := imaging.Preprocess(source, config.Default().Preprocess)
	defer binary.Close()
	for _, tc := range []struct {
		name          string
		outline, line bool
		source        checkbox.CandidateSource
	}{
		{"outline only", true, false, checkbox.SourceContour},
		{"line only", false, true, checkbox.SourceLine},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Detectors.OutlineEnabled = tc.outline
			cfg.Detectors.LineEnabled = tc.line
			result, err := checkbox.Detect(context.Background(), source, binary, cfg, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Candidates) == 0 {
				t.Fatal("enabled detector produced no candidates")
			}
			for _, candidate := range result.Candidates {
				if candidate.Source != tc.source {
					t.Fatal("disabled detector contributed a candidate")
				}
			}
			if !tc.line && result.Diagnostics.LineCount != 0 {
				t.Fatal("line detector still ran")
			}
			if !tc.outline && (len(result.Diagnostics.Contours.Candidates) != 0 || result.Diagnostics.Geometry.ReferenceCount != 0) {
				t.Fatal("outline detector still ran")
			}
		})
	}
	cfg := config.Default()
	cfg.Detectors = config.Detectors{}
	if _, err := checkbox.Detect(context.Background(), source, binary, cfg, false); err == nil {
		t.Fatal("accepted both detectors disabled")
	}
}
