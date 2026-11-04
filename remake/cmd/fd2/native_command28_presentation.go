package main

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type nativeCommand28Event struct {
	prelude, effect, tick, target, idle, impact int
	mp                                          bool
}

// The shared strike owner retains input, F2, rollback and action completion.
// This sequence supplies command28's mode0, fixed scenery and eight impacts.
type nativeCommand28Sequence struct {
	frames []nativeCompoundPresentedFrame
	events []nativeCommand28Event
	index  int
}

func nativeCommand28Pixels(base []byte, layers []modernHDBattleSource) ([]byte, error) {
	if len(base) != 64000 {
		return nil, fmt.Errorf("native command28 base unavailable")
	}
	const stride = 960 // Padding on both sides keeps the 320px viewport clipped.
	work := make([]byte, stride*200)
	for y := 0; y < 200; y++ {
		copy(work[y*stride+320:y*stride+640], base[y*320:(y+1)*320])
	}
	for _, layer := range layers {
		if err := layer.Frame.BlitAtBase(work, stride, 320+layer.DX); err != nil {
			return nil, err
		}
	}
	pixels := make([]byte, 64000)
	for y := 0; y < 200; y++ {
		copy(pixels[y*320:(y+1)*320], work[y*stride+320:y*stride+640])
	}
	return pixels, nil
}

func (g *Game) startNativeCommand28Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommand24Damage)) error {
	if g == nil || !g.nativeFullPresentationEnabled() || g.st == nil || g.m == nil || g.rng == nil || actor == nil || confirmed == nil || !actor.HasBattleFig || actor.BattleFig != 19 || len(g.nativeUIPalette) != 256 {
		return fmt.Errorf("native command28 proven selector19 context unavailable")
	}
	if g.atk != nil || g.nativeCmd29Presentation != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil ||
		g.nativeCmd9AIPresentation != nil || g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil ||
		g.nativeCmd1012 != nil || g.native2189A != nil || g.nativeCmd32Presentation != nil ||
		g.nativeCmd33Presentation != nil || g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil || g.indexedTransition != nil {
		return fmt.Errorf("native command28 presentation already active")
	}
	if len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[28].ID != 28 {
		return fmt.Errorf("native command28 record unavailable")
	}
	record := g.st.NativeCommandBook[28]
	if record.MPCost < 0 || record.MPCost > 255 || actor.MP < record.MPCost {
		return fmt.Errorf("native command28 insufficient MP")
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return err
	}
	targets, err := battle.NativeCommandEffectTargets(g.st.W, g.st.H, actor, confirmed, record.SelectionMode, record.EffectMode, record.TargetCode, flags, g.st.Units)
	if err != nil || len(targets) == 0 {
		return fmt.Errorf("native command28 final targets unavailable: %v", err)
	}
	effect, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 59)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommandDerivedStrikeSchedule(effect, 28)
	if err != nil {
		return err
	}
	if effect.HeaderByte1 != 0 || len(effect.Frames) != 19 || schedule.TargetStart != 7 || schedule.ActorImpactFrame != 6 || schedule.AudioResource != 50 || len(schedule.TargetImpactFrames) != 8 || schedule.PreludeMode != 0 || schedule.UsesBGTransition || !schedule.UsesTargetBase {
		return fmt.Errorf("native command28 resource59 signature differs")
	}
	for i, impact := range schedule.TargetImpactFrames {
		if impact != 7+i {
			return fmt.Errorf("native command28 resource59 impact order differs")
		}
	}
	var samples []int
	for _, cue := range append(append([]figani.NativeCommandFrameCue(nil), schedule.ActorCues...), schedule.TargetCues...) {
		samples = append(samples, cue.Sample)
	}
	if err := g.requireSeparatedCommandSounds(schedule.AudioResource, samples...); err != nil {
		return err
	}
	if err := g.preflightNativeDerivedStrikeSources(actor, targets, effect, 28); err != nil {
		return err
	}
	actorIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 57)
	if err != nil {
		return err
	}
	idles := make([]*figani.Animation, len(targets))
	resources := []int{57, 59}
	for i, target := range targets {
		idles[i], err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3)
		if err != nil {
			return err
		}
		resources = append(resources, target.BattleFig*3)
	}
	actorSelector, err := nativeCommand24BGSelector(g.m, actor)
	if err != nil {
		return err
	}
	targetSelector, err := nativeCommand24BGSelector(g.m, targets[0])
	if err != nil {
		return err
	}
	background, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", targetSelector)
	if err != nil {
		return err
	}
	platform, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", actorSelector)
	if err != nil {
		return err
	}
	scenery, err := modernSplitStrikeScenery(background, targetSelector, nil)
	if err != nil {
		return err
	}
	platformSource, err := modernBattlePlatformSource(platform, actorSelector)
	if err != nil {
		return err
	}
	allScenery := append(append([]modernHDBattleSource(nil), scenery...), platformSource)
	publishHD := func() {}
	if set := g.modernStoryPortraits; set != nil {
		for _, source := range allScenery {
			if err := prepareModernBattleSurfaceSource(set, source); err != nil {
				return err
			}
		}
		publishHD, err = set.stageBattleHDBanks(resources, nil)
		if err != nil {
			return err
		}
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	var panelText []*modernBattlePanelText
	panel := func(base []byte, unit *battle.Unit, index int) error {
		raw, err := battle.NativeBattlePanelRecordForUnit(unit)
		if err != nil {
			return err
		}
		return g.renderBattlePanelText(panelAssets, raw, base, index, g.handlerChapter, &panelText)
	}
	initial := make([]byte, 64000)
	if err := panel(initial, actor, actorIndex); err != nil {
		return err
	}
	background.X, background.Y = 0, 50
	if err := background.Blit(initial, 320, -1); err != nil {
		return err
	}
	firstIndex, err := nativeCommand24RuntimeUnitIndex(g.st, targets[0])
	if err != nil {
		return err
	}
	if err := panel(initial, targets[0], firstIndex); err != nil {
		return err
	}
	withPlatform := append([]byte(nil), initial...)
	platform.X, platform.Y = 164, 157
	if err := platform.Blit(withPlatform, 320, -1); err != nil {
		return err
	}
	actorAfter := *actor
	actorAfter.MP -= record.MPCost
	afterMP := append([]byte(nil), withPlatform...)
	if err := panel(afterMP, &actorAfter, actorIndex); err != nil {
		return err
	}
	dac, _, err := loadNativeBattlePalette()
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: initial, ActorIdle: actorIdle.Frames[0], FirstTargetIdle: idles[0].Frames[0],
		Platform: &platform, RawSide: actor.NativeRecordByte6, Mode: 0, BaselineDAC: dac,
	})
	if err != nil {
		return err
	}
	sequence := &nativeCommand28Sequence{}
	appendFrame := func(base []byte, layers, surfaces []modernHDBattleSource, palette color.Palette, phase string, subtract float32, event nativeCommand28Event, sound []byte, expected []byte) error {
		pixels, err := nativeCommand28Pixels(base, layers)
		if err != nil {
			return err
		}
		if expected != nil && !bytes.Equal(pixels, expected) {
			return fmt.Errorf("native command28 prelude layer reconstruction differs")
		}
		images, err := nativeCommand24IndexedImages([][]byte{pixels}, palette)
		if err != nil {
			return err
		}
		frame := modernHDCommandFrame(base, layers, palette, phase)
		if err := bindModernPhysicalBaseScenery(frame.hdSources, surfaces); err != nil {
			return err
		}
		frame.image, frame.delay, frame.sound, frame.hdSubtract = images[0], 1, sound, subtract
		sequence.frames = append(sequence.frames, frame)
		sequence.events = append(sequence.events, event)
		return nil
	}
	for i, present := range prelude {
		base, surfaces := initial, scenery
		a := modernHDCommandFigure(actorIdle, 57, 0)
		b := modernHDCommandFigure(idles[0], targets[0].BattleFig*3, 0)
		var layers []modernHDBattleSource
		if actor.NativeRecordByte6 == 0 {
			base, surfaces = withPlatform, allScenery
			a.DX = -present.Stage * 10
			layers = []modernHDBattleSource{a, b}
		} else {
			moving := platformSource
			moving.DX, a.DX = present.Stage*10, present.Stage*10
			layers = []modernHDBattleSource{b, moving, a}
		}
		palette, err := fdother.ParseVGAPalette(present.DAC)
		if err != nil {
			return err
		}
		if err := appendFrame(base, layers, surfaces, palette, "prelude", float32(present.Stage*6)/63,
			nativeCommand28Event{prelude: i}, nil, present.Pixels); err != nil {
			return err
		}
	}
	// All static input and HD checks precede the caller-owned damage RNG.
	plan, err := g.st.PlanNativeCommandDerivedStrike(actor, confirmed, 28, g.rng)
	if err != nil {
		return err
	}
	if plan.TargetImpactCount != len(schedule.TargetImpactFrames) {
		return fmt.Errorf("native command28 numeric/presentation impact count differs")
	}
	idle, idleRepeat := 0, 0
	for effectIndex := 0; effectIndex < schedule.TargetStart; effectIndex++ {
		for tick := 0; tick < schedule.FrameDelays[effectIndex]; tick++ {
			base := withPlatform
			if effectIndex >= schedule.ActorImpactFrame {
				base = afterMP
			}
			layers := []modernHDBattleSource{modernHDCommandFigure(idles[0], targets[0].BattleFig*3, idle), modernHDCommandFigure(effect, 59, effectIndex)}
			var sound []byte
			if tick == 0 && effect.Frames[effectIndex].RawByte5 != 0 {
				sound = g.separatedCommandSound(schedule.AudioResource, int(effect.Frames[effectIndex].RawByte5))
			}
			if err := appendFrame(base, layers, allScenery, g.nativeUIPalette, "actor", 0,
				nativeCommand28Event{prelude: 9, effect: effectIndex, tick: tick, idle: idle, mp: effectIndex == schedule.ActorImpactFrame && tick == 0}, sound, nil); err != nil {
				return err
			}
			idleRepeat++
			if idleRepeat >= idles[0].Frames[idle].Delay {
				idleRepeat, idle = 0, (idle+1)%len(idles[0].Frames)
			}
		}
	}
	base := append([]byte(nil), afterMP...)
	for targetIndex, target := range targets {
		index, err := nativeCommand24RuntimeUnitIndex(g.st, target)
		if err != nil {
			return err
		}
		if err := panel(base, target, index); err != nil {
			return err
		}
		idle, idleRepeat, impact, shake := 0, 0, 0, -1
		for effectIndex := schedule.TargetStart; effectIndex < len(effect.Frames); effectIndex++ {
			isImpact := effect.Frames[effectIndex].RawByte4 == 1
			if isImpact {
				impact, shake = impact+1, 5
				projection, err := battle.NativeCommand28StageResult(plan, targetIndex, impact)
				if err != nil {
					return err
				}
				frozen := *target
				frozen.HP = projection.HPAfter
				if err := panel(base, &frozen, index); err != nil {
					return err
				}
			}
			for tick := 0; tick < schedule.FrameDelays[effectIndex]; tick++ {
				b := modernHDCommandFigure(idles[targetIndex], target.BattleFig*3, idle)
				if shake >= 0 {
					b.DX = -schedule.ShakeOffsets[shake]
				}
				var sound []byte
				if tick == 0 && effect.Frames[effectIndex].RawByte5 != 0 {
					sound = g.separatedCommandSound(schedule.AudioResource, int(effect.Frames[effectIndex].RawByte5))
				}
				stage := 0
				if tick == 0 && isImpact {
					stage = impact
				}
				if err := appendFrame(append([]byte(nil), base...), []modernHDBattleSource{b, modernHDCommandFigure(effect, 59, effectIndex)}, allScenery, g.nativeUIPalette,
					fmt.Sprintf("target-%d", targetIndex), 0, nativeCommand28Event{prelude: 9, effect: effectIndex, tick: tick, target: targetIndex, idle: idle, impact: stage}, sound, nil); err != nil {
					return err
				}
				if shake >= 0 {
					shake--
				}
				idleRepeat++
				if idleRepeat >= idles[targetIndex].Frames[idle].Delay {
					idleRepeat, idle = 0, (idle+1)%len(idles[targetIndex].Frames)
				}
			}
		}
	}
	bindModernBattlePanelFrames(panelText, sequence.frames)
	publishHD()
	job := &nativeCommand29PresentationJob{command28: sequence, hd: &modernHDSplitStrike{sourceOverride: sequence.frames}, actor: actor, plan: plan, schedule: schedule,
		preludeFrames: make([]*ebiten.Image, len(prelude)), targets: make([]nativeCommand29TargetPresentation, len(targets)), transitionFrame: -1,
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, then: then}
	for i, target := range targets {
		job.targets[i] = nativeCommand29TargetPresentation{unit: target, hpBefore: target.HP, rawByte5Before: target.NativeRecordByte5}
	}
	g.nativeCmd29Presentation = job
	return nil
}

func (g *Game) stepNativeCommand28Presentation() {
	j := g.nativeCmd29Presentation
	if j == nil || j.command28 == nil || !j.drawn {
		return
	}
	j.drawn = false
	sequence := j.command28
	event := sequence.events[sequence.index]
	if event.mp {
		if j.mpPublished {
			g.failNativeCommand29Presentation(fmt.Errorf("native command28 duplicate MP marker"))
			return
		}
		if err := battle.ApplyNativeCommandDerivedStrikeMP(j.plan); err != nil {
			g.failNativeCommand29Presentation(err)
			return
		}
		j.mpPublished = true
	}
	if event.impact > 0 {
		if !j.mpPublished {
			g.failNativeCommand29Presentation(fmt.Errorf("native command28 impact before MP marker"))
			return
		}
		if err := battle.ApplyNativeCommand28TargetStage(j.plan, event.target, event.impact); err != nil {
			g.failNativeCommand29Presentation(err)
			return
		}
		j.targets[event.target].damagePublished = event.impact == 8
	}
	g.playRaw(sequence.frames[sequence.index].sound)
	sequence.index++
	if sequence.index < len(sequence.frames) {
		next := sequence.events[sequence.index]
		j.preludeFrame, j.frame, j.repeat, j.targetIndex, j.idleFrame = next.prelude, next.effect, next.tick, next.target, next.idle
		return
	}
	if err := battle.CompleteNativeCommandDerivedStrike(j.plan); err != nil {
		g.failNativeCommand29Presentation(err)
		return
	}
	then, results := j.then, append([]battle.NativeCommand24Damage(nil), j.plan.Results...)
	g.nativeCmd29Presentation = nil
	if then != nil {
		then(results)
	}
}

func (g *Game) drawNativeCommand28Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd29Presentation
	if j == nil || j.command28 == nil || j.command28.index < 0 || j.command28.index >= len(j.command28.frames) {
		return false
	}
	frame := &j.command28.frames[j.command28.index]
	if drawn, err := g.drawModernHDCompoundFrame(screen, frame); err != nil {
		g.failNativeCommand29Presentation(err)
		return true
	} else if drawn {
		j.drawn = true
		return true
	}
	if frame.image == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(frame.image, op)
	j.drawn = true
	return true
}
