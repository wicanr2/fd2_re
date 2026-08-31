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
	portraits map[int]*modernStoryPortraitFrame
}

type modernThemeCatalog struct {
	SchemaVersion int    `json:"schema_version"`
	ThemeID       string `json:"theme_id"`
	Assets        []struct {
		Role             string `json:"role"`
		Status           string `json:"status"`
		File             string `json:"file"`
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		SHA256           string `json:"sha256"`
		ConsumerContract string `json:"consumer_contract"`
		SpeakerID        int    `json:"speaker_id"`
		Frame            int    `json:"frame"`
		MouthState       string `json:"mouth_state"`
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
	set := &modernStoryPortraitSet{portraits: make(map[int]*modernStoryPortraitFrame)}
	for _, asset := range catalog.Assets {
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
