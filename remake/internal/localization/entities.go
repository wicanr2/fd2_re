package localization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type entityNameEntry struct {
	Name            string   `json:"name"`
	SourceStringIDs []string `json:"source_string_ids"`
}

type entityPack struct {
	SchemaVersion int                        `json:"schema_version"`
	Kind          string                     `json:"kind"`
	Locale        string                     `json:"locale"`
	SourceLocale  string                     `json:"source_locale"`
	ItemCount     int                        `json:"item_count"`
	Items         map[string]entityNameEntry `json:"items"`
}

// EntityCatalog 是以遊戲實體 ID 為鍵的不可變官方名稱目錄。
type EntityCatalog struct {
	Locale string
	items  map[int]string
}

// LoadOfficialEntities 驗證 <root>/<locale>/entities.json。
func LoadOfficialEntities(root, locale string) (*EntityCatalog, error) {
	if !localePattern.MatchString(locale) {
		return nil, fmt.Errorf("invalid locale %q", locale)
	}
	path := filepath.Join(root, locale, "entities.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read locale entities %q: %w", path, err)
	}
	var pack entityPack
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decode locale entities %q: %w", path, err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("decode locale entities %q: trailing JSON", path)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode locale entities %q: trailing JSON: %w", path, err)
	}
	if pack.SchemaVersion != SchemaVersion || pack.Kind != "fd2_locale_entities" ||
		pack.Locale != locale || pack.SourceLocale != "zh-Hant" ||
		pack.ItemCount != len(pack.Items) || pack.ItemCount == 0 {
		return nil, fmt.Errorf("validate locale entities %q: identity or count mismatch", path)
	}
	items := make(map[int]string, len(pack.Items))
	for rawID, entry := range pack.Items {
		id, err := strconv.Atoi(rawID)
		if err != nil || id < 0 || id > 255 || entry.Name == "" || len(entry.SourceStringIDs) == 0 {
			return nil, fmt.Errorf("validate locale entities %q: invalid item %q", path, rawID)
		}
		for _, sourceID := range entry.SourceStringIDs {
			if sourceID == "" {
				return nil, fmt.Errorf("validate locale entities %q: item %d has empty provenance", path, id)
			}
		}
		items[id] = entry.Name
	}
	return &EntityCatalog{Locale: locale, items: items}, nil
}

func (c *EntityCatalog) ItemName(id int) (string, error) {
	if c == nil {
		return "", errors.New("nil locale entity catalog")
	}
	name, ok := c.items[id]
	if !ok {
		return "", fmt.Errorf("missing localized item %d", id)
	}
	return name, nil
}
