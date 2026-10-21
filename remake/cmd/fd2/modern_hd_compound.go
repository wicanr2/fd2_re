package main

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
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
		frame.hdPhase = "effect-" + phase
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
	if frame.hdSources == nil || set == nil || !set.hasHD {
		return false, nil
	}
	wanted := false
	for _, layer := range frame.hdSources.Layers {
		if layer.Resource < 0 {
			continue
		}
		if layer.Archive == "FDOTHER.DAT" {
			wanted = wanted || set.battleEffects[layer.Resource] != nil
		} else {
			wanted = wanted || set.battleAnimations[layer.Resource] != nil
		}
	}
	if !wanted {
		return false, nil
	}
	plan, err := g.modernHDBattlePlan(frame.hdSources, frame.hdPalette, frame.hdSubtract)
	if err != nil {
		return false, err
	}
	plan.HDMix, plan.HDTarget = frame.hdMix, frame.hdTarget
	if err := g.publishModernHDBattlePlan(screen, plan); err != nil {
		return false, err
	}
	return true, nil
}

func bindModernHDCompoundPrelude(frames []nativeCompoundPresentedFrame, prelude []battlepresent.NativeCommand24PreludeFrame,
	in battlepresent.NativeCommandPreludeInput, resource int) error {
	if len(frames) != 9 || len(prelude) != len(frames) || len(in.Base) != 64000 || in.Mode != 1 || (in.RawSide != 0 && in.Platform == nil) {
		return fmt.Errorf("modern HD compound: prelude contract differs")
	}
	for index := range frames {
		original := prelude[index]
		if original.Stage != 8-index {
			return fmt.Errorf("modern HD compound: prelude stage differs")
		}
		palette, err := fdother.ParseVGAPalette(original.DAC)
		if err != nil {
			return err
		}
		sources := &modernHDBattleSources{Base: in.Base}
		actor := modernHDBattleSource{Frame: in.ActorIdle, Resource: resource, Fill: -1, DX: -original.Stage * 10}
		if in.RawSide != 0 {
			platform := in.Platform
			sources.Layers = append(sources.Layers, modernHDBattleSource{Resource: -1, Fill: -1, DX: original.Stage * 10,
				Frame: figani.Frame{X: 164, Y: 157, Width: platform.Width, Height: platform.Height, Pixels: platform.Indexed, Mask: platform.Mask}})
			actor.DX = original.Stage * 10
		}
		sources.Layers = append(sources.Layers, actor)
		frames[index].hdSources, frames[index].hdPalette = sources, palette
		frames[index].hdSubtract, frames[index].hdPhase = float32(original.Stage*6)/63, "prelude"
	}
	return nil
}

func bindModernHDCompoundActor(frames []nativeCompoundPresentedFrame, presents []battlepresent.NativeCompoundActorPresent,
	base []byte, actor, target *figani.Animation, rawSide byte, actorResource, targetResource int, dac []byte) error {
	if len(base) != 64000 || len(frames) == 0 || len(frames) != len(presents) || actor == nil || target == nil {
		return fmt.Errorf("modern HD compound: actor contract differs")
	}
	palette, err := fdother.ParseVGAPalette(dac)
	if err != nil {
		return err
	}
	for index, present := range presents {
		if present.ActorIndex < 0 || present.ActorIndex >= len(actor.Frames) || present.TargetIndex < 0 || present.TargetIndex >= len(target.Frames) {
			return fmt.Errorf("modern HD compound: source index differs")
		}
		a := modernHDBattleSource{Frame: actor.Frames[present.ActorIndex], Resource: actorResource, Index: present.ActorIndex, Fill: -1}
		b := modernHDBattleSource{Frame: target.Frames[present.TargetIndex], Resource: targetResource, Index: present.TargetIndex, Fill: -1}
		layers := []modernHDBattleSource{a, b}
		if rawSide == 0 {
			layers = []modernHDBattleSource{b, a}
		}
		frames[index].hdSources = &modernHDBattleSources{Base: base, Layers: layers}
		frames[index].hdPalette, frames[index].hdPhase = palette, "actor"
	}
	return nil
}

func bindModernHDCompoundActorSlide(frames []nativeCompoundPresentedFrame, base []byte, actor figani.Frame, resource int,
	schedule figani.NativeCompoundPresentationSchedule, dac []byte) error {
	if len(base) != 64000 || len(frames) != schedule.ActorSlideFrames || schedule.ActorSlideFrames != 8 || schedule.ActorSlideStepX != 20 {
		return fmt.Errorf("modern HD compound: actor slide contract differs")
	}
	palette, err := fdother.ParseVGAPalette(dac)
	if err != nil {
		return err
	}
	for index := range frames {
		frames[index].hdSources = &modernHDBattleSources{Base: base, Layers: []modernHDBattleSource{{Frame: actor, Resource: resource, Fill: -1, DX: 20 * index}}}
		frames[index].hdPalette, frames[index].hdPhase = palette, "actor-slide"
	}
	return nil
}

func modernCompoundFigureResources(frames []nativeCompoundPresentedFrame) []int {
	return modernCompoundArchiveResources(frames, "FIGANI.DAT")
}

func modernCompoundEffectResources(frames []nativeCompoundPresentedFrame) []int {
	return modernCompoundArchiveResources(frames, "FDOTHER.DAT")
}

func modernCompoundArchiveResources(frames []nativeCompoundPresentedFrame, archive string) []int {
	seen := make(map[int]bool)
	for _, frame := range frames {
		if frame.hdSources == nil {
			continue
		}
		for _, layer := range frame.hdSources.Layers {
			source := layer.Archive
			if source == "" {
				source = "FIGANI.DAT"
			}
			if layer.Resource >= 0 && source == archive {
				seen[layer.Resource] = true
			}
		}
	}
	resources := make([]int, 0, len(seen))
	for resource := range seen {
		resources = append(resources, resource)
	}
	sort.Ints(resources)
	return resources
}

type modernHDCommandBankSource interface {
	sourceFrames() []nativeCompoundPresentedFrame
}

func prepareModernHDCommandBanks(set *modernStoryPortraitSet, source modernHDCommandBankSource) error {
	frames := source.sourceFrames()
	return set.prepareBattleHDBanks(modernCompoundFigureResources(frames), modernCompoundEffectResources(frames))
}

func (g *Game) modernCompoundFigureResources() []int {
	switch {
	case g.nativeCmd32Presentation != nil:
		return modernCompoundFigureResources(g.nativeCmd32Presentation.frames)
	case g.nativeCmd33Presentation != nil:
		return modernCompoundFigureResources(g.nativeCmd33Presentation.frames)
	case g.nativeCmd34Presentation != nil:
		return modernCompoundFigureResources(g.nativeCmd34Presentation.frames)
	case g.nativeCmd35Presentation != nil:
		return modernCompoundFigureResources(g.nativeCmd35Presentation.frames)
	}
	return nil
}
