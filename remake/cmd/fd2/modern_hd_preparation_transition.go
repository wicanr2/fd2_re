package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type nativePreparationQuestionKind uint8

const (
	nativePreparationRecordQuestion nativePreparationQuestionKind = iota
	nativePreparationDepartureQuestion
	nativePreparationConfirmQuestion
	nativePreparationRequiredQuestion
)

// This recipe belongs to the job, independently of the next selection or
// confirmation. Indexed animation buffers remain the faithful source.
type nativePreparationTransitionText struct {
	source   []byte
	assets   nativePreparationUIAssets
	kind     nativePreparationQuestionKind
	identity int
	start    int
	closing  bool
	town     *nativeTownFrameSource
}

func (g *Game) snapshotNativePreparationTransition(source []byte, confirmation bool, start int, closing bool) *nativePreparationTransitionText {
	if g.nativePreparationUI == nil || len(source) != 320*200 || g.camp == nil || g.camp.Node() == nil {
		return nil
	}
	d := &nativePreparationTransitionText{source: append([]byte(nil), source...),
		assets: *g.nativePreparationUI, start: start, closing: closing}
	if g.prepTownSource.matches(source) {
		d.town = g.prepTownSource
	}
	switch {
	case confirmation:
		d.kind = nativePreparationConfirmQuestion
	case g.prepRequiredMissing != nil:
		d.kind, d.identity = nativePreparationRequiredQuestion, *g.prepRequiredMissing
	case g.camp.Node().Cancel != "":
		d.kind = nativePreparationDepartureQuestion
	default:
		d.kind = nativePreparationRecordQuestion
	}
	return d
}

func (d *nativePreparationTransitionText) compose(frame int, font *fdtxt.Font) ([]byte, bool) {
	if d == nil || frame < d.start {
		return nil, false
	}
	index := frame - d.start
	if (d.kind == nativePreparationRequiredQuestion && index != 0) || index >= 4 {
		return nil, false
	}
	a := &d.assets
	var question []byte
	var err error
	switch d.kind {
	case nativePreparationRecordQuestion:
		question, err = campaign.ComposeNativePreparationRecordQuestion(d.source, a.dialogue, a.portrait, a.status.Strings, font)
	case nativePreparationDepartureQuestion:
		question, err = campaign.ComposeNativePreparationDepartureQuestion(d.source, a.dialogue, a.portrait, a.status.Strings, font)
	case nativePreparationConfirmQuestion:
		question, err = campaign.ComposeNativePreparationConfirmationQuestion(d.source, a.dialogue, a.portrait, a.status.Strings, font)
	case nativePreparationRequiredQuestion:
		question, err = campaign.ComposeNativePreparationRequiredMessage(d.source, a.dialogue, a.portrait, a.status.Strings, font, d.identity)
	default:
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	if d.kind == nativePreparationRequiredQuestion {
		return question, true
	}
	var frames [][]byte
	if d.closing {
		frames, err = campaign.NativeClassConfirmationClosingFrames(question, a.choices)
	} else {
		frames, err = campaign.NativeClassConfirmationOpeningFrames(question, a.choices)
	}
	if err != nil || index >= len(frames) {
		return nil, false
	}
	return frames[index], true
}

func (g *Game) drawModernHDPreparationTransition(screen *ebiten.Image, job *nativeClassUIJob) bool {
	if g.nativeClassUI == nil || job.preparationText == nil {
		return false
	}
	d := job.preparationText
	capture := &modernHDIndexedText{palette: g.nativeClassUI.palette}
	font := g.modernHDFacilityFont(d.assets.status.Font, capture)
	compose := func(source []byte) ([]byte, bool) { return d.composeFrame(job.frame, source, font) }
	if g.drawModernHDPreparationBackground(screen, d.town, d.source, compose, capture) {
		return true
	}
	if g.modernHDText == nil || g.font == nil {
		return false
	}
	frame, ok := compose(d.source)
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	g.modernHDText.runs = append(g.modernHDText.runs, capture.runs...)
	return true
}

// Replays the whole presentation job without changing its prebuilt cache or
// acknowledgement schedule. Question stages retain their existing recipe.
func (d *nativePreparationTransitionText) composeFrame(frame int, source []byte, font *fdtxt.Font) ([]byte, bool) {
	if d == nil || len(source) != 64000 || frame < 0 {
		return nil, false
	}
	copy := *d
	copy.source = source
	if !(d.kind == nativePreparationRequiredQuestion && d.closing) {
		if question, ok := copy.compose(frame, font); ok {
			return question, true
		}
	}
	if d.kind > nativePreparationRequiredQuestion {
		return nil, false
	}
	if !d.closing && frame >= d.start {
		return nil, false
	}
	dialogue, err := campaign.ComposeNativePreparationConfirmationDialogue(source, d.assets.dialogue, d.assets.portrait)
	if err != nil {
		return nil, false
	}
	var frames [][]byte
	index := frame
	if d.closing {
		if d.kind != nativePreparationRequiredQuestion {
			index -= 4
		}
		if index == 5 {
			return append([]byte(nil), source...), true
		}
		frames, err = campaign.NativeClassListClosingFrames(source, dialogue)
	} else {
		frames, err = campaign.NativeClassListOpeningFrames(source, dialogue)
	}
	if err != nil || index < 0 || index >= len(frames) {
		return nil, false
	}
	return frames[index], true
}
