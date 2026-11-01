package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"checkbox-detector/internal/checkbox"
	"checkbox-detector/internal/config"
)

func blankImage(t *testing.T, format string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	var encoded bytes.Buffer
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&encoded, img, nil)
	} else {
		err = png.Encode(&encoded, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func TestDetectRejectsTruncatedImageAfterValidHeader(t *testing.T) {
	request := multipartRequest(t, "image", blankImage(t, "png")[:33])
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for truncated image, got %d: %s", response.Code, response.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "could not decode image" {
		t.Fatalf("unexpected decode error response: %s", response.Body)
	}
}

func multipartRequest(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile(field, "document.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/detect", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	return r
}

func TestDetectAcceptsBrowserUploadsAndRawImages(t *testing.T) {
	handler := newTestHandler(t)
	for _, format := range []string{"png", "jpeg"} {
		data := blankImage(t, format)
		for _, field := range []string{"image", "file", "raw"} {
			t.Run(format+"/"+field, func(t *testing.T) {
				var request *http.Request
				if field == "raw" {
					request = httptest.NewRequest(http.MethodPost, "/detect", bytes.NewReader(data))
					request.Header.Set("Content-Type", "image/"+format)
				} else {
					request = multipartRequest(t, field, data)
				}
				request.Header.Set("Origin", "http://localhost:5173")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusOK || response.Body.String() != "{\"boxes\":[]}\n" {
					t.Fatalf("want empty detection response, got %d: %s", response.Code, response.Body)
				}
				if response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Fatalf("missing JSON or browser CORS headers: %v", response.Header())
				}
			})
		}
	}
}

func TestDetectHandlesInvalidRequests(t *testing.T) {
	oversize := httptest.NewRequest(http.MethodPost, "/detect", nil)
	oversize.ContentLength = maxUploadBytes + 1
	invalidImage := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewBufferString("invalid image"))
	invalidImage.Header.Set("Content-Type", "image/png")
	chunkedOversize := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewReader(make([]byte, maxUploadBytes+1)))
	chunkedOversize.ContentLength = -1
	chunkedOversize.Header.Set("Content-Type", "image/png")
	for _, test := range []struct {
		name    string
		request *http.Request
		status  int
	}{
		{"method", httptest.NewRequest(http.MethodGet, "/detect", nil), http.StatusMethodNotAllowed},
		{"content type", httptest.NewRequest(http.MethodPost, "/detect", nil), http.StatusUnsupportedMediaType},
		{"missing field", multipartRequest(t, "wrong", []byte("image")), http.StatusBadRequest},
		{"invalid image", invalidImage, http.StatusBadRequest},
		{"oversized request", oversize, http.StatusRequestEntityTooLarge},
		{"oversized streaming request", chunkedOversize, http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			newTestHandler(t).ServeHTTP(response, test.request)
			if response.Code != test.status {
				t.Fatalf("want status %d, got %d: %s", test.status, response.Code, response.Body)
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] == "" {
				t.Fatalf("want JSON error, got %s", response.Body)
			}
		})
	}
}

func TestDetectPreflight(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/detect", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "Content-Type")
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "*" || response.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" || response.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Fatalf("browser preflight failed: %d %v", response.Code, response.Header())
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	detector, err := checkbox.New(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	return testHandler(t, detector)
}

type processorFunc func(context.Context, []byte) (checkbox.DetectionResult, error)

func (f processorFunc) Process(ctx context.Context, data []byte) (checkbox.DetectionResult, error) {
	return f(ctx, data)
}

func TestHandlerPreservesProcessingInputAndContext(t *testing.T) {
	request := multipartRequest(t, "image", []byte("encoded image"))
	type contextKey struct{}
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "request context"))
	called := false
	processor := processorFunc(func(ctx context.Context, data []byte) (checkbox.DetectionResult, error) {
		called = true
		if ctx.Value(contextKey{}) != "request context" || string(data) != "encoded image" {
			t.Fatal("HTTP adapter changed input or context")
		}
		return checkbox.DetectionResult{Boxes: []checkbox.Detection{}}, nil
	})
	response := httptest.NewRecorder()
	testHandler(t, processor).ServeHTTP(response, request)
	if !called || response.Code != http.StatusOK || response.Body.String() != "{\"boxes\":[]}\n" {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body)
	}
}

func TestHandlerMapsServiceErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		code    int
		message string
	}{
		{"invalid image", checkbox.ErrInvalidImage, 400, "image must be a valid PNG or JPEG"},
		{"native decode", fmt.Errorf("wrapped: %w", checkbox.ErrDecodeImage), 400, "could not decode image"},
		{"pixel limit", &checkbox.ImageSizeError{MaxPixels: 40_000_000}, 413, "image exceeds the 40 megapixel limit"},
		{"custom limit", &checkbox.ImageSizeError{MaxPixels: 500_000}, 413, "image exceeds the 0.5 megapixel limit"},
		{"cancellation", context.Canceled, 408, "image processing canceled"},
		{"deadline", context.DeadlineExceeded, 408, "image processing canceled"},
		{"internal", errors.New("private native details"), 500, "could not process image"},
	} {
		t.Run(test.name, func(t *testing.T) {
			processor := processorFunc(func(context.Context, []byte) (checkbox.DetectionResult, error) {
				return checkbox.DetectionResult{}, test.err
			})
			response := httptest.NewRecorder()
			testHandler(t, processor).ServeHTTP(response, multipartRequest(t, "image", []byte("image")))
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.code || body["error"] != test.message {
				t.Fatalf("got %d: %s", response.Code, response.Body)
			}
		})
	}
}

func TestUploadRemovesTemporaryFilesBeforeProcessing(t *testing.T) {
	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	data := bytes.Repeat([]byte{42}, (4<<20)+128)
	called := false
	processor := processorFunc(func(_ context.Context, encoded []byte) (checkbox.DetectionResult, error) {
		called = true
		if !bytes.Equal(encoded, data) {
			t.Fatal("spooled upload bytes changed")
		}
		files, err := os.ReadDir(temporary)
		if err != nil || len(files) != 0 {
			t.Fatalf("temporary files remain during processing: %v, %v", files, err)
		}
		return checkbox.DetectionResult{Boxes: []checkbox.Detection{}}, nil
	})
	response := httptest.NewRecorder()
	testHandler(t, processor).ServeHTTP(response, multipartRequest(t, "image", data))
	if !called || response.Code != http.StatusOK {
		t.Fatalf("got %d: %s", response.Code, response.Body)
	}
}

func TestMalformedMultipartSkipsProcessing(t *testing.T) {
	processor := processorFunc(func(context.Context, []byte) (checkbox.DetectionResult, error) {
		t.Fatal("processing called for malformed upload")
		return checkbox.DetectionResult{}, nil
	})
	for _, contentType := range []string{"multipart/form-data", "multipart/form-data; boundary=missing"} {
		request := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewBufferString("invalid multipart body"))
		request.Header.Set("Content-Type", contentType)
		response := httptest.NewRecorder()
		testHandler(t, processor).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("got %d: %s", response.Code, response.Body)
		}
	}
}

func testHandler(t *testing.T, processor Processor, overrides ...Options) *Handler {
	t.Helper()
	options := DefaultOptions()
	if len(overrides) > 0 {
		options = overrides[0]
	}
	handler, err := NewHandler(processor, options)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestLegacyAPIRouteIsNotRegistered(t *testing.T) {
	processor := processorFunc(func(context.Context, []byte) (checkbox.DetectionResult, error) {
		t.Fatal("legacy route invoked processing")
		return checkbox.DetectionResult{}, nil
	})
	handler := testHandler(t, processor)
	for _, method := range []string{http.MethodPost, http.MethodOptions, http.MethodGet} {
		t.Run(method, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(method, "/api/detect", nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("legacy route: want 404, got %d", response.Code)
			}
		})
	}
}
