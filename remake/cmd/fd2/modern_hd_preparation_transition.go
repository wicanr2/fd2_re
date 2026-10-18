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
}

func (g *Game) snapshotNativePreparationTransition(source []byte, confirmation bool, start int, closing bool) *nativePreparationTransitionText {
	if g.nativePreparationUI == nil || len(source) != 320*200 || g.camp == nil || g.camp.Node() == nil {
		return nil
	}
	d := &nativePreparationTransitionText{source: append([]byte(nil), source...),
		assets: *g.nativePreparationUI, start: start, closing: closing}
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
	if g.modernHDText == nil || g.font == nil || g.nativeClassUI == nil || job.preparationText == nil {
		return false
	}
	d := job.preparationText
	capture := &modernHDIndexedText{palette: g.nativeClassUI.palette}
	font := g.modernHDFacilityFont(d.assets.status.Font, capture)
	frame, ok := d.compose(job.frame, font)
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	g.modernHDText.runs = append(g.modernHDText.runs, capture.runs...)
	return true
}
