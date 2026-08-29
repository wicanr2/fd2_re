// Package localization loads editable FD2 locale packs.
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
	"strings"
)

const SchemaVersion = 1

var localePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z][a-z]{3})?(?:-[A-Z]{2}|-[0-9]{3})?$`)
var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9_]+)+$`)
var fontPathPattern = regexp.MustCompile(`^fonts/[a-z0-9][a-z0-9._/-]*\.(?:png|bin|json)$`)
var packIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]+$`)

type Font struct {
	Profile string `json:"profile"`
	Path    string `json:"path"`
}

type Entry struct {
	Text           string   `json:"text"`
	Variables      []string `json:"variables"`
	SourceStringID string   `json:"source_string_id"`
}

type Pack struct {
	SchemaVersion int              `json:"schema_version"`
	PackID        string           `json:"pack_id"`
	Locale        string           `json:"locale"`
	Kind          string           `json:"kind"`
	BaseLocale    string           `json:"base_locale,omitempty"`
	LayoutProfile string           `json:"layout_profile"`
	Font          Font             `json:"font"`
	Entries       map[string]Entry `json:"entries"`
}

// Catalog is the immutable, validated view used by the runtime.
type Catalog struct {
	Locale        string
	Kind          string
	BaseLocale    string
	LayoutProfile string
	Font          Font
	Entries       map[string]Entry
}

// Load reads one pack.json and rejects malformed or future fields.
func Load(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read locale pack %q: %w", path, err)
	}
	var pack Pack
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decode locale pack %q: %w", path, err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("decode locale pack %q: trailing JSON", path)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode locale pack %q: trailing JSON: %w", path, err)
	}
	if err := validatePack(&pack); err != nil {
		return nil, fmt.Errorf("validate locale pack %q: %w", path, err)
	}
	entries := make(map[string]Entry, len(pack.Entries))
	for key, entry := range pack.Entries {
		entry.Variables = append([]string(nil), entry.Variables...)
		entries[key] = entry
	}
	return &Catalog{Locale: pack.Locale, Kind: pack.Kind, BaseLocale: pack.BaseLocale,
		LayoutProfile: pack.LayoutProfile, Font: pack.Font, Entries: entries}, nil
}

// LoadOfficial loads <root>/<locale>/pack.json and requires an official pack.
func LoadOfficial(root, locale string) (*Catalog, error) {
	if !localePattern.MatchString(locale) {
		return nil, fmt.Errorf("invalid locale %q", locale)
	}
	catalog, err := Load(filepath.Join(root, locale, "pack.json"))
	if err != nil {
		return nil, err
	}
	if catalog.Kind != "official" || catalog.Locale != locale {
		return nil, fmt.Errorf("pack identity mismatch: kind=%q locale=%q, want official/%q", catalog.Kind, catalog.Locale, locale)
	}
	return catalog, nil
}

// Format substitutes exactly the declared printf-style variables.
func (c *Catalog) Format(key string, args ...any) (string, error) {
	if c == nil {
		return "", errors.New("nil locale catalog")
	}
	entry, ok := c.Entries[key]
	if !ok {
		return "", fmt.Errorf("unknown locale key %q", key)
	}
	if len(args) != len(entry.Variables) {
		return "", fmt.Errorf("locale key %q expects %d arguments, got %d", key, len(entry.Variables), len(args))
	}
	formatted := fmt.Sprintf(entry.Text, args...)
	if strings.Contains(formatted, "%!") {
		return "", fmt.Errorf("locale key %q received incompatible argument types", key)
	}
	return formatted, nil
}

func validatePack(p *Pack) error {
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", p.SchemaVersion)
	}
	if !packIDPattern.MatchString(p.PackID) {
		return errors.New("pack_id is invalid")
	}
	if !localePattern.MatchString(p.Locale) {
		return fmt.Errorf("invalid locale %q", p.Locale)
	}
	if p.Kind != "official" && p.Kind != "community" {
		return fmt.Errorf("invalid kind %q", p.Kind)
	}
	if p.Kind == "official" && p.BaseLocale != "" {
		return errors.New("official pack must not declare base_locale")
	}
	if p.Kind == "community" {
		if !localePattern.MatchString(p.BaseLocale) {
			return errors.New("community pack requires valid base_locale")
		}
		if p.BaseLocale == p.Locale {
			return errors.New("community base_locale must differ from locale")
		}
	}
	if p.LayoutProfile != "dos-8x8-latin" && p.LayoutProfile != "pc98-16x16-cjk" {
		return fmt.Errorf("invalid layout_profile %q", p.LayoutProfile)
	}
	if p.Font.Profile != "fd2-latin-8x8" && p.Font.Profile != "fd2-cjk-16x16" {
		return fmt.Errorf("invalid font.profile %q", p.Font.Profile)
	}
	if err := validateRelativePath(p.Font.Path); err != nil {
		return fmt.Errorf("font.path: %w", err)
	}
	if len(p.Entries) == 0 {
		return errors.New("entries must not be empty")
	}
	for key, entry := range p.Entries {
		if !keyPattern.MatchString(key) || strings.HasPrefix(key, "legacy.") {
			return fmt.Errorf("invalid semantic key %q", key)
		}
		if entry.Text == "" {
			return fmt.Errorf("entry %q has empty text", key)
		}
		if entry.SourceStringID == "" {
			return fmt.Errorf("entry %q has empty source_string_id", key)
		}
		placeholders := formatPlaceholders(entry.Text)
		if !equalStrings(placeholders, entry.Variables) {
			return fmt.Errorf("entry %q variables do not match text placeholders", key)
		}
	}
	return nil
}

func validateRelativePath(path string) error {
	if path == "" || strings.ContainsRune(path, 0) || filepath.IsAbs(path) || filepath.Separator == '/' && strings.Contains(path, `\`) || !fontPathPattern.MatchString(path) {
		return errors.New("must be a non-empty relative slash path")
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("path escapes pack root")
	}
	return nil
}

func formatPlaceholders(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] != '%' {
			continue
		}
		if i+1 < len(text) && text[i+1] == '%' {
			i++
			continue
		}
		start := i
		i++
		for i < len(text) && strings.ContainsRune("+-# 0", rune(text[i])) {
			i++
		}
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
		}
		if i < len(text) && text[i] == '.' {
			i++
			for i < len(text) && text[i] >= '0' && text[i] <= '9' {
				i++
			}
		}
		if i < len(text) && strings.ContainsRune("vTtbcdoOqxXUeEfFgGsxp", rune(text[i])) {
			out = append(out, text[start:i+1])
		}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
