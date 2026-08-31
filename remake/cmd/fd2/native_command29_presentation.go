package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type nativeCommand29TargetPresentation struct {
	unit                  *battle.Unit
	baseBefore, baseAfter *ebiten.Image
	idleFrames            []*ebiten.Image
	idlePositions         [][2]int
	idleDelays            []int
	transitionFrames      []*ebiten.Image
	hpBefore              int
	rawByte5Before        byte
	damagePublished       bool
}

type nativeCommand29PresentationJob struct {
	actor                           *battle.Unit
	plan                            *battle.NativeCommandDerivedStrikePlan
	schedule                        figani.NativeCommandDerivedStrikeSchedule
	effectFrames                    []*ebiten.Image
	effectPositions                 [][2]int
	preludeFrames                   []*ebiten.Image
	preludeFrame                    int
	actorBaseBefore, actorBaseAfter *ebiten.Image
	targets                         []nativeCommand29TargetPresentation
	targetIndex                     int
	transitionFrame                 int
	frame, repeat                   int
	idleFrame, idleRepeat           int
	shakeCounter                    int
	drawn, mpPublished              bool
	actorMPBefore                   int
	actorActedBefore                bool
	sfxActor, sfxTarget             []byte
	then                            func([]battle.NativeCommand24Damage)
}

func nativeCommand29FinalIdleFrame(delays []int, schedule figani.NativeCommandDerivedStrikeSchedule) (int, error) {
	if len(delays) == 0 || schedule.TargetStart < 0 || schedule.TargetStart >= len(schedule.FrameDelays) {
		return 0, errors.New("native command29 idle schedule unavailable")
	}
	frame, repeat, last := 0, 0, 0
	for effect := schedule.TargetStart; effect < len(schedule.FrameDelays); effect++ {
		for tick := 0; tick < schedule.FrameDelays[effect]; tick++ {
			last = frame
			repeat++
			if repeat >= delays[frame] {
				repeat = 0
				frame = (frame + 1) % len(delays)
			}
		}
	}
	return last, nil
}

func (g *Game) startNativeCommand29Presentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommand24Damage)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || g.rng == nil || actor == nil || confirmed == nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd24Presentation != nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command29 presentation context unavailable")
	}
	if !actor.HasBattleFig || actor.BattleFig != 34 || len(g.nativeUIPalette) < 256 {
		return errors.New("native command29 requires proven selector34 actor and indexed palette")
	}
	animationRoot := separatedAssetPath("animations")
	effect, err := figani.LoadSeparatedResource(animationRoot, actor.BattleFig*3+2)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommandDerivedStrikeSchedule(effect, 29)
	if err != nil {
		return err
	}
	if schedule.CommandID != 29 || schedule.AudioResource != 50 || schedule.ActorSample != 1 ||
		schedule.TargetSample != 4 || schedule.PreludeMode != 1 || !schedule.UsesBGTransition || schedule.UsesTargetBase {
		return errors.New("native command29 resource104 presentation signature mismatch")
	}
	if err := g.requireSeparatedCommandSounds(schedule.AudioResource, schedule.ActorSample, schedule.TargetSample); err != nil {
		return err
	}
	actorIdle, err := figani.LoadSeparatedResource(animationRoot, actor.BattleFig*3)
	if err != nil || len(actorIdle.Frames) == 0 {
		return errors.New("native command29 actor idle FIGANI unavailable")
	}
	plan, err := g.st.PlanNativeCommandDerivedStrike(actor, confirmed, 29, g.rng)
	if err != nil {
		return err
	}
	if len(plan.Results) == 0 {
		return errors.New("native command29 final target list is empty")
	}
	var bgLayers [3]fdother.Frame
	for i := range bgLayers {
		bgLayers[i], err = fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", i)
		if err != nil {
			return err
		}
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	paletteDAC, _, err := loadNativeBattlePalette()
	if err != nil {
		return err
	}
	actorSelector, err := nativeCommand24BGSelector(g.m, actor)
	if err != nil {
		return err
	}
	actorBG, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", actorSelector)
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	actorRecordBefore, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		return err
	}
	actorAfter := *actor
	actorAfter.MP = plan.MPAfter
	actorRecordAfter, err := battle.NativeBattlePanelRecordForUnit(&actorAfter)
	if err != nil {
		return err
	}
	var platform *fdother.Frame
	if actor.NativeRecordByte6 != 0 {
		frame, decodeErr := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", actorSelector)
		if decodeErr != nil {
			return decodeErr
		}
		platform = &frame
	}
	actorBaseBefore, err := g.localizedNativeCommand24BackgroundBase(actorBG, panelAssets, actorRecordBefore, actorIndex, g.handlerChapter, platform)
	if err != nil {
		return err
	}
	actorBaseAfter, err := g.localizedNativeCommand24BackgroundBase(actorBG, panelAssets, actorRecordAfter, actorIndex, g.handlerChapter, platform)
	if err != nil {
		return err
	}
	preludeBase, err := g.localizedNativeCommand24BackgroundBase(actorBG, panelAssets, actorRecordBefore, actorIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: preludeBase, ActorIdle: actorIdle.Frames[0], Platform: platform,
		RawSide: actor.NativeRecordByte6, Mode: schedule.PreludeMode, BaselineDAC: paletteDAC,
	})
	if err != nil {
		return err
	}
	preludeImages, err := nativeCommand24PreludeImages(prelude)
	if err != nil {
		return err
	}
	effectImages, effectPositions, err := nativeFIGANIImages(effect, g.nativeUIPalette)
	if err != nil {
		return err
	}
	actorBases, err := nativeCommand24IndexedImages([][]byte{actorBaseBefore, actorBaseAfter}, g.nativeUIPalette)
	if err != nil {
		return err
	}
	source := append([]byte(nil), actorBaseAfter...)
	if err := effect.Frames[schedule.TargetStart-1].BlitAt(source, 320); err != nil {
		return err
	}
	targetPresentations := make([]nativeCommand29TargetPresentation, 0, len(plan.Results))
	for resultIndex, result := range plan.Results {
		target := result.Target
		if target == nil || !target.HasBattleFig {
			return fmt.Errorf("native command29 target %d BattleFig unavailable", resultIndex)
		}
		idle, decodeErr := figani.LoadSeparatedResource(animationRoot, target.BattleFig*3)
		if decodeErr != nil || len(idle.Frames) == 0 {
			return fmt.Errorf("native command29 target %d idle FIGANI unavailable", resultIndex)
		}
		delays := make([]int, len(idle.Frames))
		for i, frame := range idle.Frames {
			if frame.Delay <= 0 {
				return fmt.Errorf("native command29 target %d idle frame %d delay invalid", resultIndex, i)
			}
			delays[i] = frame.Delay
		}
		selector, selectorErr := nativeCommand24BGSelector(g.m, target)
		if selectorErr != nil {
			return selectorErr
		}
		background, decodeErr := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", selector)
		if decodeErr != nil {
			return decodeErr
		}
		unitIndex, indexErr := nativeCommand24RuntimeUnitIndex(g.st, target)
		if indexErr != nil {
			return indexErr
		}
		recordBefore, recordErr := battle.NativeBattlePanelRecordForUnit(target)
		if recordErr != nil {
			return recordErr
		}
		after := *target
		after.HP = result.HPAfter
		recordAfter, recordErr := battle.NativeBattlePanelRecordForUnit(&after)
		if recordErr != nil {
			return recordErr
		}
		baseBefore, baseErr := g.localizedNativeCommand24BackgroundBase(background, panelAssets, recordBefore, unitIndex, g.handlerChapter, nil)
		if baseErr != nil {
			return baseErr
		}
		baseAfter, baseErr := g.localizedNativeCommand24BackgroundBase(background, panelAssets, recordAfter, unitIndex, g.handlerChapter, nil)
		if baseErr != nil {
			return baseErr
		}
		transitions, transitionErr := battlepresent.BuildNativeCommand24BackgroundFrames(battlepresent.NativeCommand24BackgroundInputs{
			Layers: bgLayers, Source: source, Target: baseBefore, TargetIdle: idle.Frames[0],
		})
		if transitionErr != nil {
			return transitionErr
		}
		bases, baseErr := nativeCommand24IndexedImages([][]byte{baseBefore, baseAfter}, g.nativeUIPalette)
		if baseErr != nil {
			return baseErr
		}
		transitionImages, transitionErr := nativeCommand24IndexedImages(transitions, g.nativeUIPalette)
		if transitionErr != nil {
			return transitionErr
		}
		idleImages, idlePositions, idleErr := nativeFIGANIImages(idle, g.nativeUIPalette)
		if idleErr != nil {
			return idleErr
		}
		targetPresentations = append(targetPresentations, nativeCommand29TargetPresentation{
			unit: target, baseBefore: bases[0], baseAfter: bases[1], idleFrames: idleImages,
			idlePositions: idlePositions, idleDelays: delays, transitionFrames: transitionImages,
			hpBefore: target.HP, rawByte5Before: target.NativeRecordByte5,
		})
		lastIdle, idleErr := nativeCommand29FinalIdleFrame(delays, schedule)
		if idleErr != nil {
			return idleErr
		}
		source = append([]byte(nil), baseAfter...)
		if err := idle.Frames[lastIdle].BlitAt(source, 320); err != nil {
			return err
		}
		if err := effect.Frames[len(effect.Frames)-1].BlitAt(source, 320); err != nil {
			return err
		}
	}
	sfxActor := g.separatedCommandSound(schedule.AudioResource, schedule.ActorSample)
	sfxTarget := g.separatedCommandSound(schedule.AudioResource, schedule.TargetSample)
	if !osMuteOrShot(g) && (len(sfxActor) == 0 || len(sfxTarget) == 0) {
		return errors.New("native command29 FDOTHER #50 samples1/4 unavailable")
	}
	g.nativeCmd29Presentation = &nativeCommand29PresentationJob{
		actor: actor, plan: plan, schedule: schedule, effectFrames: effectImages, effectPositions: effectPositions,
		preludeFrames: preludeImages, actorBaseBefore: actorBases[0], actorBaseAfter: actorBases[1],
		targets: targetPresentations, transitionFrame: -1, actorMPBefore: actor.MP,
		actorActedBefore: actor.Acted, sfxActor: sfxActor, sfxTarget: sfxTarget, then: then,
	}
	return nil
}

func (g *Game) failNativeCommand29Presentation(err error) {
	j := g.nativeCmd29Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	for i := range j.targets {
		target := &j.targets[i]
		target.unit.HP = target.hpBefore
		if target.unit.HasNativeRecordByte5 {
			target.unit.NativeRecordByte5 = target.rawByte5Before
		}
	}
	g.nativeCmd29Presentation = nil
	g.loadErr = "native command29 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand29Presentation() {
	j := g.nativeCmd29Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	if j.preludeFrame < len(j.preludeFrames) {
		j.preludeFrame++
		return
	}
	if j.transitionFrame >= 0 {
		j.transitionFrame++
		if j.transitionFrame >= len(j.targets[j.targetIndex].transitionFrames) {
			j.transitionFrame, j.idleFrame, j.idleRepeat = -1, 0, 0
		}
		return
	}
	if j.frame == j.schedule.ActorImpactFrame && !j.mpPublished {
		if err := battle.ApplyNativeCommandDerivedStrikeMP(j.plan); err != nil {
			g.failNativeCommand29Presentation(err)
			return
		}
		j.mpPublished = true
		g.playRaw(j.sfxActor)
	}
	if j.frame >= j.schedule.TargetStart {
		target := &j.targets[j.targetIndex]
		if target.damagePublished && j.shakeCounter >= 0 {
			j.shakeCounter--
		}
		if j.frame == j.schedule.TargetImpactFrame && !target.damagePublished {
			if !j.mpPublished {
				g.failNativeCommand29Presentation(errors.New("target marker preceded actor MP marker"))
				return
			}
			if err := battle.ApplyNativeCommandDerivedStrikeTarget(j.plan, j.targetIndex); err != nil {
				g.failNativeCommand29Presentation(err)
				return
			}
			target.damagePublished = true
			j.shakeCounter = 5
			g.playRaw(j.sfxTarget)
		}
		j.idleRepeat++
		if j.idleRepeat >= target.idleDelays[j.idleFrame] {
			j.idleRepeat = 0
			j.idleFrame = (j.idleFrame + 1) % len(target.idleFrames)
		}
	}
	j.repeat++
	if j.repeat < j.schedule.FrameDelays[j.frame] {
		return
	}
	j.repeat = 0
	j.frame++
	if j.frame == j.schedule.TargetStart {
		j.transitionFrame = 0
		return
	}
	if j.frame < len(j.effectFrames) {
		return
	}
	if !j.targets[j.targetIndex].damagePublished {
		g.failNativeCommand29Presentation(errors.New("required target marker was not presented"))
		return
	}
	if j.targetIndex+1 < len(j.targets) {
		j.targetIndex++
		j.frame, j.repeat, j.idleFrame, j.idleRepeat, j.transitionFrame = j.schedule.TargetStart, 0, 0, 0, 0
		return
	}
	if !j.mpPublished {
		g.failNativeCommand29Presentation(errors.New("required actor marker was not presented"))
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

func (g *Game) drawNativeCommand29Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd29Presentation
	if j == nil || j.frame < 0 || j.frame >= len(j.effectFrames) || j.targetIndex < 0 || j.targetIndex >= len(j.targets) {
		return false
	}
	if j.preludeFrame < len(j.preludeFrames) {
		if j.preludeFrame < 0 || j.preludeFrames[j.preludeFrame] == nil {
			return false
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(j.preludeFrames[j.preludeFrame], op)
		j.drawn = true
		return true
	}
	target := &j.targets[j.targetIndex]
	if j.transitionFrame >= 0 {
		if j.transitionFrame >= len(target.transitionFrames) {
			return false
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		screen.DrawImage(target.transitionFrames[j.transitionFrame], op)
		j.drawn = true
		return true
	}
	var base *ebiten.Image
	if j.frame < j.schedule.TargetStart {
		base = j.actorBaseBefore
		if j.frame >= j.schedule.ActorImpactFrame {
			base = j.actorBaseAfter
		}
	} else {
		base = target.baseBefore
		if target.damagePublished || j.frame >= j.schedule.TargetImpactFrame {
			base = target.baseAfter
		}
	}
	if base == nil {
		return false
	}
	baseOp := &ebiten.DrawImageOptions{}
	baseOp.GeoM.Scale(2, 2)
	screen.DrawImage(base, baseOp)
	if j.frame >= j.schedule.TargetStart {
		if j.idleFrame < 0 || j.idleFrame >= len(target.idleFrames) {
			return false
		}
		shake := 0
		if j.frame == j.schedule.TargetImpactFrame && !target.damagePublished {
			shake = j.schedule.ShakeOffsets[5]
		} else if target.damagePublished && j.shakeCounter >= 0 && j.shakeCounter < len(j.schedule.ShakeOffsets) {
			shake = j.schedule.ShakeOffsets[j.shakeCounter]
		}
		pos := target.idlePositions[j.idleFrame]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64((pos[0]-shake)*2), float64(pos[1]*2))
		screen.DrawImage(target.idleFrames[j.idleFrame], op)
	}
	pos := j.effectPositions[j.frame]
	effectOp := &ebiten.DrawImageOptions{}
	effectOp.GeoM.Scale(2, 2)
	effectOp.GeoM.Translate(float64(pos[0]*2), float64(pos[1]*2))
	screen.DrawImage(j.effectFrames[j.frame], effectOp)
	j.drawn = true
	return true
}
