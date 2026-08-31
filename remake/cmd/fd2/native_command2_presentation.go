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

type nativeCommand2PresentationPhase uint8

const (
	nativeCommand2Prelude nativeCommand2PresentationPhase = iota
	nativeCommand2Actor
	nativeCommand2Targets
	nativeCommand2Tail
)

type nativeCommand2TargetFrame struct {
	image       *ebiten.Image
	targetIndex int
	hpStage     int
	playSample1 bool
	playSample2 bool
	playSample3 bool
}

type nativeCommand2PresentationJob struct {
	actor                  *battle.Unit
	plan                   *battle.NativeCommandDamagePlan
	prelude                []*ebiten.Image
	actorBlack, actorPulse []*ebiten.Image
	actorSpecs             []battlepresent.NativeCommand0ActorFrame
	targets                []nativeCommand2TargetFrame
	tail                   []*ebiten.Image
	phase                  nativeCommand2PresentationPhase
	frame                  int
	pulseBlack             bool
	drawn, mpPublished     bool
	actorMPBefore          int
	actorActedBefore       bool
	targetHPBefore         []int
	then                   func([]battle.NativeCommandDamageResult)
}

func nativeCommand2TailPixels(base []byte, background, platform fdother.Frame, actorEffect, targetIdle *figani.Animation, targetFrame, targetRepeat int, luts [][]byte) ([][]byte, error) {
	if len(base) != 320*200 || actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil ||
		len(targetIdle.Frames) == 0 || targetFrame < 0 || targetFrame >= len(targetIdle.Frames) || len(luts) <= 14 {
		return nil, errors.New("native command2 tail inputs unavailable")
	}
	out := make([][]byte, 0, 4)
	for pass := 0; pass < 4; pass++ {
		pixels := append([]byte(nil), base...)
		bg, tai := background, platform
		bg.X, bg.Y, tai.X, tai.Y = 0, 50, 164, 157
		var err error
		if pass < 3 {
			err = bg.BlitLUTAt(pixels, 320, 0, luts[12+pass])
			if err == nil {
				err = tai.BlitLUTAt(pixels, 320, 0, luts[12+pass])
			}
		} else {
			err = bg.Blit(pixels, 320, -1)
			if err == nil {
				err = tai.Blit(pixels, 320, -1)
			}
		}
		if err != nil {
			return nil, err
		}
		if err := actorEffect.Frames[0].BlitAt(pixels, 320); err != nil {
			return nil, err
		}
		if err := targetIdle.Frames[targetFrame].BlitAt(pixels, 320); err != nil {
			return nil, err
		}
		out = append(out, pixels)
		targetRepeat++
		if targetRepeat >= targetIdle.Frames[targetFrame].Delay {
			targetRepeat = 0
			targetFrame = (targetFrame + 1) % len(targetIdle.Frames)
		}
	}
	return out, nil
}

func (g *Game) startNativeCommand2Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9AIPresentation != nil || g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil ||
		g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command2 presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command2 raw actor provenance unavailable")
	}
	var plan *battle.NativeCommandDamagePlan
	var err error
	if actor.Camp == battle.Enemy {
		plan, err = g.st.PlanNativeAICommandDamage(actor, 2, g.st.NativeCommandResistances, g.nativeRNGState)
	} else {
		plan, err = g.st.PlanNativeCommandDamage(actor, confirmed, 2, g.st.NativeCommandResistances, g.nativeRNGState)
	}
	if err != nil {
		return err
	}
	if len(plan.Results) == 0 || plan.DamageStages != figani.NativeCommand2DamageStages || len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount {
		return errors.New("native command2 final targets unavailable")
	}
	for _, result := range plan.Results {
		if result.Target == nil || !result.Target.HasBattleFig || !result.Target.HasNativeRecordByte6 {
			return errors.New("native command2 raw target provenance unavailable")
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
	actorEffect, err := nativeCommand0ActorEffect(separatedAssetPath("animations"), actor.BattleFig)
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
	effectResource := 26
	if actor.NativeRecordByte6 == 0 {
		effectResource = 27
	}
	effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", effectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand2PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	if err := g.requireSeparatedCommandSounds(figani.NativeCommand2SoundResource, 0, 1, 2, 3); err != nil {
		return fmt.Errorf("native command2 sounds: %w", err)
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand2Actor) == 0 || len(g.sfxCommand2Mode2) == 0 || len(g.sfxCommand2Mode5) == 0 || len(g.sfxCommand2Mode6) == 0) {
		return errors.New("native command2 converted #83 samples unavailable")
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
		targetBases[targetIndex] = make([][]byte, figani.NativeCommand2DamageStages+1)
		for stage := 0; stage <= figani.NativeCommand2DamageStages; stage++ {
			staged := *result.Target
			staged.HP = result.HPBefore - (result.HPBefore-result.HPAfter)*stage/figani.NativeCommand2DamageStages
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
	firstTargetRecord, err := battle.NativeBattlePanelRecordForUnit(plan.Results[0].Target)
	if err != nil {
		return err
	}
	firstTargetIndex, err := nativeCommand24RuntimeUnitIndex(g.st, plan.Results[0].Target)
	if err != nil {
		return err
	}
	preludeBase, err := g.localizedNativeCommand0Base(background, panelAssets, actorRecord, firstTargetRecord, actorIndex, firstTargetIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	actorBase, err := g.localizedNativeCommand0Base(background, panelAssets, actorRecord, firstTargetRecord, actorIndex, firstTargetIndex, g.handlerChapter, &platform)
	if err != nil {
		return err
	}
	preludePixels, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: preludeBase, ActorIdle: actorIdle.Frames[0], FirstTargetIdle: targetIdle[0].Frames[0],
		Platform: &platform, RawSide: actor.NativeRecordByte6, Mode: 0, BaselineDAC: dac,
	})
	if err != nil {
		return err
	}
	prelude, err := nativeCommand24PreludeImages(preludePixels)
	if err != nil {
		return err
	}
	actorSpecs, err := battlepresent.BuildNativeCommand0ActorFrames(battlepresent.NativeCommand0ActorInput{
		BaseBefore: actorBase, BaseAfter: targetBases[0][0], ActorEffect: actorEffect,
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
		return errors.New("native command2 DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[1][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	targetFrames := make([]nativeCommand2TargetFrame, 0)
	front, frontEnd, err := figani.BuildNativeCommand2FrontSequence(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	for _, helper := range front {
		pixels, err := battlepresent.ComposeNativeCommand2OrbitFrame(targetBases[0][0], actorEffect, targetIdle[0], effect, helper, actor.NativeRecordByte6)
		if err != nil {
			return err
		}
		images, err := nativeCommand0Images([][]byte{pixels}, g.nativeUIPalette)
		if err != nil {
			return err
		}
		targetFrames = append(targetFrames, nativeCommand2TargetFrame{image: images[0], targetIndex: -1, playSample1: helper.Sample1})
	}
	for targetIndex := range plan.Results {
		sequence, end, err := figani.BuildNativeCommand2TargetSequence()
		if err != nil {
			return err
		}
		for _, helper := range sequence {
			pixels, err := battlepresent.ComposeNativeCommand2TargetFrame(targetBases[targetIndex][helper.HPStage], targetIdle[targetIndex].Frames[0], effect, schedule, actor.NativeRecordByte6, helper.EffectFrame)
			if err != nil {
				return err
			}
			images, err := nativeCommand0Images([][]byte{pixels}, g.nativeUIPalette)
			if err != nil {
				return err
			}
			targetFrames = append(targetFrames, nativeCommand2TargetFrame{image: images[0], targetIndex: targetIndex, hpStage: helper.HPStage, playSample2: helper.Sample2})
		}
		if targetIndex+1 < len(plan.Results) {
			direction := 1
			if actor.NativeRecordByte6 == 0 {
				direction = -1
			}
			for step, helper := range figani.BuildNativeCommand2TransitionSequence(end) {
				idle, offset := targetIdle[targetIndex], direction*35*(step+1)
				if step >= 4 {
					idle, offset = targetIdle[targetIndex+1], direction*35*(8-step)
				}
				pixels, err := battlepresent.ComposeNativeCommand2TransitionFrame(targetBases[targetIndex][figani.NativeCommand2DamageStages], actorEffect, idle, effect, helper, actor.NativeRecordByte6, offset)
				if err != nil {
					return err
				}
				images, err := nativeCommand0Images([][]byte{pixels}, g.nativeUIPalette)
				if err != nil {
					return err
				}
				targetFrames = append(targetFrames, nativeCommand2TargetFrame{image: images[0], targetIndex: -1, playSample2: helper.Sample2})
			}
		}
	}
	tailSequence, err := figani.BuildNativeCommand2TailSequence(actor.NativeRecordByte6, frontEnd.Repeat, effect)
	if err != nil {
		return err
	}
	for step, helper := range tailSequence {
		pixels, err := battlepresent.ComposeNativeCommand2OrbitFrame(targetBases[len(targetBases)-1][figani.NativeCommand2DamageStages], actorEffect, targetIdle[len(targetIdle)-1], effect, helper, actor.NativeRecordByte6)
		if err != nil {
			return err
		}
		images, err := nativeCommand0Images([][]byte{pixels}, g.nativeUIPalette)
		if err != nil {
			return err
		}
		targetFrames = append(targetFrames, nativeCommand2TargetFrame{image: images[0], targetIndex: -1, playSample1: helper.Sample1, playSample3: step == 0})
	}
	tailPixels, err := nativeCommand2TailPixels(targetBases[len(targetBases)-1][figani.NativeCommand2DamageStages], background, platform, actorEffect,
		targetIdle[len(targetIdle)-1], 0, 0, g.nativeMapAssets.LUTs)
	if err != nil {
		return err
	}
	tail, err := nativeCommand0Images(tailPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	g.nativeCmd2Presentation = &nativeCommand2PresentationJob{
		actor: actor, plan: plan, prelude: prelude, actorBlack: actorBlack, actorPulse: actorPulse,
		actorSpecs: actorSpecs, targets: targetFrames, tail: tail, actorMPBefore: actor.MP,
		actorActedBefore: actor.Acted, targetHPBefore: make([]int, len(plan.Results)), then: then,
	}
	for index, result := range plan.Results {
		g.nativeCmd2Presentation.targetHPBefore[index] = result.Target.HP
	}
	return nil
}

func (g *Game) failNativeCommand2Presentation(err error) {
	j := g.nativeCmd2Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	for index, result := range j.plan.Results {
		if result.Target != nil && index < len(j.targetHPBefore) {
			result.Target.HP = j.targetHPBefore[index]
		}
	}
	g.nativeCmd2Presentation = nil
	g.loadErr = "native command2 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand2Presentation() {
	j := g.nativeCmd2Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand2Prelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand2Actor, 0
		}
	case nativeCommand2Actor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand2Presentation(err)
					return
				}
				j.mpPublished = true
				g.playRaw(g.sfxCommand2Actor)
			}
			j.pulseBlack = true
			return
		}
		if spec.PublishMP && !j.mpPublished {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand2Presentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand2Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand2Presentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand2Targets, 0
		}
	case nativeCommand2Targets:
		frame := j.targets[j.frame]
		if frame.hpStage != 0 {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, frame.targetIndex, frame.hpStage); err != nil {
				g.failNativeCommand2Presentation(err)
				return
			}
		}
		if frame.playSample1 {
			g.playRaw(g.sfxCommand2Mode2)
		}
		if frame.playSample2 {
			g.playRaw(g.sfxCommand2Mode5)
		}
		if frame.playSample3 {
			g.playRaw(g.sfxCommand2Mode6)
		}
		j.frame++
		if j.frame >= len(j.targets) {
			j.phase, j.frame = nativeCommand2Tail, 0
		}
	case nativeCommand2Tail:
		j.frame++
		if j.frame < len(j.tail) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand2Presentation(err)
			return
		}
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd2Presentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand2Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd2Presentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand2Prelude:
		if j.frame >= 0 && j.frame < len(j.prelude) {
			frame = j.prelude[j.frame]
		}
	case nativeCommand2Actor:
		if j.frame >= 0 && j.frame < len(j.actorBlack) && j.frame < len(j.actorPulse) && j.frame < len(j.actorSpecs) {
			frame = j.actorBlack[j.frame]
			if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
				frame = j.actorPulse[j.frame]
			}
		}
	case nativeCommand2Targets:
		if j.frame >= 0 && j.frame < len(j.targets) {
			frame = j.targets[j.frame].image
		}
	case nativeCommand2Tail:
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
