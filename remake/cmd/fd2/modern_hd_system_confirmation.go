package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type modernHDSystemConfirmation struct {
	ui      *nativePreparationUIAssets
	scene   *modernHDCompoundMapScene
	world   *modernHDMapFrame
	theme   *modernStoryPortraitSet
	seal    [32]byte
	job     *nativeClassUIJob
	kind    fullRewardPhase
	chrome  *modernHDDialogueChrome
	buttons *modernHDConfirmation
	palette color.Palette
}

type systemQuestionProducer func([]byte, dato.Frame, *fdtxt.Strings, *fdtxt.Font) ([]byte, error)
type systemResponseProducer func([]byte, *fdtxt.Strings, *fdtxt.Font, bool) ([][]byte, error)

func systemConfirmationProducer(s *nativeSystemEndTurnUIState) (systemQuestionProducer, systemResponseProducer) {
	switch {
	case s.exitProgram:
		return campaign.ComposeNativeBattleExitQuestion, campaign.NativeBattleExitResponseFrames
	case s.saveCurrent:
		return campaign.ComposeNativeBattleCurrentSaveQuestion, campaign.NativeBattleCurrentSaveResponseFrames
	case s.loadCurrent:
		return campaign.ComposeNativeBattleCurrentLoadQuestion, campaign.NativeBattleCurrentLoadResponseFrames
	case s.groupMarch:
		return campaign.ComposeNativeBattleGroupMarchQuestion, campaign.NativeBattleGroupMarchResponseFrames
	default:
		return campaign.ComposeNativeBattleEndTurnQuestion, campaign.NativeBattleEndTurnResponseFrames
	}
}

func systemConfirmationSeal(s *nativeSystemEndTurnUIState) [32]byte {
	h := s.hd
	var cells []any
	if h == nil || h.ui == nil || len(h.ui.choices) <= 52 {
		return [32]byte{}
	}
	for _, index := range modernHDConfirmationIndices {
		cells = append(cells, h.ui.choices[index])
	}
	b, _ := json.Marshal([]any{s.source, s.dialogue, s.question, s.accepted, s.canceled, s.portraits,
		s.exitProgram, s.saveCurrent, s.loadCurrent, s.groupMarch, h.ui.dialogue, h.ui.portrait, cells})
	return sha256.Sum256(b)
}

func (g *Game) freezeModernHDSystemConfirmation(s *nativeSystemEndTurnUIState) bool {
	in, inputErr := g.freezeNativeCommand9MapInput()
	palette, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if inputErr == nil {
		inputErr = paletteErr
	}
	s.hd = &modernHDSystemConfirmation{ui: g.nativePreparationUI, kind: fullRewardOpening,
		palette: append(color.Palette(nil), g.nativeClassUI.palette...),
		scene:   g.newModernHDCompoundMapScene(in, g.nativeMapWork, s.source, palette, inputErr)}
	s.hd.seal = systemConfirmationSeal(s)
	h, err := g.prepareModernHDSystemConfirmation(s, g.modernStoryPortraits)
	if err != nil && g.hdPresentationEnabled() {
		g.loadErr = err.Error()
		return false
	}
	if err == nil {
		s.hd = h
	}
	return true
}

// Replays only the existing display producer, never its input or transaction.
func (h *modernHDSystemConfirmation) frame(s *nativeSystemEndTurnUIState, source []byte, font *fdtxt.Font,
	kind fullRewardPhase, index, choice, pulse int, capture *modernHDIndexedText) ([]byte, int, error) {
	u := h.ui
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, u.dialogue, u.portrait)
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
	if kind == fullRewardOpening && index < 6 {
		frames, err := campaign.NativeClassListOpeningFrames(source, dialogue)
		return fullRewardFrameAt(frames, index, (5-index)*13, err)
	}
	questionFn, responseFn := systemConfirmationProducer(s)
	question, err := questionFn(dialogue, u.portrait, u.status.Strings, font)
	if err != nil {
		return nil, 0, err
	}
	switch kind {
	case fullRewardChoice:
		frame, err := campaign.ComposeNativeConfirmationChoices(question, u.choices, choice, pulse)
		return frame, 0, err
	case fullRewardOpening:
		frames, err := campaign.NativeClassConfirmationOpeningFrames(question, u.choices)
		return fullRewardFrameAt(frames, index-6, 0, err)
	case fullRewardChoiceClosing:
		frames, err := campaign.NativeClassConfirmationClosingFrames(question, u.choices)
		return fullRewardFrameAt(frames, index, 0, err)
	case fullRewardResponse:
		questionWrites := 0
		if capture != nil && capture.writes != nil {
			questionWrites = len(*capture.writes)
		}
		frames, err := responseFn(question, u.status.Strings, font, s.acceptedOutcome)
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
	return nil, 0, errors.New("modern HD system confirmation: unknown phase")
}

func (g *Game) prepareModernHDSystemConfirmation(s *nativeSystemEndTurnUIState, set *modernStoryPortraitSet) (*modernHDSystemConfirmation, error) {
	if s == nil || s.hd == nil {
		return nil, errors.New("modern HD system confirmation: source unavailable")
	}
	h := *s.hd
	h.world, h.theme = nil, set
	if !g.modernHDInterfaceEnabled() && set == nil {
		return &h, nil
	}
	if h.ui != g.nativePreparationUI || systemConfirmationSeal(s) != h.seal || g.nativeClassUI == nil {
		return nil, errors.New("modern HD system confirmation: changed source")
	}
	originalPalette, _ := json.Marshal(h.palette)
	currentPalette, _ := json.Marshal(g.nativeClassUI.palette)
	if !bytes.Equal(originalPalette, currentPalette) {
		return nil, errors.New("modern HD system confirmation: changed palette")
	}
	// Check native question and both responses independently of the HD writer.
	qf, rf := systemConfirmationProducer(s)
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(s.source, h.ui.dialogue, h.ui.portrait)
	if err != nil || !bytes.Equal(dialogue, s.dialogue) {
		return nil, errors.New("modern HD system confirmation: changed dialogue")
	}
	question, err := qf(dialogue, h.ui.portrait, h.ui.status.Strings, h.ui.status.Font)
	if err != nil || !bytes.Equal(question, s.question) {
		return nil, errors.New("modern HD system confirmation: changed question")
	}
	for _, accepted := range []bool{false, true} {
		frames, err := rf(question, h.ui.status.Strings, h.ui.status.Font, accepted)
		want := s.canceled
		if accepted {
			want = s.accepted
		}
		if err != nil || len(frames) != len(want) {
			return nil, errors.New("modern HD system confirmation: changed response")
		}
		for i := range frames {
			if !bytes.Equal(frames[i], want[i]) {
				return nil, errors.New("modern HD system confirmation: response differs")
			}
		}
	}
	if g.modernHDInterfaceEnabled() {
		h.chrome, err = freezeModernHDDialogueChrome(h.ui.dialogue, g.nativeClassUI.palette)
		if err != nil {
			return nil, err
		}
		h.buttons, err = freezeModernHDConfirmation(h.ui.choices, g.nativeClassUI.palette, g.font)
		if err != nil {
			return nil, err
		}
		candidate := *g
		candidate.modernHDText = &modernHDTextFrame{}
		for _, accepted := range []bool{false, true} {
			copy := *s
			copy.acceptedOutcome = accepted
			writes := []modernHDGlyphWrite{}
			capture := &modernHDIndexedText{palette: h.palette, writes: &writes}
			font := candidate.modernHDFacilityFont(h.ui.status.Font, capture)
			if _, _, err := h.frame(&copy, s.source, font, fullRewardResponse, -1, 0, 0, capture); err != nil || len(writes) == 0 || len(capture.runs) != 2*len(writes) {
				return nil, errors.New("modern HD system confirmation: question or response glyph unavailable")
			}
		}
	}
	if set != nil {
		portrait := set.portraits[75]
		if portrait == nil || portrait.hdImage == nil {
			return nil, errors.New("modern HD system confirmation: DATO75 unavailable")
		}
		if len(set.mapSpritesHD) != 125 || h.scene == nil {
			return nil, errors.New("modern HD system confirmation: world unavailable")
		}
		if err = h.scene.validate(g); err != nil {
			return nil, err
		}
		if !bytes.Equal(s.source, h.scene.vga) {
			return nil, errors.New("modern HD system confirmation: changed world")
		}
		h.world, err = g.prepareModernHDCommandSteadyStage(h.scene.input, h.scene.work, s.source, h.scene.palette, set, h.scene.stage)
		if err != nil {
			return nil, err
		}
	}
	if job := g.nativeClassUIJob; g.nativeSystemEndTurnUI == s && job != nil {
		counts := map[fullRewardPhase]int{fullRewardOpening: 10, fullRewardChoiceClosing: 4, fullRewardBlankClosing: 5, fullRewardResponse: len(s.canceled)}
		if h.kind == fullRewardResponse && s.acceptedOutcome {
			counts[h.kind] = len(s.accepted)
		}
		if h.job != job || len(job.frames) != counts[h.kind] {
			return nil, errors.New("modern HD system confirmation: changed timeline")
		}
		for i, frame := range job.frames {
			want, _, err := h.frame(s, s.source, h.ui.status.Font, h.kind, i, 0, 0, nil)
			if err != nil || !bytes.Equal(want, frame) {
				return nil, errors.New("modern HD system confirmation: changed native frame")
			}
		}
		if h.kind == fullRewardBlankClosing && !bytes.Equal(job.restore, s.source) {
			return nil, errors.New("modern HD system confirmation: changed restore")
		}
	}
	return &h, nil
}

func (g *Game) stageModernHDSystemConfirmation(set *modernStoryPortraitSet) (func(), error) {
	s := g.nativeSystemEndTurnUI
	if s == nil || s.hd == nil {
		return func() {}, nil
	}
	h, err := g.prepareModernHDSystemConfirmation(s, set)
	if err != nil {
		return nil, err
	}
	return func() { s.hd = h }, nil
}

func (g *Game) bindModernHDSystemConfirmation(kind fullRewardPhase) {
	if s := g.nativeSystemEndTurnUI; s != nil && s.hd != nil {
		s.hd.job, s.hd.kind = g.nativeClassUIJob, kind
	}
}

func (g *Game) drawModernHDSystemConfirmation(screen *ebiten.Image) bool {
	s := g.nativeSystemEndTurnUI
	if s == nil || s.hd == nil || !g.hdPresentationEnabled() {
		return false
	}
	h, job := s.hd, g.nativeClassUIJob
	kind, index := h.kind, -1
	if job != nil {
		if job != h.job {
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
	if g.modernHDText == nil {
		capture = nil
	}
	font := g.modernHDFacilityFont(h.ui.status.Font, capture)
	frame, shift, err := h.frame(s, s.source, font, kind, index, s.choice, g.nativeClassUIPulse/2, capture)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	if kind == fullRewardChoice {
		if question, ok := g.composeNativeConfirmationMouth(frame, s.portraits); ok {
			frame = question
		} else {
			g.loadErr = "modern HD system confirmation: mouth unavailable"
			return false
		}
	}
	region := image.Rect(5, 112+shift, 315, 198+shift).Intersect(image.Rect(0, 0, 320, 200))
	if job != nil && job.frame == len(job.frames) && len(job.restore) != 0 {
		region = image.Rectangle{}
	}
	if g.modernHDWorldVisible && h.world != nil && h.theme == g.modernStoryPortraits {
		layer, err := modernHDOverwriteLayer(64000, g.nativeClassUI.palette, func(dst []byte) error {
			pixels, _, err := h.frame(s, dst, h.ui.status.Font, kind, index, s.choice, g.nativeClassUIPulse/2, nil)
			if err == nil && kind == fullRewardChoice {
				pixels, _ = g.composeNativeConfirmationMouth(pixels, s.portraits)
			}
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
		if g.modernHDText != nil {
			g.appendModernHDDialogueChrome(h.chrome, shift)
		}
		clip := image.Rect(region.Min.X*2, region.Min.Y*2, region.Max.X*2, region.Max.Y*2)
		for _, run := range modernHDWrittenRuns(capture) {
			run = run.freezeLayout()
			run.y += float64(shift * 2)
			run.clip = run.clip.Add(image.Pt(0, shift*2)).Intersect(clip)
			if !run.clip.Empty() && g.modernHDText != nil {
				g.modernHDText.runs = append(g.modernHDText.runs, run)
			}
		}
		if set := g.modernStoryPortraits; set != nil {
			g.modernHDOverlay = &modernHDOverlay{portrait: set.portraits[75], mirrorX: true, rect: image.Rect(16, (115+shift)*2, 176, (195+shift)*2), clip: clip}
		}
		if err := g.appendModernHDConfirmation(h.buttons, kind, index, s.choice, g.nativeClassUIPulse/2); err != nil {
			g.loadErr = err.Error()
			return false
		}
	}
	if job != nil {
		job.drawn = true
	}
	return true
}
