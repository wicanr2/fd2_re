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

type nativeCommand33PresentationJob struct {
	actor                        *battle.Unit
	plan                         *battle.NativeCompoundCommand33Plan
	frames                       []nativeCompoundPresentedFrame
	frame, repeat                int
	publishAt                    int
	published, drawn, holding    bool
	hold                         int
	rngBefore                    uint16
	baselineWork, baselineVGA    []byte
	postTransactionWork, postVGA []byte
	then                         func(battle.NativeCompoundCommand33Result)
}

func nativeCommand33ClonedState(st *battle.State, result battle.NativeCompoundCommand33Result) (*battle.State, error) {
	if st == nil || len(st.Units) == 0 || len(result.Targets) == 0 {
		return nil, errors.New("native command33 cloned state unavailable")
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
	for index, target := range result.Targets {
		copied := byOriginal[target.Target]
		if copied == nil || target.HPAfter < 0 || target.HPAfter > copied.MaxHP {
			return nil, fmt.Errorf("native command33 cloned target %d unavailable", index)
		}
		copied.HP = target.HPAfter
		copied.NativeTransient = target.TransientAfter
	}
	return &clone, nil
}

func (g *Game) startNativeCommand33Presentation(actor, confirmed *battle.Unit, then func(battle.NativeCompoundCommand33Result)) error {
	if !g.nativeFullPresentationEnabled() || g == nil || g.st == nil || actor == nil || confirmed == nil {
		return errors.New("native command33 presentation context unavailable")
	}
	if g.nativeCmd34Presentation != nil || g.nativeCmd33Presentation != nil || g.nativeCmd32Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil ||
		g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return errors.New("native command33 presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native command33 indexed map state unavailable")
	}
	plan, err := g.st.PlanNativeCompoundCommand33(actor, confirmed, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Result.Targets) == 0 || len(plan.Result.Restore.Results) != len(plan.Result.Targets) {
		return errors.New("native command33 final target array unavailable")
	}

	fdotherArchive := nativeFDOTHERPath()
	bgArchive, fdtxtArchive := nativeBGPath(), nativeFDTXTPath()
	if fdotherArchive == "" || bgArchive == "" || fdtxtArchive == "" {
		return errors.New("native command33 player-provided archives unavailable")
	}
	actorIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
	if err != nil || len(actorIdle.Frames) == 0 {
		return errors.New("native command33 actor idle FIGANI unavailable")
	}
	actorEffect, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3+1)
	if err != nil {
		return err
	}
	firstTarget := plan.Result.Targets[0].Target
	if firstTarget == nil || !firstTarget.HasBattleFig {
		return errors.New("native command33 first target BattleFig unavailable")
	}
	targetIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), firstTarget.BattleFig*3)
	if err != nil || len(targetIdle.Frames) == 0 {
		return errors.New("native command33 target idle FIGANI unavailable")
	}
	commonEffect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", 66)
	if err != nil {
		return err
	}
	commonSchedule, err := figani.BuildNativeCompoundPresentationSchedule(33, commonEffect)
	if err != nil || commonSchedule.TailEnabled || commonSchedule.Sample1Frame != 6 {
		return errors.New("native command33 common schedule unavailable")
	}
	tailSchedule, err := fdother.BuildNativeCommand33TailSchedule()
	if err != nil {
		return err
	}
	a := g.nativeMapAssets
	if len(a.FDOTHER6) < tailSchedule.EffectStart+tailSchedule.EffectFrames ||
		len(a.CommandHealDigits) <= tailSchedule.DigitBias+9 {
		return errors.New("native command33 tail FDOTHER #5/#6 descriptors unavailable")
	}
	paletteDAC, err := fdother.ReadResource(fdotherArchive, 0)
	if err != nil || len(paletteDAC) != 256*3 {
		return errors.New("native command33 battle DAC unavailable")
	}
	battlePalette, err := fdother.VGAPaletteFromDAC(paletteDAC)
	if err != nil {
		return err
	}

	actorSelector, err := nativeCommand24BGSelector(g.m, actor)
	if err != nil {
		return err
	}
	background, err := fdother.DecodeArchiveSingleFrame(bgArchive, actorSelector)
	if err != nil {
		return err
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(fdotherArchive, fdtxtArchive)
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
		taiArchive := nativeTAIPath()
		if taiArchive == "" {
			return errors.New("native command33 TAI.DAT unavailable")
		}
		frame, decodeErr := fdother.DecodeArchiveSingleFrame(taiArchive, actorSelector)
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
		return errors.New("native command33 main frames unavailable")
	}

	view := g.st.NativeMapViewState
	targetUnits := make([]*battle.Unit, len(plan.Result.Targets))
	for index := range plan.Result.Targets {
		targetUnits[index] = plan.Result.Targets[index].Target
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
	tailEffectPixels := make([][]byte, 0, tailSchedule.EffectFrames)
	for frame := 0; frame < tailSchedule.EffectFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, baselineWork, a.FDOTHER6[tailSchedule.EffectStart+frame], effectTargets, view.CameraX, view.CameraY); err != nil {
			return err
		}
		tailEffectPixels = append(tailEffectPixels, vga)
	}
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return err
	}
	maskWork, maskVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeCommandHealMaskFrame(maskWork, maskVGA, baselineWork, a.Units, g.st.NativeMapSelectorCache, effectTargets, view.CameraX, view.CameraY, roster.Cycles.Idle, byte(tailSchedule.MaskIndex)); err != nil {
		return err
	}

	clonedState, err := nativeCommand33ClonedState(g.st, plan.Result)
	if err != nil {
		return err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native command33 post-transaction HUD unavailable")
	}
	postInput, err := buildNativeMapFrameInput(a, g.m, clonedState, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	postWork, postVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeFrame(postWork, postVGA, postInput); err != nil {
		return err
	}
	queue := make([]battle.NativePresentationDigit, 0, len(plan.Result.Targets)*4)
	for index, result := range plan.Result.Restore.Results {
		recordIndex := tailTargets[index].RecordIndex
		inCamera := tailTargets[index].X >= view.CameraX && tailTargets[index].X < view.CameraX+12 &&
			tailTargets[index].Y >= view.CameraY-1 && tailTargets[index].Y <= view.CameraY+7
		queue, err = battle.AppendNativePresentationDigits(queue, result.Actual, tailSchedule.DigitBias, recordIndex, inCamera)
		if err != nil {
			return err
		}
	}
	digitPixels := make([][]byte, 0, tailSchedule.DigitFrames)
	for frame := 0; frame < tailSchedule.DigitFrames; frame++ {
		work, vga := append([]byte(nil), postWork...), append([]byte(nil), postVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, postWork, a.CommandHealDigits, queue, tailTargets, view.CameraX, view.CameraY, tailSchedule.DigitVertical, frame); err != nil {
			return err
		}
		digitPixels = append(digitPixels, vga)
	}
	mapPalette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return err
	}
	tailEffectImages, err := nativeCommand24IndexedImages(tailEffectPixels, mapPalette)
	if err != nil {
		return err
	}
	maskImages, err := nativeCommand24IndexedImages([][]byte{baselineVGA, maskVGA}, mapPalette)
	if err != nil {
		return err
	}
	digitImages, err := nativeCommand24IndexedImages(digitPixels, mapPalette)
	if err != nil {
		return err
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
	frames[mainStart+commonSchedule.Sample1Frame-1].sound = loadWav(assetPath("assets/sfx/battle_92_01.wav"))
	if frames, err = appendNativeCompoundFrames(frames, rampImages, 6); err != nil {
		return err
	}
	effectStart := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, tailEffectImages, tailSchedule.EffectFrameDelayTicks); err != nil {
		return err
	}
	frames[effectStart].sound = loadWav(assetPath("assets/sfx/battle_80_12.wav"))
	maskStart := len(frames)
	for index := 0; index < tailSchedule.MaskPairs*2+1; index++ {
		frames = append(frames, nativeCompoundPresentedFrame{image: maskImages[index%2], delay: 1})
	}
	frames[maskStart].sound = loadWav(assetPath("assets/sfx/battle_80_01.wav"))
	publishAt := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, digitImages, 1); err != nil {
		return err
	}
	if !osMuteOrShot(g) {
		for _, required := range [][]byte{frames[mainStart+commonSchedule.Sample1Frame-1].sound, frames[effectStart].sound, frames[maskStart].sound} {
			if len(required) == 0 {
				return errors.New("native command33 required raw sample unavailable")
			}
		}
	}
	g.nativeCmd33Presentation = &nativeCommand33PresentationJob{
		actor: actor, plan: plan, frames: frames, publishAt: publishAt,
		rngBefore: g.nativeRNGState, baselineWork: baselineWork, baselineVGA: baselineVGA,
		postTransactionWork: postWork, postVGA: postVGA, then: then,
	}
	return nil
}

func (g *Game) cancelNativeCommand33Presentation() {
	j := g.nativeCmd33Presentation
	if j == nil {
		return
	}
	_ = battle.AbortNativeCompoundCommand33(j.plan)
	g.nativeRNGState = j.rngBefore
	g.nativeMapWork = append(g.nativeMapWork[:0], j.baselineWork...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeCmd33Presentation = nil
}

func (g *Game) failNativeCommand33Presentation(err error) {
	g.cancelNativeCommand33Presentation()
	g.loadErr = "native command33 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand33Presentation() {
	j := g.nativeCmd33Presentation
	if j == nil {
		return
	}
	if j.holding {
		if j.hold > 0 {
			j.hold--
			return
		}
		if err := battle.CompleteNativeCompoundCommand33(j.plan); err != nil {
			g.failNativeCommand33Presentation(err)
			return
		}
		then, result := j.then, j.plan.Result
		g.nativeCmd33Presentation = nil
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
	if next == j.publishAt && !j.published {
		if err := battle.PublishNativeCompoundCommand33(j.plan); err != nil {
			g.failNativeCommand33Presentation(err)
			return
		}
		g.nativeRNGState = j.plan.Result.Restore.RNGState
		g.nativeMapWork = append(g.nativeMapWork[:0], j.postTransactionWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], j.postVGA...)
		j.published = true
	}
	j.frame = next
	if j.frame >= len(j.frames) {
		if !j.published {
			g.failNativeCommand33Presentation(errors.New("required restore boundary was not presented"))
			return
		}
		j.frame = len(j.frames) - 1
		j.holding, j.hold = true, nativeDelayTicks(500)
	}
}

func (g *Game) drawNativeCommand33Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd33Presentation
	if j == nil || j.frame < 0 || j.frame >= len(j.frames) || j.frames[j.frame].image == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(j.frames[j.frame].image, op)
	j.drawn = true
	return true
}
