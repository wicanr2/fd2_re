package main

import (
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
	return nil
}
