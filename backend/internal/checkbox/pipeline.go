package checkbox

import (
	"context"

	"checkbox-detector/internal/config"
	"gocv.io/x/gocv"
)

// PipelineResult keeps optional inspection data separate from final candidates.
type PipelineResult struct {
	Candidates  []Candidate
	Diagnostics *Diagnostics
}

// Detect combines enabled candidate generators, filters geometry and background,
// then deduplicates. It borrows both matrices; diagnostics adds inspection data.
func Detect(ctx context.Context, source, binary gocv.Mat, cfg config.Config, diagnostics bool) (PipelineResult, error) {
	if err := cfg.Validate(); err != nil {
		return PipelineResult{}, err
	}
	var contours ContourResult
	var lines []Candidate
	var err error
	if cfg.Detectors.OutlineEnabled {
		contours, err = DetectCandidates(ctx, binary, cfg.Contour, diagnostics)
		if err != nil {
			return PipelineResult{}, err
		}
	}
	if cfg.Detectors.LineEnabled {
		lines, err = DetectLineCandidates(ctx, binary, cfg.Line, cfg.Interior)
		if err != nil {
			return PipelineResult{}, err
		}
	}
	combined := append(append([]Candidate(nil), contours.Candidates...), lines...)
	geometry, err := FilterByDocumentGeometry(ctx, combined, contours.Candidates, binary, cfg, diagnostics)
	if err != nil {
		return PipelineResult{}, err
	}
	backgroundFiltered, background, err := FilterByBackground(ctx, source, geometry.Candidates, cfg.Background, diagnostics)
	if err != nil {
		return PipelineResult{}, err
	}
	unique, err := deduplicate(ctx, backgroundFiltered, cfg.Dedup.FinalIoUThreshold, diagnostics)
	if err != nil {
		return PipelineResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return PipelineResult{}, err
	}
	result := PipelineResult{Candidates: unique.Candidates}
	if diagnostics {
		result.Diagnostics = &Diagnostics{
			Contours: contours, LineCount: len(lines), Geometry: geometry,
			Background: background, Dedup: unique,
		}
	}
	return result, nil
}
