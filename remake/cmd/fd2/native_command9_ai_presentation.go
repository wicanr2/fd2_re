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

type nativeCommand9AIPresentationPhase uint8

const (
	nativeCommand9AICommonPrelude nativeCommand9AIPresentationPhase = iota
	nativeCommand9AIActor
	nativeCommand9AIHandler
	nativeCommand9AICommonTail
)

type nativeCommand9AIHandlerFrame struct {
	image              *ebiten.Image
	hpStage            int
	playSub1, playSub2 bool
	holdAfter          int
}

type nativeCommand9AIPresentationJob struct {
	actor                  *battle.Unit
	plan                   *battle.NativeCommandDamagePlan
	prelude                []*ebiten.Image
	actorBlack, actorPulse []*ebiten.Image
	actorSpecs             []battlepresent.NativeCommand0ActorFrame
	handler                []nativeCommand9AIHandlerFrame
	tail                   []*ebiten.Image
	phase                  nativeCommand9AIPresentationPhase
	frame                  int
	hold                   int
	pulseBlack             bool
	drawn, mpPublished     bool
	actorMPBefore          int
	actorActedBefore       bool
	targetHPBefore         int
	then                   func([]battle.NativeCommandDamageResult)
}

// startNativeCommand9AIPresentation owns only the 0x15311 gated enemy route.
// The player 0x214AD map compositor is deliberately a separate owner.
func (g *Game) startNativeCommand9AIPresentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil ||
		g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil || g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil ||
		g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command9 AI presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || actor.NativeRecordByte6 != 0 ||
		len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command9 AI raw-side-zero actor provenance unavailable")
	}
	plan, err := g.st.PlanNativeAICommandDamageSingleTarget(actor, confirmed, 9, g.st.NativeCommandResistances, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != confirmed || plan.DamageStages != figani.NativeCommand9AIDamageStages ||
		len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[9].EffectMode != 0 {
		return errors.New("native command9 AI single final target unavailable")
	}
	target := plan.Results[0].Target
	if target == nil || !target.HasBattleFig || !target.HasNativeRecordByte6 {
		return errors.New("native command9 AI raw target provenance unavailable")
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
	targetControl, err := nativeCommand0Control(g.m, target)
	if err != nil {
		return err
	}
	targetGate, err := battle.NativeCommandBackgroundGate(target)
	if err != nil {
		return err
	}
	actorSelector := initial
	if !actorGate || actorSelector == 0 {
		actorSelector = actorControl[2]
	}
	targetSelector := fdicon.NativeCommandBackgroundSelector(initial, []fdicon.NativeCommandBackgroundTarget{{Gate: targetGate, Control: targetControl}})
	bgSelector, taiSelector := targetSelector, actorSelector

	fdotherPath := nativeFDOTHERPath()
	if fdotherPath == "" {
		return errors.New("native command9 AI player-provided archives unavailable")
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
	targetIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3)
	if err != nil {
		return err
	}
	effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", figani.NativeCommand9AIEffectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand9AISchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	for sample := 0; sample <= 2; sample++ {
		if raw, readErr := fdother.ReadNestedResource(fdotherPath, schedule.SoundResource, sample); readErr != nil || len(raw) == 0 {
			return fmt.Errorf("native command9 AI FDOTHER #90 sub%d unavailable", sample)
		}
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand8Actor) == 0 || len(g.sfxCommand8Sub1) == 0 || len(g.sfxCommand8Sub2) == 0) {
		return errors.New("native command9 AI converted #90 samples unavailable")
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	dac, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	targetIndex, err := nativeCommand24RuntimeUnitIndex(g.st, target)
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
	targetBases := make([][]byte, figani.NativeCommand9AIDamageStages+1)
	for stage := 0; stage <= figani.NativeCommand9AIDamageStages; stage++ {
		staged := *target
		staged.HP = plan.Results[0].HPBefore - (plan.Results[0].HPBefore-plan.Results[0].HPAfter)*stage/figani.NativeCommand9AIDamageStages
		record, err := battle.NativeBattlePanelRecordForUnit(&staged)
		if err != nil {
			return err
		}
		targetBases[stage], err = nativeCommand0Base(background, panelAssets, actorAfterRecord, record, actorIndex, targetIndex, g.handlerChapter, &platform)
		if err != nil {
			return err
		}
	}
	targetRecord, err := battle.NativeBattlePanelRecordForUnit(target)
	if err != nil {
		return err
	}
	preludeBase, err := nativeCommand0Base(background, panelAssets, actorRecord, targetRecord, actorIndex, targetIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	actorBaseBefore, err := nativeCommand0Base(background, panelAssets, actorRecord, targetRecord, actorIndex, targetIndex, g.handlerChapter, &platform)
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{Base: preludeBase, ActorIdle: actorIdle.Frames[0], FirstTargetIdle: targetIdle.Frames[0], Platform: &platform, RawSide: actor.NativeRecordByte6, Mode: 0, BaselineDAC: dac})
	if err != nil {
		return err
	}
	preludeImages, err := nativeCommand24PreludeImages(prelude)
	if err != nil {
		return err
	}
	actorSpecs, err := battlepresent.BuildNativeCommand0ActorFrames(battlepresent.NativeCommand0ActorInput{BaseBefore: actorBaseBefore, BaseAfter: targetBases[0], ActorEffect: actorEffect, FirstTargetIdle: targetIdle, RawSide: actor.NativeRecordByte6, Background: background, Platform: platform, LUT: g.nativeMapAssets.LUTs[11]})
	if err != nil {
		return err
	}
	actorPixels := make([][]byte, len(actorSpecs))
	for i := range actorSpecs {
		actorPixels[i] = actorSpecs[i].Pixels
	}
	actorBlack, err := nativeCommand0Images(actorPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	pulseDAC := append([]byte(nil), dac...)
	if len(pulseDAC) != 256*3 || len(g.nativeCommandPaletteFlash.Entries) != battle.NativeCommandRecordCount {
		return errors.New("native command9 AI DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[9][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	sequence, err := battlepresent.BuildNativeCommand9AISequence(battlepresent.NativeCommand9AIInput{BeforeBase: targetBases[0], AfterBase: targetBases[figani.NativeCommand9AIDamageStages], TargetBases: targetBases, Actor: actorIdle, Target: targetIdle, Effect: effect, Schedule: schedule})
	if err != nil {
		return err
	}
	handler := make([]nativeCommand9AIHandlerFrame, 0, 119)
	appendFrames := func(frames []battlepresent.NativeCommand9RenderedFrame) error {
		for _, rendered := range frames {
			images, err := nativeCommand0Images([][]byte{rendered.Pixels}, g.nativeUIPalette)
			if err != nil {
				return err
			}
			handler = append(handler, nativeCommand9AIHandlerFrame{image: images[0], hpStage: rendered.HPStage, playSub1: rendered.PlaySub1, playSub2: rendered.PlaySub2})
		}
		return nil
	}
	if err := appendFrames(sequence.SlideIn); err != nil {
		return err
	}
	if len(handler) == 0 {
		return errors.New("native command9 AI slide-in unavailable")
	}
	handler[len(handler)-1].holdAfter = nativeDelayTicks(schedule.SlideInHoldMS)
	for _, frames := range [][]battlepresent.NativeCommand9RenderedFrame{sequence.Front, sequence.Target, sequence.Tail, sequence.SlideOut} {
		if err := appendFrames(frames); err != nil {
			return err
		}
	}
	if len(handler) != 119 {
		return errors.New("native command9 AI handler frame count unavailable")
	}
	tailPixels, err := nativeCommand1TailPixels(targetBases[figani.NativeCommand9AIDamageStages], background, platform, actorEffect, targetIdle, sequence.NextIdleFrame, sequence.NextIdleRepeat, g.nativeMapAssets.LUTs)
	if err != nil {
		return err
	}
	commonTail, err := nativeCommand0Images(tailPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	g.nativeCmd9AIPresentation = &nativeCommand9AIPresentationJob{actor: actor, plan: plan, prelude: preludeImages, actorBlack: actorBlack, actorPulse: actorPulse, actorSpecs: actorSpecs, handler: handler, tail: commonTail, actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP, then: then}
	return nil
}

func (g *Game) failNativeCommand9AIPresentation(err error) {
	j := g.nativeCmd9AIPresentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	if len(j.plan.Results) == 1 && j.plan.Results[0].Target != nil {
		j.plan.Results[0].Target.HP = j.targetHPBefore
	}
	g.nativeCmd9AIPresentation = nil
	g.loadErr = "native command9 AI presentation: " + err.Error()
}

func (g *Game) stepNativeCommand9AIPresentation() {
	j := g.nativeCmd9AIPresentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand9AICommonPrelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand9AIActor, 0
		}
	case nativeCommand9AIActor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand9AIPresentation(err)
					return
				}
				j.mpPublished = true
				g.playRaw(g.sfxCommand8Actor)
			}
			j.pulseBlack = true
			return
		}
		if spec.PublishMP && !j.mpPublished {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand9AIPresentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand8Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand9AIPresentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand9AIHandler, 0
		}
	case nativeCommand9AIHandler:
		if j.hold > 0 {
			j.hold--
			if j.hold == 0 {
				j.frame++
				if j.frame >= len(j.handler) {
					j.phase, j.frame = nativeCommand9AICommonTail, 0
				}
			}
			return
		}
		frame := j.handler[j.frame]
		if frame.hpStage != 0 {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, 0, frame.hpStage); err != nil {
				g.failNativeCommand9AIPresentation(err)
				return
			}
		}
		if frame.playSub1 {
			g.playRaw(g.sfxCommand8Sub1)
		}
		if frame.playSub2 {
			g.playRaw(g.sfxCommand8Sub2)
		}
		if frame.holdAfter > 0 {
			j.hold = frame.holdAfter
			return
		}
		j.frame++
		if j.frame >= len(j.handler) {
			j.phase, j.frame = nativeCommand9AICommonTail, 0
		}
	case nativeCommand9AICommonTail:
		j.frame++
		if j.frame < len(j.tail) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand9AIPresentation(err)
			return
		}
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd9AIPresentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand9AIPresentation(screen *ebiten.Image) bool {
	j := g.nativeCmd9AIPresentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand9AICommonPrelude:
		if j.frame >= 0 && j.frame < len(j.prelude) {
			frame = j.prelude[j.frame]
		}
	case nativeCommand9AIActor:
		if j.frame >= 0 && j.frame < len(j.actorBlack) && j.frame < len(j.actorPulse) && j.frame < len(j.actorSpecs) {
			frame = j.actorBlack[j.frame]
			if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
				frame = j.actorPulse[j.frame]
			}
		}
	case nativeCommand9AIHandler:
		if j.frame >= 0 && j.frame < len(j.handler) {
			frame = j.handler[j.frame].image
		}
	case nativeCommand9AICommonTail:
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
