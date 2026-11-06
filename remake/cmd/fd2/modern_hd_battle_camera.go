package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

const modernBattlePresentationRole = "battle_presentation_profile"

type modernBattlePresentationProfile struct {
	SchemaVersion    int     `json:"schema_version"`
	MaxZoom          float64 `json:"max_zoom"`
	ImpactHoldFrames int     `json:"impact_hold_frames"`
}

func loadModernBattlePresentation(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	if set == nil || set.battlePresentation != nil || asset.Role != modernBattlePresentationRole ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.ConsumerContract != "hd1_battle_presentation_v1" || asset.File == "" ||
		filepath.Base(asset.File) != asset.File || filepath.Ext(asset.File) != ".json" {
		return fmt.Errorf("modern battle presentation: invalid asset contract")
	}
	raw, err := os.ReadFile(filepath.Join(root, asset.File))
	if err != nil || len(raw) > 4096 || !modernPhysicalSFXDigest(raw, asset.SHA256) {
		return fmt.Errorf("modern battle presentation: digest mismatch")
	}
	var profile modernBattlePresentationProfile
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&profile); err != nil || profile.SchemaVersion != 1 ||
		profile.MaxZoom != 1.07 || profile.ImpactHoldFrames != 4 {
		return fmt.Errorf("modern battle presentation: unsupported profile")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("modern battle presentation: trailing data")
	}
	set.battlePresentation = &profile
	return nil
}

// The two native panels are independently bound only when every source byte
// matches. Partial scroll panels do not admit a camera and retain their owner.
func modernBattleCameraBase(sources *modernHDBattleSources) ([]byte, bool) {
	if sources == nil || len(sources.Base) != 64000 {
		return nil, false
	}
	panels := map[image.Rectangle]bool{}
	scenery := false
	for _, source := range sources.BaseScenery {
		if source.Archive == "BG.DAT" || source.Archive == "TAI.DAT" {
			scenery = true
		}
		f := source.Frame
		if source.PanelText != nil && source.DX == 0 && source.DY == 0 && source.Clip == nil &&
			f.Width == 149 && f.Height == 42 {
			panels[image.Rect(f.X, f.Y, f.X+f.Width, f.Y+f.Height)] = true
		}
	}
	want := []image.Rectangle{image.Rect(171, 4, 320, 46), image.Rect(0, 154, 149, 196)}
	if !scenery || len(panels) != 2 || !panels[want[0]] || !panels[want[1]] {
		return nil, false
	}
	base := append([]byte(nil), sources.Base...)
	for _, panel := range want {
		for y := panel.Min.Y; y < panel.Max.Y; y++ {
			clear(base[y*320+panel.Min.X : y*320+panel.Max.X])
		}
	}
	return base, true
}

func (r *modernHDBattleGPU) presentationZoom(frame int, maximum float64) float64 {
	if !r.cameraActive || r.cameraOwner != r.owner || r.cameraTheme != r.theme {
		r.cameraStartFrame, r.cameraOwner, r.cameraTheme = frame, r.owner, r.theme
		r.cameraActive = true
	}
	t := math.Max(0, math.Min(1, float64(frame-r.cameraStartFrame)/54))
	return 1 + (maximum-1)*t*t*(3-2*t)
}

func modernBattleCameraGeometry(geometry *ebiten.GeoM, zoom float64, bounds image.Rectangle) {
	if zoom <= 1 {
		return
	}
	geometry.Scale(zoom, zoom)
	geometry.Translate((1-zoom)*float64(bounds.Dx())/2, (1-zoom)*float64(bounds.Dy())/2)
}

func modernBattleCameraClip(rect image.Rectangle, zoom float64) image.Rectangle {
	if zoom <= 1 {
		return rect
	}
	x := (1 - zoom) * modernHDWidth / 2
	y := (1 - zoom) * modernHDHeight / 2
	return image.Rect(int(math.Floor(float64(rect.Min.X)*zoom+x)), int(math.Floor(float64(rect.Min.Y)*zoom+y)),
		int(math.Ceil(float64(rect.Max.X)*zoom+x)), int(math.Ceil(float64(rect.Max.Y)*zoom+y)))
}

func (g *Game) modernPhysicalImpactHoldMillis(impact bool) float64 {
	if !impact || g.modernStoryPortraits == nil || g.modernStoryPortraits.battlePresentation == nil {
		return 0
	}
	return float64(g.modernStoryPortraits.battlePresentation.ImpactHoldFrames) * 1000 / 60
}
