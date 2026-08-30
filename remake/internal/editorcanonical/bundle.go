// Package editorcanonical validates the versioned editor bundle admitted by
// the packaged runtime. It does not interpret legacy extensions as gameplay
// semantics; it only proves that the canonical documents shipped together are
// the reviewed, deterministic set.
package editorcanonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Counts struct {
	Campaign  int `json:"campaign"`
	Scenario  int `json:"scenario"`
	Story     int `json:"story"`
	Animation int `json:"animation"`
}

type Document struct {
	Kind        string `json:"kind"`
	Source      string `json:"source"`
	Output      string `json:"output"`
	DocumentID  string `json:"document_id"`
	SHA256      string `json:"sha256"`
	Diagnostics int    `json:"diagnostics"`
}

type Summary struct {
	SchemaVersion       int               `json:"schema_version"`
	ExporterVersion     string            `json:"exporter_version"`
	ImporterVersion     string            `json:"importer_version"`
	SourceRoot          string            `json:"source_root"`
	IncludeAnimations   bool              `json:"include_animations"`
	Counts              Counts            `json:"counts"`
	Documents           []Document        `json:"documents"`
	Diagnostics         []json.RawMessage `json:"diagnostics"`
	IdentityDiagnostics int               `json:"identity_diagnostics"`
}

func decodeStrict(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("trailing JSON")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// ValidateBundle verifies the public core bundle. Private animation metadata
// is optional input to the exporter and is deliberately not required by the
// distributable runtime contract.
func ValidateBundle(root string) (*Summary, error) {
	if root == "" {
		return nil, errors.New("editor canonical root is unavailable")
	}
	var summary Summary
	if err := decodeStrict(filepath.Join(root, "bundle-summary.json"), &summary); err != nil {
		return nil, fmt.Errorf("editor canonical summary: %w", err)
	}
	if summary.SchemaVersion != 1 || summary.ExporterVersion != "fd2-editor-canonical-exporter/1.0" || summary.ImporterVersion != "fd2-editor-legacy-importer/1.0" || summary.SourceRoot != "repository-relative" || summary.IncludeAnimations {
		return nil, errors.New("editor canonical summary identity is invalid")
	}
	if summary.Counts != (Counts{Campaign: 1, Scenario: 30, Story: 35, Animation: 0}) || len(summary.Documents) != 66 {
		return nil, fmt.Errorf("editor canonical counts=%+v documents=%d", summary.Counts, len(summary.Documents))
	}
	seen := make(map[string]bool, len(summary.Documents))
	actual := Counts{}
	for _, document := range summary.Documents {
		if document.DocumentID == "" || seen[document.DocumentID] {
			return nil, fmt.Errorf("editor canonical duplicate or empty document_id %q", document.DocumentID)
		}
		seen[document.DocumentID] = true
		switch document.Kind {
		case "campaign":
			actual.Campaign++
		case "scenario":
			actual.Scenario++
		case "story":
			actual.Story++
		default:
			return nil, fmt.Errorf("editor canonical document %q has invalid kind %q", document.DocumentID, document.Kind)
		}
		clean := filepath.Clean(filepath.FromSlash(document.Output))
		if document.Output == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("editor canonical document %q escapes bundle", document.DocumentID)
		}
		digest, err := hex.DecodeString(document.SHA256)
		if err != nil || len(digest) != sha256.Size {
			return nil, fmt.Errorf("editor canonical document %q has invalid SHA-256", document.DocumentID)
		}
		raw, err := os.ReadFile(filepath.Join(root, clean))
		if err != nil {
			return nil, fmt.Errorf("editor canonical document %q: %w", document.DocumentID, err)
		}
		sum := sha256.Sum256(raw)
		if !bytes.Equal(sum[:], digest) {
			return nil, fmt.Errorf("editor canonical document %q hash mismatch", document.DocumentID)
		}
	}
	if actual != summary.Counts {
		return nil, fmt.Errorf("editor canonical typed counts=%+v want %+v", actual, summary.Counts)
	}
	var identity struct {
		SchemaVersion int               `json:"schema_version"`
		Kind          string            `json:"kind"`
		Source        map[string]string `json:"source"`
		Characters    []json.RawMessage `json:"characters"`
		Diagnostics   []json.RawMessage `json:"diagnostics"`
	}
	if err := decodeStrict(filepath.Join(root, "character-identity.json"), &identity); err != nil {
		return nil, fmt.Errorf("editor canonical character identity: %w", err)
	}
	if identity.SchemaVersion != 1 || identity.Kind != "character_identity_catalog" || identity.Source["evidence"] != "direct legacy fields" || identity.Source["root"] != "repository-relative" || len(identity.Characters) == 0 || len(identity.Diagnostics) != summary.IdentityDiagnostics {
		return nil, errors.New("editor canonical character identity contract is invalid")
	}
	return &summary, nil
}
