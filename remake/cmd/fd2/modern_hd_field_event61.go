package main

import (
	"crypto/sha256"
	"errors"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// Only the existing sixteen destination bytes cover the frozen HD world.
// The native owner keeps every full frame, clock and completion transaction.
type modernHDFieldEvent61 struct {
	scene        *modernHDCompoundMapScene
	presentation battle.NativeFieldPresentation
	hashes       [][32]byte
	frames       []*modernHDMapFrame
	theme        *modernStoryPortraitSet
	nativeHUD    *modernHDMapHUDPanel
}

func (g *Game) freezeModernHDFieldEvent61(job *nativeFieldEvent61Job) *modernHDFieldEvent61 {
	in, err := g.freezeNativeCommand9MapInput()
	palette, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err == nil {
		err = paletteErr
	}
	h := &modernHDFieldEvent61{
		scene:        g.newModernHDCompoundMapScene(in, g.nativeMapWork, g.nativeMapVGA, palette, err),
		presentation: job.plan.Presentation,
		hashes:       make([][32]byte, len(job.frames)),
	}
	for i, frame := range job.frames {
		h.hashes[i] = sha256.Sum256(frame)
	}
	return h
}

func (g *Game) prepareModernHDFieldEvent61(job *nativeFieldEvent61Job, set *modernStoryPortraitSet) ([]*modernHDMapFrame, error) {
	if set == nil || len(set.mapSpritesHD) == 0 {
		return nil, nil
	}
	if len(set.mapSpritesHD) != 125 || job == nil || job.hd == nil {
		return nil, errors.New("modern HD event61: incomplete world or frozen source")
	}
	if err := g.validateModernHDFieldEvent61(job); err != nil {
		return nil, err
	}
	h := job.hd
	base, err := g.prepareModernHDCommandSteadyStage(h.scene.input, h.scene.work, h.scene.vga, h.scene.palette, set, h.scene.stage)
	if err != nil {
		return nil, err
	}
	frames := make([]*modernHDMapFrame, len(job.frames))
	for i, source := range job.frames {
		patch := image.NewRGBA(image.Rect(0, 0, 48, 12))
		for y := 0; y < 12; y++ {
			for x := 0; x < 48; x++ {
				patch.Set(x, y, h.scene.palette[source[(151+y/6)*320+36+x/6]])
			}
		}
		frame := *base
		frame.Overlays = append(append([]modernHDMapOverlayDraw(nil), base.Overlays...), modernHDMapOverlayDraw{
			Entry: i, Image: patch, Rect: patch.Bounds().Add(image.Pt(36*6, 151*6)),
		})
		frame.OverlaysAboveUI = true
		frames[i] = &frame
	}
	return frames, nil
}

func (g *Game) validateModernHDFieldEvent61(job *nativeFieldEvent61Job) error {
	if job == nil || job.hd == nil {
		return errors.New("modern HD event61: frozen source unavailable")
	}
	h := job.hd
	p := job.plan.Presentation
	if p != h.presentation || p.Resource != 45 || p.Frames != 59 || p.Stride != 320 ||
		p.DestinationOffset != 48356 || p.Transparent != -1 || p.DelayTicks != 2 ||
		len(job.frames) != p.Frames || len(h.hashes) != p.Frames {
		return errors.New("modern HD event61: native schedule changed")
	}
	if err := h.scene.validate(g); err != nil {
		return err
	}
	for i, frame := range job.frames {
		if len(frame) != 64000 || sha256.Sum256(frame) != h.hashes[i] {
			return errors.New("modern HD event61: native frame changed")
		}
		for at, value := range frame {
			x, y := at%320, at/320
			if (x < 36 || x >= 44 || y < 151 || y >= 153) && value != h.scene.vga[at] {
				return errors.New("modern HD event61: write outside source rectangle")
			}
		}
	}
	return nil
}

func (g *Game) prepareModernHDFieldEvent61NativeHUD(job *nativeFieldEvent61Job, set *modernStoryPortraitSet) (*modernHDMapHUDPanel, error) {
	if set != nil || !g.modernHDInterfaceEnabled() {
		return nil, nil
	}
	if err := g.validateModernHDFieldEvent61(job); err != nil {
		return nil, err
	}
	view := *g
	view.modernStoryPortraits = nil
	return view.prepareModernHDMapHUDPanel(job.hd.scene.input, job.hd.scene.palette)
}

func (g *Game) stageModernHDFieldEvent61(set *modernStoryPortraitSet) (func(), error) {
	job := g.nativeFieldEvent61
	if job == nil {
		return func() {}, nil
	}
	frames, err := g.prepareModernHDFieldEvent61(job, set)
	if err != nil {
		return nil, err
	}
	hud, err := g.prepareModernHDFieldEvent61NativeHUD(job, set)
	if err != nil {
		return nil, err
	}
	if job.hd == nil {
		return func() {}, nil
	}
	return func() { job.hd.frames, job.hd.theme, job.hd.nativeHUD = frames, set, hud }, nil
}
