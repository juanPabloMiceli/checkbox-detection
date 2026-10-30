// Package testfixtures loads reviewed characterization data for integration tests.
package testfixtures

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type Fixture struct {
	Name         string
	ImagePath    string
	Image        []byte
	ExpectedJSON []byte
}

// Load verifies the source-image hashes before reading expected outputs. A
// changed image must be reviewed together with its characterization fixture.
func Load(t *testing.T, dataDir string) []Fixture {
	t.Helper()
	var manifest struct {
		Fixtures []struct {
			Image    string `json:"image"`
			SHA256   string `json:"sha256"`
			Expected string `json:"expected"`
		} `json:"fixtures"`
	}
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	expectedDir := filepath.Join(dataDir, "expected")
	if err := json.Unmarshal(read(filepath.Join(expectedDir, "manifest.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Fixtures) == 0 {
		t.Fatal("characterization manifest contains no fixtures")
	}
	fixtures := make([]Fixture, 0, len(manifest.Fixtures))
	for _, entry := range manifest.Fixtures {
		path := filepath.Join(dataDir, entry.Image)
		data := read(path)
		if actual := fmt.Sprintf("%x", sha256.Sum256(data)); actual != entry.SHA256 {
			t.Fatalf("%s changed: review the image and expected output together", entry.Image)
		}
		fixtures = append(fixtures, Fixture{
			Name:         entry.Image,
			ImagePath:    path,
			Image:        data,
			ExpectedJSON: read(filepath.Join(expectedDir, entry.Expected)),
		})
	}
	return fixtures
}
