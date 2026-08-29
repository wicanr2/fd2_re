package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type nativeCommand3PresentationPhase uint8

const (
	nativeCommand3CommonPrelude nativeCommand3PresentationPhase = iota
	nativeCommand3Actor
	nativeCommand3Handler
	nativeCommand3CommonTail
)

type nativeCommand3HandlerFrame struct {
	image       *ebiten.Image
	targetIndex int
	hpStage     int
	playSub1    bool
	playSub2    bool
}

type nativeCommand3PresentationJob struct {
	actor                  *battle.Unit
	plan                   *battle.NativeCommandDamagePlan
	prelude                []*ebiten.Image
	actorBlack, actorPulse []*ebiten.Image
	actorSpecs             []battlepresent.NativeCommand0ActorFrame
	handler                []nativeCommand3HandlerFrame
	tail                   []*ebiten.Image
	phase                  nativeCommand3PresentationPhase
	frame                  int
	pulseBlack             bool
	drawn, mpPublished     bool
	actorMPBefore          int
	actorActedBefore       bool
	targetHPBefore         []int
	then                   func([]battle.NativeCommandDamageResult)
}

func (g *Game) startNativeCommand3Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil ||
		g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command3 presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command3 raw actor provenance unavailable")
	}
	var plan *battle.NativeCommandDamagePlan
	var err error
	if actor.Camp == battle.Enemy {
		plan, err = g.st.PlanNativeAICommandDamage(actor, 3, g.st.NativeCommandResistances, g.nativeRNGState)
	} else {
		plan, err = g.st.PlanNativeCommandDamage(actor, confirmed, 3, g.st.NativeCommandResistances, g.nativeRNGState)
	}
	if err != nil {
		return err
	}
	if len(plan.Results) == 0 || plan.DamageStages != figani.NativeCommand3DamageStages ||
		len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[3].EffectMode != 2 {
		return errors.New("native command3 final targets unavailable")
	}
	for _, result := range plan.Results {
		if result.Target == nil || !result.Target.HasBattleFig || !result.Target.HasNativeRecordByte6 {
			return errors.New("native command3 raw target provenance unavailable")
		}
	}

	initial, err := g.nativeCommandScene.InitialBackground(g.handlerChapter)
	if err != nil {
		return err
	}
	actorControl, err := nativeCommand0Control(g.m, actor)
	if err != nil {
		return err
	}
	actorGate, err := battle.NativeCommandBackgroundGate(actor)
	if err != nil {
		return err
	}
	backgroundTargets := make([]fdicon.NativeCommandBackgroundTarget, 0, len(plan.Results))
	for _, result := range plan.Results {
		control, err := nativeCommand0Control(g.m, result.Target)
		if err != nil {
			return err
		}
		gate, err := battle.NativeCommandBackgroundGate(result.Target)
		if err != nil {
			return err
		}
		backgroundTargets = append(backgroundTargets, fdicon.NativeCommandBackgroundTarget{Gate: gate, Control: control})
	}
	actorSelector := initial
	if !actorGate || actorSelector == 0 {
		actorSelector = actorControl[2]
	}
	targetSelector := fdicon.NativeCommandBackgroundSelector(initial, backgroundTargets)
	bgSelector, taiSelector := targetSelector, actorSelector
	if actor.NativeRecordByte6 == 0 {
		bgSelector, taiSelector = actorSelector, targetSelector
	}
	background, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", int(bgSelector))
	if err != nil {
		return err
	}
	platform, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", int(taiSelector))
	if err != nil {
		return err
	}
	actorIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
	if err != nil {
		return err
	}
	actorEffect, err := nativeCommand6ActorEffect(separatedAssetPath("animations"), actor.BattleFig)
	if err != nil {
		return err
	}
	targetIdle := make([]*figani.Animation, len(plan.Results))
	for index, result := range plan.Results {
		targetIdle[index], err = figani.LoadSeparatedResource(separatedAssetPath("animations"), result.Target.BattleFig*3)
		if err != nil {
			return err
		}
	}
	effectResource := 39
	if actor.NativeRecordByte6 == 0 {
		effectResource = 43
	}
	effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand3PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	if err := g.requireSeparatedCommandSounds(schedule.SoundResource, 0, 1, 2); err != nil {
		return fmt.Errorf("native command3 sounds: %w", err)
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand3Actor) == 0 || len(g.sfxCommand3Sub1) == 0 || len(g.sfxCommand3Sub2) == 0) {
		return errors.New("native command3 converted #84 samples unavailable")
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	dac, _, err := loadNativeBattlePalette()
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	actorRecord, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		return err
	}
	actorAfter := *actor
	actorAfter.MP = plan.MPAfter
	actorAfterRecord, err := battle.NativeBattlePanelRecordForUnit(&actorAfter)
	if err != nil {
		return err
	}
	targetBases := make([][][]byte, len(plan.Results))
	for targetIndex, result := range plan.Results {
		runtimeIndex, err := nativeCommand24RuntimeUnitIndex(g.st, result.Target)
		if err != nil {
			return err
		}
		targetBases[targetIndex] = make([][]byte, figani.NativeCommand3DamageStages+1)
		for stage := 0; stage <= figani.NativeCommand3DamageStages; stage++ {
			staged := *result.Target
			staged.HP = result.HPBefore - (result.HPBefore-result.HPAfter)*stage/figani.NativeCommand3DamageStages
			record, err := battle.NativeBattlePanelRecordForUnit(&staged)
			if err != nil {
				return err
			}
			targetBases[targetIndex][stage], err = nativeCommand0Base(background, panelAssets, actorAfterRecord, record, actorIndex, runtimeIndex, g.handlerChapter, &platform)
			if err != nil {
				return err
			}
		}
	}
	transitionBases := make([][]byte, len(plan.Results)-1)
	for index := range transitionBases {
		transitionBases[index] = targetBases[index][figani.NativeCommand3DamageStages]
	}
	firstTargetRecord, err := battle.NativeBattlePanelRecordForUnit(plan.Results[0].Target)
	if err != nil {
		return err
	}
	firstTargetIndex, err := nativeCommand24RuntimeUnitIndex(g.st, plan.Results[0].Target)
	if err != nil {
		return err
	}
	preludeBase, err := nativeCommand0Base(background, panelAssets, actorRecord, firstTargetRecord, actorIndex, firstTargetIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	actorBaseBefore, err := nativeCommand0Base(background, panelAssets, actorRecord, firstTargetRecord, actorIndex, firstTargetIndex, g.handlerChapter, &platform)
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: preludeBase, ActorIdle: actorIdle.Frames[0], FirstTargetIdle: targetIdle[0].Frames[0],
		Platform: &platform, RawSide: actor.NativeRecordByte6, Mode: 0, BaselineDAC: dac,
	})
	if err != nil {
		return err
	}
	preludeImages, err := nativeCommand24PreludeImages(prelude)
	if err != nil {
		return err
	}
	actorSpecs, err := battlepresent.BuildNativeCommand0ActorFrames(battlepresent.NativeCommand0ActorInput{
		BaseBefore: actorBaseBefore, BaseAfter: targetBases[0][0], ActorEffect: actorEffect,
		FirstTargetIdle: targetIdle[0], RawSide: actor.NativeRecordByte6,
		Background: background, Platform: platform, LUT: g.nativeMapAssets.LUTs[11],
	})
	if err != nil {
		return err
	}
	actorPixels := make([][]byte, len(actorSpecs))
	for index := range actorSpecs {
		actorPixels[index] = actorSpecs[index].Pixels
	}
	actorBlack, err := nativeCommand0Images(actorPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	pulseDAC := append([]byte(nil), dac...)
	if len(pulseDAC) != 256*3 || len(g.nativeCommandPaletteFlash.Entries) != battle.NativeCommandRecordCount {
		return errors.New("native command3 DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[3][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	effectSequence, err := battlepresent.BuildNativeCommand3EffectSequence(battlepresent.NativeCommand3EffectInput{
		FrontBase: targetBases[0][0], TailBase: targetBases[len(targetBases)-1][figani.NativeCommand3DamageStages],
		TargetBases: targetBases, TransitionBases: transitionBases, ActorEffect: actorEffect,
		TargetIdle: targetIdle, Effect: effect, Schedule: schedule, RawSide: actor.NativeRecordByte6,
	})
	if err != nil {
		return err
	}
	handler := make([]nativeCommand3HandlerFrame, 0)
	appendFrames := func(frames []battlepresent.NativeCommand3RenderedFrame, targetIndex int) error {
		for _, rendered := range frames {
			images, err := nativeCommand0Images([][]byte{rendered.Pixels}, g.nativeUIPalette)
			if err != nil {
				return err
			}
			handler = append(handler, nativeCommand3HandlerFrame{
				image: images[0], targetIndex: targetIndex, hpStage: rendered.HPStage,
				playSub1: rendered.PlaySub1, playSub2: rendered.PlaySub2,
			})
		}
		return nil
	}
	if err := appendFrames(effectSequence.Front, -1); err != nil {
		return err
	}
	for targetIndex, frames := range effectSequence.Targets {
		if err := appendFrames(frames, targetIndex); err != nil {
			return err
		}
		if targetIndex < len(effectSequence.Transitions) {
			if err := appendFrames(effectSequence.Transitions[targetIndex], -1); err != nil {
				return err
			}
		}
	}
	if err := appendFrames(effectSequence.Tail, -1); err != nil {
		return err
	}
	tailPixels, err := nativeCommand1TailPixels(
		targetBases[len(targetBases)-1][figani.NativeCommand3DamageStages], background, platform, actorEffect,
		targetIdle[len(targetIdle)-1], effectSequence.NextIdleFrame, effectSequence.NextIdleRepeat, g.nativeMapAssets.LUTs,
	)
	if err != nil {
		return err
	}
	commonTail, err := nativeCommand0Images(tailPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	if len(handler) == 0 {
		return errors.New("native command3 handler frames unavailable")
	}
	g.nativeCmd3Presentation = &nativeCommand3PresentationJob{
		actor: actor, plan: plan, prelude: preludeImages, actorBlack: actorBlack, actorPulse: actorPulse,
		actorSpecs: actorSpecs, handler: handler, tail: commonTail, actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: make([]int, len(plan.Results)), then: then,
	}
	for index, result := range plan.Results {
		g.nativeCmd3Presentation.targetHPBefore[index] = result.Target.HP
	}
	return nil
}

func (g *Game) failNativeCommand3Presentation(err error) {
	j := g.nativeCmd3Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	for index, result := range j.plan.Results {
		if result.Target != nil && index < len(j.targetHPBefore) {
			result.Target.HP = j.targetHPBefore[index]
		}
	}
	g.nativeCmd3Presentation = nil
	g.loadErr = "native command3 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand3Presentation() {
	j := g.nativeCmd3Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand3CommonPrelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand3Actor, 0
		}
	case nativeCommand3Actor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand3Presentation(err)
					return
				}
				j.mpPublished = true
				g.playRaw(g.sfxCommand3Actor)
			}
			j.pulseBlack = true
			return
		}
		if spec.PublishMP && !j.mpPublished {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand3Presentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand3Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand3Presentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand3Handler, 0
		}
	case nativeCommand3Handler:
		frame := j.handler[j.frame]
		if frame.hpStage != 0 {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, frame.targetIndex, frame.hpStage); err != nil {
				g.failNativeCommand3Presentation(err)
				return
			}
		}
		if frame.playSub1 {
			g.playRaw(g.sfxCommand3Sub1)
		}
		if frame.playSub2 {
			g.playRaw(g.sfxCommand3Sub2)
		}
		j.frame++
		if j.frame < len(j.handler) {
			return
		}
		j.phase, j.frame = nativeCommand3CommonTail, 0
	case nativeCommand3CommonTail:
		j.frame++
		if j.frame < len(j.tail) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand3Presentation(err)
			return
		}
		// 數值計畫發布自身 RNGAfter；handler 軌道本身沒有額外亂數來源。
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd3Presentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand3Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd3Presentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand3CommonPrelude:
		if j.frame >= 0 && j.frame < len(j.prelude) {
			frame = j.prelude[j.frame]
		}
	case nativeCommand3Actor:
		if j.frame >= 0 && j.frame < len(j.actorBlack) && j.frame < len(j.actorPulse) && j.frame < len(j.actorSpecs) {
			frame = j.actorBlack[j.frame]
			if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
				frame = j.actorPulse[j.frame]
			}
		}
	case nativeCommand3Handler:
		if j.frame >= 0 && j.frame < len(j.handler) {
			frame = j.handler[j.frame].image
		}
	case nativeCommand3CommonTail:
		if j.frame >= 0 && j.frame < len(j.tail) {
			frame = j.tail[j.frame]
		}
	}
	if frame == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(frame, op)
	j.drawn = true
	return true
}
