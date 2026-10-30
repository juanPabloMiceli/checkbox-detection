package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"checkbox-detector/internal/checkbox"
	"checkbox-detector/internal/config"
)

type configurableProcessor interface {
	DetectorConfig() config.Config
	ProcessWithSettings(context.Context, []byte, config.DetectionSettings) (checkbox.DetectionResult, error)
}

func settingsHandler(processor Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, OPTIONS")
			writeError(w, 405, "use GET for detection settings")
			return
		}
		configurable, ok := processor.(configurableProcessor)
		if !ok {
			writeError(w, 503, "detection settings are unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(configurable.DetectorConfig().SettingsDefinition()); err != nil {
			log.Printf("write settings response: %v", err)
		}
	}
}
