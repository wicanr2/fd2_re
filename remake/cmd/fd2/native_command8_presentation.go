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

type nativeCommand8PresentationPhase uint8

const (
	nativeCommand8CommonPrelude nativeCommand8PresentationPhase = iota
	nativeCommand8Actor
	nativeCommand8Handler
	nativeCommand8CommonTail
)

type nativeCommand8HandlerFrame struct {
	image              *ebiten.Image
	hpStage            int
	playSub1, playSub2 bool
}

type nativeCommand8PresentationJob struct {
	actor                  *battle.Unit
	plan                   *battle.NativeCommandDamagePlan
	prelude                []*ebiten.Image
	actorBlack, actorPulse []*ebiten.Image
	actorSpecs             []battlepresent.NativeCommand0ActorFrame
	handler                []nativeCommand8HandlerFrame
	tail                   []*ebiten.Image
	phase                  nativeCommand8PresentationPhase
	frame                  int
	pulseBlack             bool
	drawn, mpPublished     bool
	actorMPBefore          int
	actorActedBefore       bool
	targetHPBefore         int
	then                   func([]battle.NativeCommandDamageResult)
}

func (g *Game) startNativeCommand8Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil ||
		g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command8 presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command8 raw actor provenance unavailable")
	}
	var plan *battle.NativeCommandDamagePlan
	var err error
	if actor.Camp == battle.Enemy {
		plan, err = g.st.PlanNativeAICommandDamage(actor, 8, g.st.NativeCommandResistances, g.nativeRNGState)
	} else {
		plan, err = g.st.PlanNativeCommandDamage(actor, confirmed, 8, g.st.NativeCommandResistances, g.nativeRNGState)
	}
	if err != nil {
		return err
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != confirmed || plan.DamageStages != figani.NativeCommand8DamageStages ||
		len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[8].EffectMode != 0 {
		return errors.New("native command8 single final target unavailable")
	}
	target := plan.Results[0].Target
	if target == nil || !target.HasBattleFig || !target.HasNativeRecordByte6 {
		return errors.New("native command8 raw target provenance unavailable")
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
	if actor.NativeRecordByte6 == 0 {
		bgSelector, taiSelector = actorSelector, targetSelector
	}
	fdotherPath, fdtxtPath := nativeFDOTHERPath(), nativeFDTXTPath()
	if fdotherPath == "" || fdtxtPath == "" {
		return errors.New("native command8 player-provided archives unavailable")
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
	effectResource := 28
	if actor.NativeRecordByte6 == 0 {
		effectResource = 30
	}
	effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand8PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	for sample := 0; sample <= 2; sample++ {
		if raw, readErr := fdother.ReadNestedResource(fdotherPath, schedule.SoundResource, sample); readErr != nil || len(raw) == 0 {
			return fmt.Errorf("native command8 FDOTHER #90 sub%d unavailable", sample)
		}
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand8Actor) == 0 || len(g.sfxCommand8Sub1) == 0 || len(g.sfxCommand8Sub2) == 0) {
		return errors.New("native command8 converted #90 samples unavailable")
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(fdotherPath, fdtxtPath)
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
	targetBases := make([][]byte, figani.NativeCommand8DamageStages+1)
	for stage := 0; stage <= figani.NativeCommand8DamageStages; stage++ {
		staged := *target
		staged.HP = plan.Results[0].HPBefore - (plan.Results[0].HPBefore-plan.Results[0].HPAfter)*stage/figani.NativeCommand8DamageStages
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
		return errors.New("native command8 DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[8][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	sequence, err := battlepresent.BuildNativeCommand8EffectSequence(battlepresent.NativeCommand8EffectInput{FrontBase: targetBases[0], TailBase: targetBases[figani.NativeCommand8DamageStages], TargetBases: [][][]byte{targetBases}, ActorEffect: actorEffect, TargetIdle: []*figani.Animation{targetIdle}, Effect: effect, Schedule: schedule, RawSide: actor.NativeRecordByte6})
	if err != nil {
		return err
	}
	handler := make([]nativeCommand8HandlerFrame, 0, len(sequence.Front)+len(sequence.Targets[0])+len(sequence.Tail))
	appendFrames := func(frames []battlepresent.NativeCommand8RenderedFrame) error {
		for _, rendered := range frames {
			images, err := nativeCommand0Images([][]byte{rendered.Pixels}, g.nativeUIPalette)
			if err != nil {
				return err
			}
			handler = append(handler, nativeCommand8HandlerFrame{image: images[0], hpStage: rendered.HPStage, playSub1: rendered.PlaySub1, playSub2: rendered.PlaySub2})
		}
		return nil
	}
	if err := appendFrames(sequence.Front); err != nil {
		return err
	}
	if err := appendFrames(sequence.Targets[0]); err != nil {
		return err
	}
	if err := appendFrames(sequence.Tail); err != nil {
		return err
	}
	tailPixels, err := nativeCommand1TailPixels(targetBases[figani.NativeCommand8DamageStages], background, platform, actorEffect, targetIdle, sequence.NextIdleFrame, sequence.NextIdleRepeat, g.nativeMapAssets.LUTs)
	if err != nil {
		return err
	}
	commonTail, err := nativeCommand0Images(tailPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	if len(handler) != figani.NativeCommand8FrontFrames+figani.NativeCommand8TargetFrames+figani.NativeCommand8TailFrames {
		return errors.New("native command8 handler frames unavailable")
	}
	g.nativeCmd8Presentation = &nativeCommand8PresentationJob{actor: actor, plan: plan, prelude: preludeImages, actorBlack: actorBlack, actorPulse: actorPulse, actorSpecs: actorSpecs, handler: handler, tail: commonTail, actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP, then: then}
	return nil
}

func (g *Game) failNativeCommand8Presentation(err error) {
	j := g.nativeCmd8Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	if len(j.plan.Results) == 1 && j.plan.Results[0].Target != nil {
		j.plan.Results[0].Target.HP = j.targetHPBefore
	}
	g.nativeCmd8Presentation = nil
	g.loadErr = "native command8 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand8Presentation() {
	j := g.nativeCmd8Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand8CommonPrelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand8Actor, 0
		}
	case nativeCommand8Actor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand8Presentation(err)
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
				g.failNativeCommand8Presentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand8Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand8Presentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand8Handler, 0
		}
	case nativeCommand8Handler:
		frame := j.handler[j.frame]
		if frame.hpStage != 0 {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, 0, frame.hpStage); err != nil {
				g.failNativeCommand8Presentation(err)
				return
			}
		}
		if frame.playSub1 {
			g.playRaw(g.sfxCommand8Sub1)
		}
		if frame.playSub2 {
			g.playRaw(g.sfxCommand8Sub2)
		}
		j.frame++
		if j.frame >= len(j.handler) {
			j.phase, j.frame = nativeCommand8CommonTail, 0
		}
	case nativeCommand8CommonTail:
		j.frame++
		if j.frame < len(j.tail) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand8Presentation(err)
			return
		}
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd8Presentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand8Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd8Presentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand8CommonPrelude:
		if j.frame >= 0 && j.frame < len(j.prelude) {
			frame = j.prelude[j.frame]
		}
	case nativeCommand8Actor:
		if j.frame >= 0 && j.frame < len(j.actorBlack) && j.frame < len(j.actorPulse) && j.frame < len(j.actorSpecs) {
			frame = j.actorBlack[j.frame]
			if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
				frame = j.actorPulse[j.frame]
			}
		}
	case nativeCommand8Handler:
		if j.frame >= 0 && j.frame < len(j.handler) {
			frame = j.handler[j.frame].image
		}
	case nativeCommand8CommonTail:
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
