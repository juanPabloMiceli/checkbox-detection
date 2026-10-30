package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"checkbox-detector/internal/checkbox"
)

// Processor isolates HTTP parsing and error mapping from image processing.
type Processor interface {
	Process(context.Context, []byte) (checkbox.DetectionResult, error)
}

type Handler struct {
	mux       *http.ServeMux
	admission *admission
	options   Options
}

func NewHandler(processor Processor, options Options) (*Handler, error) {
	if processor == nil {
		return nil, fmt.Errorf("processor is required")
	}
	if err := options.validate(); err != nil {
		return nil, err
	}
	handler := &Handler{admission: newAdmission(options), options: options}
	mux := http.NewServeMux()
	mux.HandleFunc("/detect", handler.detect(processor))
	mux.HandleFunc("/detect/settings", settingsHandler(processor))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusOK, "ok") })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := handler.admission.available(r.Context()); err != nil {
			writeStatus(w, http.StatusServiceUnavailable, "draining")
			return
		}
		writeStatus(w, http.StatusOK, "ready")
	})
	handler.mux = mux
	return handler, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

// BeginShutdown rejects new work and wakes queued requests while active work
// retains capacity until Process actually returns, even after cancellation.
func (h *Handler) BeginShutdown() { h.admission.close() }

func (h *Handler) detect(processor Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Local frontend development across ports, without cookies/credentials.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST, OPTIONS")
			writeError(w, http.StatusMethodNotAllowed, "use POST with a PNG or JPEG image")
			return
		}
		release, err := h.admission.acquire(r.Context())
		if err != nil {
			if errors.Is(err, errBusy) || errors.Is(err, errDraining) {
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusServiceUnavailable, err.Error())
			} else {
				writeProcessingError(w, err)
			}
			return
		}
		defer release()
		data, err := readUpload(w, r)
		if err != nil {
			var requestErr *requestError
			if errors.As(err, &requestErr) {
				writeError(w, requestErr.status, requestErr.message)
			} else {
				log.Printf("read upload: %v", err)
				writeError(w, http.StatusInternalServerError, "could not read image upload")
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), h.options.ProcessTimeout)
		defer cancel()
		var result checkbox.DetectionResult
		if values, exists := r.PostForm["settings"]; exists {
			configurable, ok := processor.(configurableProcessor)
			if !ok {
				writeError(w, http.StatusBadRequest, "per-request settings are not supported")
				return
			}
			if len(values) != 1 {
				writeError(w, http.StatusBadRequest, "provide settings exactly once")
				return
			}
			settings, settingsErr := configurable.DetectorConfig().DecodeSettings([]byte(values[0]))
			if settingsErr != nil {
				writeError(w, http.StatusBadRequest, settingsErr.Error())
				return
			}
			result, err = configurable.ProcessWithSettings(ctx, data, settings)
		} else {
			result, err = processor.Process(ctx, data)
		}
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			writeProcessingError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("write detection response: %v", err)
		}
	}
}

func writeStatus(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{Status: status}); err != nil {
		log.Printf("write status response: %v", err)
	}
}

func writeProcessingError(w http.ResponseWriter, err error) {
	var sizeError *checkbox.ImageSizeError
	switch {
	case errors.Is(err, checkbox.ErrInvalidImage):
		writeError(w, http.StatusBadRequest, checkbox.ErrInvalidImage.Error())
	case errors.Is(err, checkbox.ErrDecodeImage):
		writeError(w, http.StatusBadRequest, checkbox.ErrDecodeImage.Error())
	case errors.As(err, &sizeError):
		writeError(w, http.StatusRequestEntityTooLarge, sizeError.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusRequestTimeout, "image processing canceled")
	default:
		log.Printf("process image: %v", err)
		writeError(w, http.StatusInternalServerError, "could not process image")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message}); err != nil {
		log.Printf("write error response: %v", err)
	}
}
