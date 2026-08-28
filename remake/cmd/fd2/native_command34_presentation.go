package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeCommand34PresentationJob struct {
	actor                     *battle.Unit
	plan                      *battle.NativeCompoundCommand34Plan
	frames                    []nativeCompoundPresentedFrame
	frame, repeat             int
	publishAt                 [3]int
	stageEnd                  [3]int
	published                 int
	drawn, holding            bool
	hold, holdStage           int
	rngBefore                 uint16
	baselineWork, baselineVGA []byte
	stageWork, stageVGA       [3][]byte
	then                      func(battle.NativeCompoundCommand34Result)
}

func nativeCommand34ClonedState(st *battle.State, states []battle.NativeCommand34TargetState) (*battle.State, error) {
	if st == nil || len(st.Units) == 0 || len(states) == 0 {
		return nil, errors.New("native command34 cloned state unavailable")
	}
	clone := *st
	clone.Units = make([]*battle.Unit, len(st.Units))
	byOriginal := make(map[*battle.Unit]*battle.Unit, len(st.Units))
	for index, unit := range st.Units {
		if unit == nil {
			continue
		}
		copied := *unit
		clone.Units[index] = &copied
		byOriginal[unit] = &copied
	}
	for index, state := range states {
		copied := byOriginal[state.Target]
		if copied == nil {
			return nil, fmt.Errorf("native command34 cloned target %d unavailable", index)
		}
		copied.NativeTransient = state.NativeTransient
		copied.AP, copied.DP, copied.HIT, copied.EV = state.AP, state.DP, state.HIT, state.EV
	}
	return &clone, nil
}

func (g *Game) startNativeCommand34Presentation(actor, confirmed *battle.Unit, then func(battle.NativeCompoundCommand34Result)) error {
	if !g.nativeFullPresentationEnabled() || g == nil || g.st == nil || actor == nil || confirmed == nil {
		return errors.New("native command34 presentation context unavailable")
	}
	if g.nativeCmd35Presentation != nil || g.nativeCmd34Presentation != nil || g.nativeCmd33Presentation != nil || g.nativeCmd32Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil ||
		g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return errors.New("native command34 presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native command34 indexed map state unavailable")
	}
	plan, err := g.st.PlanNativeCompoundCommand34(actor, confirmed, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Result.Stages) != 3 || len(plan.Result.StageStates) != 3 || len(plan.Result.StageStates[0]) == 0 {
		return errors.New("native command34 final target array unavailable")
	}

	fdotherArchive := nativeFDOTHERPath()
	if fdotherArchive == "" {
		return errors.New("native command34 player-provided archives unavailable")
	}
	actorIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
	if err != nil || len(actorIdle.Frames) == 0 {
		return errors.New("native command34 actor idle FIGANI unavailable")
	}
	actorEffect, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3+1)
	if err != nil {
		return err
	}
	firstTarget := plan.Result.StageStates[0][0].Target
	if firstTarget == nil || !firstTarget.HasBattleFig {
		return errors.New("native command34 first target BattleFig unavailable")
	}
	targetIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), firstTarget.BattleFig*3)
	if err != nil || len(targetIdle.Frames) == 0 {
		return errors.New("native command34 target idle FIGANI unavailable")
	}
	commonEffect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", 67)
	if err != nil {
		return err
	}
	commonSchedule, err := figani.BuildNativeCompoundPresentationSchedule(34, commonEffect)
	if err != nil || commonSchedule.TailEnabled || commonSchedule.Sample1Frame != 2 {
		return errors.New("native command34 common schedule unavailable")
	}
	tailSchedule, err := fdother.BuildNativeCommand34TailSchedule()
	if err != nil {
		return err
	}
	a := g.nativeMapAssets
	for _, stage := range tailSchedule {
		if len(a.FDOTHER6) < stage.EffectStart+stage.EffectFrames || len(a.CommandHealDigits) <= stage.DigitBias+9 {
			return errors.New("native command34 tail FDOTHER #5/#6 descriptors unavailable")
		}
	}
	paletteDAC, battlePalette, err := loadNativeBattlePalette()
	if err != nil || len(paletteDAC) != 256*3 {
		return errors.New("native command34 battle DAC unavailable")
	}

	actorSelector, err := nativeCommand24BGSelector(g.m, actor)
	if err != nil {
		return err
	}
	background, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", actorSelector)
	if err != nil {
		return err
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	actorRecord, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
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
	actorBase, err := nativeCommand24BackgroundBase(background, panelAssets, actorRecord, actorIndex, g.handlerChapter, platform)
	if err != nil {
		return err
	}
	preludeBase, err := nativeCommand24BackgroundBase(background, panelAssets, actorRecord, actorIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: preludeBase, ActorIdle: actorIdle.Frames[0], Platform: platform,
		RawSide: actor.NativeRecordByte6, Mode: 1, BaselineDAC: paletteDAC,
	})
	if err != nil {
		return err
	}
	preludeImages, err := nativeCommand24PreludeImages(prelude)
	if err != nil {
		return err
	}
	actorPixels, err := battlepresent.BuildNativeCompoundActorFrames(actorBase, actorEffect, targetIdle, actor.NativeRecordByte6)
	if err != nil {
		return err
	}
	actorImages, err := nativeCommand24IndexedImages(actorPixels, battlePalette)
	if err != nil {
		return err
	}
	common, err := battlepresent.BuildNativeCompoundCommonFrames(actorBase, actorEffect.Frames[0], commonEffect, commonSchedule)
	if err != nil {
		return err
	}
	actorSlide, err := nativeCommand24IndexedImages(common.ActorSlide, battlePalette)
	if err != nil {
		return err
	}
	effectSlide, err := nativeCommand24IndexedImages(common.EffectSlide, battlePalette)
	if err != nil {
		return err
	}
	mainImages, err := nativeCommand24IndexedImages(common.Main, battlePalette)
	if err != nil || len(mainImages) < commonSchedule.Sample1Frame {
		return errors.New("native command34 main frames unavailable")
	}

	view := g.st.NativeMapViewState
	targetUnits := make([]*battle.Unit, len(plan.Result.StageStates[0]))
	for index := range plan.Result.StageStates[0] {
		targetUnits[index] = plan.Result.StageStates[0][index].Target
	}
	tailTargets, err := nativeCommandHealTailTargets(g.st, targetUnits)
	if err != nil {
		return err
	}
	effectTargets := make([]indexedmap.NativeCommandHealTailTarget, 0, len(tailTargets))
	for _, target := range tailTargets {
		if target.X >= view.CameraX-1 && target.X <= view.CameraX+12 && target.Y >= view.CameraY-1 && target.Y <= view.CameraY+8 {
			effectTargets = append(effectTargets, target)
		}
	}
	baselineWork, baselineVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native command34 post-transaction HUD unavailable")
	}
	mapPalette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return err
	}
	var tailEffectImages, maskImages, digitImages [3][]*ebiten.Image
	var stageWork, stageVGA [3][]byte
	stageBaseWork, stageBaseVGA := baselineWork, baselineVGA
	for stageIndex, schedule := range tailSchedule {
		effectPixels := make([][]byte, 0, schedule.EffectFrames)
		for frame := 0; frame < schedule.EffectFrames; frame++ {
			work, vga := append([]byte(nil), stageBaseWork...), append([]byte(nil), stageBaseVGA...)
			if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, stageBaseWork, a.FDOTHER6[schedule.EffectStart+frame], effectTargets, view.CameraX, view.CameraY); err != nil {
				return err
			}
			effectPixels = append(effectPixels, vga)
		}
		maskWork, maskVGA := append([]byte(nil), stageBaseWork...), append([]byte(nil), stageBaseVGA...)
		if err := indexedmap.ComposeNativeCommandHealMaskFrame(maskWork, maskVGA, stageBaseWork, a.Units, g.st.NativeMapSelectorCache, effectTargets, view.CameraX, view.CameraY, roster.Cycles.Idle, byte(schedule.MaskIndex)); err != nil {
			return err
		}
		clonedState, cloneErr := nativeCommand34ClonedState(g.st, plan.Result.StageStates[stageIndex])
		if cloneErr != nil {
			return cloneErr
		}
		postInput, inputErr := buildNativeMapFrameInput(a, g.m, clonedState, nativeMapFrameRuntime{HUD: hud})
		if inputErr != nil {
			return inputErr
		}
		postWork, postVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeFrame(postWork, postVGA, postInput); err != nil {
			return err
		}
		stageWork[stageIndex], stageVGA[stageIndex] = postWork, postVGA
		stageBaseWork, stageBaseVGA = postWork, postVGA
		queue := make([]battle.NativePresentationDigit, 0, len(tailTargets)*4)
		for targetIndex, target := range tailTargets {
			inCamera := target.X >= view.CameraX && target.X < view.CameraX+12 && target.Y >= view.CameraY-1 && target.Y <= view.CameraY+7
			processed, value := false, 0
			if stageIndex < 2 {
				processed, value = plan.Result.Stages[stageIndex].WordSteps[targetIndex].Processed, int(plan.Result.Stages[stageIndex].WordSteps[targetIndex].Delta)
			} else {
				processed, value = plan.Result.Stages[stageIndex].PairSteps[targetIndex].Processed, 0x0f
			}
			if processed {
				queue, err = battle.AppendNativePresentationDigits(queue, value, schedule.DigitBias, target.RecordIndex, inCamera)
				if err != nil {
					return err
				}
			} else if inCamera {
				for slot, glyph := range [...]int{74, 75, 76, 76} {
					queue = append(queue, battle.NativePresentationDigit{PositionCode: [...]int{2, 8, 12, 17}[slot], Target: target.RecordIndex, Digit: glyph})
				}
			}
		}
		digitPixels := make([][]byte, 0, schedule.DigitFrames)
		for frame := 0; frame < schedule.DigitFrames; frame++ {
			work, vga := append([]byte(nil), postWork...), append([]byte(nil), postVGA...)
			if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, postWork, a.CommandHealDigits, queue, tailTargets, view.CameraX, view.CameraY, schedule.DigitVertical, frame); err != nil {
				return err
			}
			digitPixels = append(digitPixels, vga)
		}
		tailEffectImages[stageIndex], err = nativeCommand24IndexedImages(effectPixels, mapPalette)
		if err != nil {
			return err
		}
		maskImages[stageIndex], err = nativeCommand24IndexedImages([][]byte{baselineVGA, maskVGA}, mapPalette)
		if err != nil {
			return err
		}
		digitImages[stageIndex], err = nativeCommand24IndexedImages(digitPixels, mapPalette)
		if err != nil {
			return err
		}
	}
	rampPixels, rampDeltas := make([][]byte, 0, 41), make([]int, 0, 41)
	for delta := 0; delta <= 40; delta++ {
		rampPixels = append(rampPixels, append([]byte(nil), baselineVGA...))
		rampDeltas = append(rampDeltas, delta)
	}
	rampImages, err := nativeCompoundImagesWithDAC(rampPixels, g.nativeMapDAC, rampDeltas, commonSchedule.PaletteRGB)
	if err != nil {
		return err
	}

	frames := make([]nativeCompoundPresentedFrame, 0)
	if frames, err = appendNativeCompoundFrames(frames, preludeImages, 1); err != nil {
		return err
	}
	if frames, err = appendNativeCompoundFrames(frames, actorImages, 1); err != nil {
		return err
	}
	frames[len(frames)-1].delay += commonSchedule.PreludeHoldTicks
	if frames, err = appendNativeCompoundFrames(frames, actorSlide, 1); err != nil {
		return err
	}
	if frames, err = appendNativeCompoundFrames(frames, effectSlide, 1); err != nil {
		return err
	}
	mainStart := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, mainImages, commonSchedule.MainDelayTicks); err != nil {
		return err
	}
	frames[mainStart+commonSchedule.Sample1Frame-1].sound = loadWav(assetPath("assets/sfx/battle_93_01.wav"))
	if frames, err = appendNativeCompoundFrames(frames, rampImages, 6); err != nil {
		return err
	}
	var publishAt, stageEnd [3]int
	required := [][]byte{frames[mainStart+commonSchedule.Sample1Frame-1].sound}
	for stageIndex, schedule := range tailSchedule {
		effectStart := len(frames)
		if frames, err = appendNativeCompoundFrames(frames, tailEffectImages[stageIndex], 1); err != nil {
			return err
		}
		sample := loadWav(assetPath(fmt.Sprintf("assets/sfx/battle_80_%02d.wav", schedule.EffectSample)))
		frames[effectStart].sound = sample
		for _, rawFrame := range schedule.ExtraSampleFrameIndices {
			if rawFrame <= 0 || rawFrame >= schedule.EffectFrames {
				return errors.New("native command34 extra sample frame unavailable")
			}
			frames[effectStart+rawFrame].sound = sample
		}
		maskStart := len(frames)
		for index := 0; index < schedule.MaskPairs*2+1; index++ {
			frames = append(frames, nativeCompoundPresentedFrame{image: maskImages[stageIndex][index%2], delay: 1})
		}
		frames[maskStart].sound = loadWav(assetPath("assets/sfx/battle_80_01.wav"))
		publishAt[stageIndex] = len(frames)
		if frames, err = appendNativeCompoundFrames(frames, digitImages[stageIndex], 1); err != nil {
			return err
		}
		stageEnd[stageIndex] = len(frames)
		required = append(required, frames[effectStart].sound, frames[maskStart].sound)
	}
	if !osMuteOrShot(g) {
		for _, sample := range required {
			if len(sample) == 0 {
				return errors.New("native command34 required raw sample unavailable")
			}
		}
	}
	g.nativeCmd34Presentation = &nativeCommand34PresentationJob{
		actor: actor, plan: plan, frames: frames, publishAt: publishAt, stageEnd: stageEnd,
		rngBefore: g.nativeRNGState, baselineWork: baselineWork, baselineVGA: baselineVGA,
		stageWork: stageWork, stageVGA: stageVGA, then: then,
	}
	return nil
}

func (g *Game) cancelNativeCommand34Presentation() {
	j := g.nativeCmd34Presentation
	if j == nil {
		return
	}
	_ = battle.AbortNativeCompoundCommand34(j.plan)
	g.nativeRNGState = j.rngBefore
	g.nativeMapWork = append(g.nativeMapWork[:0], j.baselineWork...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeCmd34Presentation = nil
}

func (g *Game) failNativeCommand34Presentation(err error) {
	g.cancelNativeCommand34Presentation()
	g.loadErr = "native command34 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand34Presentation() {
	j := g.nativeCmd34Presentation
	if j == nil {
		return
	}
	if j.holding {
		if j.hold > 0 {
			j.hold--
			return
		}
		j.holding = false
		if j.holdStage < 2 {
			j.frame = j.stageEnd[j.holdStage]
			return
		}
		if err := battle.CompleteNativeCompoundCommand34(j.plan); err != nil {
			g.failNativeCommand34Presentation(err)
			return
		}
		then, result := j.then, j.plan.Result
		g.nativeCmd34Presentation = nil
		if then != nil {
			then(result)
		}
		return
	}
	if !j.drawn || j.frame < 0 || j.frame >= len(j.frames) {
		return
	}
	j.drawn = false
	frame := &j.frames[j.frame]
	if !frame.soundPlayed && len(frame.sound) > 0 {
		g.playRaw(frame.sound)
		frame.soundPlayed = true
	}
	j.repeat++
	if j.repeat < frame.delay {
		return
	}
	j.repeat = 0
	next := j.frame + 1
	for stage := 0; stage < 3; stage++ {
		if next == j.publishAt[stage] && j.published == stage {
			if err := battle.PublishNativeCompoundCommand34Stage(j.plan, stage); err != nil {
				g.failNativeCommand34Presentation(err)
				return
			}
			g.nativeRNGState = j.plan.Result.Stages[stage].RNGState
			g.nativeMapWork = append(g.nativeMapWork[:0], j.stageWork[stage]...)
			g.nativeMapVGA = append(g.nativeMapVGA[:0], j.stageVGA[stage]...)
			j.published++
		}
	}
	for stage := 0; stage < 3; stage++ {
		if next == j.stageEnd[stage] {
			if j.published != stage+1 {
				g.failNativeCommand34Presentation(errors.New("required modifier boundary was not presented"))
				return
			}
			j.frame = next - 1
			j.holding, j.hold, j.holdStage = true, nativeDelayTicks(500), stage
			return
		}
	}
	j.frame = next
	if j.frame >= len(j.frames) {
		g.failNativeCommand34Presentation(errors.New("command34 frame boundary unavailable"))
	}
}

func (g *Game) drawNativeCommand34Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd34Presentation
	if j == nil || j.frame < 0 || j.frame >= len(j.frames) || j.frames[j.frame].image == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(j.frames[j.frame].image, op)
	j.drawn = true
	return true
}
