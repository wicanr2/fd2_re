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
	Status          string   `json:"status,omitempty"`
}

type entityPack struct {
	SchemaVersion    int                        `json:"schema_version"`
	Kind             string                     `json:"kind"`
	Locale           string                     `json:"locale"`
	SourceLocale     string                     `json:"source_locale"`
	ItemCount        int                        `json:"item_count"`
	Items            map[string]entityNameEntry `json:"items"`
	CharacterCount   int                        `json:"character_count"`
	Characters       map[string]entityNameEntry `json:"characters"`
	BattleNameCount  int                        `json:"battle_name_count"`
	BattleNames      map[string]entityNameEntry `json:"battle_names"`
	CommandNameCount int                        `json:"command_name_count"`
	CommandNames     map[string]entityNameEntry `json:"command_names"`
	ClassNameCount   int                        `json:"class_name_count"`
	ClassNames       map[string]entityNameEntry `json:"class_names"`
}

// EntityCatalog 是以遊戲實體 ID 為鍵的不可變官方名稱目錄。
type EntityCatalog struct {
	Locale       string
	items        map[int]string
	characters   map[int]string
	battleNames  map[int]string
	commandNames map[int]string
	classNames   map[int]string
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
		pack.ItemCount != len(pack.Items) || pack.ItemCount == 0 ||
		pack.CharacterCount != len(pack.Characters) || pack.CharacterCount != 32 {
		return nil, fmt.Errorf("validate locale entities %q: identity or count mismatch", path)
	}
	if pack.BattleNameCount != len(pack.BattleNames) || pack.BattleNameCount != 94 {
		return nil, fmt.Errorf("validate locale entities %q: battle name count mismatch", path)
	}
	if pack.CommandNameCount != len(pack.CommandNames) || pack.CommandNameCount != 35 {
		return nil, fmt.Errorf("validate locale entities %q: command name count mismatch", path)
	}
	if pack.ClassNameCount != len(pack.ClassNames) || pack.ClassNameCount != 29 {
		return nil, fmt.Errorf("validate locale entities %q: class name count mismatch", path)
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
		if entry.Status != "original_confirmed" && entry.Status != "deterministic_script_conversion" &&
			entry.Status != "machine_draft" && entry.Status != "curated_remake_translation" {
			return nil, fmt.Errorf("validate locale entities %q: item %d has invalid status %q", path, id, entry.Status)
		}
		items[id] = entry.Name
	}
	characters := make(map[int]string, len(pack.Characters))
	for rawID, entry := range pack.Characters {
		id, err := strconv.Atoi(rawID)
		if err != nil || id < 0 || id >= 32 || entry.Name == "" || len(entry.SourceStringIDs) != 2 {
			return nil, fmt.Errorf("validate locale entities %q: invalid character %q", path, rawID)
		}
		if entry.Status != "original_confirmed" && entry.Status != "deterministic_script_conversion" &&
			entry.Status != "curated_remake_transliteration" {
			return nil, fmt.Errorf("validate locale entities %q: invalid character status %q", path, entry.Status)
		}
		characters[id] = entry.Name
	}
	battleNames := make(map[int]string, len(pack.BattleNames))
	for rawID, entry := range pack.BattleNames {
		id, err := strconv.Atoi(rawID)
		if err != nil || id < 0 || id > 138 || entry.Name == "" || len(entry.SourceStringIDs) != 1 {
			return nil, fmt.Errorf("validate locale entities %q: invalid battle name %q", path, rawID)
		}
		if entry.Status != "original_confirmed" && entry.Status != "deterministic_script_conversion" &&
			entry.Status != "curated_remake_transliteration" && entry.Status != "curated_remake_translation" {
			return nil, fmt.Errorf("validate locale entities %q: invalid battle name status %q", path, entry.Status)
		}
		battleNames[id] = entry.Name
	}
	commandNames := make(map[int]string, len(pack.CommandNames))
	for rawID, entry := range pack.CommandNames {
		id, err := strconv.Atoi(rawID)
		if err != nil || id < 0 || id > 35 || id == 31 || entry.Name == "" || len(entry.SourceStringIDs) != 1 {
			return nil, fmt.Errorf("validate locale entities %q: invalid command name %q", path, rawID)
		}
		if entry.Status != "original_confirmed" && entry.Status != "deterministic_script_conversion" &&
			entry.Status != "curated_remake_translation" && entry.Status != "curated_source_glyph_correction" {
			return nil, fmt.Errorf("validate locale entities %q: invalid command name status %q", path, entry.Status)
		}
		commandNames[id] = entry.Name
	}
	classNames := make(map[int]string, len(pack.ClassNames))
	for rawID, entry := range pack.ClassNames {
		id, err := strconv.Atoi(rawID)
		if err != nil || id < 0 || id >= 29 || entry.Name == "" || len(entry.SourceStringIDs) != 1 {
			return nil, fmt.Errorf("validate locale entities %q: invalid class name %q", path, rawID)
		}
		placeholder := id == 26 || id == 27 || id == 28
		if placeholder && entry.Status != "original_placeholder" {
			return nil, fmt.Errorf("validate locale entities %q: class %d lost placeholder status", path, id)
		}
		if !placeholder && entry.Status != "original_confirmed" &&
			entry.Status != "deterministic_script_conversion" &&
			entry.Status != "curated_remake_translation" {
			return nil, fmt.Errorf("validate locale entities %q: invalid class status %q", path, entry.Status)
		}
		classNames[id] = entry.Name
	}
	return &EntityCatalog{
		Locale: locale, items: items, characters: characters,
		battleNames: battleNames, commandNames: commandNames, classNames: classNames,
	}, nil
}

func (c *EntityCatalog) CharacterName(nativeIdentity int) (string, error) {
	if c == nil {
		return "", errors.New("nil locale entity catalog")
	}
	name, ok := c.characters[nativeIdentity]
	if !ok {
		return "", fmt.Errorf("missing localized character %d", nativeIdentity)
	}
	return name, nil
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

// BattleName 以原版戰鬥 record +8 的 raw selector 查詢面板名稱。
// 它與隊伍 native_identity 是兩個不同契約，不能互相替代。
func (c *EntityCatalog) BattleName(rawID int) (string, error) {
	if c == nil {
		return "", errors.New("nil locale entity catalog")
	}
	name, ok := c.battleNames[rawID]
	if !ok {
		return "", fmt.Errorf("missing localized battle name %d", rawID)
	}
	return name, nil
}

// CommandName 以原版指令 ID 查詢玩家指令格名稱；ID31 是已證實空槽。
func (c *EntityCatalog) CommandName(rawID int) (string, error) {
	if c == nil {
		return "", errors.New("nil locale entity catalog")
	}
	name, ok := c.commandNames[rawID]
	if !ok {
		return "", fmt.Errorf("missing localized command name %d", rawID)
	}
	return name, nil
}

// ClassName 以原版ClassID查詢職業名稱；26／27／28保留原版占位語意。
func (c *EntityCatalog) ClassName(classID int) (string, error) {
	if c == nil {
		return "", errors.New("nil locale entity catalog")
	}
	name, ok := c.classNames[classID]
	if !ok {
		return "", fmt.Errorf("missing localized class %d", classID)
	}
	return name, nil
}
