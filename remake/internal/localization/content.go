package localization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var contentPlaceholderPattern = regexp.MustCompile(`%(?:%|[-+0-9.#]*[a-zA-Z])`)

// ContentEntry is one player-visible occurrence in the full locale catalog.
// Source remains provenance only; StringID is the runtime identity.
type ContentEntry struct {
	StringID  string         `json:"string_id"`
	IDStatus  string         `json:"id_status"`
	Role      string         `json:"role"`
	Text      string         `json:"text"`
	Variables []string       `json:"variables"`
	Status    string         `json:"status"`
	Source    map[string]any `json:"source"`
}

type contentPack struct {
	SchemaVersion      int            `json:"schema_version"`
	Kind               string         `json:"kind"`
	Locale             string         `json:"locale"`
	SourceLocale       string         `json:"source_locale"`
	EntryCount         int            `json:"entry_count"`
	InventorySHA256    string         `json:"inventory_sha256"`
	GlossarySHA256     string         `json:"glossary_sha256"`
	PivotContentSHA256 *string        `json:"pivot_content_sha256"`
	TranslationEngine  string         `json:"translation_engine"`
	Entries            []ContentEntry `json:"entries"`
}

// ContentCatalog is an immutable lookup view of one complete official locale.
type ContentCatalog struct {
	Locale            string
	InventorySHA256   string
	GlossarySHA256    string
	TranslationEngine string
	entries           map[string]ContentEntry
}

// LoadOfficialContent validates <root>/<locale>/content.json. It intentionally
// stays separate from pack.json: semantic UI messages and full authored content
// have different schemas and rollout gates.
func LoadOfficialContent(root, locale string) (*ContentCatalog, error) {
	if !localePattern.MatchString(locale) {
		return nil, fmt.Errorf("invalid locale %q", locale)
	}
	path := filepath.Join(root, locale, "content.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read locale content %q: %w", path, err)
	}
	var pack contentPack
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decode locale content %q: %w", path, err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("decode locale content %q: trailing JSON", path)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode locale content %q: trailing JSON: %w", path, err)
	}
	if err := validateContentPack(&pack, locale); err != nil {
		return nil, fmt.Errorf("validate locale content %q: %w", path, err)
	}
	entries := make(map[string]ContentEntry, len(pack.Entries))
	for _, entry := range pack.Entries {
		entry.Variables = append([]string(nil), entry.Variables...)
		entries[entry.StringID] = entry
	}
	return &ContentCatalog{
		Locale: locale, InventorySHA256: pack.InventorySHA256,
		GlossarySHA256: pack.GlossarySHA256, TranslationEngine: pack.TranslationEngine,
		entries: entries,
	}, nil
}

func validateContentPack(pack *contentPack, locale string) error {
	if pack.SchemaVersion != SchemaVersion || pack.Kind != "fd2_full_locale_content" {
		return errors.New("schema_version or kind is invalid")
	}
	if pack.Locale != locale || pack.SourceLocale != "zh-Hant" {
		return fmt.Errorf("identity mismatch: locale=%q source_locale=%q", pack.Locale, pack.SourceLocale)
	}
	if pack.EntryCount != len(pack.Entries) || pack.EntryCount == 0 {
		return errors.New("entry_count does not match entries")
	}
	if !sha256Pattern.MatchString(pack.InventorySHA256) || !sha256Pattern.MatchString(pack.GlossarySHA256) {
		return errors.New("inventory or glossary SHA-256 is invalid")
	}
	if pack.PivotContentSHA256 != nil && !sha256Pattern.MatchString(*pack.PivotContentSHA256) {
		return errors.New("pivot content SHA-256 is invalid")
	}
	if pack.TranslationEngine == "" {
		return errors.New("translation_engine is empty")
	}
	seen := make(map[string]struct{}, len(pack.Entries))
	for _, entry := range pack.Entries {
		if entry.StringID == "" || entry.Role == "" || entry.Text == "" || entry.IDStatus == "" {
			return errors.New("content entry has an empty required field")
		}
		if _, exists := seen[entry.StringID]; exists {
			return fmt.Errorf("duplicate string_id %q", entry.StringID)
		}
		seen[entry.StringID] = struct{}{}
		if entry.Status != "source" && entry.Status != "machine_draft" && entry.Status != "reviewed" && entry.Status != "blocked" {
			return fmt.Errorf("entry %q has invalid status %q", entry.StringID, entry.Status)
		}
		if !equalStrings(contentPlaceholders(entry.Text), entry.Variables) {
			return fmt.Errorf("entry %q variables do not match text placeholders", entry.StringID)
		}
	}
	return nil
}

func contentPlaceholders(text string) []string {
	matches := contentPlaceholderPattern.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if match != "%%" {
			out = append(out, match)
		}
	}
	return out
}

// StoryText resolves one canonical authored dialogue line. Provisional source
// identities and non-dialogue roles cannot enter the formal story path.
func (c *ContentCatalog) StoryText(lineID string) (string, error) {
	if c == nil {
		return "", errors.New("nil locale content catalog")
	}
	key := lineID + "/text"
	entry, ok := c.entries[key]
	if !ok {
		return "", fmt.Errorf("missing story content %q", key)
	}
	if entry.IDStatus != "stable_canonical" || entry.Role != "dialogue" {
		return "", fmt.Errorf("story content %q has identity=%q role=%q", key, entry.IDStatus, entry.Role)
	}
	if entry.Status == "blocked" {
		return "", fmt.Errorf("story content %q is blocked", key)
	}
	return entry.Text, nil
}
