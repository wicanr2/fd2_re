package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) modernHDTimedDialogueEligible() bool {
	if g.nativeLevelUpDialogue == nil || len(g.dialog) != 1 || g.storyBG || g.walk != nil || g.bannerT > 0 {
		return false
	}
	n := g.dialog[0].NativeDialogue
	return n != nil && n.Control == "FFEC" && n.HasMotionTargetY && n.MotionTargetY == 0 &&
		g.nativeDialogueLayout != nil && g.nativeClassUI != nil && len(g.nativeDialogueClosing) == 5
}

func (g *Game) freezeModernHDTimedDialogue(frame *modernHDMapFrame) (*modernHDMapFrame, error) {
	if !g.modernHDTimedDialogueEligible() {
		return frame, nil
	}
	s := g.nativeLevelUpDialogue
	if s.hdWorld == nil {
		coverage := make(map[campaign.NativeDialogueFramePhase][][]bool)
		for _, spec := range []struct {
			phase campaign.NativeDialogueFramePhase
			count int
		}{{campaign.NativeDialogueOpening, 5}, {campaign.NativeDialogueSteady, 1}, {campaign.NativeDialogueClosing, 5}} {
			for index := 0; index < spec.count; index++ {
				mask, err := campaign.NativeStoryDialogueCoverage(g.nativeClassUI.dialogue, g.nativeDialogueLayout.Control, spec.phase, index)
				if err != nil {
					return nil, err
				}
				coverage[spec.phase] = append(coverage[spec.phase], mask)
			}
		}
		// Frame owns its pixel buffers and draw plans; subsequent transactions
		// allocate another frame. The original clock still advances as before.
		s.hdWorld, s.hdCoverage = frame, coverage
	}
	return s.hdWorld, nil
}

func modernHDTimedDialoguePixels(frame []byte, mask []bool, palette color.Palette) (*image.RGBA, error) {
	if len(frame) != 320*200 || len(mask) != len(frame) || len(palette) != 256 {
		return nil, errors.New("modern HD dialogue: incomplete pixels")
	}
	out := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for i, covered := range mask {
		if !covered {
			continue
		}
		if palette[frame[i]] == nil {
			return nil, errors.New("modern HD dialogue: missing palette color")
		}
		r, g, b, _ := palette[frame[i]].RGBA()
		out.SetRGBA(i%320, i/320, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255})
	}
	return out, nil
}

func (g *Game) presentNativeStoryDialogueFrame(screen *ebiten.Image, frame []byte, phase campaign.NativeDialogueFramePhase, index int, portrait image.Image, rect image.Rectangle) error {
	if portrait != nil && (portrait.Bounds().Dx() != rect.Dx() || portrait.Bounds().Dy() != rect.Dy()) {
		return errors.New("native RGBA overlay geometry is invalid")
	}
	if g.modernHDWorldVisible && g.modernHDTimedDialogueEligible() && g.nativeLevelUpDialogue.hdWorld != nil {
		masks := g.nativeLevelUpDialogue.hdCoverage[phase]
		if index < 0 || index >= len(masks) {
			return errors.New("modern HD dialogue: coverage unavailable")
		}
		pixels, err := modernHDTimedDialoguePixels(frame, masks[index], g.nativeClassUI.palette)
		if err != nil {
			return err
		}
		g.clearModernHDText()
		if g.modernHDMapGPU.Dialogue == nil {
			g.modernHDMapGPU.Dialogue = ebiten.NewImage(320, 200)
		}
		g.modernHDMapGPU.Dialogue.WritePixels(pixels.Pix)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(g.modernHDMapGPU.Dialogue, op)
		g.modernHDTimedDialogueVisible = true
		g.modernHDTimedDialoguePhase, g.modernHDTimedDialogueIndex = phase, index
	} else if mask := g.modernHDStoryDialogueMask(phase, index); g.modernHDWorldVisible && g.modernHDMapGPU != nil && mask != nil {
		// 一般故事與戰場事件對白：HD 世界已畫好，只疊原生對白實際寫入的像素，
		// 不再用完整 indexed 幀把背景與地圖人物蓋回原版。
		pixels, err := modernHDTimedDialoguePixels(frame, mask, g.nativeClassUI.palette)
		if err != nil {
			return err
		}
		g.clearModernHDText()
		if g.modernHDMapGPU.Dialogue == nil {
			g.modernHDMapGPU.Dialogue = ebiten.NewImage(320, 200)
		}
		g.modernHDMapGPU.Dialogue.WritePixels(pixels.Pix)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(g.modernHDMapGPU.Dialogue, op)
	} else {
		g.presentNativeClassFrame(screen, frame)
	}
	if portrait != nil {
		texture := ebiten.NewImageFromImage(portrait)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(rect.Min.X*2), float64(rect.Min.Y*2))
		screen.DrawImage(texture, op)
	}
	if g.modernHDText != nil && g.nativeDialogueLayout != nil {
		bounds := g.nativeDialogueChromeBounds[phase]
		if index < 0 || index >= len(bounds) || g.nativeDialogueChrome == nil {
			return errors.New("modern HD dialogue: story border preflight unavailable")
		}
		if !bounds[index].Empty() {
			g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{
				chrome: g.nativeDialogueChrome, rect: bounds[index], gold: true,
			})
		}
	}
	return nil
}

// modernHDStoryDialogueMasks 是一句原生對白在 HD 世界上要保留的原生像素：開框、各頁穩定
// 狀態與收框（含滑向游標的尾段）各一張。遮罩以 0／255 兩種字面背景重放同一個合成器，
// 兩次相同的位置就是合成器實際寫入的位置（含零色），與設施場景的 marker 重放同一做法。
type modernHDStoryDialogueMasks struct {
	opening, closing, pages [][]bool
}

func modernHDMarkerMasks(compose func(base []byte) ([][]byte, error)) ([][]bool, error) {
	low, err := compose(make([]byte, 320*200))
	if err != nil {
		return nil, err
	}
	high, err := compose(bytes.Repeat([]byte{0xff}, 320*200))
	if err != nil {
		return nil, err
	}
	if len(low) == 0 || len(low) != len(high) {
		return nil, errors.New("modern HD dialogue: marker replay changed frame count")
	}
	masks := make([][]bool, len(low))
	for frame := range low {
		if len(low[frame]) != 320*200 || len(high[frame]) != 320*200 {
			return nil, errors.New("modern HD dialogue: marker replay frame is incomplete")
		}
		mask := make([]bool, 320*200)
		for i := range mask {
			mask[i] = low[frame][i] == high[frame][i]
		}
		masks[frame] = mask
	}
	return masks, nil
}

func freezeModernHDStoryDialogueMasks(pages int,
	opening, closing func(base []byte) ([][]byte, error),
	page func(base []byte, page int) ([][]byte, error)) *modernHDStoryDialogueMasks {
	m := &modernHDStoryDialogueMasks{}
	var err error
	if m.opening, err = modernHDMarkerMasks(opening); err != nil {
		return nil
	}
	if m.closing, err = modernHDMarkerMasks(closing); err != nil {
		return nil
	}
	for index := 0; index < pages; index++ {
		masks, err := modernHDMarkerMasks(func(base []byte) ([][]byte, error) { return page(base, index) })
		if err != nil {
			return nil
		}
		// 一頁所有逐字幀（含三行窗口捲動）寫入集合的聯集；嘴型與等鍵箭頭都畫在
		// 已寫入的頭像格與框格內。
		union := make([]bool, 320*200)
		for _, mask := range masks {
			for i, written := range mask {
				union[i] = union[i] || written
			}
		}
		m.pages = append(m.pages, union)
	}
	return m
}

func (g *Game) modernHDStoryDialogueMask(phase campaign.NativeDialogueFramePhase, index int) []bool {
	m := g.nativeDialogueHDMasks
	// 故事地圖（storyBG）與戰場都由 drawModernHDMapFrame 先畫 HD 世界；呼叫端另以
	// modernHDWorldVisible 確認本次 Draw 確實有 HD 世界在底下。
	if m == nil || g.nativeClassUI == nil {
		return nil
	}
	var masks [][]bool
	switch phase {
	case campaign.NativeDialogueOpening:
		masks = m.opening
	case campaign.NativeDialogueClosing:
		masks = m.closing
	case campaign.NativeDialogueSteady:
		if g.dlgScrollT != 0 {
			return nil // 非繁中分頁捲動同時畫兩頁，沿用完整原生幀
		}
		masks, index = m.pages, g.dlgPage
	}
	if index < 0 || index >= len(masks) {
		return nil
	}
	return masks[index]
}
