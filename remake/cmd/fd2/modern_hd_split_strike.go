package main

import (
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 在原生plan消耗Go亂數前檢查HD，但所有銀行維持未發布。
func (g *Game) stageModernHDSplitStrikeBanks(actor, confirmed *battle.Unit, command int) (func(), error) {
	if g.modernStoryPortraits == nil {
		return func() {}, nil
	}
	if g.st == nil || actor == nil || len(g.st.NativeCommandBook) != 36 || command < 0 || command >= 36 ||
		g.st.NativeCommandBook[command].ID != command {
		return nil, fmt.Errorf("modern HD split strike: command source unavailable")
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	r := g.st.NativeCommandBook[command]
	targets, err := battle.NativeCommandEffectTargets(g.st.W, g.st.H, actor, confirmed,
		r.SelectionMode, r.EffectMode, r.TargetCode, flags, g.st.Units)
	if err != nil {
		return nil, err
	}
	resources := []int{actor.BattleFig * 3, actor.BattleFig*3 + 2}
	for _, target := range targets {
		if target == nil || !target.HasBattleFig {
			return nil, fmt.Errorf("modern HD split strike: target source unavailable")
		}
		resources = append(resources, target.BattleFig*3)
	}
	for _, unit := range append([]*battle.Unit{actor}, targets...) {
		selector, err := nativeCommand24BGSelector(g.m, unit)
		if err != nil {
			return nil, err
		}
		background, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", selector)
		if err != nil {
			return nil, err
		}
		var platform *fdother.Frame
		if unit == actor && actor.NativeRecordByte6 != 0 {
			frame, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", selector)
			if err != nil {
				return nil, err
			}
			platform = &frame
		}
		sources, err := modernSplitStrikeScenery(background, selector, platform)
		if err != nil {
			return nil, err
		}
		for _, source := range sources {
			if _, _, err := g.modernStoryPortraits.battleSurfaceHD(source); err != nil {
				return nil, err
			}
		}
	}
	transitions, err := modernBattleTransitionSources()
	if err != nil {
		return nil, err
	}
	for _, source := range transitions {
		if _, _, err := g.modernStoryPortraits.battleSurfaceHD(source); err != nil {
			return nil, err
		}
	}
	return g.modernStoryPortraits.stageBattleHDBanks(resources, nil)
}

func modernSplitStrikeScenery(background fdother.Frame, selector int, platform *fdother.Frame) ([]modernHDBattleSource, error) {
	if background.Width != 320 || background.Height != 100 {
		return nil, nil
	}
	frame, err := modernHDBattleSurfaceFrame(background)
	if err != nil {
		return nil, err
	}
	frame.X, frame.Y = 0, 50
	sources := []modernHDBattleSource{{Archive: "BG.DAT", Resource: selector, Frame: frame, Fill: -1}}
	if platform != nil {
		p, err := modernBattlePlatformSource(*platform, selector)
		if err != nil {
			return nil, err
		}
		sources = append(sources, p)
	}
	return sources, nil
}

// 消費既有24／29 owner的索引及乾淨base，不寫入排程或交易。
type modernHDSplitStrike struct {
	prelude                          []nativeCompoundPresentedFrame
	effect                           *figani.Animation
	effectID                         int
	actorBefore, actorAfter          []byte
	actorImpact, targetStart, impact int
	palette                          color.Palette
	targets                          []modernHDSplitStrikeTarget
	actorScenery                     []modernHDBattleSource
}

type modernHDSplitStrikeTarget struct {
	before, after []byte
	idle          *figani.Animation
	idleID        int
	transitions   []nativeCompoundPresentedFrame
	scenery       []modernHDBattleSource
}

func bindModernHDSplitStrike(prelude battlepresent.NativeCommandPreludeInput,
	presents []battlepresent.NativeCommand24PreludeFrame, actorIdleID int,
	effect *figani.Animation, effectID, actorImpact, targetStart, impact int,
	before, after []byte, palette color.Palette, scenery ...modernHDBattleSource) (*modernHDSplitStrike, error) {
	if effect == nil || actorImpact < 0 || actorImpact >= targetStart || targetStart <= 0 ||
		targetStart >= len(effect.Frames) || impact < targetStart || impact >= len(effect.Frames) ||
		len(before) != 64000 || len(after) != 64000 || len(palette) != 256 || actorIdleID < 0 || effectID < 0 {
		return nil, fmt.Errorf("modern HD split strike: native inputs unavailable")
	}
	hd := &modernHDSplitStrike{effect: effect, effectID: effectID, actorBefore: before, actorAfter: after,
		actorImpact: actorImpact, targetStart: targetStart, impact: impact, palette: palette,
		prelude: make([]nativeCompoundPresentedFrame, len(presents))}
	if err := bindModernHDCompoundPrelude(hd.prelude, presents, prelude, actorIdleID); err != nil {
		return nil, err
	}
	hd.actorScenery = scenery
	for _, base := range []*[]byte{&hd.actorBefore, &hd.actorAfter} {
		source := &modernHDBattleSources{Base: *base}
		if err := bindModernPhysicalBaseScenery(source, scenery); err != nil {
			return nil, err
		}
		*base = source.Base
	}
	for i := range hd.prelude {
		source := hd.prelude[i].hdSources
		for _, layer := range scenery {
			if layer.Archive == "BG.DAT" {
				source.BaseScenery = append(source.BaseScenery, layer)
			}
			if layer.Archive == "TAI.DAT" && prelude.RawSide != 0 {
				// bindModernHDCompoundPrelude明示先放platform，再放actor。
				if len(source.Layers) != 2 || source.Layers[0].Resource != -1 {
					return nil, fmt.Errorf("modern HD split strike: prelude platform differs")
				}
				layer.DX = source.Layers[0].DX
				source.Layers[0] = layer
			}
		}
	}
	return hd, nil
}

func (h *modernHDSplitStrike) actorEndSources() *modernHDBattleSources {
	return &modernHDBattleSources{Base: h.actorAfter, BaseScenery: h.actorScenery,
		Layers: []modernHDBattleSource{modernHDCommandFigure(h.effect, h.effectID, h.targetStart-1)}}
}

func (h *modernHDSplitStrike) addTarget(before, after []byte, idle *figani.Animation, idleID int,
	source *modernHDBattleSources, presents []battlepresent.NativeCommand24BackgroundPresent, bg [3]fdother.Frame, scenery ...modernHDBattleSource) error {
	if h == nil || len(before) != 64000 || len(after) != 64000 || idle == nil || len(idle.Frames) == 0 || idleID < 0 {
		return fmt.Errorf("modern HD split strike: target unavailable")
	}
	cleanBefore, cleanAfter := &modernHDBattleSources{Base: before}, &modernHDBattleSources{Base: after}
	if err := bindModernPhysicalBaseScenery(cleanBefore, scenery); err != nil {
		return err
	}
	if err := bindModernPhysicalBaseScenery(cleanAfter, scenery); err != nil {
		return err
	}
	sources, err := modernHDBattleScrollScenerySources(presents, source, cleanBefore, idle.Frames[0], idleID, bg)
	if err != nil {
		return err
	}
	target := modernHDSplitStrikeTarget{before: cleanBefore.Base, after: cleanAfter.Base, idle: idle, idleID: idleID, scenery: scenery}
	for i, value := range sources {
		phase := fmt.Sprintf("transition-out-%d", len(h.targets))
		if presents[i].Incoming {
			phase = fmt.Sprintf("transition-in-%d", len(h.targets))
		}
		f := modernHDCommandFrame(value.Base, value.Layers, h.palette, phase)
		target.transitions = append(target.transitions, f)
	}
	h.targets = append(h.targets, target)
	return nil
}

func (h *modernHDSplitStrike) targetEndSources(targetIndex, idleFrame int) *modernHDBattleSources {
	t := &h.targets[targetIndex]
	return &modernHDBattleSources{Base: t.after, BaseScenery: t.scenery, Layers: []modernHDBattleSource{
		modernHDCommandFigure(t.idle, t.idleID, idleFrame),
		modernHDCommandFigure(h.effect, h.effectID, len(h.effect.Frames)-1),
	}}
}

func (h *modernHDSplitStrike) sourceFrames() []nativeCompoundPresentedFrame {
	if h == nil {
		return nil
	}
	frames := append([]nativeCompoundPresentedFrame(nil), h.prelude...)
	f := modernHDCommandFrame(h.actorBefore, []modernHDBattleSource{modernHDCommandFigure(h.effect, h.effectID, 0)}, h.palette, "actor")
	f.hdSources.BaseScenery = h.actorScenery
	frames = append(frames, f)
	for _, target := range h.targets {
		frames = append(frames, target.transitions...)
		f := modernHDCommandFrame(target.before, []modernHDBattleSource{modernHDCommandFigure(target.idle, target.idleID, 0)}, h.palette, "target")
		f.hdSources.BaseScenery = target.scenery
		frames = append(frames, f)
	}
	return frames
}

func (h *modernHDSplitStrike) frame(prelude, transition, target, effect, idle int, damaged bool, shake int) *nativeCompoundPresentedFrame {
	if h == nil || target < 0 || target >= len(h.targets) || effect < 0 || effect >= len(h.effect.Frames) {
		return nil
	}
	if prelude >= 0 && prelude < len(h.prelude) {
		return &h.prelude[prelude]
	}
	t := &h.targets[target]
	if transition >= 0 {
		if transition >= len(t.transitions) {
			return nil
		}
		return &t.transitions[transition]
	}
	base, phase := h.actorBefore, "actor"
	scenery := h.actorScenery
	var layers []modernHDBattleSource
	if effect < h.targetStart {
		if effect >= h.actorImpact {
			base = h.actorAfter
		}
	} else {
		if idle < 0 || idle >= len(t.idle.Frames) {
			return nil
		}
		base, phase = t.before, fmt.Sprintf("target-%d", target)
		scenery = t.scenery
		if damaged || effect >= h.impact {
			base = t.after
		}
		layer := modernHDCommandFigure(t.idle, t.idleID, idle)
		layer.DX = -shake
		layers = append(layers, layer)
	}
	layers = append(layers, modernHDCommandFigure(h.effect, h.effectID, effect))
	f := modernHDCommandFrame(base, layers, h.palette, phase)
	f.hdSources.BaseScenery = scenery
	return &f
}

func (j *nativeCommand24PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil {
		return nil
	}
	shake := 0
	if j.frame == j.schedule.TargetImpactFrame && !j.damagePublished {
		shake = j.schedule.ShakeOffsets[5]
	} else if j.damagePublished && j.shakeCounter >= 0 && j.shakeCounter < len(j.schedule.ShakeOffsets) {
		shake = j.schedule.ShakeOffsets[j.shakeCounter]
	}
	return j.hd.frame(j.preludeFrame, j.transitionFrame, 0, j.frame, j.targetFrame, j.damagePublished, shake)
}

func (j *nativeCommand29PresentationJob) modernFrame() *nativeCompoundPresentedFrame {
	if j == nil || j.hd == nil || j.targetIndex < 0 || j.targetIndex >= len(j.targets) {
		return nil
	}
	t := &j.targets[j.targetIndex]
	shake := 0
	if j.frame == j.schedule.TargetImpactFrame && !t.damagePublished {
		shake = j.schedule.ShakeOffsets[5]
	} else if t.damagePublished && j.shakeCounter >= 0 && j.shakeCounter < len(j.schedule.ShakeOffsets) {
		shake = j.schedule.ShakeOffsets[j.shakeCounter]
	}
	return j.hd.frame(j.preludeFrame, j.transitionFrame, j.targetIndex, j.frame, j.idleFrame, t.damagePublished, shake)
}
