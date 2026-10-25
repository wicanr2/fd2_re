package main

import (
	"bytes"
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

var nativeTransientExpiryLocaleKeys = [6]string{
	"battle.status.expiry.attack", "battle.status.expiry.defence", "battle.status.expiry.speed",
	"battle.status.expiry.poison", "battle.status.expiry.paralysis", "battle.status.expiry.silence",
}

// Original Chinese retains the source glyph path. Translations use the same
// text origin and a bounded three-line field to the right of the portrait.
func (g *Game) composeNativeTransientExpiryFrame(source []byte, portrait dato.Frame, counter int, font *fdtxt.Font) ([]byte, error) {
	if g.localeID == "" || g.localeID == "zh-Hant" {
		return campaign.ComposeNativeTransientExpiryFrame(source, g.nativeClassUI.dialogue, portrait, g.nativeClassUI.strings, font, counter)
	}
	if counter < 0 || counter >= len(nativeTransientExpiryLocaleKeys) || g.localeCatalog == nil || g.font == nil {
		return nil, errors.New("native expiry: locale assets unavailable")
	}
	message, err := g.localeCatalog.Format(nativeTransientExpiryLocaleKeys[counter])
	if err != nil {
		return nil, err
	}
	frame, err := campaign.ComposeNativeTransientExpiryDialogue(source, g.nativeClassUI.dialogue, portrait)
	if err != nil {
		return nil, err
	}
	lines := g.font.Wrap(message, 0.78, 205)
	if len(lines) == 0 || len(lines) > 3 {
		return nil, errors.New("native expiry: translated text exceeds three lines")
	}
	for i, line := range lines {
		if err := drawIndexedLocalizedText(frame, g.font, line, 99, 127+i*19, 205, 19, 0.78, 0xcd, 0x4c); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

// This state belongs to one expiry job, including its final restore frame.
// The scene is captured after that record's native focus, never during Draw.
type nativeTransientExpiryPresentation struct {
	selector, counter int
	portrait          dato.Frame
	source            []byte
	scene             *modernHDCompoundMapScene
	world             *modernHDMapFrame
	theme             *modernStoryPortraitSet
}

func (g *Game) newNativeTransientExpiryPresentation(unit *battle.Unit, counter int, portrait dato.Frame) (*nativeTransientExpiryPresentation, error) {
	p := &nativeTransientExpiryPresentation{selector: unit.BattleFig, counter: counter,
		portrait: portrait, source: append([]byte(nil), g.nativeMapVGA...)}
	in, inputErr := g.freezeNativeCommand9MapInput()
	palette, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if inputErr == nil {
		inputErr = paletteErr
	}
	p.scene = g.newModernHDCompoundMapScene(in, g.nativeMapWork, p.source, palette, inputErr)
	world, err := g.prepareModernHDTransientExpiryWorld(p, g.modernStoryPortraits)
	if err != nil {
		return nil, err
	}
	p.world, p.theme = world, g.modernStoryPortraits
	return p, nil
}

func (g *Game) prepareModernHDTransientExpiryWorld(p *nativeTransientExpiryPresentation, set *modernStoryPortraitSet) (*modernHDMapFrame, error) {
	if set == nil || len(set.mapSpritesHD) == 0 {
		return nil, nil // Original and legacy portrait-only themes retain their world.
	}
	if len(set.mapSpritesHD) != 125 || p == nil || p.scene == nil || !bytes.Equal(p.source, p.scene.vga) {
		return nil, errors.New("modern HD expiry: frozen source unavailable")
	}
	if err := p.scene.validate(g); err != nil {
		return nil, err
	}
	s := p.scene
	return g.prepareModernHDCommandSteadyStage(s.input, s.work, s.vga, s.palette, set, s.stage)
}

// Check the new HD sources on private states before publishing the countdown.
// Activation seals them again after the actual damage/death/focus sequence.
func (g *Game) preflightModernHDTransientExpiry(candidate *battle.State, expired []battle.NativeTransientExpiry, damage []battle.NativeTransientDamage) error {
	if len(expired) == 0 || g.modernStoryPortraits == nil || len(g.modernStoryPortraits.mapSpritesHD) == 0 {
		return nil
	}
	preview, err := g.nativeTransientDeathPreview(candidate, damage)
	if err != nil {
		return err
	}
	for _, event := range expired {
		slot := -1
		for i, unit := range candidate.Units {
			if unit == event.Unit {
				slot = i
				break
			}
		}
		preview, err = preview.nativeTransientFocusPreview(preview.st, []int{slot})
		if err != nil {
			return err
		}
		unit, counter, err := nativeTransientPresentationEvent(event)
		if err != nil {
			return err
		}
		portraits, err := loadNativeSeparatedPortrait(unit.BattleFig)
		if err != nil || len(portraits) == 0 {
			return errors.New("modern HD expiry: portrait unavailable")
		}
		if _, err := preview.newNativeTransientExpiryPresentation(preview.st.Units[slot], counter, portraits[0]); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) stageModernHDTransientExpiry(set *modernStoryPortraitSet) (func(), error) {
	job := g.nativeClassUIJob
	if job == nil || job.expiryText == nil {
		return func() {}, nil
	}
	p := job.expiryText
	world, err := g.prepareModernHDTransientExpiryWorld(p, set)
	if err != nil {
		return nil, err
	}
	return func() { p.world, p.theme = world, set }, nil
}

func modernHDTransientExpiryRegion(job *nativeClassUIJob) (int, image.Rectangle, bool) {
	if job == nil || job.expiryText == nil || len(job.frames) != 11 || len(job.restore) != 64000 || job.frame < 0 || job.frame > 11 {
		return 0, image.Rectangle{}, false
	}
	if job.frame == 11 {
		return 0, image.Rectangle{}, true
	}
	shift := (5 - job.frame) * 13
	if job.frame >= 6 {
		shift = (job.frame - 5) * 13
	}
	return shift, image.Rect(5, 112+shift, 315, 198+shift).Intersect(image.Rect(0, 0, 320, 200)), true
}

func (g *Game) modernHDTransientExpiryPortraitAllowed() bool {
	job := g.nativeClassUIJob
	return job != nil && job.expiryText != nil && job.expiryText.theme == g.modernStoryPortraits && g.modernStoryPortraits != nil
}

func (g *Game) drawModernHDTransientExpiryMessage(screen *ebiten.Image, job *nativeClassUIJob) bool {
	shift, region, ok := modernHDTransientExpiryRegion(job)
	if !ok || g.modernHDText == nil || g.font == nil || g.nativeClassUI == nil {
		return false
	}
	p := job.expiryText
	frame := job.restore
	var capture *modernHDIndexedText
	if !region.Empty() {
		capture = &modernHDIndexedText{palette: g.nativeClassUI.palette}
		font := g.modernHDFacilityFont(g.nativeClassUI.font, capture)
		previous := g.font.indexedHD
		g.font.indexedHD = capture
		final, err := g.composeNativeTransientExpiryFrame(p.source, p.portrait, p.counter, font)
		g.font.indexedHD = previous
		if err != nil {
			return false
		}
		var frames [][]byte
		index := job.frame
		if job.frame < 6 {
			frames, err = campaign.NativeClassListOpeningFrames(p.source, final)
		} else {
			frames, err = campaign.NativeClassListClosingFrames(p.source, final)
			index -= 6
		}
		if err != nil {
			return false
		}
		frame = frames[index]
	}
	if g.modernHDWorldVisible && p.world != nil && p.theme == g.modernStoryPortraits {
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
		if set := g.modernStoryPortraits; set != nil {
			portrait := set.portraits[p.selector]
			if portrait != nil && portrait.hdImage != nil {
				g.modernHDOverlay = &modernHDOverlay{portrait: portrait, mirrorX: true,
					rect: image.Rect(16, (115+shift)*2, 176, (195+shift)*2), clip: clip}
			}
		}
	}
	job.drawn = true
	return true
}
