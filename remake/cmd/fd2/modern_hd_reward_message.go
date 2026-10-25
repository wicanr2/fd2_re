package main

import (
	"bytes"
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) prepareModernHDRewardWorld(message *nativeDeathRewardMessageState, source []byte, set *modernStoryPortraitSet) (*modernHDMapFrame, error) {
	if set == nil || len(set.mapSpritesHD) != 125 {
		return nil, nil // Existing portrait-only themes retain the native world.
	}
	if message == nil || len(source) != 64000 || message.hdScene == nil {
		return nil, errors.New("modern HD reward: frozen sources unavailable")
	}
	scene := message.hdScene
	if err := scene.validate(g); err != nil {
		return nil, err
	}
	if !bytes.Equal(source, scene.vga) {
		return nil, errors.New("modern HD reward: native panel source changed")
	}
	return g.prepareModernHDCommandSteadyStage(scene.input, scene.work, source, scene.palette, set, scene.stage)
}

// Returns the actual source-to-destination shift of the existing native job.
// A restore frame deliberately has no panel. Unrelated jobs are never adapted.
func modernHDRewardRegion(message *nativeDeathRewardMessageState, job *nativeClassUIJob) (int, image.Rectangle, bool) {
	shift := 0
	if job != nil {
		switch {
		case job == message.opening && len(job.frames) == 6 && job.frame >= 0 && job.frame < 6:
			shift = (5 - job.frame) * 13
		case job == message.closing && len(job.frames) == 5 && job.frame >= 0 && job.frame < 5:
			shift = (job.frame + 1) * 13
		case job == message.closing && job.frame == 5 && len(job.restore) == 64000:
			return 0, image.Rectangle{}, true
		default:
			return 0, image.Rectangle{}, false
		}
	} else if !message.awaitAck {
		return 0, image.Rectangle{}, false
	}
	return shift, image.Rect(5, 112+shift, 315, 198+shift).Intersect(image.Rect(0, 0, 320, 200)), true
}

func (g *Game) drawModernHDRewardMessage(screen *ebiten.Image) bool {
	state := g.nativeSystemEndTurnUI
	if state == nil || state.rewardMessage == nil || g.modernHDText == nil || g.font == nil || g.nativePreparationUI == nil || g.nativeClassUI == nil {
		return false
	}
	message, job := state.rewardMessage, g.nativeClassUIJob
	shift, region, ok := modernHDRewardRegion(message, job)
	if !ok {
		return false
	}
	var frame []byte
	var capture *modernHDIndexedText
	if !region.Empty() {
		capture = &modernHDIndexedText{palette: g.nativeClassUI.palette}
		font := g.modernHDFacilityFont(g.nativePreparationUI.status.Font, capture)
		dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(state.source, g.nativePreparationUI.dialogue, message.portrait)
		if err != nil {
			return false
		}
		frame, err = campaign.ComposeNativeDeathRewardMessageFrame(dialogue, g.nativePreparationUI.status.Strings, font, message.kind, message.value)
		if err != nil {
			return false
		}
		if job != nil {
			var frames [][]byte
			if job == message.opening {
				frames, err = campaign.NativeClassListOpeningFrames(state.source, frame)
			} else {
				frames, err = campaign.NativeClassListClosingFrames(state.source, frame)
			}
			if err != nil || job.frame >= len(frames) {
				return false
			}
			frame = frames[job.frame]
		}
	} else {
		frame = job.restore
	}
	// Only the native panel writes cover the prepared HD world. Legacy themes
	// without a complete HD world still present the original full background.
	if g.modernHDWorldVisible && message.hdWorld != nil && message.hdTheme == g.modernStoryPortraits {
		mask := make([]bool, 64000)
		for y := region.Min.Y; y < region.Max.Y; y++ {
			for x := region.Min.X; x < region.Max.X; x++ {
				mask[y*320+x] = true
			}
		}
		pixels, err := modernHDTimedDialoguePixels(frame, mask, g.nativeClassUI.palette)
		if err != nil {
			return false
		}
		g.clearModernHDText()
		texture := ebiten.NewImageFromImage(pixels)
		defer texture.Dispose()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(texture, op)
	} else {
		g.presentNativeClassFrame(screen, frame)
	}
	if capture != nil {
		clip := image.Rect(region.Min.X*2, region.Min.Y*2, region.Max.X*2, region.Max.Y*2)
		for _, run := range capture.runs {
			run.y += float64(shift * 2)
			run.clip = run.clip.Add(image.Pt(0, shift*2)).Intersect(clip)
			if !run.clip.Empty() {
				g.modernHDText.runs = append(g.modernHDText.runs, run)
			}
		}
		if set := g.modernStoryPortraits; set != nil && message.killer != nil {
			portrait := set.portraits[message.killer.BattleFig]
			if portrait != nil && portrait.hdImage != nil {
				g.modernHDOverlay = &modernHDOverlay{portrait: portrait, mirrorX: true,
					rect: image.Rect(16, (115+shift)*2, 176, (195+shift)*2), clip: clip}
			}
		}
	}
	if job != nil {
		job.drawn = true
	}
	return true
}
