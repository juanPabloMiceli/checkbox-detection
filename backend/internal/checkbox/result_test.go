package checkbox

import (
	"encoding/json"
	"image"
	"image/color"
	"testing"

	"checkbox-detector/internal/config"

	"gocv.io/x/gocv"
)

func TestDetectionResultMatchesTakeHomeJSON(t *testing.T) {
	binary := gocv.NewMatWithSize(100, 150, gocv.MatTypeCV8U)
	defer binary.Close()
	gocv.Rectangle(&binary, image.Rect(10, 20, 40, 60), color.RGBA{255, 255, 255, 255}, -1)
	candidates := []Candidate{
		{Bounds: image.Rect(10, 20, 40, 60)},
		{Bounds: image.Rect(80, 20, 110, 60)},
	}
	result := DetectionResult{Boxes: ToDetections(candidates, binary, config.Default().Classification)}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"boxes":[{"bbox":[10,20,40,60],"is_checked":true},{"bbox":[80,20,110,60],"is_checked":false}]}`
	if string(data) != want {
		t.Fatalf("JSON does not match the required contract:\ngot  %s\nwant %s", data, want)
	}
}

func TestDetectionResultWithNoBoxes(t *testing.T) {
	binary := gocv.NewMat()
	defer binary.Close()
	data, err := json.Marshal(DetectionResult{Boxes: ToDetections(nil, binary, config.Default().Classification)})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"boxes":[]}` {
		t.Fatalf("empty detections must serialize as an empty boxes array, got %s", data)
	}
}
