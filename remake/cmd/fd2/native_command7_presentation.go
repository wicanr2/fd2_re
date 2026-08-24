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

type nativeCommand7PresentationPhase uint8

const (
	nativeCommand7CommonPrelude nativeCommand7PresentationPhase = iota
	nativeCommand7Actor
	nativeCommand7Handler
	nativeCommand7CommonTail
)

type nativeCommand7HandlerFrame struct {
	image       *ebiten.Image
	targetIndex int
	hpStage     int
	playHandleA bool
	playHandleB bool
}

type nativeCommand7PresentationJob struct {
	actor                  *battle.Unit
	plan                   *battle.NativeCommandDamagePlan
	prelude                []*ebiten.Image
	actorBlack, actorPulse []*ebiten.Image
	actorSpecs             []battlepresent.NativeCommand0ActorFrame
	handler                []nativeCommand7HandlerFrame
	tail                   []*ebiten.Image
	phase                  nativeCommand7PresentationPhase
	frame                  int
	pulseBlack             bool
	drawn, mpPublished     bool
	actorMPBefore          int
	actorActedBefore       bool
	targetHPBefore         []int
	then                   func([]battle.NativeCommandDamageResult)
}

func (g *Game) startNativeCommand7Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command7 presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command7 raw actor provenance unavailable")
	}
	plan, err := g.st.PlanNativeCommandDamage(actor, confirmed, 7, g.st.NativeCommandResistances, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Results) == 0 || plan.DamageStages != figani.NativeCommand7DamageStages ||
		len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[7].EffectMode != 2 {
		return errors.New("native command7 final targets unavailable")
	}
	for _, result := range plan.Results {
		if result.Target == nil || !result.Target.HasBattleFig || !result.Target.HasNativeRecordByte6 {
			return errors.New("native command7 raw target provenance unavailable")
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
	figaniPath, bgPath, taiPath := nativeFIGANIPath(), nativeBGPath(), nativeTAIPath()
	fdotherPath, fdtxtPath := nativeFDOTHERPath(), nativeFDTXTPath()
	if figaniPath == "" || bgPath == "" || taiPath == "" || fdotherPath == "" || fdtxtPath == "" {
		return errors.New("native command7 player-provided archives unavailable")
	}
	background, err := fdother.DecodeArchiveSingleFrame(bgPath, int(bgSelector))
	if err != nil {
		return err
	}
	platform, err := fdother.DecodeArchiveSingleFrame(taiPath, int(taiSelector))
	if err != nil {
		return err
	}
	actorIdle, err := figani.DecodeResource(figaniPath, actor.BattleFig*3)
	if err != nil {
		return err
	}
	actorEffect, err := nativeCommand6ActorEffect(figaniPath, actor.BattleFig)
	if err != nil {
		return err
	}
	targetIdle := make([]*figani.Animation, len(plan.Results))
	for index, result := range plan.Results {
		targetIdle[index], err = figani.DecodeResource(figaniPath, result.Target.BattleFig*3)
		if err != nil {
			return err
		}
	}
	effectResource := 37
	if actor.NativeRecordByte6 == 0 {
		effectResource = 38
	}
	effect, err := figani.DecodeResource(fdotherPath, effectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand7PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	for sample := 0; sample <= 1; sample++ {
		if raw, readErr := fdother.ReadNestedResource(fdotherPath, schedule.SoundResource, sample); readErr != nil || len(raw) == 0 {
			return fmt.Errorf("native command7 FDOTHER #88 sub%d unavailable", sample)
		}
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand7Actor) == 0 || len(g.sfxCommand7Target) == 0) {
		return errors.New("native command7 converted #88 samples unavailable")
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
		targetBases[targetIndex] = make([][]byte, figani.NativeCommand7DamageStages+1)
		for stage := 0; stage <= figani.NativeCommand7DamageStages; stage++ {
			staged := *result.Target
			staged.HP = result.HPBefore - (result.HPBefore-result.HPAfter)*stage/figani.NativeCommand7DamageStages
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
		transitionBases[index] = targetBases[index][figani.NativeCommand7DamageStages]
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
		return errors.New("native command7 DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[7][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	effectSequence, err := battlepresent.BuildNativeCommand7EffectSequence(battlepresent.NativeCommand7EffectInput{
		FrontBase: targetBases[0][0], TailBase: targetBases[len(targetBases)-1][figani.NativeCommand7DamageStages],
		TargetBases: targetBases, TransitionBases: transitionBases, ActorEffect: actorEffect,
		TargetIdle: targetIdle, Effect: effect, Schedule: schedule, RawSide: actor.NativeRecordByte6,
	})
	if err != nil {
		return err
	}
	handler := make([]nativeCommand7HandlerFrame, 0)
	appendFrames := func(frames []battlepresent.NativeCommand7RenderedFrame, targetIndex int) error {
		for _, rendered := range frames {
			images, err := nativeCommand0Images([][]byte{rendered.Pixels}, g.nativeUIPalette)
			if err != nil {
				return err
			}
			handler = append(handler, nativeCommand7HandlerFrame{
				image: images[0], targetIndex: targetIndex, hpStage: rendered.HPStage,
				playHandleA: rendered.PlayHandleA, playHandleB: rendered.PlayHandleB,
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
		targetBases[len(targetBases)-1][figani.NativeCommand7DamageStages], background, platform, actorEffect,
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
		return errors.New("native command7 handler frames unavailable")
	}
	g.nativeCmd7Presentation = &nativeCommand7PresentationJob{
		actor: actor, plan: plan, prelude: preludeImages, actorBlack: actorBlack, actorPulse: actorPulse,
		actorSpecs: actorSpecs, handler: handler, tail: commonTail, actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: make([]int, len(plan.Results)), then: then,
	}
	for index, result := range plan.Results {
		g.nativeCmd7Presentation.targetHPBefore[index] = result.Target.HP
	}
	return nil
}

func (g *Game) failNativeCommand7Presentation(err error) {
	j := g.nativeCmd7Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	for index, result := range j.plan.Results {
		if result.Target != nil && index < len(j.targetHPBefore) {
			result.Target.HP = j.targetHPBefore[index]
		}
	}
	g.nativeCmd7Presentation = nil
	g.loadErr = "native command7 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand7Presentation() {
	j := g.nativeCmd7Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand7CommonPrelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand7Actor, 0
		}
	case nativeCommand7Actor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand7Presentation(err)
					return
				}
				j.mpPublished = true
				g.playRaw(g.sfxCommand7Actor)
			}
			j.pulseBlack = true
			return
		}
		if spec.PublishMP && !j.mpPublished {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand7Presentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand7Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand7Presentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand7Handler, 0
		}
	case nativeCommand7Handler:
		frame := j.handler[j.frame]
		if frame.hpStage != 0 {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, frame.targetIndex, frame.hpStage); err != nil {
				g.failNativeCommand7Presentation(err)
				return
			}
		}
		if frame.playHandleA {
			g.playRaw(g.sfxCommand7Target)
		}
		if frame.playHandleB {
			g.playRaw(g.sfxCommand7Target)
		}
		j.frame++
		if j.frame < len(j.handler) {
			return
		}
		j.phase, j.frame = nativeCommand7CommonTail, 0
	case nativeCommand7CommonTail:
		j.frame++
		if j.frame < len(j.tail) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand7Presentation(err)
			return
		}
		// 數值計畫發布自身 RNGAfter；handler 軌道本身沒有額外亂數來源。
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd7Presentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand7Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd7Presentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand7CommonPrelude:
		if j.frame >= 0 && j.frame < len(j.prelude) {
			frame = j.prelude[j.frame]
		}
	case nativeCommand7Actor:
		if j.frame >= 0 && j.frame < len(j.actorBlack) && j.frame < len(j.actorPulse) && j.frame < len(j.actorSpecs) {
			frame = j.actorBlack[j.frame]
			if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
				frame = j.actorPulse[j.frame]
			}
		}
	case nativeCommand7Handler:
		if j.frame >= 0 && j.frame < len(j.handler) {
			frame = j.handler[j.frame].image
		}
	case nativeCommand7CommonTail:
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
