package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// nativePhysicalScene 是一次正常 0x28A6C 的場景快照，反擊沿用同一份。
// READY 子規格見 docs/data/ida/fd2_physical_background_selection_20261004.json。
// 非零 header 的滑入與完整 DAC 演出仍由既有 E1 owner 處理。
type nativePhysicalScene struct {
	selection            battle.NativePhysicalSceneSelection
	background, pedestal *ebiten.Image
}

func (g *Game) nativePhysicalTerrainControl(unit *battle.Unit) ([4]byte, error) {
	if g == nil || g.st == nil || g.m == nil || unit == nil ||
		g.m.W != g.st.W || g.m.H != g.st.H || g.m.W <= 0 || g.m.H <= 0 ||
		unit.X < 0 || unit.Y < 0 || unit.X >= g.m.W || unit.Y >= g.m.H ||
		len(g.m.NativeTerrainControl) == 0 || len(g.m.NativeTerrainControl)%4 != 0 {
		return [4]byte{}, errors.New("native physical terrain control unavailable")
	}
	var tile int
	if g.st.HasNativeMapEventGrid {
		var ok bool
		tile, ok = g.st.NativeMapTileAt(unit.X, unit.Y)
		if !ok {
			return [4]byte{}, errors.New("native physical mutable map is malformed")
		}
	} else {
		if len(g.m.Tiles) != g.m.W*g.m.H {
			return [4]byte{}, errors.New("native physical map tiles unavailable")
		}
		tile = g.m.Tiles[unit.Y*g.m.W+unit.X]
	}
	if tile < 0 || tile > 0x3ff || tile >= len(g.m.NativeTerrainControl)/4 {
		return [4]byte{}, errors.New("native physical terrain selector out of range")
	}
	var control [4]byte
	copy(control[:], g.m.NativeTerrainControl[tile*4:tile*4+4])
	return control, nil
}

// nativePhysicalSurfaceImage 保留 mask 與 index0 的差別，不以透明 PNG 補值。
func nativePhysicalSurfaceRGBA(frame fdother.Frame, palette color.Palette) (*image.RGBA, error) {
	if frame.Width <= 0 || frame.Height <= 0 || frame.Width > 320 || frame.Height > 200 ||
		len(palette) != 256 || len(frame.Indexed) != frame.Width*frame.Height ||
		len(frame.Mask) != len(frame.Indexed) || len(frame.Pixels) != 0 {
		return nil, errors.New("native physical surface is malformed")
	}
	img := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for at, index := range frame.Indexed {
		if frame.Mask[at] == 0 {
			continue
		}
		if frame.Mask[at] != 255 {
			return nil, errors.New("native physical surface mask is malformed")
		}
		r, gr, b, _ := palette[index].RGBA()
		img.SetRGBA(at%frame.Width, at/frame.Width, color.RGBA{byte(r >> 8), byte(gr >> 8), byte(b >> 8), 255})
	}
	return img, nil
}

func nativePhysicalSurfaceImage(frame fdother.Frame, palette color.Palette) (*ebiten.Image, error) {
	img, err := nativePhysicalSurfaceRGBA(frame, palette)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

// prepareNativePhysicalScene 在任何結算寫入前驗證正常場景的 BG／TAI 來源。
// 無原生地圖狀態的相容場景仍由自己的既有 owner 處理。
func (g *Game) prepareNativePhysicalScene(actor, target *battle.Unit) (*nativePhysicalScene, error) {
	if g == nil || g.st == nil {
		return nil, errors.New("native physical scene state unavailable")
	}
	if !g.st.HasNativeMapViewState {
		return nil, nil
	}
	initial, err := g.nativeCommandScene.InitialBackground(g.handlerChapter)
	if err != nil {
		return nil, err
	}
	actorControl, err := g.nativePhysicalTerrainControl(actor)
	if err != nil {
		return nil, err
	}
	targetControl, err := g.nativePhysicalTerrainControl(target)
	if err != nil {
		return nil, err
	}
	if actor == nil || !actor.HasBattleFig || actor.BattleFig < 0 || actor.BattleFig > 255 {
		return nil, errors.New("native physical actor FIGANI selector unavailable")
	}
	animation, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3+1)
	if err != nil {
		return nil, err
	}
	selection, err := battle.SelectNativePhysicalScene(initial, actor, target, actorControl, targetControl, animation.HeaderByte1)
	if err != nil {
		return nil, err
	}
	root := separatedAssetPath("surfaces")
	bg, err := fdother.LoadSeparatedSingleFrame(root, "BG.DAT", int(selection.BaseBG))
	if err != nil {
		return nil, fmt.Errorf("native physical BG%d: %w", selection.BaseBG, err)
	}
	tai, err := fdother.LoadSeparatedSingleFrame(root, "TAI.DAT", int(selection.TAI))
	if err != nil {
		return nil, fmt.Errorf("native physical TAI%d: %w", selection.TAI, err)
	}
	background, err := nativePhysicalSurfaceImage(bg, g.nativeUIPalette)
	if err != nil {
		return nil, err
	}
	pedestal, err := nativePhysicalSurfaceImage(tai, g.nativeUIPalette)
	if err != nil {
		return nil, err
	}
	return &nativePhysicalScene{selection: selection, background: background, pedestal: pedestal}, nil
}
