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

type nativeCommand6PresentationPhase uint8

const (
	nativeCommand6CommonPrelude nativeCommand6PresentationPhase = iota
	nativeCommand6Actor
	nativeCommand6Handler
)

type nativeCommand6HandlerFrame struct {
	image       *ebiten.Image
	targetIndex int
	hpStage     int
	playTarget  bool
	playFront   bool
	playTail    bool
}

type nativeCommand6PresentationJob struct {
	actor                  *battle.Unit
	plan                   *battle.NativeCommandDamagePlan
	prelude                []*ebiten.Image
	actorBlack, actorPulse []*ebiten.Image
	actorSpecs             []battlepresent.NativeCommand0ActorFrame
	handler                []nativeCommand6HandlerFrame
	phase                  nativeCommand6PresentationPhase
	frame                  int
	pulseBlack             bool
	drawn, mpPublished     bool
	actorMPBefore          int
	actorActedBefore       bool
	targetHPBefore         []int
	then                   func([]battle.NativeCommandDamageResult)
}

func nativeCommand6ActorEffect(animationRoot string, selector int) (*figani.Animation, error) {
	if selector < 0 || selector > 0xff {
		return nil, errors.New("native command6 actor FIGANI selector unavailable")
	}
	return figani.LoadSeparatedResourceWithZeroHeaderFallback(animationRoot, selector*3+2)
}

func (g *Game) startNativeCommand6Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9AIPresentation != nil || g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command6 presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command6 raw actor provenance unavailable")
	}
	var plan *battle.NativeCommandDamagePlan
	var err error
	if actor.Camp == battle.Enemy {
		plan, err = g.st.PlanNativeAICommandDamage(actor, 6, g.st.NativeCommandResistances, g.nativeRNGState)
	} else {
		plan, err = g.st.PlanNativeCommandDamage(actor, confirmed, 6, g.st.NativeCommandResistances, g.nativeRNGState)
	}
	if err != nil {
		return err
	}
	if len(plan.Results) == 0 || plan.DamageStages != figani.NativeCommand6DamageStages ||
		len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[6].EffectMode != 2 {
		return errors.New("native command6 final targets unavailable")
	}
	for _, result := range plan.Results {
		if result.Target == nil || !result.Target.HasBattleFig || !result.Target.HasNativeRecordByte6 {
			return errors.New("native command6 raw target provenance unavailable")
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
	effectResource := 32
	if actor.NativeRecordByte6 == 0 {
		effectResource = 33
	}
	effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand6PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	if err := g.requireSeparatedCommandSounds(schedule.SoundResource, 0, 1, 2, 3); err != nil {
		return fmt.Errorf("native command6 sounds: %w", err)
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand6Actor) == 0 || len(g.sfxCommand6Target) == 0 || len(g.sfxCommand6Front) == 0 || len(g.sfxCommand6Tail) == 0) {
		return errors.New("native command6 converted #87 samples unavailable")
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
		targetBases[targetIndex] = make([][]byte, figani.NativeCommand6DamageStages+1)
		for stage := 0; stage <= figani.NativeCommand6DamageStages; stage++ {
			staged := *result.Target
			staged.HP = result.HPBefore - (result.HPBefore-result.HPAfter)*stage/figani.NativeCommand6DamageStages
			record, err := battle.NativeBattlePanelRecordForUnit(&staged)
			if err != nil {
				return err
			}
			targetBases[targetIndex][stage], err = g.localizedNativeCommand0Base(background, panelAssets, actorAfterRecord, record, actorIndex, runtimeIndex, g.handlerChapter, &platform)
			if err != nil {
				return err
			}
		}
	}
	transitionBases := make([][]byte, len(plan.Results)-1)
	for index := range transitionBases {
		transitionBases[index] = targetBases[index][figani.NativeCommand6DamageStages]
	}
	preludeTargetRecord, err := battle.NativeBattlePanelRecordForUnit(plan.Results[0].Target)
	if err != nil {
		return err
	}
	firstTargetIndex, err := nativeCommand24RuntimeUnitIndex(g.st, plan.Results[0].Target)
	if err != nil {
		return err
	}
	preludeBase, err := g.localizedNativeCommand0Base(background, panelAssets, actorRecord, preludeTargetRecord, actorIndex, firstTargetIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	actorBaseBefore, err := g.localizedNativeCommand0Base(background, panelAssets, actorRecord, preludeTargetRecord, actorIndex, firstTargetIndex, g.handlerChapter, &platform)
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
		return errors.New("native command6 DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[6][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	effectSequence, err := battlepresent.BuildNativeCommand6EffectSequence(battlepresent.NativeCommand6EffectInput{
		FrontBase: targetBases[0][0], TailBase: targetBases[len(targetBases)-1][figani.NativeCommand6DamageStages],
		TargetBases: targetBases, TransitionBases: transitionBases, ActorEffect: actorEffect,
		TargetIdle: targetIdle, Effect: effect, Schedule: schedule, RawSide: actor.NativeRecordByte6,
	})
	if err != nil {
		return err
	}
	handler := make([]nativeCommand6HandlerFrame, 0)
	appendPixels := func(pixels [][]byte, template nativeCommand6HandlerFrame) error {
		images, err := nativeCommand0Images(pixels, g.nativeUIPalette)
		if err != nil {
			return err
		}
		for _, image := range images {
			frame := template
			frame.image = image
			handler = append(handler, frame)
		}
		return nil
	}
	if err := appendPixels(effectSequence.Front, nativeCommand6HandlerFrame{targetIndex: -1, playFront: true}); err != nil {
		return err
	}
	for targetIndex, targetSequence := range effectSequence.Targets {
		images, err := nativeCommand0Images(targetSequence.Frames, g.nativeUIPalette)
		if err != nil {
			return err
		}
		for frameIndex, image := range images {
			handler = append(handler, nativeCommand6HandlerFrame{
				image: image, targetIndex: targetIndex, hpStage: targetSequence.HPStages[frameIndex], playTarget: true,
			})
		}
		if targetIndex < len(effectSequence.Transitions) {
			if err := appendPixels(effectSequence.Transitions[targetIndex], nativeCommand6HandlerFrame{targetIndex: -1, playTarget: true}); err != nil {
				return err
			}
		}
	}
	if err := appendPixels(effectSequence.Tail, nativeCommand6HandlerFrame{targetIndex: -1, playTail: true}); err != nil {
		return err
	}
	if len(handler) == 0 {
		return errors.New("native command6 handler frames unavailable")
	}
	g.nativeCmd6Presentation = &nativeCommand6PresentationJob{
		actor: actor, plan: plan, prelude: preludeImages, actorBlack: actorBlack, actorPulse: actorPulse,
		actorSpecs: actorSpecs, handler: handler, actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: make([]int, len(plan.Results)), then: then,
	}
	for index, result := range plan.Results {
		g.nativeCmd6Presentation.targetHPBefore[index] = result.Target.HP
	}
	return nil
}

func (g *Game) failNativeCommand6Presentation(err error) {
	j := g.nativeCmd6Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	for index, result := range j.plan.Results {
		if result.Target != nil && index < len(j.targetHPBefore) {
			result.Target.HP = j.targetHPBefore[index]
		}
	}
	g.nativeCmd6Presentation = nil
	g.loadErr = "native command6 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand6Presentation() {
	j := g.nativeCmd6Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand6CommonPrelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand6Actor, 0
		}
	case nativeCommand6Actor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand6Presentation(err)
					return
				}
				j.mpPublished = true
				g.playRaw(g.sfxCommand6Actor)
			}
			j.pulseBlack = true
			return
		}
		if spec.PublishMP && !j.mpPublished {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand6Presentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand6Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand6Presentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand6Handler, 0
		}
	case nativeCommand6Handler:
		frame := j.handler[j.frame]
		if frame.hpStage != 0 {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, frame.targetIndex, frame.hpStage); err != nil {
				g.failNativeCommand6Presentation(err)
				return
			}
		}
		if frame.playFront && j.frame == 0 {
			g.playRaw(g.sfxCommand6Front)
		}
		if frame.playTarget {
			g.playRaw(g.sfxCommand6Target)
		}
		if frame.playTail && (j.frame == 0 || !j.handler[j.frame-1].playTail) {
			g.playRaw(g.sfxCommand6Tail)
		}
		j.frame++
		if j.frame < len(j.handler) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand6Presentation(err)
			return
		}
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd6Presentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand6Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd6Presentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand6CommonPrelude:
		if j.frame >= 0 && j.frame < len(j.prelude) {
			frame = j.prelude[j.frame]
		}
	case nativeCommand6Actor:
		if j.frame >= 0 && j.frame < len(j.actorBlack) && j.frame < len(j.actorPulse) && j.frame < len(j.actorSpecs) {
			frame = j.actorBlack[j.frame]
			if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
				frame = j.actorPulse[j.frame]
			}
		}
	case nativeCommand6Handler:
		if j.frame >= 0 && j.frame < len(j.handler) {
			frame = j.handler[j.frame].image
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
