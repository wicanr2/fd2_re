package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

const modernHandpaintedThemeID = "modern-handpainted-a"

type modernStoryPortraitFrame struct {
	image image.Image
}

type modernStoryPortraitSet struct {
	portraits         map[int]*modernStoryPortraitFrame
	mapSprites        map[int][]image.Image
	mapTilesets       map[int]image.Image
	battleHUDPanel    image.Image
	battleActionIcons []image.Image
}

type modernThemeCatalog struct {
	SchemaVersion int    `json:"schema_version"`
	ThemeID       string `json:"theme_id"`
	Assets        []struct {
		Role             string   `json:"role"`
		Status           string   `json:"status"`
		File             string   `json:"file"`
		Width            int      `json:"width"`
		Height           int      `json:"height"`
		SHA256           string   `json:"sha256"`
		ConsumerContract string   `json:"consumer_contract"`
		SpeakerID        int      `json:"speaker_id"`
		Frame            int      `json:"frame"`
		MouthState       string   `json:"mouth_state"`
		Files            []string `json:"files"`
		FrameSHA256      []string `json:"frame_sha256"`
		FrameCount       int      `json:"frame_count"`
		SourceGroup      int      `json:"source_group"`
		AlphaContract    string   `json:"alpha_contract"`
		CyclePolicy      string   `json:"cycle_policy"`
		MapID            int      `json:"map_id"`
		TileWidth        int      `json:"tile_width"`
		TileHeight       int      `json:"tile_height"`
		Columns          int      `json:"columns"`
		TileCount        int      `json:"tile_count"`
		Semantics        []string `json:"semantics"`
	} `json:"assets"`
}

// loadModernStoryPortraitSet 只承認已通過受版控 catalog 的獨立故事頭像。
// catalog 可公開；packRoot 指向玩家本機或私人完整版中的實體 PNG。
func loadModernStoryPortraitSet(catalogPath, packRoot string) (*modernStoryPortraitSet, error) {
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("modern theme catalog: %w", err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("modern theme catalog: %w", err)
	}
	if catalog.SchemaVersion != 1 || catalog.ThemeID != modernHandpaintedThemeID {
		return nil, errors.New("modern theme catalog: unsupported identity")
	}
	set := &modernStoryPortraitSet{
		portraits:   make(map[int]*modernStoryPortraitFrame),
		mapSprites:  make(map[int][]image.Image),
		mapTilesets: make(map[int]image.Image),
	}
	for _, asset := range catalog.Assets {
		if asset.Role == "battle_action_icon_set" {
			wantSemantics := []string{"attack", "spell", "item", "wait"}
			if asset.Status != "runtime_candidate" && asset.Status != "runtime_ready" {
				return nil, fmt.Errorf("modern action icons have unsupported status %q", asset.Status)
			}
			if asset.ConsumerContract != "battle_action_icons_4x28x26_v1" ||
				asset.Width != 28 || asset.Height != 26 || asset.FrameCount != 4 ||
				len(asset.Files) != 4 || len(asset.FrameSHA256) != 4 || len(asset.Semantics) != 4 {
				return nil, errors.New("modern action icons violate the set contract")
			}
			for index, semantic := range wantSemantics {
				if asset.Semantics[index] != semantic {
					return nil, errors.New("modern action icons have an invalid semantic order")
				}
			}
			if set.battleActionIcons != nil {
				return nil, errors.New("modern action icons are duplicated")
			}
			icons := make([]image.Image, 0, 4)
			for index, file := range asset.Files {
				name := filepath.Base(file)
				if name != file || filepath.Ext(name) != ".png" {
					return nil, fmt.Errorf("modern action icon %d has an unsafe path", index)
				}
				pngRaw, err := os.ReadFile(filepath.Join(packRoot, name))
				if err != nil {
					return nil, fmt.Errorf("modern action icon %d: %w", index, err)
				}
				digest := sha256.Sum256(pngRaw)
				if hex.EncodeToString(digest[:]) != asset.FrameSHA256[index] {
					return nil, fmt.Errorf("modern action icon %d has a sha256 mismatch", index)
				}
				decoded, _, err := image.Decode(bytes.NewReader(pngRaw))
				if err != nil || decoded.Bounds().Dx() != 28 || decoded.Bounds().Dy() != 26 {
					return nil, fmt.Errorf("modern action icon %d has invalid PNG geometry", index)
				}
				for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
					for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
						_, _, _, alpha := decoded.At(x, y).RGBA()
						if alpha != 0xffff {
							return nil, fmt.Errorf("modern action icon %d is not fully opaque", index)
						}
					}
				}
				icons = append(icons, decoded)
			}
			set.battleActionIcons = icons
			continue
		}
		if asset.Role == "battle_hud_panel" {
			if asset.Status != "runtime_candidate" && asset.Status != "runtime_ready" {
				return nil, fmt.Errorf("modern battle HUD has unsupported status %q", asset.Status)
			}
			if asset.ConsumerContract != "battle_status_panel_149x42_v1" || asset.Width != 149 || asset.Height != 42 {
				return nil, errors.New("modern battle HUD violates the panel contract")
			}
			if set.battleHUDPanel != nil {
				return nil, errors.New("modern battle HUD is duplicated")
			}
			name := filepath.Base(asset.File)
			if name != asset.File || filepath.Ext(name) != ".png" {
				return nil, errors.New("modern battle HUD has an unsafe path")
			}
			pngRaw, err := os.ReadFile(filepath.Join(packRoot, name))
			if err != nil {
				return nil, fmt.Errorf("modern battle HUD: %w", err)
			}
			digest := sha256.Sum256(pngRaw)
			if hex.EncodeToString(digest[:]) != asset.SHA256 {
				return nil, errors.New("modern battle HUD has a sha256 mismatch")
			}
			decoded, _, err := image.Decode(bytes.NewReader(pngRaw))
			if err != nil || decoded.Bounds().Dx() != 149 || decoded.Bounds().Dy() != 42 {
				return nil, errors.New("modern battle HUD has invalid PNG geometry")
			}
			for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
				for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
					_, _, _, alpha := decoded.At(x, y).RGBA()
					if alpha != 0xffff {
						return nil, errors.New("modern battle HUD is not fully opaque")
					}
				}
			}
			set.battleHUDPanel = decoded
			continue
		}
		if asset.Role == "map_tileset_set" {
			if asset.Status != "runtime_candidate" && asset.Status != "runtime_ready" {
				return nil, fmt.Errorf("modern theme map %d has unsupported status %q", asset.MapID, asset.Status)
			}
			if asset.ConsumerContract != "map_tileset_indexed_geometry_v1" ||
				asset.MapID < 0 || asset.MapID >= 33 || asset.TileWidth != 24 || asset.TileHeight != 24 ||
				asset.Columns != 16 || asset.Width != asset.Columns*asset.TileWidth ||
				asset.Height <= 0 || asset.Height%asset.TileHeight != 0 ||
				asset.TileCount != (asset.Height/asset.TileHeight)*asset.Columns {
				return nil, fmt.Errorf("modern theme map %d violates the tileset contract", asset.MapID)
			}
			if _, duplicate := set.mapTilesets[asset.MapID]; duplicate {
				return nil, fmt.Errorf("modern theme map %d is duplicated", asset.MapID)
			}
			name := filepath.Base(asset.File)
			if name != asset.File || filepath.Ext(name) != ".png" {
				return nil, fmt.Errorf("modern theme map %d has an unsafe path", asset.MapID)
			}
			pngRaw, err := os.ReadFile(filepath.Join(packRoot, name))
			if err != nil {
				return nil, fmt.Errorf("modern theme map %d: %w", asset.MapID, err)
			}
			digest := sha256.Sum256(pngRaw)
			if hex.EncodeToString(digest[:]) != asset.SHA256 {
				return nil, fmt.Errorf("modern theme map %d has a sha256 mismatch", asset.MapID)
			}
			decoded, _, err := image.Decode(bytes.NewReader(pngRaw))
			if err != nil || decoded.Bounds().Dx() != asset.Width || decoded.Bounds().Dy() != asset.Height {
				return nil, fmt.Errorf("modern theme map %d has invalid PNG geometry", asset.MapID)
			}
			set.mapTilesets[asset.MapID] = decoded
			continue
		}
		if asset.Role == "map_sprite_set" {
			if asset.Status != "runtime_candidate" && asset.Status != "runtime_ready" {
				return nil, fmt.Errorf("modern theme map sprite %d has unsupported status %q", asset.SourceGroup, asset.Status)
			}
			walkContract := asset.ConsumerContract == "fdicon_map_sprite_12x24_v1" &&
				asset.FrameCount == 12 && len(asset.Files) == 12 && len(asset.FrameSHA256) == 12 &&
				(asset.CyclePolicy == "three_distinct_cycles" || asset.CyclePolicy == "source_exact_repeats")
			staticContract := asset.ConsumerContract == "fdicon_static_sprite_3x24_v1" &&
				asset.FrameCount == 3 && len(asset.Files) == 3 && len(asset.FrameSHA256) == 3 &&
				asset.CyclePolicy == "static_three_phase"
			if (!walkContract && !staticContract) || asset.Width != 24 || asset.Height != 24 ||
				asset.SourceGroup < 0 || asset.SourceGroup >= 96 || asset.AlphaContract != "binary" {
				return nil, fmt.Errorf("modern theme map sprite %d violates the frame contract", asset.SourceGroup)
			}
			if _, duplicate := set.mapSprites[asset.SourceGroup]; duplicate {
				return nil, fmt.Errorf("modern theme map sprite %d is duplicated", asset.SourceGroup)
			}
			frames := make([]image.Image, 0, asset.FrameCount)
			seenDigest := make(map[string]struct{}, asset.FrameCount)
			repeatedFrame := false
			for frame, file := range asset.Files {
				name := filepath.Base(file)
				if name != file || filepath.Ext(name) != ".png" {
					return nil, fmt.Errorf("modern theme map sprite %d frame %d has an unsafe path", asset.SourceGroup, frame)
				}
				pngRaw, err := os.ReadFile(filepath.Join(packRoot, name))
				if err != nil {
					return nil, fmt.Errorf("modern theme map sprite %d frame %d: %w", asset.SourceGroup, frame, err)
				}
				digest := sha256.Sum256(pngRaw)
				digestText := hex.EncodeToString(digest[:])
				if digestText != asset.FrameSHA256[frame] {
					return nil, fmt.Errorf("modern theme map sprite %d frame %d has a sha256 mismatch", asset.SourceGroup, frame)
				}
				if _, duplicate := seenDigest[digestText]; duplicate && asset.CyclePolicy != "source_exact_repeats" {
					return nil, fmt.Errorf("modern theme map sprite %d repeats frame %d", asset.SourceGroup, frame)
				} else if duplicate {
					repeatedFrame = true
				}
				seenDigest[digestText] = struct{}{}
				decoded, _, err := image.Decode(bytes.NewReader(pngRaw))
				if err != nil || decoded.Bounds().Dx() != 24 || decoded.Bounds().Dy() != 24 {
					return nil, fmt.Errorf("modern theme map sprite %d frame %d has invalid PNG geometry", asset.SourceGroup, frame)
				}
				for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
					for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
						_, _, _, alpha := decoded.At(x, y).RGBA()
						if alpha != 0 && alpha != 0xffff {
							return nil, fmt.Errorf("modern theme map sprite %d frame %d has non-binary alpha", asset.SourceGroup, frame)
						}
					}
				}
				frames = append(frames, decoded)
			}
			if asset.CyclePolicy == "source_exact_repeats" && !repeatedFrame {
				return nil, fmt.Errorf("modern theme map sprite %d declares source repeats but has none", asset.SourceGroup)
			}
			if staticContract && len(seenDigest) != 3 {
				return nil, fmt.Errorf("modern theme static map sprite %d does not have three distinct phases", asset.SourceGroup)
			}
			set.mapSprites[asset.SourceGroup] = frames
			continue
		}
		if asset.Role != "story_portrait_frame" {
			continue
		}
		if asset.Status != "runtime_candidate" && asset.Status != "runtime_ready" {
			return nil, fmt.Errorf("modern theme portrait %d has unsupported status %q", asset.SpeakerID, asset.Status)
		}
		if asset.ConsumerContract != "native_story_dialogue_rgba_overlay_v1" ||
			asset.Width != 80 || asset.Height != 80 || asset.SpeakerID < 0 ||
			asset.Frame != 0 || asset.MouthState != "closed" {
			return nil, fmt.Errorf("modern theme portrait %d violates the static frame contract", asset.SpeakerID)
		}
		if _, duplicate := set.portraits[asset.SpeakerID]; duplicate {
			return nil, fmt.Errorf("modern theme portrait %d is duplicated", asset.SpeakerID)
		}
		name := filepath.Base(asset.File)
		if name != asset.File || filepath.Ext(name) != ".png" {
			return nil, fmt.Errorf("modern theme portrait %d has an unsafe path", asset.SpeakerID)
		}
		pngPath := filepath.Join(packRoot, name)
		pngRaw, err := os.ReadFile(pngPath)
		if err != nil {
			return nil, fmt.Errorf("modern theme portrait %d: %w", asset.SpeakerID, err)
		}
		digest := sha256.Sum256(pngRaw)
		if hex.EncodeToString(digest[:]) != asset.SHA256 {
			return nil, fmt.Errorf("modern theme portrait %d has a sha256 mismatch", asset.SpeakerID)
		}
		decoded, _, err := image.DecodeConfig(bytes.NewReader(pngRaw))
		if err != nil || decoded.Width != 80 || decoded.Height != 80 {
			return nil, fmt.Errorf("modern theme portrait %d has invalid PNG geometry", asset.SpeakerID)
		}
		portrait, _, err := image.Decode(bytes.NewReader(pngRaw))
		if err != nil || portrait.Bounds().Dx() != 80 || portrait.Bounds().Dy() != 80 {
			return nil, fmt.Errorf("modern theme portrait %d cannot be decoded", asset.SpeakerID)
		}
		for y := portrait.Bounds().Min.Y; y < portrait.Bounds().Max.Y; y++ {
			for x := portrait.Bounds().Min.X; x < portrait.Bounds().Max.X; x++ {
				_, _, _, alpha := portrait.At(x, y).RGBA()
				if alpha != 0xffff {
					return nil, fmt.Errorf("modern theme portrait %d is not fully opaque", asset.SpeakerID)
				}
			}
		}
		set.portraits[asset.SpeakerID] = &modernStoryPortraitFrame{image: portrait}
	}
	if len(set.portraits) == 0 {
		return nil, errors.New("modern theme catalog has no story portrait")
	}
	return set, nil
}
