package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadFile applies an optional JSON object over the defaults.
// The detector constructor validates settings exposed by the detection API.
func LoadFile(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	defer file.Close()
	return Load(file)
}

func Load(reader io.Reader) (Config, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return Config{}, fmt.Errorf("config must be a JSON object")
	}

	cfg := Default()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Config{}, fmt.Errorf("config must contain exactly one JSON object")
	}

	// encoding/json silently ignores null for numeric fields and structs.
	// Reject it explicitly so a typo cannot silently keep a default value.
	decoder = json.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Config{}, fmt.Errorf("read config JSON: %w", err)
		}
		if token == nil {
			return Config{}, fmt.Errorf("config values must not be null")
		}
	}
	return cfg, nil
}
