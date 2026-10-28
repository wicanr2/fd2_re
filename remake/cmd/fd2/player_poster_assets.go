package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
)

const playerPosterAssetRole = "presentation_poster_pair"
const playerPosterWidth = 132

type playerPosterSettings struct {
	Set    string `json:"set"`
	Edge   int    `json:"edge"`
	FadeMS int    `json:"fade_ms"`
}

func defaultPlayerPosters() playerPosterSettings {
	return playerPosterSettings{Set: "off", Edge: 48, FadeMS: 600}
}

func validPosterMode(mode string) bool {
	return mode == "off" || mode == "hd" || mode == "classic"
}

func normalizePlayerPosters(p *playerPosterSettings) playerPosterSettings {
	if p == nil {
		return defaultPlayerPosters()
	}
	v := *p
	if !validPosterMode(v.Set) {
		v.Set = "off"
	}
	if v.Edge < 0 || v.Edge > 96 {
		v.Edge = 48
	}
	switch v.FadeMS {
	case 0, 300, 600, 1200:
	default:
		v.FadeMS = 600
	}
	return v
}

type playerPosterPair struct {
	// Both sets retain their unmodified approved pixels. Only the selected
	// left/right rectangles enter the game; the baked concept UI never does.
	Sides [2][2]*image.RGBA
}

func loadPlayerPosterPair(catalogPath, root string) (*playerPosterPair, error) {
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, err
	}
	if catalog.SchemaVersion != 1 || catalog.ThemeID != modernHandpaintedThemeID {
		return nil, fmt.Errorf("posters: unsupported catalog identity")
	}
	var entries [2]*modernThemeAsset
	for i := range catalog.Assets {
		a := &catalog.Assets[i]
		if a.Role != playerPosterAssetRole {
			continue
		}
		if a.Frame < 0 || a.Frame > 1 || entries[a.Frame] != nil ||
			(a.Status != "runtime_candidate" && a.Status != "runtime_ready") ||
			a.ConsumerContract != "composite_poster_sides_v1" ||
			a.LayoutPolicy != "native_game_center_sides_v1" || a.AlphaContract != "opaque" ||
			a.Width != 1672 || a.Height != 941 || filepath.Base(a.File) != a.File || filepath.Ext(a.File) != ".png" {
			return nil, fmt.Errorf("posters: invalid or duplicate pair contract")
		}
		entries[a.Frame] = a
	}
	if entries[0] == nil || entries[1] == nil {
		return nil, fmt.Errorf("posters: both approved sets are required")
	}
	result := &playerPosterPair{}
	for index, a := range entries {
		path := filepath.Join(root, a.File)
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() > 32<<20 {
			return nil, fmt.Errorf("posters: invalid file size")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != a.SHA256 {
			return nil, fmt.Errorf("posters: sha256 mismatch for set %d", index)
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || format != "png" || cfg.Width != 1672 || cfg.Height != 941 {
			return nil, fmt.Errorf("posters: invalid PNG dimensions")
		}
		picture, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		for y := 0; y < 941; y++ {
			for x := 0; x < 1672; x++ {
				if _, _, _, alpha := picture.At(x, y).RGBA(); alpha != 65535 {
					return nil, fmt.Errorf("posters: source must be opaque")
				}
			}
		}
		for side, x := range []int{0, 1376} {
			crop := image.NewRGBA(image.Rect(0, 0, 296, 900))
			draw.Draw(crop, crop.Bounds(), picture, image.Pt(x, 0), draw.Src)
			result.Sides[index][side] = crop
		}
	}
	return result, nil
}

// The prototype's inner edge has alpha 0, midpoint .25, outer edge 1.
func playerPosterEdgeAlpha(distance, width int) float64 {
	if width <= 0 || distance >= width {
		return 1
	}
	if distance <= 0 {
		return 0
	}
	t := float64(distance) / float64(width)
	if t < .5 {
		return .5 * t
	}
	return 1.5*t - .5
}

func maskPlayerPoster(source *image.RGBA, side, width int) *image.NRGBA {
	result := image.NewNRGBA(source.Bounds())
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			c := source.RGBAAt(x, y)
			distance := source.Bounds().Dx() - 1 - x
			if side == 1 {
				distance = x
			}
			i := result.PixOffset(x, y)
			result.Pix[i], result.Pix[i+1], result.Pix[i+2] = c.R, c.G, c.B
			result.Pix[i+3] = uint8(playerPosterEdgeAlpha(distance, width)*255 + .5)
		}
	}
	return result
}
