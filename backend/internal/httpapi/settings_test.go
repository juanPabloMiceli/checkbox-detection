package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"checkbox-detector/internal/checkbox"
	"checkbox-detector/internal/config"
)

func TestSettingsEndpointAndRequestOverride(t *testing.T) {
	handler := newTestHandler(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/detect/settings", nil))
	if response.Code != 200 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("settings response: %d %s", response.Code, response.Body)
	}
	var definition config.SettingsDefinition
	if err := json.Unmarshal(response.Body.Bytes(), &definition); err != nil {
		t.Fatal(err)
	}
	if len(definition.Limits) != 12 || definition.Defaults.CheckedInkRatio != 0.12 || !definition.Defaults.OutlineEnabled || !definition.Defaults.LineEnabled {
		t.Fatal("incorrect defaults")
	}
	image, err := os.ReadFile("../../testdata/im3.png")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		settings string
		status   int
	}{{`{"checkedInkRatio":1}`, 200}, {`{"minSidePx":91}`, 400}, {`{"outlineEnabled":false,"lineEnabled":false}`, 400}, {`{"preprocess":{}}`, 400}} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("image", "document.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = part.Write(image); err != nil {
			t.Fatal(err)
		}
		if err = writer.WriteField("settings", tc.settings); err != nil {
			t.Fatal(err)
		}
		if err = writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/detect", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.settings, response.Code, response.Body)
		}
		if tc.status == 200 {
			var result checkbox.DetectionResult
			if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Boxes) == 0 {
				t.Fatal("expected detections")
			}
			// Check JSON without relying on internal detection field naming.
			var raw struct {
				Boxes []struct {
					Checked bool `json:"is_checked"`
				}
			}
			if err = json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			for _, box := range raw.Boxes {
				if box.Checked {
					t.Fatal("100% ink threshold classified a checked box")
				}
			}
		}
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/detect/settings", nil))
	var after config.SettingsDefinition
	if err = json.Unmarshal(response.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if after.Defaults.CheckedInkRatio != 0.12 {
		t.Fatal("request changed shared defaults")
	}
}
