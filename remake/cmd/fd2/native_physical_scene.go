package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// nativePhysicalScene 是一次正常 0x28A6C 的場景快照，反擊沿用同一份。
// READY 子規格見 docs/data/ida/fd2_physical_background_selection_20261004.json。
// 首次非零 header departure／scroll 已接線；完整 DAC 與尾段仍屬 E1。
type nativePhysicalScene struct {
	selection                 battle.NativePhysicalSceneSelection
	background, pedestal      *ebiten.Image
	backgroundFrame           fdother.Frame
	panelAssets               battle.NativeItemPanelDataAssets
	actorRecord, targetRecord []byte
	actorIndex, targetIndex   int
	prelude                   []battlepresent.NativeCommand24PreludeFrame
	preludeImages             []*ebiten.Image
	preludeFrame              int
	preludeDrawn              bool
	counterActive             bool
	baseImage                 *ebiten.Image
	baseActorHP, baseTargetHP int
	departure                 [][]byte
	transition                [][]byte
	leadImages                []*ebiten.Image
	leadDelays                []int
	leadFrame, leadTicks      int
	leadDrawn                 bool
	leadBodyTicks             int
	attackStart               int
	targetBackground          fdother.Frame
	platformFrame             fdother.Frame
	rawSide                   byte
	bodyResources             *nativePhysicalBodyResources
	body                      *nativePhysicalBodyPlayback
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
	if err := g.preflightNativePhysicalMapReturn(); err != nil {
		return nil, fmt.Errorf("native physical map return preflight: %w", err)
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
	scene := &nativePhysicalScene{selection: selection, background: background, pedestal: pedestal}
	if err := g.prepareNativePhysicalPrelude(scene, actor, target, animation, bg, tai); err != nil {
		return nil, err
	}
	if err := g.prepareNativePhysicalBodyResources(scene, actor, target, animation); err != nil {
		return nil, err
	}
	return scene, nil
}

// preflightNativePhysicalMapReturn 用候選資料驗證既有11CAC合成器。
// 不發布候選的State、clock或畫布；資產與單位保持唯讀。
func (g *Game) preflightNativePhysicalMapReturn() error {
	if g.nativeMapAssets == nil {
		return nil
	}
	probe := *g
	state := *g.st
	probe.st = &state
	probe.nativeMapWork = append([]byte(nil), g.nativeMapWork...)
	probe.nativeMapVGA = append([]byte(nil), g.nativeMapVGA...)
	probe.nativeMapDAC = append([]byte(nil), g.nativeMapDAC...)
	return probe.composeNativeMapFrameForPhysicalReturn()
}

// nativePhysicalBase 保留 0x28CE7→0x28D48→0x28D62 的順序。
// 320×200 base 的清零來自 0x28B41..0x28B4E 明確 memset。
func (g *Game) nativePhysicalBase(scene *nativePhysicalScene, actorRecord, targetRecord []byte) ([]byte, error) {
	base := make([]byte, 320*200)
	if err := g.renderLocalizedNativeBattlePanel(scene.panelAssets, actorRecord, base, scene.actorIndex, g.handlerChapter); err != nil {
		return nil, err
	}
	bg := scene.backgroundFrame
	bg.X, bg.Y = 0, 50
	if err := bg.Blit(base, 320, -1); err != nil {
		return nil, err
	}
	if !scene.selection.HasSeparateBackgrounds {
		if err := g.renderLocalizedNativeBattlePanel(scene.panelAssets, targetRecord, base, scene.targetIndex, g.handlerChapter); err != nil {
			return nil, err
		}
	}
	return base, nil
}

func (g *Game) prepareNativePhysicalPrelude(scene *nativePhysicalScene, actor, target *battle.Unit, attack *figani.Animation, bg, tai fdother.Frame) error {
	var err error
	scene.backgroundFrame = bg
	scene.actorRecord, err = battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		return err
	}
	scene.targetRecord, err = battle.NativeBattlePanelRecordForUnit(target)
	if err != nil {
		return err
	}
	scene.actorIndex, err = nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	scene.targetIndex, err = nativeCommand24RuntimeUnitIndex(g.st, target)
	if err != nil {
		return err
	}
	scene.panelAssets, err = battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	base, err := g.nativePhysicalBase(scene, scene.actorRecord, scene.targetRecord)
	if err != nil {
		return err
	}
	actorIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
	if err != nil {
		return err
	}
	targetIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3)
	if err != nil {
		return err
	}
	if len(actorIdle.Frames) == 0 || len(targetIdle.Frames) == 0 {
		return errors.New("native physical prelude idle unavailable")
	}
	dac, _, err := loadNativeBattlePalette()
	if err != nil {
		return err
	}
	mode := byte(0)
	if scene.selection.HasSeparateBackgrounds {
		mode = 1
	}
	scene.prelude, err = battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: base, ActorIdle: actorIdle.Frames[0], FirstTargetIdle: targetIdle.Frames[0],
		Platform: &tai, RawSide: actor.NativeRecordByte6, Mode: mode, BaselineDAC: dac,
	})
	if err != nil {
		return err
	}
	scene.preludeImages, err = nativeCommand24PreludeImages(scene.prelude)
	if err != nil || !scene.selection.HasSeparateBackgrounds {
		return err
	}
	return g.prepareNativePhysicalDeparture(scene, actor, attack, targetIdle, base, tai)
}

func (g *Game) nativePhysicalTargetBase(scene *nativePhysicalScene, record []byte) ([]byte, error) {
	base := make([]byte, 320*200) // 29C90／29DED 明確清除 base。
	bg := scene.targetBackground
	bg.X, bg.Y = 0, 50
	if err := bg.Blit(base, 320, -1); err != nil {
		return nil, err
	}
	if scene.rawSide == 0 {
		platform := scene.platformFrame
		platform.X, platform.Y = 164, 157
		if err := platform.Blit(base, 320, -1); err != nil {
			return nil, err
		}
	}
	if err := g.renderLocalizedNativeBattlePanel(scene.panelAssets, record, base, scene.targetIndex, g.handlerChapter); err != nil {
		return nil, err
	}
	return base, nil
}

func (g *Game) prepareNativePhysicalDeparture(scene *nativePhysicalScene, actor *battle.Unit, attack, targetIdle *figani.Animation, base []byte, tai fdother.Frame) error {
	if attack == nil || int(attack.HeaderByte2) > len(attack.Frames) {
		return errors.New("native physical departure header unavailable")
	}
	scene.attackStart = int(attack.HeaderByte2)
	scene.rawSide, scene.platformFrame = actor.NativeRecordByte6, tai
	root := separatedAssetPath("surfaces")
	targetBG := scene.selection.PrimaryBG
	if scene.rawSide == 0 {
		targetBG = scene.selection.SecondaryBG
	}
	var err error
	scene.targetBackground, err = fdother.LoadSeparatedSingleFrame(root, "BG.DAT", int(targetBG))
	if err != nil {
		return err
	}
	targetBase, err := g.nativePhysicalTargetBase(scene, scene.targetRecord)
	if err != nil {
		return err
	}
	var layers [3]fdother.Frame
	for i := range layers {
		layers[i], err = fdother.LoadSeparatedSingleFrame(root, "BG.DAT", i)
		if err != nil {
			return err
		}
	}
	source := make([]byte, 320*200) // 2952A 明確清除 work；header2=0 時保持零。
	for i := 0; i < scene.attackStart; i++ {
		frame := attack.Frames[i]
		pixels := append([]byte(nil), base...)
		if err := frame.BlitAt(pixels, 320); err != nil {
			return err
		}
		scene.departure = append(scene.departure, pixels)
		scene.leadDelays = append(scene.leadDelays, frame.Delay)
		source = pixels
	}
	in := battlepresent.NativeCommand24BackgroundInputs{Layers: layers, Source: source, Target: targetBase, TargetIdle: targetIdle.Frames[0]}
	if scene.rawSide == 0 {
		scene.transition, err = battlepresent.BuildNativePhysicalRightBackgroundFrames(in)
	} else {
		scene.transition, err = battlepresent.BuildNativeCommand24BackgroundFrames(in)
	}
	if err != nil {
		return err
	}
	for range scene.transition {
		scene.leadDelays = append(scene.leadDelays, 0)
	}
	frames := append(append([][]byte(nil), scene.departure...), scene.transition...)
	scene.leadImages, err = nativeCommand24IndexedImages(frames, g.nativeUIPalette)
	return err
}

func (g *Game) drawNativePhysicalPrelude(screen *ebiten.Image) bool {
	if g.atk == nil || g.atk.nativeScene == nil {
		return false
	}
	scene := g.atk.nativeScene
	var img *ebiten.Image
	if scene.preludeFrame < len(scene.preludeImages) {
		img = scene.preludeImages[scene.preludeFrame]
		scene.preludeDrawn = true
	} else if scene.leadFrame < len(scene.leadImages) {
		if scene.body != nil {
			if err := g.drawNativePhysicalBody(screen, scene); err != nil {
				g.loadErr = "native physical body: " + err.Error()
			}
			return true
		}
		img = scene.leadImages[scene.leadFrame]
		scene.leadDrawn = true
	} else {
		if scene.body != nil {
			if err := g.drawNativePhysicalBody(screen, scene); err != nil {
				g.loadErr = "native physical body: " + err.Error()
			}
			return true
		}
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(img, op)
	return true
}

func (g *Game) drawNativePhysicalBase(screen *ebiten.Image, defenderHP int) (bool, error) {
	a := g.atk
	if a == nil || a.nativeScene == nil || len(a.nativeScene.prelude) == 0 {
		return false, nil
	}
	scene := a.nativeScene
	actorHP, targetHP := a.atkHP, defenderHP
	if scene.counterActive {
		actorHP, targetHP = defenderHP, a.atkHP
	}
	if scene.baseImage != nil && scene.baseActorHP == actorHP && scene.baseTargetHP == targetHP {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(scene.baseImage, op)
		return true, nil
	}
	actorRecord := append([]byte(nil), scene.actorRecord...)
	targetRecord := append([]byte(nil), scene.targetRecord...)
	currentActor, currentTarget := actorRecord, targetRecord
	if scene.counterActive {
		currentActor, currentTarget = targetRecord, actorRecord
	}
	binary.LittleEndian.PutUint16(currentActor[0x40:], uint16(int16(a.atkHP)))
	binary.LittleEndian.PutUint16(currentTarget[0x40:], uint16(int16(defenderHP)))
	var base []byte
	var err error
	if scene.selection.HasSeparateBackgrounds {
		base, err = g.nativePhysicalTargetBase(scene, targetRecord)
	} else {
		base, err = g.nativePhysicalBase(scene, actorRecord, targetRecord)
	}
	if err != nil {
		return false, err
	}
	images, err := nativeCommand24IndexedImages([][]byte{base}, g.nativeUIPalette)
	if err != nil {
		return false, err
	}
	if scene.baseImage != nil {
		scene.baseImage.Dispose()
	}
	scene.baseImage = images[0]
	scene.baseActorHP, scene.baseTargetHP = actorHP, targetHP
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(images[0], op)
	return true, nil
}
