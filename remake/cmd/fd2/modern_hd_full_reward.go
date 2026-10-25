package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type fullRewardPhase uint8

const (
	fullRewardOpening fullRewardPhase = iota
	fullRewardChoice
	fullRewardChoiceClosing
	fullRewardBlankClosing
	fullRewardResponse
	fullRewardCancelOpening
	fullRewardCancelClosing
)

type modernHDGlyphWrite struct {
	start      int
	clear      image.Rectangle
	background bool
}

type modernHDFullReward struct {
	scene     *modernHDCompoundMapScene
	world     *modernHDMapFrame
	theme     *modernStoryPortraitSet
	ui        *nativePreparationUIAssets
	question  []byte
	canceled  [][]byte
	seal      [32]byte
	job       *nativeClassUIJob
	kind      fullRewardPhase
	chrome    *modernHDDialogueChrome
	chromeErr error
}

func modernHDFullRewardSeal(p *nativeDeathRewardUIState) [32]byte {
	if p == nil || p.hd == nil || p.killer == nil {
		return [32]byte{}
	}
	b, _ := json.Marshal([]any{p.item, p.killer.BattleFig, p.portrait, p.source, p.dialogue, p.hd.question, p.hd.canceled})
	return sha256.Sum256(b)
}

func (g *Game) bindModernHDFullReward(kind fullRewardPhase) {
	if p := g.nativeDeathRewardUI; p != nil && p.hd != nil {
		p.hd.job, p.hd.kind = g.nativeClassUIJob, kind
	}
}

// Replays the existing native producer. It owns no clock, input or transaction.
func (p *nativeDeathRewardUIState) modernHDFrame(source []byte, font *fdtxt.Font, kind fullRewardPhase, index, choice, pulse int, capture *modernHDIndexedText) ([]byte, int, error) {
	ui := p.hd.ui
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, ui.dialogue, p.portrait)
	if err != nil {
		return nil, 0, err
	}
	if kind == fullRewardBlankClosing {
		if index == 5 {
			return source, 0, nil
		}
		frames, err := campaign.NativeClassListClosingFrames(source, dialogue)
		return fullRewardFrameAt(frames, index, (index+1)*13, err)
	}
	if kind == fullRewardCancelOpening || kind == fullRewardCancelClosing {
		final, err := campaign.ComposeNativeDeathRewardSelectorCancel(dialogue, p.portrait, ui.status.Strings, font, p.item)
		if err != nil {
			return nil, 0, err
		}
		if index < 0 {
			return final, 0, nil
		}
		if kind == fullRewardCancelOpening {
			frames, err := campaign.NativeClassListOpeningFrames(source, final)
			return fullRewardFrameAt(frames, index, (5-index)*13, err)
		}
		if index == 5 {
			return source, 0, nil
		}
		frames, err := campaign.NativeClassListClosingFrames(source, final)
		return fullRewardFrameAt(frames, index, (index+1)*13, err)
	}
	// The first six opening frames contain only the blank dialogue.
	if kind == fullRewardOpening && index < 6 {
		frames, err := campaign.NativeClassListOpeningFrames(source, dialogue)
		return fullRewardFrameAt(frames, index, (5-index)*13, err)
	}
	question, err := campaign.ComposeNativeDeathRewardDiscardQuestion(dialogue, p.portrait, ui.status.Strings, font)
	if err != nil {
		return nil, 0, err
	}
	switch kind {
	case fullRewardChoice:
		frame, err := campaign.ComposeNativeConfirmationChoices(question, ui.choices, choice, pulse)
		return frame, 0, err
	case fullRewardOpening:
		frames, err := campaign.NativeClassConfirmationOpeningFrames(question, ui.choices)
		return fullRewardFrameAt(frames, index-6, 0, err)
	case fullRewardChoiceClosing:
		frames, err := campaign.NativeClassConfirmationClosingFrames(question, ui.choices)
		return fullRewardFrameAt(frames, index, 0, err)
	case fullRewardResponse:
		questionWrites := 0
		if capture != nil && capture.writes != nil {
			questionWrites = len(*capture.writes)
		}
		frames, err := campaign.NativeDeathRewardAbandonResponseFrames(question, ui.status.Strings, font, p.item)
		if err != nil {
			return nil, 0, err
		}
		if index < 0 {
			index = len(frames) - 1
		}
		if capture != nil && capture.writes != nil {
			writes := *capture.writes
			limit := questionWrites + index + 1
			if limit < len(writes) {
				capture.runs = capture.runs[:writes[limit].start]
				*capture.writes = writes[:limit]
			}
		}
		return fullRewardFrameAt(frames, index, 0, nil)
	}
	return nil, 0, errors.New("modern HD full reward: unknown producer")
}

func fullRewardFrameAt(frames [][]byte, index, shift int, err error) ([]byte, int, error) {
	if err != nil {
		return nil, 0, err
	}
	if index < 0 || index >= len(frames) {
		return nil, 0, errors.New("modern HD full reward: frame outside producer")
	}
	return frames[index], shift, nil
}

func (g *Game) prepareModernHDFullRewardWorld(p *nativeDeathRewardUIState, set *modernStoryPortraitSet) (*modernHDMapFrame, error) {
	if set != nil && set.hasHD {
		if p == nil || p.hd == nil {
			return nil, errors.New("modern HD full reward: dialogue chrome unavailable")
		}
		if p.hd.chromeErr != nil {
			return nil, p.hd.chromeErr
		}
		if p.hd.chrome == nil {
			return nil, errors.New("modern HD full reward: dialogue chrome unavailable")
		}
	}
	if set == nil || len(set.mapSpritesHD) != 125 {
		return nil, nil
	}
	if p == nil || p.hd == nil || p.hd.scene == nil || p.hd.ui != g.nativePreparationUI || modernHDFullRewardSeal(p) != p.hd.seal {
		return nil, errors.New("modern HD full reward: changed prompt source")
	}
	h := p.hd
	if portrait := set.portraits[p.killer.BattleFig]; portrait == nil || portrait.hdImage == nil {
		return nil, errors.New("modern HD full reward: HD portrait unavailable")
	}
	if err := h.scene.validate(g); err != nil {
		return nil, err
	}
	if !bytes.Equal(p.source, h.scene.vga) {
		return nil, errors.New("modern HD full reward: changed world source")
	}
	if state := g.nativeSystemEndTurnUI; state != nil && state.deathReward == p {
		if !bytes.Equal(state.source, p.source) {
			return nil, errors.New("modern HD full reward: changed modal source")
		}
		if h.kind == fullRewardCancelOpening || h.kind == fullRewardCancelClosing {
			want, _, err := p.modernHDFrame(p.source, h.ui.status.Font, fullRewardCancelOpening, -1, 0, 0, nil)
			if err != nil || !bytes.Equal(state.dialogue, want) || !bytes.Equal(state.question, want) || len(state.canceled) != 1 || !bytes.Equal(state.canceled[0], want) {
				return nil, errors.New("modern HD full reward: changed cancel response")
			}
		} else {
			if !bytes.Equal(state.dialogue, p.dialogue) || !bytes.Equal(state.question, h.question) || len(state.canceled) != len(h.canceled) {
				return nil, errors.New("modern HD full reward: changed modal text")
			}
			for i := range state.canceled {
				if !bytes.Equal(state.canceled[i], h.canceled[i]) {
					return nil, errors.New("modern HD full reward: changed response frame")
				}
			}
		}
	}
	if job := g.nativeClassUIJob; job != nil && g.nativeDeathRewardUI == p {
		if job != h.job {
			return nil, errors.New("modern HD full reward: unbound job")
		}
		count := map[fullRewardPhase]int{fullRewardOpening: 10, fullRewardChoiceClosing: 4, fullRewardBlankClosing: 5,
			fullRewardResponse: len(h.canceled), fullRewardCancelOpening: 6, fullRewardCancelClosing: 5}[h.kind]
		restore := h.kind == fullRewardBlankClosing || h.kind == fullRewardCancelClosing
		if count == 0 || len(job.frames) != count || (len(job.restore) != 0) != restore {
			return nil, errors.New("modern HD full reward: changed native timeline")
		}
		for i, frame := range job.frames {
			want, _, err := p.modernHDFrame(p.source, h.ui.status.Font, h.kind, i, 0, 0, nil)
			if err != nil || !bytes.Equal(frame, want) {
				return nil, errors.New("modern HD full reward: changed native job")
			}
		}
		if len(job.restore) != 0 && !bytes.Equal(job.restore, p.source) {
			return nil, errors.New("modern HD full reward: changed restore source")
		}
	}
	return g.prepareModernHDCommandSteadyStage(h.scene.input, h.scene.work, p.source, h.scene.palette, set, h.scene.stage)
}

// Later native glyph clears erase earlier vector ink too. A shadow may extend
// outside the 16x16 clear; the wider text-safe clip is not an erase rectangle.
func modernHDWrittenRuns(c *modernHDIndexedText) []modernHDTextRun {
	if c.writes == nil {
		return c.runs
	}
	writes := *c.writes
	var result []modernHDTextRun
	for i, write := range writes {
		end := len(c.runs)
		if i+1 < len(writes) {
			end = writes[i+1].start
		}
		for _, run := range c.runs[write.start:end] {
			clips := []image.Rectangle{run.clip}
			for _, later := range writes[i+1:] {
				if !later.background {
					continue
				}
				var next []image.Rectangle
				for _, clip := range clips {
					next = append(next, modernHDSubtractRect(clip, later.clear)...)
				}
				clips = next
			}
			for _, clip := range clips {
				part := run
				part.clip = clip
				result = append(result, part)
			}
		}
	}
	return result
}

func modernHDSubtractRect(r, cut image.Rectangle) []image.Rectangle {
	i := r.Intersect(cut)
	if i.Empty() {
		return []image.Rectangle{r}
	}
	var result []image.Rectangle
	for _, part := range []image.Rectangle{image.Rect(r.Min.X, r.Min.Y, r.Max.X, i.Min.Y), image.Rect(r.Min.X, i.Max.Y, r.Max.X, r.Max.Y), image.Rect(r.Min.X, i.Min.Y, i.Min.X, i.Max.Y), image.Rect(i.Max.X, i.Min.Y, r.Max.X, i.Max.Y)} {
		if !part.Empty() {
			result = append(result, part)
		}
	}
	return result
}

func (g *Game) drawModernHDFullReward(screen *ebiten.Image) bool {
	state := g.nativeSystemEndTurnUI
	if state == nil || state.deathReward == nil || state.deathReward.hd == nil || g.modernHDText == nil {
		return false
	}
	p, job := state.deathReward, g.nativeClassUIJob
	kind, index := p.hd.kind, -1
	if job != nil {
		if job != p.hd.job {
			return false
		}
		index = job.frame
	} else if g.nativeSystemEndTurnConfirm {
		kind = fullRewardChoice
	} else if g.nativeSystemEndTurnDelay <= 0 {
		return false
	}
	writes := []modernHDGlyphWrite{}
	capture := &modernHDIndexedText{palette: g.nativeClassUI.palette, writes: &writes}
	font := g.modernHDFacilityFont(p.hd.ui.status.Font, capture)
	frame, shift, err := p.modernHDFrame(p.source, font, kind, index, state.choice, g.nativeClassUIPulse/2, capture)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	restore := job != nil && job.frame == len(job.frames) && len(job.restore) != 0
	region := image.Rect(5, 112+shift, 315, 198+shift).Intersect(image.Rect(0, 0, 320, 200))
	if restore {
		region = image.Rectangle{}
	}
	if g.modernHDWorldVisible && p.hd.world != nil && p.hd.theme == g.modernStoryPortraits {
		layer, err := modernHDOverwriteLayer(64000, g.nativeClassUI.palette, func(dst []byte) error {
			pixels, _, err := p.modernHDFrame(dst, p.hd.ui.status.Font, kind, index, state.choice, g.nativeClassUIPulse/2, nil)
			if err == nil {
				copy(dst, pixels)
			}
			return err
		})
		if err != nil {
			g.loadErr = err.Error()
			return false
		}
		mask := make([]bool, 64000)
		for i := range mask {
			mask[i] = layer[i].A != 0
		}
		pixels, err := modernHDTimedDialoguePixels(frame, mask, g.nativeClassUI.palette)
		if err != nil {
			g.loadErr = err.Error()
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
	if !region.Empty() {
		g.appendModernHDDialogueChrome(p.hd.chrome, shift)
		clip := image.Rect(region.Min.X*2, region.Min.Y*2, region.Max.X*2, region.Max.Y*2)
		for _, run := range modernHDWrittenRuns(capture) {
			run.y += float64(shift * 2)
			run.clip = run.clip.Add(image.Pt(0, shift*2)).Intersect(clip)
			if !run.clip.Empty() {
				g.modernHDText.runs = append(g.modernHDText.runs, run)
			}
		}
		if set := g.modernStoryPortraits; set != nil {
			portrait := set.portraits[p.killer.BattleFig]
			if portrait != nil && portrait.hdImage != nil {
				g.modernHDOverlay = &modernHDOverlay{portrait: portrait, mirrorX: true, rect: image.Rect(16, (115+shift)*2, 176, (195+shift)*2), clip: clip}
			}
		}
	}
	if job != nil {
		job.drawn = true
	}
	return true
}
