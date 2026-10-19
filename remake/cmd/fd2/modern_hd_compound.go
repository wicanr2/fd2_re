package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 只附加0x27FC9已閉合的圖源，不重建或推進任何交易／音效／present排程。
func bindModernHDCompoundFrames(frames []nativeCompoundPresentedFrame, base []byte, effect *figani.Animation,
	schedule figani.NativeCompoundPresentationSchedule, phase string, baseline []byte) error {
	if len(base) != 64000 || effect == nil || len(effect.Frames) < 2 {
		return fmt.Errorf("modern HD compound: native sources unavailable")
	}
	want, err := figani.BuildNativeCompoundPresentationSchedule(schedule.CommandID, effect)
	if err != nil || want != schedule {
		return fmt.Errorf("modern HD compound: native schedule differs")
	}
	count := 0
	switch phase {
	case "slide":
		count = schedule.EffectSlideFrames
	case "main":
		count = schedule.MainFrameCount
	case "tail":
		count = schedule.TailFrames
	default:
		return fmt.Errorf("modern HD compound: unknown phase")
	}
	if count == 0 || len(frames) != count {
		return fmt.Errorf("modern HD compound: phase length differs")
	}
	palette, err := fdother.VGAPaletteFromDAC(baseline)
	if err != nil {
		return err
	}
	for index := range frames {
		frame := &frames[index]
		frame.hdSources = &modernHDBattleSources{Base: base}
		frame.hdResource, frame.hdPalette = schedule.EffectResource, palette
		layer := func(which, dx int) {
			frame.hdSources.Layers = append(frame.hdSources.Layers, modernHDBattleSource{
				Archive: "FDOTHER.DAT", Resource: schedule.EffectResource, Index: which,
				Frame: effect.Frames[which], DX: dx, Fill: -1,
			})
		}
		switch phase {
		case "slide":
			layer(0, schedule.EffectSlideStepX*(count-1-index))
		case "main":
			if schedule.OverlayFirstFrame {
				layer(0, 0)
			}
			layer(index+1, 0)
		case "tail":
			layer(len(effect.Frames)-2+index%2, 0)
			delta := 40 - index*4
			dac, err := fdother.InterpolateNativeCompoundDAC(baseline, 0, 0xff, delta, schedule.PaletteRGB)
			if err != nil {
				return err
			}
			frame.hdPalette, err = fdother.VGAPaletteFromDAC(dac)
			if err != nil {
				return err
			}
			frame.hdMix = 1 - float32(delta)/40
			for component, raw := range schedule.PaletteRGB {
				frame.hdTarget[component] = float32((raw<<2)|(raw>>4)) / 255
			}
		}
	}
	return nil
}

func (g *Game) modernCompoundEffectResource() int {
	switch {
	case g.nativeCmd32Presentation != nil:
		return 65
	case g.nativeCmd33Presentation != nil:
		return 66
	case g.nativeCmd34Presentation != nil:
		return 67
	case g.nativeCmd35Presentation != nil:
		return 68
	}
	return -1
}

func (g *Game) drawModernHDCompoundFrame(screen *ebiten.Image, frame *nativeCompoundPresentedFrame) (bool, error) {
	set := g.modernStoryPortraits
	if frame.hdSources == nil || set == nil || !set.hasHD || set.battleEffects[frame.hdResource] == nil {
		return false, nil
	}
	plan, err := g.modernHDBattlePlan(frame.hdSources, frame.hdPalette, 0)
	if err != nil {
		return false, err
	}
	plan.HDMix, plan.HDTarget = frame.hdMix, frame.hdTarget
	if err := g.publishModernHDBattlePlan(screen, plan); err != nil {
		return false, err
	}
	return true, nil
}
